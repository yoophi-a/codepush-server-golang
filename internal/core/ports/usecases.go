package ports

import (
	"context"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type AuthService interface {
	Authenticate(context.Context, string) (domain.Account, error)
}

// HTTPAPI is the driving port consumed by the HTTP adapter.
// It represents the use-case surface exposed to inbound transports.
type HTTPAPI interface {
	AuthService
	Health(context.Context) error
	GetAccount(context.Context, string) (domain.Account, error)
	ListAccessKeys(context.Context, string) ([]domain.AccessKey, error)
	CreateAccessKey(context.Context, string, string, domain.AccessKeyRequest) (domain.AccessKey, error)
	GetAccessKey(context.Context, string, string) (domain.AccessKey, error)
	UpdateAccessKey(context.Context, string, string, domain.AccessKeyRequest) (domain.AccessKey, error)
	DeleteAccessKey(context.Context, string, string) error
	DeleteSessionsByCreator(context.Context, string, string) error
	ListApps(context.Context, string) ([]domain.App, error)
	CreateApp(context.Context, string, domain.AppCreationRequest) (domain.App, error)
	GetApp(context.Context, string, string) (domain.App, error)
	UpdateApp(context.Context, string, string, domain.AppPatchRequest) (domain.App, error)
	DeleteApp(context.Context, string, string) error
	TransferApp(context.Context, string, string, string) error
	AddCollaborator(context.Context, string, string, string) error
	ListCollaborators(context.Context, string, string) (map[string]domain.CollaboratorProperties, error)
	RemoveCollaborator(context.Context, string, string, string) error
	ListDeployments(context.Context, string, string) ([]domain.Deployment, error)
	CreateDeployment(context.Context, string, string, domain.DeploymentRequest) (domain.Deployment, error)
	GetDeployment(context.Context, string, string, string) (domain.Deployment, error)
	UpdateDeployment(context.Context, string, string, string, domain.DeploymentPatchRequest) (domain.Deployment, error)
	DeleteDeployment(context.Context, string, string, string) error
	GetHistory(context.Context, string, string, string) ([]domain.Package, error)
	ClearHistory(context.Context, string, string, string) error
	Rollback(context.Context, string, string, string, string) (domain.Package, error)
	GetMetrics(context.Context, string, string, string) (map[string]domain.UpdateMetrics, error)
	UpdateCheck(context.Context, domain.UpdateCheckRequest) (domain.UpdateCheckResponse, error)
	ReportDeploy(context.Context, domain.DeploymentStatusReport) error
	ReportDownload(context.Context, domain.DownloadReport) error
}
