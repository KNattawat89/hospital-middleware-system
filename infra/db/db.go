package db

import (
	"fmt"

	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	zapgorm "moul.io/zapgorm2"
)

func NewClient(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {

	logger := zapgorm.New(log)
	logger.SetAsDefault()

	if cfg.App.Debug {
		logger.LogMode(gormlogger.Info)
	} else {
		logger.LogMode(gormlogger.Error)
	}

	db, err := gorm.Open(postgres.Open(cfg.Db.Dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
		Logger: logger,
	})

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
		log.Fatal("Failed to enable uuid-ossp extension", zap.Error(err))
	}

	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}

	return db, nil
}
