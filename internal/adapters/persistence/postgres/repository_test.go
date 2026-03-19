package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yoophi/codepush-server-golang/internal/core/domain"
)

type fakeDB struct {
	beginFn func(context.Context) (pgx.Tx, error)
	execFn  func(context.Context, string, ...any) (pgconn.CommandTag, error)
	queryFn func(context.Context, string, ...any) (pgx.Rows, error)
	rowFn   func(context.Context, string, ...any) pgx.Row
	pingErr error
	closed  bool
}

func (f *fakeDB) Begin(ctx context.Context) (pgx.Tx, error) {
	if f.beginFn != nil {
		return f.beginFn(ctx)
	}
	return nil, errors.New("unexpected Begin")
}
func (f *fakeDB) Close()                     { f.closed = true }
func (f *fakeDB) Ping(context.Context) error { return f.pingErr }
func (f *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag(""), nil
}
func (f *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if f.queryFn != nil {
		return f.queryFn(ctx, sql, args...)
	}
	return &fakeRows{}, nil
}
func (f *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if f.rowFn != nil {
		return f.rowFn(ctx, sql, args...)
	}
	return fakeRow{err: pgx.ErrNoRows}
}

type fakeTx struct {
	fakeDB
	commitErr      error
	rollbackErr    error
	commitCalled   bool
	rollbackCalled bool
}

func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (f *fakeTx) Commit(context.Context) error {
	f.commitCalled = true
	return f.commitErr
}
func (f *fakeTx) Rollback(context.Context) error {
	f.rollbackCalled = true
	return f.rollbackErr
}
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int64:
			*d = r.values[i].(int64)
		case *int:
			*d = r.values[i].(int)
		case *bool:
			*d = r.values[i].(bool)
		case **int:
			if r.values[i] == nil {
				*d = nil
			} else {
				v := r.values[i].(int)
				*d = &v
			}
		default:
			panic("unsupported scan type")
		}
	}
	return nil
}

type fakeRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("") }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	return fakeRow{values: row}.Scan(dest...)
}
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

func testStore(db db) *Store {
	return &Store{pool: db}
}

func TestStoreBasics(t *testing.T) {
	db := &fakeDB{}
	store := testStore(db)
	store.Close()
	if !db.closed {
		t.Fatalf("expected Close() to close the db")
	}
	if store.Accounts() == nil || store.AccessKeys() == nil || store.Apps() == nil || store.Deployments() == nil || store.Packages() == nil {
		t.Fatalf("expected repository constructors")
	}
	if store.Pool() != nil {
		t.Fatalf("expected nil raw pool in fake-backed store")
	}
	if _, err := NewStore(context.Background(), "://bad-url"); err == nil {
		t.Fatalf("expected invalid database URL error")
	}
}

func TestAccountRepoMethods(t *testing.T) {
	t.Run("CheckHealth and Migrate", func(t *testing.T) {
		db := &fakeDB{}
		store := testStore(db)
		if err := store.Accounts().CheckHealth(context.Background()); err != nil {
			t.Fatalf("CheckHealth() error = %v", err)
		}
		if err := store.Migrate(context.Background()); err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}
	})

	t.Run("EnsureBootstrap inserts account and access key", func(t *testing.T) {
		tx := &fakeTx{}
		var queryRowCount int
		tx.rowFn = func(_ context.Context, sql string, _ ...any) pgx.Row {
			queryRowCount++
			switch queryRowCount {
			case 1:
				return fakeRow{err: pgx.ErrNoRows}
			case 2:
				return fakeRow{values: []any{"acc-1"}}
			default:
				return fakeRow{values: []any{false}}
			}
		}
		tx.execFn = func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 1"), nil
		}
		store := testStore(&fakeDB{
			beginFn: func(context.Context) (pgx.Tx, error) { return tx, nil },
		})
		err := store.Accounts().EnsureBootstrap(context.Background(), domain.Account{Email: "owner@example.com", Name: "Owner"}, domain.AccessKey{Name: "token"})
		if err != nil {
			t.Fatalf("EnsureBootstrap() error = %v", err)
		}
		if !tx.commitCalled || !tx.rollbackCalled {
			t.Fatalf("expected commit and deferred rollback path to be exercised")
		}
	})

	t.Run("GetByID GetByEmail and ResolveAccessKey", func(t *testing.T) {
		rows := []fakeRow{
			{values: []any{"acc-1", "owner@example.com", "Owner", int64(10)}},
			{values: []any{"acc-1", "owner@example.com", "Owner", int64(10)}},
			{values: []any{"acc-1", int64(time.Now().Add(time.Hour).UnixMilli())}},
			{err: pgx.ErrNoRows},
			{values: []any{"acc-1", int64(time.Now().Add(-time.Hour).UnixMilli())}},
		}
		i := 0
		store := testStore(&fakeDB{
			rowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				row := rows[i]
				i++
				return row
			},
		})
		account, err := store.Accounts().GetByID(context.Background(), "acc-1")
		if err != nil || account.Email != "owner@example.com" || len(account.LinkedProviders) != 0 {
			t.Fatalf("unexpected GetByID() result %#v err=%v", account, err)
		}
		account, err = store.Accounts().GetByEmail(context.Background(), "owner@example.com")
		if err != nil || account.Email != "owner@example.com" {
			t.Fatalf("unexpected GetByEmail() result %#v err=%v", account, err)
		}
		id, err := store.Accounts().ResolveAccountIDByAccessKey(context.Background(), "token")
		if err != nil || id != "acc-1" {
			t.Fatalf("unexpected ResolveAccountIDByAccessKey() result %q err=%v", id, err)
		}
		_, err = store.Accounts().ResolveAccountIDByAccessKey(context.Background(), "missing")
		if !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		_, err = store.Accounts().ResolveAccountIDByAccessKey(context.Background(), "expired")
		if !errors.Is(err, domain.ErrExpired) {
			t.Fatalf("expected ErrExpired, got %v", err)
		}
	})
}

func TestAccessKeyRepoMethods(t *testing.T) {
	t.Run("List GetByName Update Delete", func(t *testing.T) {
		rowValues := [][]any{
			{"id-1", "acc-1", "token-1", "friendly", "desc", "creator", int64(10), int64(20), false},
		}
		rowIdx := 0
		execIdx := 0
		store := testStore(&fakeDB{
			queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
				return &fakeRows{rows: rowValues}, nil
			},
			rowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				row := fakeRow{values: rowValues[rowIdx]}
				rowIdx++
				return row
			},
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				execIdx++
				if execIdx == 1 {
					return pgconn.NewCommandTag("UPDATE 1"), nil
				}
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		})
		got, err := store.AccessKeys().List(context.Background(), "acc-1")
		if err != nil || len(got) != 1 || got[0].Name != "token-1" {
			t.Fatalf("unexpected List() result %#v err=%v", got, err)
		}
		key, err := store.AccessKeys().GetByName(context.Background(), "acc-1", "token-1")
		if err != nil || key.ID != "id-1" {
			t.Fatalf("unexpected GetByName() result %#v err=%v", key, err)
		}
		key, err = store.AccessKeys().Update(context.Background(), domain.AccessKey{ID: "id-1", FriendlyName: "new", Description: "new", Expires: 30})
		if err != nil || key.FriendlyName != "new" {
			t.Fatalf("unexpected Update() result %#v err=%v", key, err)
		}
		if err := store.AccessKeys().Delete(context.Background(), "acc-1", "token-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("Create DeleteSessions and errors", func(t *testing.T) {
		execCount := 0
		store := testStore(&fakeDB{
			rowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				return fakeRow{values: []any{"id-1", "acc-1", "token-1", "friendly", "desc", "creator", int64(10), int64(20), false}}
			},
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				execCount++
				if execCount == 1 {
					return pgconn.NewCommandTag("DELETE 2"), nil
				}
				return pgconn.NewCommandTag("DELETE 0"), nil
			},
		})
		key, err := store.AccessKeys().Create(context.Background(), domain.AccessKey{AccountID: "acc-1"}, "token-1")
		if err != nil || key.ID != "id-1" {
			t.Fatalf("unexpected Create() result %#v err=%v", key, err)
		}
		rows, err := store.AccessKeys().DeleteSessionsByCreator(context.Background(), "acc-1", "creator")
		if err != nil || rows != 2 {
			t.Fatalf("unexpected DeleteSessionsByCreator() result %d err=%v", rows, err)
		}
		if err := (&AccessKeyRepo{store: testStore(&fakeDB{
			execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 0"), nil
			},
		})}).Delete(context.Background(), "acc-1", "missing"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestAppDeploymentAndPackageHelpers(t *testing.T) {
	t.Run("permission and helper functions", func(t *testing.T) {
		db := &fakeDB{
			rowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				return fakeRow{values: []any{string(domain.PermissionOwner)}}
			},
			queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
				return &fakeRows{rows: [][]any{{"app-1"}}}, nil
			},
		}
		if err := requirePermission(context.Background(), db, "acc-1", "app-1", domain.PermissionOwner); err != nil {
			t.Fatalf("requirePermission() error = %v", err)
		}
		unique, err := isUniqueAppName(context.Background(), db, "acc-1", "demo", "app-1")
		if err != nil || !unique {
			t.Fatalf("expected name to be unique with excluded ID, got unique=%v err=%v", unique, err)
		}
	})

	t.Run("list collaborators deployments history and scanners", func(t *testing.T) {
		queryCount := 0
		store := testStore(&fakeDB{
			queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
				queryCount++
				switch queryCount {
				case 1:
					return &fakeRows{rows: [][]any{{"owner@example.com", string(domain.PermissionOwner), true}}}, nil
				case 2:
					return &fakeRows{rows: [][]any{{"dep-1", "app-1", "Production", "dep-key", int64(10)}}}, nil
				default:
					return &fakeRows{rows: [][]any{{"pkg-1", "dep-1", 1, "v1", "1.0.0", "desc", false, false, "hash", "blob", "manifest", nil, int64(1), int64(2), "Rollback", "v0", "Production", "user"}}}, nil
				}
			},
			rowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
				switch {
				case sql == `SELECT permission FROM app_collaborators WHERE app_id = $1 AND account_id = $2`:
					return fakeRow{values: []any{string(domain.PermissionOwner)}}
				case sql == `
			SELECT id, deployment_id, ordinal, label, app_version, description, is_disabled, is_mandatory, package_hash,
				blob_url, manifest_blob_url, rollout, size, upload_time, release_method, original_label, original_deployment, released_by
			FROM packages WHERE deployment_id = $1 ORDER BY ordinal DESC LIMIT 1
		`:
					return fakeRow{err: pgx.ErrNoRows}
				case sql == `SELECT COALESCE(MAX(ordinal), 0) + 1 FROM packages WHERE deployment_id = $1`:
					return fakeRow{values: []any{2}}
				default:
					return fakeRow{values: []any{"pkg-1", "dep-1", 1, "v1", "1.0.0", "desc", false, false, "hash", "blob", "manifest", nil, int64(1), int64(2), "Rollback", "v0", "Production", "user"}}
				}
			},
		})

		collabs, err := store.Apps().ListCollaborators(context.Background(), "acc-1", "app-1")
		if err != nil || collabs["owner@example.com"].Permission != domain.PermissionOwner {
			t.Fatalf("unexpected ListCollaborators() result %#v err=%v", collabs, err)
		}
		deps, err := store.Deployments().List(context.Background(), "acc-1", "app-1")
		if err != nil || len(deps) != 1 || deps[0].Name != "Production" {
			t.Fatalf("unexpected List() result %#v err=%v", deps, err)
		}
		history, err := store.Packages().ListHistory(context.Background(), "acc-1", "app-1", "dep-1")
		if err != nil || len(history) != 1 || history[0].Label != "v1" {
			t.Fatalf("unexpected ListHistory() result %#v err=%v", history, err)
		}
		next, err := nextPackageOrdinal(context.Background(), store.pool, "dep-1")
		if err != nil || next != 2 {
			t.Fatalf("unexpected nextPackageOrdinal() result %d err=%v", next, err)
		}
	})

	t.Run("current package and scans", func(t *testing.T) {
		row := fakeRow{values: []any{"pkg-1", "dep-1", 1, "v1", "1.0.0", "desc", false, true, "hash", "blob", "manifest", 50, int64(1), int64(2), "Rollback", "v0", "Production", "user"}}
		key, err := scanAccessKey(fakeRow{values: []any{"id-1", "acc-1", "token", "friendly", "desc", "creator", int64(1), int64(2), false}})
		if err != nil || key.Name != "token" {
			t.Fatalf("unexpected scanAccessKey() result %#v err=%v", key, err)
		}
		pkg, err := scanPackage(row)
		if err != nil || pkg.Label != "v1" || pkg.Rollout == nil || *pkg.Rollout != 50 {
			t.Fatalf("unexpected scanPackage() result %#v err=%v", pkg, err)
		}
		got, err := currentPackage(context.Background(), &fakeDB{
			rowFn: func(context.Context, string, ...any) pgx.Row { return row },
		}, "dep-1")
		if err != nil || got.Label != "v1" {
			t.Fatalf("unexpected currentPackage() result %#v err=%v", got, err)
		}
	})
}

func TestRepositoryErrorPaths(t *testing.T) {
	if !isUniqueViolation(errors.New("duplicate key")) {
		t.Fatalf("expected duplicate key detection")
	}
	if isUniqueViolation(errors.New("other")) {
		t.Fatalf("unexpected unique violation detection")
	}

	store := testStore(&fakeDB{
		execFn: func(context.Context, string, ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})
	if _, err := store.AccessKeys().Update(context.Background(), domain.AccessKey{ID: "missing"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := (&DeploymentRepo{store: store}).Update(context.Background(), "acc-1", "app-1", domain.Deployment{ID: "missing", Name: "Production"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAppRepoCrudMethods(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		tx := &fakeTx{}
		tx.queryFn = func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if strings.Contains(sql, "FROM apps a") {
				return &fakeRows{}, nil
			}
			return nil, errors.New("unexpected query")
		}
		tx.rowFn = func(_ context.Context, sql string, _ ...any) pgx.Row {
			if strings.Contains(sql, "INSERT INTO apps") {
				return fakeRow{values: []any{"app-1"}}
			}
			return fakeRow{err: pgx.ErrNoRows}
		}
		tx.execFn = func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "INSERT INTO app_collaborators") {
				return pgconn.NewCommandTag("INSERT 1"), nil
			}
			return pgconn.NewCommandTag(""), nil
		}
		store := testStore(&fakeDB{beginFn: func(context.Context) (pgx.Tx, error) { return tx, nil }})
		app, err := store.Apps().Create(context.Background(), "acc-1", domain.App{Name: "demo"})
		if err != nil || app.ID != "app-1" {
			t.Fatalf("unexpected Create() result %#v err=%v", app, err)
		}
		if !tx.commitCalled {
			t.Fatalf("expected transaction commit")
		}
	})

	t.Run("GetByName Update Delete Transfer AddAndRemoveCollaborator", func(t *testing.T) {
		beginCount := 0
		base := &fakeDB{}
		base.beginFn = func(context.Context) (pgx.Tx, error) {
			beginCount++
			tx := &fakeTx{}
			tx.execFn = func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
			return tx, nil
		}
		base.queryFn = func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			switch {
			case strings.Contains(sql, "FROM apps a"):
				return &fakeRows{rows: [][]any{{"app-1", "demo", int64(1)}}}, nil
			case strings.Contains(sql, "JOIN accounts a ON a.id = ac.account_id"):
				return &fakeRows{rows: [][]any{
					{"owner@example.com", string(domain.PermissionOwner), true},
					{"collab@example.com", string(domain.PermissionCollaborator), false},
				}}, nil
			case strings.Contains(sql, "FROM deployments WHERE app_id"):
				return &fakeRows{rows: [][]any{{"dep-1", "app-1", "Production", "dep-key", int64(1)}}}, nil
			default:
				return &fakeRows{}, nil
			}
		}
		base.rowFn = func(_ context.Context, sql string, args ...any) pgx.Row {
			switch {
			case strings.Contains(sql, "SELECT permission FROM app_collaborators"):
				return fakeRow{values: []any{string(domain.PermissionOwner)}}
			case strings.Contains(sql, "FROM packages WHERE deployment_id = $1 ORDER BY ordinal DESC LIMIT 1"):
				return fakeRow{err: pgx.ErrNoRows}
			case strings.Contains(sql, "SELECT id, email, name, created_at FROM accounts WHERE email = $1"):
				email := args[0].(string)
				if email == "collab@example.com" {
					return fakeRow{values: []any{"acc-2", email, "Collab", int64(1)}}
				}
				return fakeRow{values: []any{"acc-1", email, "Owner", int64(1)}}
			default:
				return fakeRow{err: pgx.ErrNoRows}
			}
		}
		base.execFn = func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			switch {
			case strings.Contains(sql, "UPDATE apps SET name"):
				return pgconn.NewCommandTag("UPDATE 1"), nil
			case strings.Contains(sql, "DELETE FROM apps"):
				return pgconn.NewCommandTag("DELETE 1"), nil
			case strings.Contains(sql, "INSERT INTO app_collaborators"):
				return pgconn.NewCommandTag("INSERT 1"), nil
			case strings.Contains(sql, "DELETE FROM app_collaborators"):
				return pgconn.NewCommandTag("DELETE 1"), nil
			default:
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
		}
		store := testStore(base)

		app, err := store.Apps().GetByName(context.Background(), "acc-1", "owner@example.com:demo")
		if err != nil || app.ID != "app-1" {
			t.Fatalf("unexpected GetByName() result %#v err=%v", app, err)
		}
		app, err = store.Apps().Update(context.Background(), "acc-1", domain.App{ID: "app-1", Name: "demo-renamed"})
		if err != nil || app.ID != "app-1" {
			t.Fatalf("unexpected Update() result %#v err=%v", app, err)
		}
		if err := store.Apps().Delete(context.Background(), "acc-1", "app-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if err := store.Apps().Transfer(context.Background(), "acc-1", "app-1", "collab@example.com"); err != nil {
			t.Fatalf("Transfer() error = %v", err)
		}
		if err := store.Apps().AddCollaborator(context.Background(), "acc-1", "app-1", "collab@example.com"); err != nil {
			t.Fatalf("AddCollaborator() error = %v", err)
		}
		if err := store.Apps().RemoveCollaborator(context.Background(), "acc-1", "app-1", "collab@example.com"); err != nil {
			t.Fatalf("RemoveCollaborator() error = %v", err)
		}
		if beginCount == 0 {
			t.Fatalf("expected transaction-backed methods to begin a transaction")
		}
	})
}

func TestDeploymentAndPackageCrudMethods(t *testing.T) {
	db := &fakeDB{
		rowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
			switch {
			case strings.Contains(sql, "SELECT permission FROM app_collaborators"):
				return fakeRow{values: []any{string(domain.PermissionOwner)}}
			case strings.Contains(sql, "INSERT INTO deployments"):
				return fakeRow{values: []any{"dep-1"}}
			case strings.Contains(sql, "FROM deployments WHERE app_id = $1 AND name = $2"):
				return fakeRow{values: []any{"dep-1", "app-1", "Production", "dep-key", int64(1)}}
			case strings.Contains(sql, "FROM deployments WHERE deployment_key = $1"):
				return fakeRow{values: []any{"dep-1", "app-1", "Production", "dep-key", int64(1)}}
			case strings.Contains(sql, "FROM packages WHERE deployment_id = $1 ORDER BY ordinal DESC LIMIT 1"):
				return fakeRow{values: []any{"pkg-1", "dep-1", 1, "v1", "1.0.0", "desc", false, false, "hash", "blob", "manifest", nil, int64(1), int64(2), "Rollback", "v0", "Production", "user"}}
			case strings.Contains(sql, "SELECT COALESCE(MAX(ordinal), 0) + 1 FROM packages WHERE deployment_id = $1"):
				return fakeRow{values: []any{2}}
			case strings.Contains(sql, "INSERT INTO packages"):
				return fakeRow{values: []any{"pkg-2"}}
			default:
				return fakeRow{err: pgx.ErrNoRows}
			}
		},
		execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			switch {
			case strings.Contains(sql, "UPDATE deployments SET name"):
				return pgconn.NewCommandTag("UPDATE 1"), nil
			case strings.Contains(sql, "DELETE FROM deployments"):
				return pgconn.NewCommandTag("DELETE 1"), nil
			case strings.Contains(sql, "DELETE FROM packages"):
				return pgconn.NewCommandTag("DELETE 1"), nil
			default:
				return pgconn.NewCommandTag("INSERT 1"), nil
			}
		},
	}
	store := testStore(db)

	deployment, err := store.Deployments().Create(context.Background(), "acc-1", "app-1", domain.Deployment{Name: "Production", Key: "dep-key"})
	if err != nil || deployment.ID != "dep-1" {
		t.Fatalf("unexpected Create() result %#v err=%v", deployment, err)
	}
	deployment, err = store.Deployments().GetByName(context.Background(), "acc-1", "app-1", "Production")
	if err != nil || deployment.Key != "dep-key" {
		t.Fatalf("unexpected GetByName() result %#v err=%v", deployment, err)
	}
	deployment, err = store.Deployments().GetByKey(context.Background(), "dep-key")
	if err != nil || deployment.ID != "dep-1" {
		t.Fatalf("unexpected GetByKey() result %#v err=%v", deployment, err)
	}
	if err := store.Deployments().Delete(context.Background(), "acc-1", "app-1", "dep-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := store.Packages().ClearHistory(context.Background(), "acc-1", "app-1", "dep-1"); err != nil {
		t.Fatalf("ClearHistory() error = %v", err)
	}
	pkg, err := store.Packages().CommitRollback(context.Background(), "acc-1", "app-1", "dep-1", domain.Package{AppVersion: "1.0.0"})
	if err != nil || pkg.ID != "pkg-2" || pkg.Label != "v2" {
		t.Fatalf("unexpected CommitRollback() result %#v err=%v", pkg, err)
	}

	deployment, err = store.Deployments().Update(context.Background(), "acc-1", "app-1", domain.Deployment{ID: "dep-1", Name: "Production-2"})
	if err != nil || deployment.ID != "dep-1" {
		t.Fatalf("unexpected Update() result %#v err=%v", deployment, err)
	}
}

func TestAppRepoListAndRemovalBranches(t *testing.T) {
	queryCount := 0
	store := testStore(&fakeDB{
		queryFn: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			queryCount++
			switch {
			case strings.Contains(sql, "FROM apps a"):
				return &fakeRows{rows: [][]any{{"app-1", "demo", int64(1)}}}, nil
			case strings.Contains(sql, "FROM deployments") && strings.Contains(sql, "ANY($1)"):
				return &fakeRows{rows: [][]any{{"app-1", "Production"}}}, nil
			case strings.Contains(sql, "JOIN accounts a ON a.id = ac.account_id") && strings.Contains(sql, "ANY($1)"):
				return &fakeRows{rows: [][]any{{"app-1", "owner@example.com", string(domain.PermissionOwner), true}}}, nil
			default:
				return &fakeRows{}, nil
			}
		},
		rowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
			switch {
			case strings.Contains(sql, "FROM packages WHERE deployment_id = $1 ORDER BY ordinal DESC LIMIT 1"):
				return fakeRow{err: pgx.ErrNoRows}
			case strings.Contains(sql, "SELECT id, email, name, created_at FROM accounts WHERE email = $1"):
				return fakeRow{values: []any{"acc-2", "other@example.com", "Other", int64(1)}}
			case strings.Contains(sql, "SELECT permission FROM app_collaborators"):
				return fakeRow{values: []any{string(domain.PermissionOwner)}}
			default:
				return fakeRow{err: pgx.ErrNoRows}
			}
		},
		execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "DELETE FROM app_collaborators") {
				return pgconn.NewCommandTag("DELETE 0"), nil
			}
			return pgconn.NewCommandTag(""), nil
		},
	})

	apps, err := store.Apps().List(context.Background(), "acc-1")
	if err != nil || len(apps) != 1 || apps[0].Deployments[0] != "Production" {
		t.Fatalf("unexpected List() result %#v err=%v", apps, err)
	}
	if queryCount != 3 {
		t.Fatalf("expected batched app list queries, got %d", queryCount)
	}
	if err := store.Apps().RemoveCollaborator(context.Background(), "acc-1", "app-1", "other@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAppRepoListReturnsEmptyWithoutBatchQueries(t *testing.T) {
	queryCount := 0
	store := testStore(&fakeDB{
		queryFn: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			queryCount++
			if strings.Contains(sql, "FROM apps a") {
				return &fakeRows{}, nil
			}
			t.Fatalf("unexpected batch query for empty app list: %s", sql)
			return nil, nil
		},
	})

	apps, err := store.Apps().List(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("expected empty app list, got %#v", apps)
	}
	if queryCount != 1 {
		t.Fatalf("expected only initial app query, got %d", queryCount)
	}
}
