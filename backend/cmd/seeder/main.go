package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	TotalUsers    = 10000
	BatchSize     = 500
	MinRating     = 100
	MaxRating     = 5000
	UsernamePrefix = "user_"
)

func main() {
	log.Println("🌱 Starting seeder for Kinetix Leaderboard System...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize PostgreSQL
	db, err := initPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// Initialize Redis
	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✓ Connected to Redis")

	// Initialize repositories
	postgresRepo := repository.NewPostgresRepository(db)
	redisRepo := repository.NewRedisRepository(redisClient)

	// Run migrations
	if err := postgresRepo.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Seed data
	ctx := context.Background()
	
	log.Printf("🌱 Generating %d users...", TotalUsers)
	users := generateUsers(TotalUsers)
	
	log.Println("📦 Inserting users into PostgreSQL...")
	if err := seedPostgres(ctx, postgresRepo, users); err != nil {
		log.Fatalf("Failed to seed PostgreSQL: %v", err)
	}
	
	log.Println("⚡ Populating Redis leaderboard...")
	if err := seedRedis(ctx, redisRepo, users); err != nil {
		log.Fatalf("Failed to seed Redis: %v", err)
	}

	// Verify seeding
	total, err := redisRepo.GetTotalUsers(ctx)
	if err != nil {
		log.Fatalf("Failed to verify Redis: %v", err)
	}
	
	log.Printf("✅ Seeding completed successfully!")
	log.Printf("   - PostgreSQL: %d users", TotalUsers)
	log.Printf("   - Redis: %d users", total)
	
	// Show sample of top 10
	log.Println("\n📊 Top 10 Users:")
	topUsers, err := redisRepo.GetTopUsers(ctx, 0, 10)
	if err != nil {
		log.Fatalf("Failed to get top users: %v", err)
	}
	
	for i, user := range topUsers {
		username := user.Member.(string)
		// Fetch actual rating from metadata hash (not composite score from sorted set)
		rating, err := redisRepo.GetUserScore(ctx, username)
		if err != nil {
			log.Printf("   %d. %s - Rating: ERROR (%v)", i+1, username, err)
			continue
		}
		log.Printf("   %d. %s - Rating: %d", i+1, username, rating)
	}

	// Close connections
	postgresRepo.Close()
	redisRepo.Close()
	
	log.Println("\n🎉 Seeder finished!")
}

// generateUsers creates random users with ratings between MinRating and MaxRating
func generateUsers(count int) []models.User {
	users := make([]models.User, count)
	
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
	
	for i := 0; i < count; i++ {
		rating := rand.Intn(MaxRating-MinRating+1) + MinRating
		
		users[i] = models.User{
			Username: fmt.Sprintf("%s%d", UsernamePrefix, i+1),
			Rating:   rating,
		}
	}
	
	return users
}

// seedPostgres inserts users into PostgreSQL in batches
func seedPostgres(ctx context.Context, repo *repository.PostgresRepository, users []models.User) error {
	startTime := time.Now()
	
	if err := repo.BulkInsertUsers(ctx, users, BatchSize); err != nil {
		return fmt.Errorf("bulk insert failed: %w", err)
	}
	
	duration := time.Since(startTime)
	log.Printf("   ✓ Inserted %d users in %v (%.0f users/sec)", 
		len(users), duration, float64(len(users))/duration.Seconds())
	
	return nil
}

// seedRedis populates Redis leaderboard using pipelining for efficiency
func seedRedis(ctx context.Context, repo *repository.RedisRepository, users []models.User) error {
	startTime := time.Now()
	
	// Build map for bulk update
	userMap := make(map[string]int, len(users))
	for _, user := range users {
		userMap[user.Username] = user.Rating
	}
	
	if err := repo.BulkUpdateScores(ctx, userMap); err != nil {
		return fmt.Errorf("bulk update failed: %w", err)
	}
	
	duration := time.Since(startTime)
	log.Printf("   ✓ Populated Redis with %d users in %v (%.0f users/sec)", 
		len(users), duration, float64(len(users))/duration.Seconds())
	
	return nil
}

// initPostgres initializes PostgreSQL connection
func initPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pool for bulk operations
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

// initRedis initializes Redis connection
func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
