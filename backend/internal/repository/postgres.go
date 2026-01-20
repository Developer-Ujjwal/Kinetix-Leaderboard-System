package repository

import (
	"context"
	"fmt"

	"backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresRepository handles all PostgreSQL operations
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository creates a new Postgres repository
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// UpsertUser creates or updates a user in PostgreSQL
// Uses ON CONFLICT to handle upserts efficiently
func (r *PostgresRepository) UpsertUser(ctx context.Context, username string, rating int) error {
	user := models.User{
		Username: username,
		Rating:   rating,
	}

	// Use GORM's Clauses for UPSERT (INSERT ... ON CONFLICT ... DO UPDATE)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "username"}},
		DoUpdates: clause.AssignmentColumns([]string{"rating", "updated_at"}),
	}).Create(&user).Error
}

// GetUser retrieves a user by username
func (r *PostgresRepository) GetUser(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetAllUsers retrieves all users (used for seeding Redis)
func (r *PostgresRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Order("rating DESC").Find(&users).Error
	return users, err
}

// BulkInsertUsers efficiently inserts multiple users
func (r *PostgresRepository) BulkInsertUsers(ctx context.Context, users []models.User, batchSize int) error {
	return r.db.WithContext(ctx).CreateInBatches(users, batchSize).Error
}

// GetTotalUsers returns the total count of users
func (r *PostgresRepository) GetTotalUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// DeleteUser removes a user from the database
func (r *PostgresRepository) DeleteUser(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Where("username = ?", username).Delete(&models.User{}).Error
}

// Ping checks if database is reachable
func (r *PostgresRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs database migrations
func (r *PostgresRepository) AutoMigrate() error {
	return r.db.AutoMigrate(&models.User{})
}
