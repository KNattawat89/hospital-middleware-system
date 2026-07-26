package patient

import (
	"errors"
	"fmt"
	"time"

	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	repo Repo
	his  HISClient
	log  *zap.Logger
}

func NewService(repo Repo, his HISClient, log *zap.Logger) *Service {
	return &Service{repo: repo, his: his, log: log}
}

// SearchInput mirrors the /patient/search request. HospitalID always comes
// from the logged-in staff's token, never from client input, so a staff
// member can never search another hospital's patients.
type SearchInput struct {
	HospitalID  string
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *time.Time
	PhoneNumber *string
	Email       *string
}

// Search returns patients scoped to input.HospitalID matching whatever
// optional filters were supplied. When a national_id or passport_id is
// given, it first refreshes the local record from the hospital's HIS -
// that's the only identifier the upstream HIS itself can look up by - then
// runs the local query so every filter (including name/phone/email) is
// evaluated against up-to-date data.
func (s *Service) Search(input SearchInput) ([]*model.Patient, error) {
	if input.NationalID != nil || input.PassportID != nil {
		if err := s.refreshFromHIS(input.HospitalID, input.NationalID, input.PassportID); err != nil {
			// The HIS is an enhancement to the local search, not a
			// precondition for it: a staff member should still see whatever
			// is already on file even if the upstream hospital system is
			// unreachable or erroring.
			if errors.Is(err, ErrPatientNotFoundUpstream) {
				s.log.Debug("HIS has no record for the given id", zap.Error(err))
			} else {
				s.log.Warn("refresh from HIS failed, continuing with local data", zap.Error(err))
			}
		}
	}

	patients, err := s.repo.SearchPatients(SearchFilters{
		HospitalID:  input.HospitalID,
		NationalID:  input.NationalID,
		PassportID:  input.PassportID,
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		DateOfBirth: input.DateOfBirth,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}

	return patients, nil
}

func (s *Service) refreshFromHIS(hospitalID string, nationalID, passportID *string) error {
	hospital, err := s.repo.FindHospitalByID(hospitalID)
	if err != nil {
		return fmt.Errorf("find hospital: %w", err)
	}

	id := ""
	if nationalID != nil {
		id = *nationalID
	} else if passportID != nil {
		id = *passportID
	}

	hisPatient, err := s.his.SearchByID(*hospital.APIBaseURL, id)
	if err != nil {
		return err
	}

	var existing *model.Patient
	if nationalID != nil {
		existing, err = s.repo.FindByHospitalAndNationalID(hospitalID, *nationalID)
	} else {
		existing, err = s.repo.FindByHospitalAndPassportID(hospitalID, *passportID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find existing patient: %w", err)
	}

	patientRecord, err := hisPatient.toModel(hospitalID)
	if err != nil {
		return fmt.Errorf("map HIS response: %w", err)
	}
	if existing != nil {
		patientRecord.ID = existing.ID
	}

	return s.repo.UpsertPatient(patientRecord)
}

func (p *HISPatient) toModel(hospitalID string) (*model.Patient, error) {
	hid, err := uuid.Parse(hospitalID)
	if err != nil {
		return nil, fmt.Errorf("parse hospital id: %w", err)
	}

	return &model.Patient{
		HospitalID:   &hid,
		FirstnameTh:  nilIfEmpty(p.FirstNameTh),
		MiddlenameTh: nilIfEmpty(p.MiddleNameTh),
		LastnameTh:   nilIfEmpty(p.LastNameTh),
		FirstnameEn:  nilIfEmpty(p.FirstNameEn),
		MiddlenameEn: nilIfEmpty(p.MiddleNameEn),
		LastnameEn:   nilIfEmpty(p.LastNameEn),
		PatientHn:    nilIfEmpty(p.PatientHN),
		NationalID:   nilIfEmpty(p.NationalID),
		PassportID:   nilIfEmpty(p.PassportID),
		PhoneNumber:  nilIfEmpty(p.PhoneNumber),
		Email:        nilIfEmpty(p.Email),
		Gender:       nilIfEmpty(p.Gender),
		DateOfBirth:  parseDate(p.DateOfBirth),
	}, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}
