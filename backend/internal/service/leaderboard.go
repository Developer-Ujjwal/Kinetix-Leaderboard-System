package service

import (
	"context"
	"fmt"
	"log"

	"backend/internal/models"
	"backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

// LeaderboardService handles business logic for the leaderboard
type LeaderboardService struct {
	redisRepo    *repository.RedisRepository
	postgresRepo *repository.PostgresRepository
}

// NewLeaderboardService creates a new leaderboard service
func NewLeaderboardService(
	redisRepo *repository.RedisRepository,
	postgresRepo *repository.PostgresRepository,
) *LeaderboardService {
	return &LeaderboardService{
		redisRepo:    redisRepo,
		postgresRepo: postgresRepo,
	}
}

// UpdateScore updates a user's score using write-through cache strategy
// Redis is updated synchronously, PostgreSQL asynchronously
func (s *LeaderboardService) UpdateScore(ctx context.Context, username string, rating int) error {
	// Step 1: Update Redis synchronously (critical path for low latency)
	if err := s.redisRepo.UpdateScore(ctx, username, rating); err != nil {
		return fmt.Errorf("failed to update Redis: %w", err)
	}

	// Step 2: Update PostgreSQL asynchronously in background goroutine
	go func() {
		// Use background context to avoid cancellation
		bgCtx := context.Background()
		
		if err := s.postgresRepo.UpsertUser(bgCtx, username, rating); err != nil {
			// Log error but don't fail the request
			log.Printf("ERROR: Failed to update PostgreSQL for user %s: %v", username, err)
		}
	}()

	return nil
}

// GetLeaderboard retrieves the leaderboard with tie-aware ranking (1224)
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, offset, limit int) (*models.LeaderboardResponse, error) {
	// Validate pagination parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Get users from Redis
	users, err := s.redisRepo.GetTopUsers(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}

	// Get total count
	total, err := s.redisRepo.GetTotalUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}

	// Apply tie-aware ranking logic (1224)
	entries := s.applyTieAwareRanking(users, offset)

	return &models.LeaderboardResponse{
		Data:   entries,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}, nil
}

// SearchUser searches for a user and returns their rank
func (s *LeaderboardService) SearchUser(ctx context.Context, username string) (*models.SearchResponse, error) {
	// Get user's rank
	rank, err := s.redisRepo.GetUserRank(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user rank: %w", err)
	}

	// Get user's score
	rating, err := s.redisRepo.GetUserScore(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user score: %w", err)
	}

	return &models.SearchResponse{
		GlobalRank: rank,
		Username:   username,
		Rating:     rating,
	}, nil
}

// applyTieAwareRanking applies the 1224 ranking system
// Users with the same score get the same rank
// The next rank is offset by the number of users sharing the previous rank
func (s *LeaderboardService) applyTieAwareRanking(users []redis.Z, offset int) []models.LeaderboardEntry {
	entries := make([]models.LeaderboardEntry, 0, len(users))
	
	if len(users) == 0 {
		return entries
	}

	// Start rank at offset + 1 (1-indexed)
	currentRank := offset + 1
	var previousScore float64
	sameRankCount := 0

	for i, user := range users {
		score := user.Score
		username := user.Member.(string)

		// If this is the first user or score is different from previous
		if i == 0 {
			previousScore = score
			sameRankCount = 1
		} else if score == previousScore {
			// Same score as previous user - keep same rank
			sameRankCount++
		} else {
			// Different score - update rank by adding the count of users with previous score
			currentRank += sameRankCount
			previousScore = score
			sameRankCount = 1
		}

		entries = append(entries, models.LeaderboardEntry{
			Rank:     currentRank,
			Username: username,
			Rating:   int(score),
		})
	}

	return entries
}

// SyncRedisFromPostgres syncs all data from PostgreSQL to Redis
// Useful for initialization or recovery
func (s *LeaderboardService) SyncRedisFromPostgres(ctx context.Context) error {
	users, err := s.postgresRepo.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users from PostgreSQL: %w", err)
	}

	if len(users) == 0 {
		log.Println("No users to sync")
		return nil
	}

	// Build map for bulk update
	userMap := make(map[string]int, len(users))
	for _, user := range users {
		userMap[user.Username] = user.Rating
	}

	// Bulk update Redis
	if err := s.redisRepo.BulkUpdateScores(ctx, userMap); err != nil {
		return fmt.Errorf("failed to sync to Redis: %w", err)
	}

	log.Printf("Successfully synced %d users to Redis", len(users))
	return nil
}

// HealthCheck checks the health of both Redis and PostgreSQL
func (s *LeaderboardService) HealthCheck(ctx context.Context) error {
	if err := s.redisRepo.Ping(ctx); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	if err := s.postgresRepo.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL health check failed: %w", err)
	}

	return nil
}
