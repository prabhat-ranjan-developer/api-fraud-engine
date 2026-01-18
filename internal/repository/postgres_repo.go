package repository

import (
	"fraud-engine/internal/domain"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	DB *gorm.DB
}

// NewPostgresRepository creates a new instance of the Postgres repository
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{
		DB: db,
	}
}

// LogFraud saves the fraud check result to the database
func (r *PostgresRepository) LogFraud(log domain.FraudLog) error {
	return r.DB.Create(&log).Error
}
