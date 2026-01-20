package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	// LeaderboardKey is the Redis sorted set key for the leaderboard
	LeaderboardKey = "leaderboard:ratings"
)

// RedisRepository handles all Redis operations
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// UpdateScore updates a user's score in Redis (using sorted set)
// The score is used directly as the Redis score for ZADD
func (r *RedisRepository) UpdateScore(ctx context.Context, username string, rating int) error {
	// ZADD adds or updates the member with the score
	// We use float64(rating) as the score for proper sorting
	return r.client.ZAdd(ctx, LeaderboardKey, redis.Z{
		Score:  float64(rating),
		Member: username,
	}).Err()
}

// GetUserScore retrieves a user's score from Redis
func (r *RedisRepository) GetUserScore(ctx context.Context, username string) (int, error) {
	score, err := r.client.ZScore(ctx, LeaderboardKey, username).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	return int(score), nil
}

// GetUserRank calculates a user's rank using tie-aware logic (1224)
// Returns the rank (1-indexed) or error if user not found
func (r *RedisRepository) GetUserRank(ctx context.Context, username string) (int, error) {
	// Use pipeline to minimize round-trips
	pipe := r.client.Pipeline()
	
	// Get the user's score
	scoreCmd := pipe.ZScore(ctx, LeaderboardKey, username)
	
	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	
	score, err := scoreCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	
	// Count users with score strictly greater than current user's score
	// ZCOUNT key (score +inf counts users with score > current score
	count, err := r.client.ZCount(ctx, LeaderboardKey, fmt.Sprintf("(%f", score), "+inf").Result()
	if err != nil {
		return 0, err
	}
	
	// Rank = count of users with higher score + 1
	return int(count) + 1, nil
}

// GetTopUsers retrieves top users from the leaderboard with pagination
// Returns users sorted by score in descending order
func (r *RedisRepository) GetTopUsers(ctx context.Context, offset, limit int) ([]redis.Z, error) {
	// ZREVRANGE with scores returns users sorted by score (high to low)
	// Using WITHSCORES to get both username and rating
	start := int64(offset)
	stop := int64(offset + limit - 1)
	
	return r.client.ZRevRangeWithScores(ctx, LeaderboardKey, start, stop).Result()
}

// GetTotalUsers returns the total number of users in the leaderboard
func (r *RedisRepository) GetTotalUsers(ctx context.Context) (int64, error) {
	return r.client.ZCard(ctx, LeaderboardKey).Result()
}

// DeleteUser removes a user from the leaderboard
func (r *RedisRepository) DeleteUser(ctx context.Context, username string) error {
	return r.client.ZRem(ctx, LeaderboardKey, username).Err()
}

// BulkUpdateScores updates multiple users' scores efficiently using pipeline
func (r *RedisRepository) BulkUpdateScores(ctx context.Context, users map[string]int) error {
	pipe := r.client.Pipeline()
	
	for username, rating := range users {
		pipe.ZAdd(ctx, LeaderboardKey, redis.Z{
			Score:  float64(rating),
			Member: username,
		})
	}
	
	_, err := pipe.Exec(ctx)
	return err
}

// Ping checks if Redis is reachable
func (r *RedisRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisRepository) Close() error {
	return r.client.Close()
}
