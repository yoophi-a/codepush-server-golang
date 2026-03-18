package application

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
	"github.com/yoophi/codepush-server-golang/internal/core/ports"
)

const defaultAccessKeyTTL = int64(60 * 24 * time.Hour / time.Millisecond)

type Service struct {
	accounts    ports.AccountRepository
	accessKeys  ports.AccessKeyRepository
	apps        ports.AppRepository
	deployments ports.DeploymentRepository
	packages    ports.PackageRepository
	metrics     ports.MetricsRepository
}

func NewService(
	accounts ports.AccountRepository,
	accessKeys ports.AccessKeyRepository,
	apps ports.AppRepository,
	deployments ports.DeploymentRepository,
	packages ports.PackageRepository,
	metrics ports.MetricsRepository,
) *Service {
	return &Service{
		accounts:    accounts,
		accessKeys:  accessKeys,
		apps:        apps,
		deployments: deployments,
		packages:    packages,
		metrics:     metrics,
	}
}

func (s *Service) Health(ctx context.Context) error {
	if err := s.accounts.CheckHealth(ctx); err != nil {
		return err
	}
	return s.metrics.CheckHealth(ctx)
}

func (s *Service) Authenticate(ctx context.Context, token string) (domain.Account, error) {
	accountID, err := s.accounts.ResolveAccountIDByAccessKey(ctx, token)
	if err != nil {
		return domain.Account{}, err
	}
	return s.accounts.GetByID(ctx, accountID)
}

func (s *Service) GetAccount(ctx context.Context, accountID string) (domain.Account, error) {
	return s.accounts.GetByID(ctx, accountID)
}

func (s *Service) ListAccessKeys(ctx context.Context, accountID string) ([]domain.AccessKey, error) {
	keys, err := s.accessKeys.List(ctx, accountID)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		keys[i].Name = domain.AccessKeyMask
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].CreatedTime < keys[j].CreatedTime })
	return keys, nil
}

func (s *Service) CreateAccessKey(ctx context.Context, accountID, createdBy string, req domain.AccessKeyRequest) (domain.AccessKey, error) {
	now := time.Now().UnixMilli()
	ttl := req.TTL
	if ttl == 0 {
		ttl = defaultAccessKeyTTL
	}
	if req.Name == "" {
		req.Name = generateToken(accountID, now)
	}
	if req.FriendlyName == "" {
		req.FriendlyName = req.Name
	}
	if createdBy == "" {
		createdBy = "unknown"
	}
	return s.accessKeys.Create(ctx, domain.AccessKey{
		AccountID:    accountID,
		Name:         req.Name,
		FriendlyName: req.FriendlyName,
		Description:  req.Description,
		CreatedBy:    createdBy,
		CreatedTime:  now,
		Expires:      now + ttl,
	}, req.Name)
}

func (s *Service) GetAccessKey(ctx context.Context, accountID, name string) (domain.AccessKey, error) {
	key, err := s.accessKeys.GetByName(ctx, accountID, name)
	if err != nil {
		return domain.AccessKey{}, err
	}
	key.Name = ""
	return key, nil
}

func (s *Service) UpdateAccessKey(ctx context.Context, accountID, name string, req domain.AccessKeyRequest) (domain.AccessKey, error) {
	key, err := s.accessKeys.GetByName(ctx, accountID, name)
	if err != nil {
		return domain.AccessKey{}, err
	}
	if req.FriendlyName != "" {
		key.FriendlyName = req.FriendlyName
		key.Description = req.FriendlyName
	}
	if req.Description != "" {
		key.Description = req.Description
	}
	if req.TTL > 0 {
		key.Expires = time.Now().UnixMilli() + req.TTL
	}
	key, err = s.accessKeys.Update(ctx, key)
	if err != nil {
		return domain.AccessKey{}, err
	}
	key.Name = ""
	return key, nil
}

func (s *Service) DeleteAccessKey(ctx context.Context, accountID, name string) error {
	return s.accessKeys.Delete(ctx, accountID, name)
}

func (s *Service) DeleteSessionsByCreator(ctx context.Context, accountID, createdBy string) error {
	n, err := s.accessKeys.DeleteSessionsByCreator(ctx, accountID, createdBy)
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Service) ListApps(ctx context.Context, accountID string) ([]domain.App, error) {
	apps, err := s.apps.List(ctx, accountID)
	if err != nil {
		return nil, err
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps, nil
}

func (s *Service) CreateApp(ctx context.Context, accountID string, req domain.AppCreationRequest) (domain.App, error) {
	app, err := s.apps.Create(ctx, accountID, domain.App{Name: req.Name})
	if err != nil {
		return domain.App{}, err
	}
	if !req.ManuallyProvisionDeployments {
		for _, name := range []string{"Production", "Staging"} {
			if _, err := s.deployments.Create(ctx, accountID, app.ID, domain.Deployment{Name: name, Key: generateToken(accountID, time.Now().UnixNano())}); err != nil {
				return domain.App{}, err
			}
		}
	}
	return s.apps.GetByName(ctx, accountID, app.Name)
}

func (s *Service) GetApp(ctx context.Context, accountID, appName string) (domain.App, error) {
	return s.apps.GetByName(ctx, accountID, appName)
}

func (s *Service) UpdateApp(ctx context.Context, accountID, appName string, req domain.AppPatchRequest) (domain.App, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return domain.App{}, err
	}
	if req.Name != "" {
		app.Name = req.Name
	}
	return s.apps.Update(ctx, accountID, app)
}

func (s *Service) DeleteApp(ctx context.Context, accountID, appName string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	return s.apps.Delete(ctx, accountID, app.ID)
}

func (s *Service) TransferApp(ctx context.Context, accountID, appName, email string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	return s.apps.Transfer(ctx, accountID, app.ID, email)
}

func (s *Service) AddCollaborator(ctx context.Context, accountID, appName, email string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	return s.apps.AddCollaborator(ctx, accountID, app.ID, email)
}

func (s *Service) ListCollaborators(ctx context.Context, accountID, appName string) (map[string]domain.CollaboratorProperties, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return nil, err
	}
	return s.apps.ListCollaborators(ctx, accountID, app.ID)
}

func (s *Service) RemoveCollaborator(ctx context.Context, accountID, appName, email string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	return s.apps.RemoveCollaborator(ctx, accountID, app.ID, email)
}

func (s *Service) ListDeployments(ctx context.Context, accountID, appName string) ([]domain.Deployment, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return nil, err
	}
	deployments, err := s.deployments.List(ctx, accountID, app.ID)
	if err != nil {
		return nil, err
	}
	sort.Slice(deployments, func(i, j int) bool { return deployments[i].Name < deployments[j].Name })
	return deployments, nil
}

func (s *Service) CreateDeployment(ctx context.Context, accountID, appName string, req domain.DeploymentRequest) (domain.Deployment, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return domain.Deployment{}, err
	}
	key := req.Key
	if key == "" {
		key = generateToken(accountID, time.Now().UnixNano())
	}
	return s.deployments.Create(ctx, accountID, app.ID, domain.Deployment{Name: req.Name, Key: key})
}

func (s *Service) GetDeployment(ctx context.Context, accountID, appName, deploymentName string) (domain.Deployment, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return domain.Deployment{}, err
	}
	return s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
}

func (s *Service) UpdateDeployment(ctx context.Context, accountID, appName, deploymentName string, req domain.DeploymentPatchRequest) (domain.Deployment, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return domain.Deployment{}, err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return domain.Deployment{}, err
	}
	if req.Name != "" {
		deployment.Name = req.Name
	}
	return s.deployments.Update(ctx, accountID, app.ID, deployment)
}

func (s *Service) DeleteDeployment(ctx context.Context, accountID, appName, deploymentName string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return err
	}
	return s.deployments.Delete(ctx, accountID, app.ID, deployment.ID)
}

func (s *Service) GetHistory(ctx context.Context, accountID, appName, deploymentName string) ([]domain.Package, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return nil, err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return nil, err
	}
	return s.packages.ListHistory(ctx, accountID, app.ID, deployment.ID)
}

func (s *Service) ClearHistory(ctx context.Context, accountID, appName, deploymentName string) error {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return err
	}
	if err := s.packages.ClearHistory(ctx, accountID, app.ID, deployment.ID); err != nil {
		return err
	}
	return s.metrics.Clear(ctx, deployment.Key)
}

func (s *Service) Rollback(ctx context.Context, accountID, appName, deploymentName, targetLabel string) (domain.Package, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return domain.Package{}, err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return domain.Package{}, err
	}
	history, err := s.packages.ListHistory(ctx, accountID, app.ID, deployment.ID)
	if err != nil {
		return domain.Package{}, err
	}
	if len(history) == 0 {
		return domain.Package{}, domain.ErrNotFound
	}
	var target *domain.Package
	if targetLabel == "" {
		if len(history) < 2 {
			return domain.Package{}, domain.ErrNotFound
		}
		target = &history[len(history)-2]
	} else {
		for i := range history {
			if history[i].Label == targetLabel {
				target = &history[i]
				break
			}
		}
		if target == nil {
			return domain.Package{}, domain.ErrNotFound
		}
	}
	latest := history[len(history)-1]
	if latest.AppVersion != target.AppVersion {
		return domain.Package{}, domain.ErrConflict
	}
	clone := *target
	clone.ID = ""
	clone.ReleaseMethod = domain.ReleaseRollback
	clone.OriginalDeployment = deployment.Name
	clone.OriginalLabel = target.Label
	clone.UploadTime = time.Now().UnixMilli()
	return s.packages.CommitRollback(ctx, accountID, app.ID, deployment.ID, clone)
}

func (s *Service) GetMetrics(ctx context.Context, accountID, appName, deploymentName string) (map[string]domain.UpdateMetrics, error) {
	app, err := s.apps.GetByName(ctx, accountID, appName)
	if err != nil {
		return nil, err
	}
	deployment, err := s.deployments.GetByName(ctx, accountID, app.ID, deploymentName)
	if err != nil {
		return nil, err
	}
	return s.metrics.GetMetrics(ctx, deployment.Key)
}

func (s *Service) UpdateCheck(ctx context.Context, req domain.UpdateCheckRequest) (domain.UpdateCheckResponse, error) {
	deployment, err := s.deployments.GetByKey(ctx, req.DeploymentKey)
	if err != nil {
		return domain.UpdateCheckResponse{}, err
	}
	history, err := s.packages.ListHistory(ctx, "", deployment.AppID, deployment.ID)
	if err != nil {
		return domain.UpdateCheckResponse{}, err
	}
	selected := selectPackage(history, req)
	if selected == nil || selected.PackageHash == req.PackageHash {
		return domain.UpdateCheckResponse{IsAvailable: false}, nil
	}
	return domain.UpdateCheckResponse{
		Label:             selected.Label,
		AppVersion:        selected.AppVersion,
		Description:       selected.Description,
		IsDisabled:        selected.IsDisabled,
		IsMandatory:       selected.IsMandatory,
		PackageHash:       selected.PackageHash,
		DownloadURL:       selected.BlobURL,
		PackageSize:       selected.Size,
		IsAvailable:       true,
		TargetBinaryRange: selected.AppVersion,
	}, nil
}

func (s *Service) ReportDeploy(ctx context.Context, report domain.DeploymentStatusReport) error {
	return s.metrics.ReportDeploy(ctx, report)
}

func (s *Service) ReportDownload(ctx context.Context, report domain.DownloadReport) error {
	return s.metrics.IncrementDownload(ctx, report.DeploymentKey, report.Label)
}

func generateToken(seed string, value int64) string {
	h := sha1.New()
	_, _ = fmt.Fprintf(h, "%s-%d", seed, value)
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func selectPackage(history []domain.Package, req domain.UpdateCheckRequest) *domain.Package {
	if len(history) == 0 {
		return nil
	}
	for i := len(history) - 1; i >= 0; i-- {
		pkg := history[i]
		if pkg.IsDisabled || !matchesVersion(pkg.AppVersion, req.AppVersion) {
			continue
		}
		if pkg.Rollout != nil && *pkg.Rollout < 100 {
			if req.ClientUniqueID == "" || !selectedForRollout(req.ClientUniqueID, *pkg.Rollout, pkg.Label) {
				continue
			}
		}
		return &pkg
	}
	return nil
}

func matchesVersion(target, current string) bool {
	normalizedCurrent := normalizeVersion(current)
	v, err := semver.NewVersion(normalizedCurrent)
	if err != nil {
		return strings.EqualFold(target, current)
	}
	constraint, err := semver.NewConstraint(normalizeConstraint(target))
	if err != nil {
		return strings.EqualFold(target, current)
	}
	return constraint.Check(v)
}

func normalizeVersion(version string) string {
	switch strings.Count(version, ".") {
	case 0:
		return version + ".0.0"
	case 1:
		return version + ".0"
	default:
		return version
	}
}

func normalizeConstraint(value string) string {
	if strings.ContainsAny(value, "<>=*") || strings.Contains(value, "||") {
		return value
	}
	return normalizeVersion(value)
}

func selectedForRollout(clientID string, rollout int, label string) bool {
	hash := sha1.Sum([]byte(clientID + ":" + label))
	value := binary.BigEndian.Uint32(hash[:4]) % 100
	return int(value)+1 <= rollout
}

func HTTPStatus(err error) int {
	switch err {
	case nil:
		return http.StatusOK
	case domain.ErrUnauthorized, domain.ErrExpired:
		return http.StatusUnauthorized
	case domain.ErrForbidden:
		return http.StatusForbidden
	case domain.ErrNotFound:
		return http.StatusNotFound
	case domain.ErrConflict:
		return http.StatusConflict
	case domain.ErrMalformedRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
