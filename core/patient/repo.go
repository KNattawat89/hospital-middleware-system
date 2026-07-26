// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.

package patient

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repo interface {
	WithTx(TxFunc) error

	// Auto Generated
}

func NewRepo(db *gorm.DB, log *zap.Logger) Repo {
	return &repo{db: db, log: log}
}

type repo struct {
	db  *gorm.DB
	log *zap.Logger
}

type TxFunc func(Repo) error

func (q *repo) WithTx(fn TxFunc) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		q := NewRepo(tx, q.log)
		return fn(q)
	})
}
