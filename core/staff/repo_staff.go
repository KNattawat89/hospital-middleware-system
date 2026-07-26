package staff

import (
	"time"

	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
)

// FindByHospitalAndUsername looks up a staff member scoped to a single
// hospital, since username uniqueness is only guaranteed per hospital.
func (q *repo) FindByHospitalAndUsername(hospitalID, username string) (*model.Staff, error) {
	var staffRecord model.Staff
	if err := q.db.Where("hospital_id = ? AND username = ?", hospitalID, username).First(&staffRecord).Error; err != nil {
		return nil, err
	}
	return &staffRecord, nil
}

// CreateStaff inserts a new staff record.
func (q *repo) CreateStaff(staffRecord *model.Staff) error {
	return q.db.Create(staffRecord).Error
}

// TouchLastLoginAt stamps the staff's last_login_at with the current time.
func (q *repo) TouchLastLoginAt(id string) error {
	return q.db.Model(&model.Staff{}).Where("id = ?", id).Update("last_login_at", time.Now()).Error
}
