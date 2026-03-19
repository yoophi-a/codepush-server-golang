package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	pool    db
	rawPool *pgxpool.Pool
}

type db interface {
	Begin(context.Context) (pgx.Tx, error)
	Close()
	Ping(context.Context) error
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	store := &Store{pool: pool, rawPool: pool}
	if err := store.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.rawPool
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

func (s *Store) Accounts() *AccountRepo       { return &AccountRepo{store: s} }
func (s *Store) AccessKeys() *AccessKeyRepo   { return &AccessKeyRepo{store: s} }
func (s *Store) Apps() *AppRepo               { return &AppRepo{store: s} }
func (s *Store) Deployments() *DeploymentRepo { return &DeploymentRepo{store: s} }
func (s *Store) Packages() *PackageRepo       { return &PackageRepo{store: s} }

type AccountRepo struct{ store *Store }
type AccessKeyRepo struct{ store *Store }
type AppRepo struct{ store *Store }
type DeploymentRepo struct{ store *Store }
type PackageRepo struct{ store *Store }

func (r *AccountRepo) CheckHealth(ctx context.Context) error {
	return r.store.pool.Ping(ctx)
}

func (r *AccountRepo) EnsureBootstrap(ctx context.Context, account domain.Account, key domain.AccessKey) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var accountID string
	err = tx.QueryRow(ctx, `SELECT id FROM accounts WHERE email = $1`, account.Email).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO accounts (email, name, created_at) VALUES ($1, $2, $3) RETURNING id`,
			account.Email, account.Name, time.Now().UnixMilli(),
		).Scan(&accountID)
	}
	if err != nil {
		return err
	}

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM access_keys WHERE token = $1)`, key.Name).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		_, err = tx.Exec(ctx, `
			INSERT INTO access_keys (account_id, token, friendly_name, description, created_by, created_at, expires, is_session)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, accountID, key.Name, key.FriendlyName, key.Description, key.CreatedBy, key.CreatedTime, key.Expires, key.IsSession)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *AccountRepo) GetByID(ctx context.Context, id string) (domain.Account, error) {
	var account domain.Account
	err := r.store.pool.QueryRow(ctx, `SELECT id, email, name, created_at FROM accounts WHERE id = $1`, id).
		Scan(&account.ID, &account.Email, &account.Name, &account.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	account.LinkedProviders = []string{}
	return account, nil
}

func (r *AccountRepo) GetByEmail(ctx context.Context, email string) (domain.Account, error) {
	var account domain.Account
	err := r.store.pool.QueryRow(ctx, `SELECT id, email, name, created_at FROM accounts WHERE email = $1`, email).
		Scan(&account.ID, &account.Email, &account.Name, &account.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	account.LinkedProviders = []string{}
	return account, nil
}

func (r *AccountRepo) ResolveAccountIDByAccessKey(ctx context.Context, token string) (string, error) {
	var accountID string
	var expires int64
	err := r.store.pool.QueryRow(ctx, `SELECT account_id, expires FROM access_keys WHERE token = $1`, token).Scan(&accountID, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	if expires < time.Now().UnixMilli() {
		return "", domain.ErrExpired
	}
	return accountID, nil
}

func (r *AccessKeyRepo) List(ctx context.Context, accountID string) ([]domain.AccessKey, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT id, account_id, token, friendly_name, description, created_by, created_at, expires, is_session
		FROM access_keys WHERE account_id = $1 ORDER BY created_at ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.AccessKey
	for rows.Next() {
		key, err := scanAccessKey(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

func (r *AccessKeyRepo) Create(ctx context.Context, key domain.AccessKey, token string) (domain.AccessKey, error) {
	err := r.store.pool.QueryRow(ctx, `
		INSERT INTO access_keys (account_id, token, friendly_name, description, created_by, created_at, expires, is_session)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, account_id, token, friendly_name, description, created_by, created_at, expires, is_session
	`, key.AccountID, token, key.FriendlyName, key.Description, key.CreatedBy, key.CreatedTime, key.Expires, key.IsSession).
		Scan(&key.ID, &key.AccountID, &key.Name, &key.FriendlyName, &key.Description, &key.CreatedBy, &key.CreatedTime, &key.Expires, &key.IsSession)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.AccessKey{}, domain.ErrConflict
		}
		return domain.AccessKey{}, err
	}
	return key, nil
}

func (r *AccessKeyRepo) GetByName(ctx context.Context, accountID, name string) (domain.AccessKey, error) {
	row := r.store.pool.QueryRow(ctx, `
		SELECT id, account_id, token, friendly_name, description, created_by, created_at, expires, is_session
		FROM access_keys
		WHERE account_id = $1 AND (token = $2 OR friendly_name = $2)
	`, accountID, name)
	key, err := scanAccessKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AccessKey{}, domain.ErrNotFound
	}
	return key, err
}

func (r *AccessKeyRepo) Update(ctx context.Context, key domain.AccessKey) (domain.AccessKey, error) {
	command, err := r.store.pool.Exec(ctx, `UPDATE access_keys SET friendly_name = $1, description = $2, expires = $3 WHERE id = $4`,
		key.FriendlyName, key.Description, key.Expires, key.ID)
	if err != nil {
		return domain.AccessKey{}, err
	}
	if command.RowsAffected() == 0 {
		return domain.AccessKey{}, domain.ErrNotFound
	}
	return key, nil
}

func (r *AccessKeyRepo) Delete(ctx context.Context, accountID, name string) error {
	command, err := r.store.pool.Exec(ctx, `DELETE FROM access_keys WHERE account_id = $1 AND (token = $2 OR friendly_name = $2)`, accountID, name)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AccessKeyRepo) DeleteSessionsByCreator(ctx context.Context, accountID, createdBy string) (int64, error) {
	command, err := r.store.pool.Exec(ctx, `DELETE FROM access_keys WHERE account_id = $1 AND created_by = $2 AND is_session = TRUE`, accountID, createdBy)
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}

func (r *AppRepo) List(ctx context.Context, accountID string) ([]domain.App, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT a.id, a.name, a.created_at
		FROM apps a
		JOIN app_collaborators ac ON ac.app_id = a.id
		WHERE ac.account_id = $1
		ORDER BY a.name
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.App
	var appIDs []string
	for rows.Next() {
		var app domain.App
		if err := rows.Scan(&app.ID, &app.Name, &app.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, app)
		appIDs = append(appIDs, app.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	deploymentsByApp, err := r.listDeploymentNamesByApp(ctx, appIDs)
	if err != nil {
		return nil, err
	}
	collaboratorsByApp, err := r.listCollaboratorsByApps(ctx, accountID, appIDs)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Deployments = deploymentsByApp[result[i].ID]
		result[i].Collaborators = collaboratorsByApp[result[i].ID]
		if result[i].Collaborators == nil {
			result[i].Collaborators = map[string]domain.CollaboratorProperties{}
		}
	}
	return result, nil
}

func (r *AppRepo) Create(ctx context.Context, accountID string, app domain.App) (domain.App, error) {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return domain.App{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	unique, err := isUniqueAppName(ctx, tx, accountID, app.Name, "")
	if err != nil {
		return domain.App{}, err
	}
	if !unique {
		return domain.App{}, domain.ErrConflict
	}
	app.CreatedAt = time.Now().UnixMilli()
	if err := tx.QueryRow(ctx, `INSERT INTO apps (name, created_at) VALUES ($1, $2) RETURNING id`, app.Name, app.CreatedAt).Scan(&app.ID); err != nil {
		return domain.App{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_collaborators (app_id, account_id, permission) VALUES ($1, $2, $3)`,
		app.ID, accountID, domain.PermissionOwner); err != nil {
		return domain.App{}, err
	}
	return app, tx.Commit(ctx)
}

func (r *AppRepo) GetByName(ctx context.Context, accountID, appName string) (domain.App, error) {
	name := appName
	ownerEmail := ""
	if parts := strings.SplitN(appName, ":", 2); len(parts) == 2 {
		ownerEmail = parts[0]
		name = parts[1]
	}

	rows, err := r.store.pool.Query(ctx, `
		SELECT a.id, a.name, a.created_at
		FROM apps a
		JOIN app_collaborators ac ON ac.app_id = a.id
		WHERE ac.account_id = $1 AND a.name = $2
	`, accountID, name)
	if err != nil {
		return domain.App{}, err
	}
	defer rows.Close()

	var candidates []domain.App
	for rows.Next() {
		var app domain.App
		if err := rows.Scan(&app.ID, &app.Name, &app.CreatedAt); err != nil {
			return domain.App{}, err
		}
		collabs, err := r.ListCollaborators(ctx, accountID, app.ID)
		if err != nil {
			return domain.App{}, err
		}
		if ownerEmail != "" {
			props, ok := collabs[ownerEmail]
			if !ok || props.Permission != domain.PermissionOwner {
				continue
			}
		}
		deps, err := (&DeploymentRepo{store: r.store}).List(ctx, accountID, app.ID)
		if err != nil {
			return domain.App{}, err
		}
		app.Collaborators = collabs
		for _, dep := range deps {
			app.Deployments = append(app.Deployments, dep.Name)
		}
		candidates = append(candidates, app)
	}
	if len(candidates) == 0 {
		return domain.App{}, domain.ErrNotFound
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	for _, app := range candidates {
		for _, props := range app.Collaborators {
			if props.IsCurrentAccount && props.Permission == domain.PermissionOwner {
				return app, nil
			}
		}
	}
	return domain.App{}, domain.ErrConflict
}

func (r *AppRepo) Update(ctx context.Context, accountID string, app domain.App) (domain.App, error) {
	if err := requirePermission(ctx, r.store.pool, accountID, app.ID, domain.PermissionOwner); err != nil {
		return domain.App{}, err
	}
	unique, err := isUniqueAppName(ctx, r.store.pool, accountID, app.Name, app.ID)
	if err != nil {
		return domain.App{}, err
	}
	if !unique {
		return domain.App{}, domain.ErrConflict
	}
	command, err := r.store.pool.Exec(ctx, `UPDATE apps SET name = $1 WHERE id = $2`, app.Name, app.ID)
	if err != nil {
		return domain.App{}, err
	}
	if command.RowsAffected() == 0 {
		return domain.App{}, domain.ErrNotFound
	}
	return r.GetByName(ctx, accountID, app.Name)
}

func (r *AppRepo) Delete(ctx context.Context, accountID, appID string) error {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	command, err := r.store.pool.Exec(ctx, `DELETE FROM apps WHERE id = $1`, appID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AppRepo) Transfer(ctx context.Context, accountID, appID, email string) error {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	account, err := (&AccountRepo{store: r.store}).GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `UPDATE app_collaborators SET permission = $1 WHERE app_id = $2 AND account_id = $3`,
		domain.PermissionCollaborator, appID, accountID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO app_collaborators (app_id, account_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (app_id, account_id) DO UPDATE SET permission = EXCLUDED.permission
	`, appID, account.ID, domain.PermissionOwner); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AppRepo) AddCollaborator(ctx context.Context, accountID, appID, email string) error {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	account, err := (&AccountRepo{store: r.store}).GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	_, err = r.store.pool.Exec(ctx, `
		INSERT INTO app_collaborators (app_id, account_id, permission)
		VALUES ($1, $2, $3)
		ON CONFLICT (app_id, account_id) DO UPDATE SET permission = EXCLUDED.permission
	`, appID, account.ID, domain.PermissionCollaborator)
	return err
}

func (r *AppRepo) ListCollaborators(ctx context.Context, accountID, appID string) (map[string]domain.CollaboratorProperties, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT a.email, ac.permission, a.id = $2
		FROM app_collaborators ac
		JOIN accounts a ON a.id = ac.account_id
		WHERE ac.app_id = $1
	`, appID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]domain.CollaboratorProperties{}
	for rows.Next() {
		var email string
		var permission string
		var current bool
		if err := rows.Scan(&email, &permission, &current); err != nil {
			return nil, err
		}
		result[email] = domain.CollaboratorProperties{
			IsCurrentAccount: current,
			Permission:       domain.Permission(permission),
		}
	}
	return result, rows.Err()
}

func (r *AppRepo) listDeploymentNamesByApp(ctx context.Context, appIDs []string) (map[string][]string, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT app_id, name
		FROM deployments
		WHERE app_id = ANY($1)
		ORDER BY app_id, name
	`, appIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string, len(appIDs))
	for rows.Next() {
		var appID string
		var name string
		if err := rows.Scan(&appID, &name); err != nil {
			return nil, err
		}
		result[appID] = append(result[appID], name)
	}
	return result, rows.Err()
}

func (r *AppRepo) listCollaboratorsByApps(ctx context.Context, accountID string, appIDs []string) (map[string]map[string]domain.CollaboratorProperties, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT ac.app_id, a.email, ac.permission, a.id = $2
		FROM app_collaborators ac
		JOIN accounts a ON a.id = ac.account_id
		WHERE ac.app_id = ANY($1)
		ORDER BY ac.app_id, a.email
	`, appIDs, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]domain.CollaboratorProperties, len(appIDs))
	for rows.Next() {
		var appID string
		var email string
		var permission string
		var current bool
		if err := rows.Scan(&appID, &email, &permission, &current); err != nil {
			return nil, err
		}
		if result[appID] == nil {
			result[appID] = map[string]domain.CollaboratorProperties{}
		}
		result[appID][email] = domain.CollaboratorProperties{
			IsCurrentAccount: current,
			Permission:       domain.Permission(permission),
		}
	}
	return result, rows.Err()
}

func (r *AppRepo) RemoveCollaborator(ctx context.Context, accountID, appID, email string) error {
	collabs, err := r.ListCollaborators(ctx, accountID, appID)
	if err != nil {
		return err
	}
	if props, ok := collabs[email]; ok && props.IsCurrentAccount {
		if props.Permission != domain.PermissionCollaborator {
			return domain.ErrForbidden
		}
	} else if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	account, err := (&AccountRepo{store: r.store}).GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	command, err := r.store.pool.Exec(ctx, `DELETE FROM app_collaborators WHERE app_id = $1 AND account_id = $2`, appID, account.ID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeploymentRepo) List(ctx context.Context, accountID, appID string) ([]domain.Deployment, error) {
	if accountID != "" {
		if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionCollaborator); err != nil {
			return nil, err
		}
	}
	rows, err := r.store.pool.Query(ctx, `
		SELECT id, app_id, name, deployment_key, created_at
		FROM deployments WHERE app_id = $1 ORDER BY name
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Deployment
	var depIDs []string
	for rows.Next() {
		var dep domain.Deployment
		if err := rows.Scan(&dep.ID, &dep.AppID, &dep.Name, &dep.Key, &dep.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, dep)
		depIDs = append(depIDs, dep.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	packagesByDeployment, err := listCurrentPackagesByDeployment(ctx, r.store.pool, depIDs)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Package = packagesByDeployment[result[i].ID]
	}
	return result, nil
}

func (r *DeploymentRepo) Create(ctx context.Context, accountID, appID string, dep domain.Deployment) (domain.Deployment, error) {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return domain.Deployment{}, err
	}
	dep.CreatedAt = time.Now().UnixMilli()
	err := r.store.pool.QueryRow(ctx, `
		INSERT INTO deployments (app_id, name, deployment_key, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, appID, dep.Name, dep.Key, dep.CreatedAt).Scan(&dep.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Deployment{}, domain.ErrConflict
		}
		return domain.Deployment{}, err
	}
	dep.AppID = appID
	return dep, nil
}

func (r *DeploymentRepo) GetByName(ctx context.Context, accountID, appID, name string) (domain.Deployment, error) {
	if accountID != "" {
		if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionCollaborator); err != nil {
			return domain.Deployment{}, err
		}
	}
	var dep domain.Deployment
	err := r.store.pool.QueryRow(ctx, `
		SELECT id, app_id, name, deployment_key, created_at
		FROM deployments WHERE app_id = $1 AND name = $2
	`, appID, name).Scan(&dep.ID, &dep.AppID, &dep.Name, &dep.Key, &dep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deployment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	dep.Package, _ = currentPackage(ctx, r.store.pool, dep.ID)
	return dep, nil
}

func (r *DeploymentRepo) GetByKey(ctx context.Context, key string) (domain.Deployment, error) {
	var dep domain.Deployment
	err := r.store.pool.QueryRow(ctx, `
		SELECT id, app_id, name, deployment_key, created_at
		FROM deployments WHERE deployment_key = $1
	`, key).Scan(&dep.ID, &dep.AppID, &dep.Name, &dep.Key, &dep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deployment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deployment{}, err
	}
	dep.Package, _ = currentPackage(ctx, r.store.pool, dep.ID)
	return dep, nil
}

func (r *DeploymentRepo) Update(ctx context.Context, accountID, appID string, dep domain.Deployment) (domain.Deployment, error) {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return domain.Deployment{}, err
	}
	command, err := r.store.pool.Exec(ctx, `UPDATE deployments SET name = $1 WHERE id = $2`, dep.Name, dep.ID)
	if err != nil {
		return domain.Deployment{}, err
	}
	if command.RowsAffected() == 0 {
		return domain.Deployment{}, domain.ErrNotFound
	}
	return r.GetByName(ctx, accountID, appID, dep.Name)
}

func (r *DeploymentRepo) Delete(ctx context.Context, accountID, appID, depID string) error {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	command, err := r.store.pool.Exec(ctx, `DELETE FROM deployments WHERE id = $1 AND app_id = $2`, depID, appID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PackageRepo) ListHistory(ctx context.Context, accountID, appID, depID string) ([]domain.Package, error) {
	if accountID != "" {
		if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionCollaborator); err != nil {
			return nil, err
		}
	}
	rows, err := r.store.pool.Query(ctx, `
		SELECT id, deployment_id, ordinal, label, app_version, description, is_disabled, is_mandatory, package_hash,
			blob_url, manifest_blob_url, rollout, size, upload_time, release_method, original_label, original_deployment, released_by
		FROM packages WHERE deployment_id = $1 ORDER BY ordinal ASC
	`, depID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Package
	for rows.Next() {
		pkg, err := scanPackage(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, pkg)
	}
	return result, rows.Err()
}

func (r *PackageRepo) ClearHistory(ctx context.Context, accountID, appID, depID string) error {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionOwner); err != nil {
		return err
	}
	_, err := r.store.pool.Exec(ctx, `DELETE FROM packages WHERE deployment_id = $1`, depID)
	return err
}

func (r *PackageRepo) CommitRollback(ctx context.Context, accountID, appID, depID string, pkg domain.Package) (domain.Package, error) {
	if err := requirePermission(ctx, r.store.pool, accountID, appID, domain.PermissionCollaborator); err != nil {
		return domain.Package{}, err
	}
	nextOrdinal, err := nextPackageOrdinal(ctx, r.store.pool, depID)
	if err != nil {
		return domain.Package{}, err
	}
	pkg.DeploymentID = depID
	pkg.Ordinal = nextOrdinal
	pkg.Label = fmt.Sprintf("v%d", nextOrdinal)
	err = r.store.pool.QueryRow(ctx, `
		INSERT INTO packages (
			deployment_id, ordinal, label, app_version, description, is_disabled, is_mandatory, package_hash,
			blob_url, manifest_blob_url, rollout, size, upload_time, release_method, original_label, original_deployment, released_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id
	`, depID, pkg.Ordinal, pkg.Label, pkg.AppVersion, pkg.Description, pkg.IsDisabled, pkg.IsMandatory, pkg.PackageHash,
		pkg.BlobURL, pkg.ManifestBlobURL, pkg.Rollout, pkg.Size, pkg.UploadTime, pkg.ReleaseMethod, pkg.OriginalLabel, pkg.OriginalDeployment, pkg.ReleasedBy).
		Scan(&pkg.ID)
	if err != nil {
		return domain.Package{}, err
	}
	return pkg, nil
}

func requirePermission(ctx context.Context, q queryRower, accountID, appID string, permission domain.Permission) error {
	var current string
	err := q.QueryRow(ctx, `SELECT permission FROM app_collaborators WHERE app_id = $1 AND account_id = $2`, appID, accountID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if current == string(domain.PermissionOwner) {
		return nil
	}
	if permission == domain.PermissionCollaborator && current == string(domain.PermissionCollaborator) {
		return nil
	}
	return domain.ErrForbidden
}

func isUniqueAppName(ctx context.Context, q queryer, accountID, name, excludeID string) (bool, error) {
	rows, err := q.Query(ctx, `
		SELECT a.id
		FROM apps a
		JOIN app_collaborators ac ON ac.app_id = a.id
		WHERE ac.account_id = $1 AND a.name = $2
	`, accountID, name)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return false, err
		}
		if id != excludeID {
			return false, nil
		}
	}
	return true, rows.Err()
}

func currentPackage(ctx context.Context, q queryRower, depID string) (*domain.Package, error) {
	row := q.QueryRow(ctx, `
		SELECT id, deployment_id, ordinal, label, app_version, description, is_disabled, is_mandatory, package_hash,
			blob_url, manifest_blob_url, rollout, size, upload_time, release_method, original_label, original_deployment, released_by
		FROM packages WHERE deployment_id = $1 ORDER BY ordinal DESC LIMIT 1
	`, depID)
	pkg, err := scanPackage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

func listCurrentPackagesByDeployment(ctx context.Context, q queryer, depIDs []string) (map[string]*domain.Package, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ON (deployment_id)
			id, deployment_id, ordinal, label, app_version, description, is_disabled, is_mandatory, package_hash,
			blob_url, manifest_blob_url, rollout, size, upload_time, release_method, original_label, original_deployment, released_by
		FROM packages
		WHERE deployment_id = ANY($1)
		ORDER BY deployment_id, ordinal DESC
	`, depIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*domain.Package, len(depIDs))
	for rows.Next() {
		pkg, err := scanPackage(rows)
		if err != nil {
			return nil, err
		}
		pkgCopy := pkg
		result[pkg.DeploymentID] = &pkgCopy
	}
	return result, rows.Err()
}

func nextPackageOrdinal(ctx context.Context, q queryRower, depID string) (int, error) {
	var ordinal int
	if err := q.QueryRow(ctx, `SELECT COALESCE(MAX(ordinal), 0) + 1 FROM packages WHERE deployment_id = $1`, depID).Scan(&ordinal); err != nil {
		return 0, err
	}
	return ordinal, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type scanner interface {
	Scan(...any) error
}

func scanAccessKey(row scanner) (domain.AccessKey, error) {
	var key domain.AccessKey
	err := row.Scan(&key.ID, &key.AccountID, &key.Name, &key.FriendlyName, &key.Description, &key.CreatedBy, &key.CreatedTime, &key.Expires, &key.IsSession)
	return key, err
}

func scanPackage(row scanner) (domain.Package, error) {
	var pkg domain.Package
	var rollout *int
	err := row.Scan(&pkg.ID, &pkg.DeploymentID, &pkg.Ordinal, &pkg.Label, &pkg.AppVersion, &pkg.Description, &pkg.IsDisabled,
		&pkg.IsMandatory, &pkg.PackageHash, &pkg.BlobURL, &pkg.ManifestBlobURL, &rollout, &pkg.Size,
		&pkg.UploadTime, &pkg.ReleaseMethod, &pkg.OriginalLabel, &pkg.OriginalDeployment, &pkg.ReleasedBy)
	if rollout != nil {
		pkg.Rollout = rollout
	}
	return pkg, err
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key")
}
