package redis

import (
	"context"
	"strconv"

	goredis "github.com/redis/go-redis/v9"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

const (
	deploySucceeded = "DeploymentSucceeded"
	deployFailed    = "DeploymentFailed"
)

type Metrics struct {
	client *goredis.Client
}

func New(addr, password string, db int) *Metrics {
	return &Metrics{
		client: goredis.NewClient(&goredis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (m *Metrics) Close() error {
	return m.client.Close()
}

func (m *Metrics) CheckHealth(ctx context.Context) error {
	return m.client.Ping(ctx).Err()
}

func (m *Metrics) IncrementDownload(ctx context.Context, deploymentKey, label string) error {
	pipe := m.client.TxPipeline()
	pipe.HIncrBy(ctx, countersKey(deploymentKey, label), "downloaded", 1)
	pipe.SAdd(ctx, labelsKey(deploymentKey), label)
	_, err := pipe.Exec(ctx)
	return err
}

func (m *Metrics) ReportDeploy(ctx context.Context, report domain.DeploymentStatusReport) error {
	label := report.Label
	if label == "" {
		label = report.AppVersion
	}
	pipe := m.client.TxPipeline()
	pipe.SAdd(ctx, labelsKey(report.DeploymentKey), label)
	switch report.Status {
	case "", deploySucceeded:
		pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "installed", 1)
	case deployFailed:
		pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "failed", 1)
	default:
		pipe.HIncrBy(ctx, countersKey(report.DeploymentKey, label), "installed", 1)
	}
	if report.ClientUniqueID != "" {
		currentKey := activeClientKey(report.DeploymentKey, report.ClientUniqueID)
		prev, _ := m.client.Get(ctx, currentKey).Result()
		if prev != "" && prev != label {
			pipe.SRem(ctx, activeSetKey(report.DeploymentKey, prev), report.ClientUniqueID)
		}
		pipe.Set(ctx, currentKey, label, 0)
		pipe.SAdd(ctx, activeSetKey(report.DeploymentKey, label), report.ClientUniqueID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (m *Metrics) GetMetrics(ctx context.Context, deploymentKey string) (map[string]domain.UpdateMetrics, error) {
	labels, err := m.client.SMembers(ctx, labelsKey(deploymentKey)).Result()
	if err != nil {
		return nil, err
	}
	result := map[string]domain.UpdateMetrics{}
	for _, label := range labels {
		counters, err := m.client.HGetAll(ctx, countersKey(deploymentKey, label)).Result()
		if err != nil {
			return nil, err
		}
		active, err := m.client.SCard(ctx, activeSetKey(deploymentKey, label)).Result()
		if err != nil {
			return nil, err
		}
		metrics := domain.UpdateMetrics{Active: active}
		metrics.Downloaded = parseCounter(counters["downloaded"])
		metrics.Failed = parseCounter(counters["failed"])
		metrics.Installed = parseCounter(counters["installed"])
		result[label] = metrics
	}
	return result, nil
}

func (m *Metrics) Clear(ctx context.Context, deploymentKey string) error {
	labels, err := m.client.SMembers(ctx, labelsKey(deploymentKey)).Result()
	if err != nil {
		return err
	}
	keys := []string{labelsKey(deploymentKey)}
	for _, label := range labels {
		keys = append(keys, countersKey(deploymentKey, label), activeSetKey(deploymentKey, label))
	}
	if len(keys) == 0 {
		return nil
	}
	return m.client.Del(ctx, keys...).Err()
}

func labelsKey(deploymentKey string) string {
	return "metrics:" + deploymentKey + ":labels"
}

func countersKey(deploymentKey, label string) string {
	return "metrics:" + deploymentKey + ":label:" + label + ":counters"
}

func activeSetKey(deploymentKey, label string) string {
	return "metrics:" + deploymentKey + ":label:" + label + ":active"
}

func activeClientKey(deploymentKey, clientID string) string {
	return "metrics:" + deploymentKey + ":client:" + clientID
}

func parseCounter(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
