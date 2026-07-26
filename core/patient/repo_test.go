package patient

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

func TestRepo_FindHospitalByID(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)

	t.Run("found", func(t *testing.T) {
		got, err := r.FindHospitalByID(hospital.ID.String())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID.String() != hospital.ID.String() {
			t.Fatalf("unexpected hospital returned")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.FindHospitalByID(uuid.NewString())
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected ErrRecordNotFound, got %v", err)
		}
	})
}

// Subtests below share sequential state (patientRecord.ID is set by the
// first subtest and read by later ones) - do not add t.Parallel() to any of
// them without also removing that dependency.
func TestRepo_SearchAndUpsertPatients(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)
	hospitalID := hospital.ID.String()

	nationalID := "1234567890123"
	patientRecord := &model.Patient{
		HospitalID: hospital.ID,
		NationalID: &nationalID,
		LastnameEn: strPtr("Jaidee"),
	}
	t.Cleanup(func() { db.Unscoped().Where("hospital_id = ?", hospitalID).Delete(&model.Patient{}) })

	t.Run("upsert inserts a new patient", func(t *testing.T) {
		if err := r.UpsertPatient(patientRecord); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if patientRecord.ID == nil {
			t.Fatalf("expected patient ID to be populated after insert")
		}
	})

	t.Run("find by national id", func(t *testing.T) {
		got, err := r.FindByHospitalAndNationalID(hospitalID, nationalID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID.String() != patientRecord.ID.String() {
			t.Fatalf("expected to find the inserted patient")
		}
	})

	t.Run("find by national id - not found", func(t *testing.T) {
		_, err := r.FindByHospitalAndNationalID(hospitalID, "0000000000000")
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("find by passport id - not found", func(t *testing.T) {
		_, err := r.FindByHospitalAndPassportID(hospitalID, "AB123456")
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected ErrRecordNotFound, got %v", err)
		}
	})

	t.Run("upsert updates the existing patient", func(t *testing.T) {
		updated := &model.Patient{
			ID:         patientRecord.ID,
			HospitalID: hospital.ID,
			NationalID: &nationalID,
			LastnameEn: strPtr("UpdatedName"),
		}
		if err := r.UpsertPatient(updated); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := r.FindByHospitalAndNationalID(hospitalID, nationalID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *got.LastnameEn != "UpdatedName" {
			t.Fatalf("expected updated lastname, got %s", *got.LastnameEn)
		}
	})

	t.Run("search by last name matches th or en", func(t *testing.T) {
		results, err := r.SearchPatients(SearchFilters{HospitalID: hospitalID, LastName: strPtr("UpdatedName")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("search scoped to a different hospital returns nothing", func(t *testing.T) {
		other := createTestHospital(t, db)
		results, err := r.SearchPatients(SearchFilters{HospitalID: other.ID.String(), NationalID: &nationalID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results scoped to a different hospital, got %d", len(results))
		}
	})
}

func TestRepo_WithTx(t *testing.T) {
	db := openTestDB(t)
	r := NewRepo(db, zap.NewNop())
	hospital := createTestHospital(t, db)

	t.Run("commits on success", func(t *testing.T) {
		nationalID := "9990000000000"
		t.Cleanup(func() {
			db.Unscoped().Where("hospital_id = ? AND national_id = ?", hospital.ID.String(), nationalID).Delete(&model.Patient{})
		})

		err := r.WithTx(func(txRepo Repo) error {
			return txRepo.UpsertPatient(&model.Patient{HospitalID: hospital.ID, NationalID: &nationalID})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := r.FindByHospitalAndNationalID(hospital.ID.String(), nationalID); err != nil {
			t.Fatalf("expected committed row to be findable: %v", err)
		}
	})

	t.Run("rolls back on error", func(t *testing.T) {
		nationalID := "9990000000001"
		wantErr := errors.New("boom")

		err := r.WithTx(func(txRepo Repo) error {
			if err := txRepo.UpsertPatient(&model.Patient{HospitalID: hospital.ID, NationalID: &nationalID}); err != nil {
				return err
			}
			return wantErr
		})
		if err != wantErr {
			t.Fatalf("expected wantErr to propagate, got %v", err)
		}

		if _, err := r.FindByHospitalAndNationalID(hospital.ID.String(), nationalID); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected rolled-back row to not exist, got %v", err)
		}
	})
}
