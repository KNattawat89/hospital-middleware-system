package staff

import (
	"testing"

	"github.com/KNattawat89/hospital-middleware-system/infra/auth"
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type fakeRepo struct {
	findHospitalByCode        func(code string) (*model.Hospital, error)
	findByHospitalAndUsername func(hospitalID, username string) (*model.Staff, error)
	createStaff               func(staffRecord *model.Staff) error
	touchLastLoginAt          func(id string) error
}

func (f *fakeRepo) WithTx(fn TxFunc) error { return fn(f) }

func (f *fakeRepo) FindHospitalByCode(code string) (*model.Hospital, error) {
	return f.findHospitalByCode(code)
}

func (f *fakeRepo) FindByHospitalAndUsername(hospitalID, username string) (*model.Staff, error) {
	return f.findByHospitalAndUsername(hospitalID, username)
}

func (f *fakeRepo) CreateStaff(staffRecord *model.Staff) error {
	return f.createStaff(staffRecord)
}

func (f *fakeRepo) TouchLastLoginAt(id string) error {
	return f.touchLastLoginAt(id)
}

type fakeTokenIssuer struct {
	pair          *auth.TokenPair
	err           error
	called        bool
	gotSubject    string
	gotHospitalID string
}

func (f *fakeTokenIssuer) GenerateTokenPair(subject, hospitalID string) (*auth.TokenPair, error) {
	f.called = true
	f.gotSubject = subject
	f.gotHospitalID = hospitalID
	return f.pair, f.err
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hashed)
}

func TestService_Create(t *testing.T) {
	hospitalID := uuid.New()

	t.Run("success", func(t *testing.T) {
		var createdStaff *model.Staff

		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return nil, gorm.ErrRecordNotFound
			},
			createStaff: func(staffRecord *model.Staff) error {
				createdStaff = staffRecord
				return nil
			},
		}

		s := &Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()}

		got, err := s.Create(CreateInput{Username: "nurse001", Password: "s3cret!", Hospital: "HOSPITAL_A"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != createdStaff {
			t.Fatalf("expected returned staff to be the created record")
		}
		if *got.Username != "nurse001" {
			t.Fatalf("expected username nurse001, got %s", *got.Username)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*got.PasswordHashed), []byte("s3cret!")); err != nil {
			t.Fatalf("password was not hashed correctly: %v", err)
		}
		if got.HospitalID != &hospitalID {
			t.Fatalf("expected staff to be scoped to the resolved hospital ID")
		}
	})

	t.Run("hospital not found", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		s := &Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()}

		_, err := s.Create(CreateInput{Username: "nurse001", Password: "s3cret!", Hospital: "UNKNOWN"})
		if err != ErrHospitalNotFound {
			t.Fatalf("expected ErrHospitalNotFound, got %v", err)
		}
	})

	t.Run("duplicate username in same hospital", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return &model.Staff{}, nil
			},
			createStaff: func(staffRecord *model.Staff) error {
				t.Fatalf("CreateStaff should not be called when username is taken")
				return nil
			},
		}

		s := &Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()}

		_, err := s.Create(CreateInput{Username: "nurse001", Password: "s3cret!", Hospital: "HOSPITAL_A"})
		if err != ErrUsernameTaken {
			t.Fatalf("expected ErrUsernameTaken, got %v", err)
		}
	})
}

func TestService_Login(t *testing.T) {
	hospitalID := uuid.New()
	staffID := uuid.New()
	hashed := hashPassword(t, "correct-password")

	newRepo := func() *fakeRepo {
		return &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return &model.Staff{ID: &staffID, PasswordHashed: &hashed}, nil
			},
			touchLastLoginAt: func(id string) error { return nil },
		}
	}

	t.Run("success", func(t *testing.T) {
		tokens := &fakeTokenIssuer{pair: &auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}}
		s := &Service{repo: newRepo(), tokens: tokens, log: zap.NewNop()}

		pair, err := s.Login(LoginInput{Username: "nurse001", Password: "correct-password", Hospital: "HOSPITAL_A"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pair.AccessToken != "access" {
			t.Fatalf("expected access token to be returned")
		}
		if !tokens.called {
			t.Fatalf("expected GenerateTokenPair to be called")
		}
		if tokens.gotSubject != staffID.String() {
			t.Fatalf("expected token subject to be staff ID, got %s", tokens.gotSubject)
		}
		if tokens.gotHospitalID != hospitalID.String() {
			t.Fatalf("expected token hospital ID to be the staff's hospital, got %s", tokens.gotHospitalID)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		tokens := &fakeTokenIssuer{pair: &auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}}
		s := &Service{repo: newRepo(), tokens: tokens, log: zap.NewNop()}

		_, err := s.Login(LoginInput{Username: "nurse001", Password: "wrong-password", Hospital: "HOSPITAL_A"})
		if err != ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if tokens.called {
			t.Fatalf("expected GenerateTokenPair not to be called on bad password")
		}
	})

	t.Run("unknown username does not leak which field was wrong", func(t *testing.T) {
		repo := newRepo()
		repo.findByHospitalAndUsername = func(hospitalID, username string) (*model.Staff, error) {
			return nil, gorm.ErrRecordNotFound
		}
		s := &Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()}

		_, err := s.Login(LoginInput{Username: "ghost", Password: "correct-password", Hospital: "HOSPITAL_A"})
		if err != ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown hospital does not leak which field was wrong", func(t *testing.T) {
		repo := newRepo()
		repo.findHospitalByCode = func(code string) (*model.Hospital, error) {
			return nil, gorm.ErrRecordNotFound
		}
		s := &Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()}

		_, err := s.Login(LoginInput{Username: "nurse001", Password: "correct-password", Hospital: "UNKNOWN"})
		if err != ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}
