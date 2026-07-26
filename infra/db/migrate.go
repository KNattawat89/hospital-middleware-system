package db

import (
	"embed"

	"github.com/KNattawat89/hospital-middleware-system/infra/config"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// go:embed migrations/*
var embedMigrations embed.FS

func Migrate(config *config.Config, db *gorm.DB) error {
	if !config.App.Migrate {
		return nil
	}

	goose.SetBaseFS(embedMigrations)
	goose.SetTableName("public.goose_db_version")

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return err
	}

	return nil
}
