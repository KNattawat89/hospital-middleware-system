package patient

import (
	"errors"
	"testing"

	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fakeRepo struct {
	findHospitalByID            func(id string) (*model.Hospital, error)
	findByHospitalAndNationalID func(hospitalID, nationalID string) (*model.Patient, error)
	findByHospitalAndPassportID func(hospitalID, passportID string) (*model.Patient, error)
	searchPatients              func(filters SearchFilters) ([]*model.Patient, error)
	upsertPatient               func(patientRecord *model.Patient) error
	gotSearchFilters            SearchFilters
}

func (f *fakeRepo) WithTx(fn TxFunc) error { return fn(f) }

func (f *fakeRepo) FindHospitalByID(id string) (*model.Hospital, error) {
	return f.findHospitalByID(id)
}

func (f *fakeRepo) FindByHospitalAndNationalID(hospitalID, nationalID string) (*model.Patient, error) {
	return f.findByHospitalAndNationalID(hospitalID, nationalID)
}

func (f *fakeRepo) FindByHospitalAndPassportID(hospitalID, passportID string) (*model.Patient, error) {
	return f.findByHospitalAndPassportID(hospitalID, passportID)
}

func (f *fakeRepo) SearchPatients(filters SearchFilters) ([]*model.Patient, error) {
	f.gotSearchFilters = filters
	return f.searchPatients(filters)
}

func (f *fakeRepo) UpsertPatient(patientRecord *model.Patient) error {
	return f.upsertPatient(patientRecord)
}

type fakeHISClient struct {
	patient    *HISPatient
	err        error
	calledID   string
	calledBase string
}

func (f *fakeHISClient) SearchByID(baseURL, id string) (*HISPatient, error) {
	f.calledBase = baseURL
	f.calledID = id
	return f.patient, f.err
}

func strPtr(s string) *string { return &s }

func TestService_Search_ScopesToHospital(t *testing.T) {
	hospitalAID := uuid.New()
	hospitalBID := uuid.New()

	var wantPatients []*model.Patient

	repo := &fakeRepo{
		searchPatients: func(filters SearchFilters) ([]*model.Patient, error) {
			if filters.HospitalID == hospitalAID.String() {
				return wantPatients, nil
			}
			return nil, nil
		},
	}
	s := &Service{repo: repo, his: &fakeHISClient{}, log: zap.NewNop()}

	// A Hospital A staff member searching (no id filters, so no HIS call)
	// only ever sees Hospital A's patients - the hospital comes from the
	// authenticated context, not from the request body.
	got, err := s.Search(SearchInput{HospitalID: hospitalAID.String(), LastName: strPtr("Somchai")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotSearchFilters.HospitalID != hospitalAID.String() {
		t.Fatalf("expected search to be scoped to hospital A")
	}
	_ = got

	// The same staff can never see Hospital B's patients - there is no
	// client-controlled way to change filters.HospitalID away from what the
	// handler put in from the token.
	gotB, err := s.Search(SearchInput{HospitalID: hospitalBID.String(), LastName: strPtr("Somchai")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotB) != 0 {
		t.Fatalf("expected zero results for a different hospital, got %d", len(gotB))
	}
}

func TestService_Search_RefreshesFromHISWhenIDGiven(t *testing.T) {
	hospitalID := uuid.New()
	baseURL := "https://hospital-a.api.co.th"

	var upserted *model.Patient

	repo := &fakeRepo{
		findHospitalByID: func(id string) (*model.Hospital, error) {
			return &model.Hospital{ID: &hospitalID, APIBaseURL: &baseURL}, nil
		},
		findByHospitalAndNationalID: func(hospitalID, nationalID string) (*model.Patient, error) {
			return nil, gorm.ErrRecordNotFound
		},
		upsertPatient: func(patientRecord *model.Patient) error {
			upserted = patientRecord
			return nil
		},
		searchPatients: func(filters SearchFilters) ([]*model.Patient, error) {
			return []*model.Patient{upserted}, nil
		},
	}

	his := &fakeHISClient{patient: &HISPatient{
		FirstNameTh: "สมชาย",
		LastNameTh:  "ใจดี",
		NationalID:  "1234567890123",
		PatientHN:   "HN-001",
		Gender:      "M",
	}}

	s := &Service{repo: repo, his: his, log: zap.NewNop()}

	nationalID := "1234567890123"
	got, err := s.Search(SearchInput{HospitalID: hospitalID.String(), NationalID: &nationalID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if his.calledBase != baseURL || his.calledID != nationalID {
		t.Fatalf("expected HIS to be called with hospital's base URL and the given national ID")
	}
	if upserted == nil {
		t.Fatalf("expected patient to be upserted from the HIS response")
	}
	if *upserted.FirstnameTh != "สมชาย" {
		t.Fatalf("expected HIS response to be mapped into the local model")
	}
	if len(got) != 1 {
		t.Fatalf("expected the refreshed patient to come back from the local search")
	}
}

func TestService_Search_UpstreamNotFoundStillSearchesLocally(t *testing.T) {
	hospitalID := uuid.New()
	baseURL := "https://hospital-a.api.co.th"

	repo := &fakeRepo{
		findHospitalByID: func(id string) (*model.Hospital, error) {
			return &model.Hospital{ID: &hospitalID, APIBaseURL: &baseURL}, nil
		},
		upsertPatient: func(patientRecord *model.Patient) error {
			t.Fatalf("should not upsert when HIS has no record")
			return nil
		},
		searchPatients: func(filters SearchFilters) ([]*model.Patient, error) {
			return nil, nil
		},
	}

	his := &fakeHISClient{err: ErrPatientNotFoundUpstream}
	s := &Service{repo: repo, his: his, log: zap.NewNop()}

	nationalID := "0000000000000"
	got, err := s.Search(SearchInput{HospitalID: hospitalID.String(), NationalID: &nationalID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no local results either")
	}
}

func TestService_Search_HISConnectionErrorStillSearchesLocally(t *testing.T) {
	hospitalID := uuid.New()
	baseURL := "https://hospital-a.api.co.th"
	existing := &model.Patient{PatientHn: strPtr("HN-001")}

	repo := &fakeRepo{
		findHospitalByID: func(id string) (*model.Hospital, error) {
			return &model.Hospital{ID: &hospitalID, APIBaseURL: &baseURL}, nil
		},
		upsertPatient: func(patientRecord *model.Patient) error {
			t.Fatalf("should not upsert when the HIS call itself failed")
			return nil
		},
		searchPatients: func(filters SearchFilters) ([]*model.Patient, error) {
			return []*model.Patient{existing}, nil
		},
	}

	// A network/timeout/5xx failure from the HIS is not the same as "no
	// record" - the upstream is just unreachable - but the endpoint must
	// still degrade to whatever is already on file rather than 500.
	his := &fakeHISClient{err: errors.New("connection refused")}
	s := &Service{repo: repo, his: his, log: zap.NewNop()}

	nationalID := "1234567890123"
	got, err := s.Search(SearchInput{HospitalID: hospitalID.String(), NationalID: &nationalID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the local record to still be returned despite the HIS error, got %d results", len(got))
	}
}
