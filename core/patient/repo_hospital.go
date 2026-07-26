package patient

import (
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
)

// FindHospitalByID looks up a hospital by its primary key, used to resolve
// the staff's hospital (from the JWT) to its HIS base URL.
func (q *repo) FindHospitalByID(id string) (*model.Hospital, error) {
	var hospital model.Hospital
	if err := q.db.Where("id = ?", id).First(&hospital).Error; err != nil {
		return nil, err
	}
	return &hospital, nil
}
