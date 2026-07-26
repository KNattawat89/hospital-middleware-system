package staff

import (
	"errors"
	"os"
	"testing"

	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openTestDB connects to a dev Postgres migrated with this project's schema.
// These are integration tests for the thin gorm wrappers in repo_*.go -
// they're skipped, not failed, when no such DB is reachable so `go test
// ./...` still works in environments without one. The DSN is overridable via
// TEST_DATABASE_DSN/GOOSE_DBSTRING because the default below only matches
// this project's .env, not .env.example (which uses a different db name) -
// a reachable-but-unmigrated database would otherwise fail confusingly
// instead of skipping, so the table-existence check below is what actually
// decides skip vs. run, not just connectivity.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = os.Getenv("GOOSE_DBSTRING")
	}
	if dsn == "" {
		dsn = "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Skip("skipping integration test, postgres not reachable")
	}
	if err := db.Exec("SELECT 1 FROM hospitals LIMIT 0").Error; err != nil {
		t.Skipf("skipping integration test, schema not migrated on this database: %v", err)
	}
	return db
}

func createTestHospital(t *testing.T, db *gorm.DB) *model.Hospital {
	t.Helper()
	code := "TEST_" + uuid.NewString()[:8]
	name := "Test Hospital"
	baseURL := "https://example.test"
	hospital := &model.Hospital{Code: &code, Name: &name, APIBaseURL: &baseURL}
	if err := db.Create(hospital).Error; err != nil {
		t.Fatalf("create test hospital: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(hospital) })
	return hospital
}

func TestRepo_FindHospitalByCode(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)

	t.Run("found", func(t *testing.T) {
		got, err := r.FindHospitalByCode(*hospital.Code)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID.String() != hospital.ID.String() {
			t.Fatalf("unexpected hospital returned")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.FindHospitalByCode("NOPE_" + uuid.NewString())
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected ErrRecordNotFound, got %v", err)
		}
	})
}

func TestRepo_StaffLifecycle(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)
	hospitalID := hospital.ID.String()

	username := "test_" + uuid.NewString()[:8]
	hashed := "hashed-password"
	staffRecord := &model.Staff{
		HospitalID:     hospital.ID,
		Username:       &username,
		PasswordHashed: &hashed,
	}
	t.Cleanup(func() { db.Unscoped().Where("hospital_id = ?", hospitalID).Delete(&model.Staff{}) })

	t.Run("create", func(t *testing.T) {
		if err := r.CreateStaff(staffRecord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if staffRecord.ID == nil {
			t.Fatalf("expected staff ID to be populated")
		}
	})

	t.Run("find by hospital and username", func(t *testing.T) {
		got, err := r.FindByHospitalAndUsername(hospitalID, username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID.String() != staffRecord.ID.String() {
			t.Fatalf("expected to find the created staff")
		}
	})

	t.Run("find by hospital and username - not found", func(t *testing.T) {
		_, err := r.FindByHospitalAndUsername(hospitalID, "ghost")
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("touch last login at", func(t *testing.T) {
		if err := r.TouchLastLoginAt(staffRecord.ID.String()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := r.FindByHospitalAndUsername(hospitalID, username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.LastLoginAt == nil {
			t.Fatalf("expected last_login_at to be set")
		}
	})
}

func TestRepo_WithTx(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)

	t.Run("commits on success", func(t *testing.T) {
		username := "txok_" + uuid.NewString()[:8]
		hashed := "x"
		t.Cleanup(func() { db.Unscoped().Where("hospital_id = ? AND username = ?", hospital.ID.String(), username).Delete(&model.Staff{}) })

		err := r.WithTx(func(txRepo Repo) error {
			return txRepo.CreateStaff(&model.Staff{HospitalID: hospital.ID, Username: &username, PasswordHashed: &hashed})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := r.FindByHospitalAndUsername(hospital.ID.String(), username); err != nil {
			t.Fatalf("expected committed row to be findable: %v", err)
		}
	})

	t.Run("rolls back on error", func(t *testing.T) {
		username := "txfail_" + uuid.NewString()[:8]
		hashed := "x"
		wantErr := errors.New("boom")

		err := r.WithTx(func(txRepo Repo) error {
			if err := txRepo.CreateStaff(&model.Staff{HospitalID: hospital.ID, Username: &username, PasswordHashed: &hashed}); err != nil {
				return err
			}
			return wantErr
		})
		if err != wantErr {
			t.Fatalf("expected wantErr to propagate, got %v", err)
		}

		if _, err := r.FindByHospitalAndUsername(hospital.ID.String(), username); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected rolled-back row to not exist, got %v", err)
		}
	})
}
