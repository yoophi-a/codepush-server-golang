package ports

import (
	"context"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type HealthChecker interface {
	CheckHealth(context.Context) error
}

type AccountRepository interface {
	HealthChecker
	EnsureBootstrap(context.Context, domain.Account, domain.AccessKey) error
	GetByID(context.Context, string) (domain.Account, error)
	GetByEmail(context.Context, string) (domain.Account, error)
	ResolveAccountIDByAccessKey(context.Context, string) (string, error)
}

type AccessKeyRepository interface {
	List(context.Context, string) ([]domain.AccessKey, error)
	Create(context.Context, domain.AccessKey, string) (domain.AccessKey, error)
	GetByName(context.Context, string, string) (domain.AccessKey, error)
	Update(context.Context, domain.AccessKey) (domain.AccessKey, error)
	Delete(context.Context, string, string) error
	DeleteSessionsByCreator(context.Context, string, string) (int64, error)
}

type AppRepository interface {
	List(context.Context, string) ([]domain.App, error)
	Create(context.Context, string, domain.App) (domain.App, error)
	GetByName(context.Context, string, string) (domain.App, error)
	Update(context.Context, string, domain.App) (domain.App, error)
	Delete(context.Context, string, string) error
	Transfer(context.Context, string, string, string) error
	AddCollaborator(context.Context, string, string, string) error
	ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error)
	RemoveCollaborator(context.Context, string, string, string) error
}

type DeploymentRepository interface {
	List(context.Context, string, string) ([]domain.Deployment, error)
	Create(context.Context, string, string, domain.Deployment) (domain.Deployment, error)
	GetByName(context.Context, string, string, string) (domain.Deployment, error)
	GetByKey(context.Context, string) (domain.Deployment, error)
	Update(context.Context, string, string, domain.Deployment) (domain.Deployment, error)
	Delete(context.Context, string, string, string) error
}

type PackageRepository interface {
	ListHistory(context.Context, string, string, string) ([]domain.Package, error)
	ClearHistory(context.Context, string, string, string) error
	CommitRollback(context.Context, string, string, string, domain.Package) (domain.Package, error)
}

type MetricsRepository interface {
	HealthChecker
	IncrementDownload(context.Context, string, string) error
	ReportDeploy(context.Context, domain.DeploymentStatusReport) error
	GetMetrics(context.Context, string) (map[string]domain.UpdateMetrics, error)
	Clear(context.Context, string) error
}

type BlobStorage interface {
	HealthChecker
	PutObject(context.Context, string, []byte, string) (string, error)
	DeleteObject(context.Context, string) error
}
