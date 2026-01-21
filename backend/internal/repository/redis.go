package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// LeaderboardKey is the Redis sorted set key for the leaderboard
	LeaderboardKey = "leaderboard:ratings"
	
	// MetadataKey is the Redis hash key for user metadata (username, profile URL, etc.)
	MetadataKey = "leaderboard:metadata"
	
	// VersionKey tracks the global leaderboard version for efficient change detection
	VersionKey = "leaderboard:version"
	
	// TimestampDivisor is used in composite score calculation to prevent precision loss
	// Using 10^10 ensures timestamp doesn't significantly affect the score
	TimestampDivisor = 10_000_000_000
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

// ComputeCompositeScore calculates a composite score for consistent tie-breaking
// Formula: score + (1 - timestamp/10^10)
// This ensures users who reached the same score earlier have a slightly higher value
// Example: User A reaches 5000 at t=1000 → 5000 + (1 - 0.0000001) = 5000.9999999
//          User B reaches 5000 at t=2000 → 5000 + (1 - 0.0000002) = 5000.9999998
//          User A ranks higher (older timestamp wins)
func ComputeCompositeScore(score int, timestamp int64) float64 {
	return float64(score) + (1.0 - float64(timestamp)/TimestampDivisor)
}

// ExtractBaseScore extracts the integer score from a composite score
func ExtractBaseScore(compositeScore float64) int {
	return int(compositeScore)
}

// UpdateScore updates a user's score in Redis using composite scoring for tie-breaking
// Also stores metadata in Redis hash for fast retrieval
func (r *RedisRepository) UpdateScore(ctx context.Context, username string, rating int) error {
	timestamp := time.Now().UnixNano()
	compositeScore := ComputeCompositeScore(rating, timestamp)
	
	pipe := r.client.Pipeline()
	
	// Update sorted set with composite score
	pipe.ZAdd(ctx, LeaderboardKey, redis.Z{
		Score:  compositeScore,
		Member: username,
	})
	
	// Store metadata in hash for fast lookup
	pipe.HSet(ctx, MetadataKey, username, rating) // Store base score for display
	
	// Increment global version for change detection
	pipe.Incr(ctx, VersionKey)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserScore retrieves a user's score from Redis metadata hash
func (r *RedisRepository) GetUserScore(ctx context.Context, username string) (int, error) {
	scoreStr, err := r.client.HGet(ctx, MetadataKey, username).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	
	score, err := strconv.Atoi(scoreStr)
	if err != nil {
		return 0, fmt.Errorf("invalid score format: %w", err)
	}
	
	return score, nil
}

// GetUserScoreBatch retrieves scores for multiple users using HMGET
func (r *RedisRepository) GetUserScoreBatch(ctx context.Context, usernames []string) (map[string]int, error) {
	if len(usernames) == 0 {
		return make(map[string]int), nil
	}
	
	results, err := r.client.HMGet(ctx, MetadataKey, usernames...).Result()
	if err != nil {
		return nil, err
	}
	
	scores := make(map[string]int, len(usernames))
	for i, result := range results {
		if result == nil {
			continue // User not found
		}
		
		scoreStr, ok := result.(string)
		if !ok {
			continue
		}
		
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			continue
		}
		
		scores[usernames[i]] = score
	}
	
	return scores, nil
}

// GetUserRank calculates a user's rank using composite score comparison
// Returns the rank (1-indexed) or error if user not found
func (r *RedisRepository) GetUserRank(ctx context.Context, username string) (int, error) {
	// Get the user's composite score from sorted set
	compositeScore, err := r.client.ZScore(ctx, LeaderboardKey, username).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("user not found")
		}
		return 0, err
	}
	
	// Count users with composite score strictly greater than current user
	// This provides consistent tie-breaking: earlier timestamps rank higher
	count, err := r.client.ZCount(ctx, LeaderboardKey, fmt.Sprintf("(%f", compositeScore), "+inf").Result()
	if err != nil {
		return 0, err
	}
	
	// Rank = count of users with higher composite score + 1
	return int(count) + 1, nil
}

// GetLeaderboardVersion returns the current global version number
func (r *RedisRepository) GetLeaderboardVersion(ctx context.Context) (int64, error) {
	version, err := r.client.Get(ctx, VersionKey).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // Version not set yet, return 0
		}
		return 0, err
	}
	return version, nil
}

// GetTopUsers retrieves top users from the leaderboard sorted by composite score in descending order
// Note: Composite scores need to be converted back to base scores for display
func (r *RedisRepository) GetTopUsers(ctx context.Context, offset, limit int) ([]redis.Z, error) {
	// ZREVRANGE with scores returns users sorted by composite score (high to low)
	start := int64(offset)
	stop := int64(offset + limit - 1)
	
	results, err := r.client.ZRevRangeWithScores(ctx, LeaderboardKey, start, stop).Result()
	if err != nil {
		return nil, err
	}
	
	// Extract base scores from composite scores for display
	for i := range results {
		results[i].Score = float64(ExtractBaseScore(results[i].Score))
	}
	
	return results, nil
}

// GetTotalUsers returns the total number of users in the leaderboard
func (r *RedisRepository) GetTotalUsers(ctx context.Context) (int64, error) {
	return r.client.ZCard(ctx, LeaderboardKey).Result()
}

// BulkUpdateScores updates multiple users' scores efficiently using pipeline
func (r *RedisRepository) BulkUpdateScores(ctx context.Context, users map[string]int) error {
	pipe := r.client.Pipeline()
	
	timestamp := time.Now().UnixNano()
	
	for username, rating := range users {
		compositeScore := ComputeCompositeScore(rating, timestamp)
		
		// Update sorted set with composite score
		pipe.ZAdd(ctx, LeaderboardKey, redis.Z{
			Score:  compositeScore,
			Member: username,
		})
		
		// Store metadata
		pipe.HSet(ctx, MetadataKey, username, rating)
		
		// Small timestamp increment for deterministic ordering within batch
		timestamp++
	}
	
	// Increment version once for entire batch
	pipe.Incr(ctx, VersionKey)
	
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
