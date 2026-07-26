package patient

import (
	"time"

	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
)

// SearchFilters holds optional patient search filters, always scoped to a
// single hospital so one hospital's staff can never see another's patients.
type SearchFilters struct {
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

// SearchPatients returns patients in filters.HospitalID matching every
// supplied filter. Name filters match against both the Thai and English
// name columns, since the request doesn't distinguish locale.
func (q *repo) SearchPatients(filters SearchFilters) ([]*model.Patient, error) {
	tx := q.db.Where("hospital_id = ?", filters.HospitalID)

	if filters.NationalID != nil {
		tx = tx.Where("national_id = ?", *filters.NationalID)
	}
	if filters.PassportID != nil {
		tx = tx.Where("passport_id = ?", *filters.PassportID)
	}
	if filters.FirstName != nil {
		tx = tx.Where("firstname_th = ? OR firstname_en = ?", *filters.FirstName, *filters.FirstName)
	}
	if filters.MiddleName != nil {
		tx = tx.Where("middlename_th = ? OR middlename_en = ?", *filters.MiddleName, *filters.MiddleName)
	}
	if filters.LastName != nil {
		tx = tx.Where("lastname_th = ? OR lastname_en = ?", *filters.LastName, *filters.LastName)
	}
	if filters.DateOfBirth != nil {
		tx = tx.Where("date_of_birth = ?", *filters.DateOfBirth)
	}
	if filters.PhoneNumber != nil {
		tx = tx.Where("phone_number = ?", *filters.PhoneNumber)
	}
	if filters.Email != nil {
		tx = tx.Where("email = ?", *filters.Email)
	}

	var patients []*model.Patient
	if err := tx.Find(&patients).Error; err != nil {
		return nil, err
	}
	return patients, nil
}

// FindByHospitalAndNationalID looks up a single hospital-scoped patient by
// national ID, used to decide insert vs. update when refreshing from HIS.
func (q *repo) FindByHospitalAndNationalID(hospitalID, nationalID string) (*model.Patient, error) {
	var patientRecord model.Patient
	if err := q.db.Where("hospital_id = ? AND national_id = ?", hospitalID, nationalID).First(&patientRecord).Error; err != nil {
		return nil, err
	}
	return &patientRecord, nil
}

// FindByHospitalAndPassportID is the passport_id equivalent of
// FindByHospitalAndNationalID.
func (q *repo) FindByHospitalAndPassportID(hospitalID, passportID string) (*model.Patient, error) {
	var patientRecord model.Patient
	if err := q.db.Where("hospital_id = ? AND passport_id = ?", hospitalID, passportID).First(&patientRecord).Error; err != nil {
		return nil, err
	}
	return &patientRecord, nil
}

// UpsertPatient inserts patientRecord, or updates it in place when it
// already carries an ID (i.e. a prior local record was found).
func (q *repo) UpsertPatient(patientRecord *model.Patient) error {
	if patientRecord.ID != nil {
		return q.db.Model(&model.Patient{}).Where("id = ?", patientRecord.ID).Updates(patientRecord).Error
	}
	return q.db.Create(patientRecord).Error
}
