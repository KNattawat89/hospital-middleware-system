package staff

import (
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
)

// FindHospitalByCode looks up a hospital by its unique code (e.g. "HOSPITAL_A").
func (q *repo) FindHospitalByCode(code string) (*model.Hospital, error) {
	var hospital model.Hospital
	if err := q.db.Where("code = ?", code).First(&hospital).Error; err != nil {
		return nil, err
	}
	return &hospital, nil
}
