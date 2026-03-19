package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Stack struct {
	Postgres *postgres.PostgresContainer
	Redis    testcontainers.Container
	MinIO    *minio.MinioContainer
}

type Endpoints struct {
	DatabaseURL string
	RedisAddr   string
	MinIOAddr   string
}

var startStackFn = startStack

func StartStack(ctx context.Context) (stack *Stack, endpoints Endpoints, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stack = nil
			endpoints = Endpoints{}
			err = fmt.Errorf("testcontainers unavailable: %v", recovered)
		}
	}()
	return startStackFn(ctx)
}

func startStack(ctx context.Context) (*Stack, Endpoints, error) {
	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("codepush"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, Endpoints{}, err
	}
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	if err != nil {
		pg.Terminate(ctx)
		return nil, Endpoints{}, err
	}

	minioC, err := minio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
	)
	if err != nil {
		redisC.Terminate(ctx)
		pg.Terminate(ctx)
		return nil, Endpoints{}, err
	}

	pgURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, Endpoints{}, err
	}
	redisHost, err := redisC.Host(ctx)
	if err != nil {
		return nil, Endpoints{}, err
	}
	redisPort, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		return nil, Endpoints{}, err
	}
	minioHost, err := minioC.Host(ctx)
	if err != nil {
		return nil, Endpoints{}, err
	}
	minioPort, err := minioC.MappedPort(ctx, "9000")
	if err != nil {
		return nil, Endpoints{}, err
	}

	return &Stack{
			Postgres: pg,
			Redis:    redisC,
			MinIO:    minioC,
		}, Endpoints{
			DatabaseURL: pgURL,
			RedisAddr:   fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
			MinIOAddr:   fmt.Sprintf("%s:%s", minioHost, minioPort.Port()),
		}, nil
}

func (s *Stack) Terminate(ctx context.Context) {
	if s.MinIO != nil {
		_ = s.MinIO.Terminate(ctx)
	}
	if s.Redis != nil {
		_ = s.Redis.Terminate(ctx)
	}
	if s.Postgres != nil {
		_ = s.Postgres.Terminate(ctx)
	}
}
