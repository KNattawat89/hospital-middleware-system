package staff

import (
	"errors"
	"fmt"

	"github.com/KNattawat89/hospital-middleware-system/infra/auth"
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrUsernameTaken      = errors.New("username already exists in this hospital")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// TokenIssuer is the subset of auth.Service used by staff login, pulled out
// as an interface so tests can mock token issuance without real JWTs.
type TokenIssuer interface {
	GenerateTokenPair(subject, hospitalID string) (*auth.TokenPair, error)
}

type Service struct {
	repo   Repo
	tokens TokenIssuer
	log    *zap.Logger
}

func NewService(repo Repo, tokens *auth.Service, log *zap.Logger) *Service {
	return &Service{repo: repo, tokens: tokens, log: log}
}

type CreateInput struct {
	Username string
	Password string
	Hospital string
}

// Create registers a new staff member under the given hospital code.
func (s *Service) Create(input CreateInput) (*model.Staff, error) {
	hospital, err := s.repo.FindHospitalByCode(input.Hospital)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrHospitalNotFound
		}
		return nil, fmt.Errorf("find hospital: %w", err)
	}

	if _, err := s.repo.FindByHospitalAndUsername(hospital.ID.String(), input.Username); err == nil {
		return nil, ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing staff: %w", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	hashedStr := string(hashed)

	staffRecord := &model.Staff{
		HospitalID:     hospital.ID,
		Username:       &input.Username,
		PasswordHashed: &hashedStr,
	}

	if err := s.repo.CreateStaff(staffRecord); err != nil {
		return nil, fmt.Errorf("create staff: %w", err)
	}

	return staffRecord, nil
}

type LoginInput struct {
	Username string
	Password string
	Hospital string
}

// Login verifies credentials scoped to a hospital and issues a token pair
// carrying the staff's hospital ID, so later requests can be scoped without
// a further DB lookup.
func (s *Service) Login(input LoginInput) (*auth.TokenPair, error) {
	hospital, err := s.repo.FindHospitalByCode(input.Hospital)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find hospital: %w", err)
	}

	staffRecord, err := s.repo.FindByHospitalAndUsername(hospital.ID.String(), input.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find staff: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*staffRecord.PasswordHashed), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	pair, err := s.tokens.GenerateTokenPair(staffRecord.ID.String(), hospital.ID.String())
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}

	// A failure here is a bookkeeping hiccup, not a reason to fail a login
	// that already succeeded and already has a valid token pair.
	if err := s.repo.TouchLastLoginAt(staffRecord.ID.String()); err != nil {
		s.log.Warn("failed to update last_login_at", zap.Error(err))
	}

	return pair, nil
}
