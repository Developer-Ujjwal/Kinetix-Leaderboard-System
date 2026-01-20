package models

import (
	"time"
)

// User represents a user in the leaderboard system
type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Rating    int       `gorm:"not null;index" json:"rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "users"
}

// ScoreRequest represents the request payload for updating scores
type ScoreRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Rating   int    `json:"rating" validate:"required,min=100,max=5000"`
}

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

// LeaderboardResponse represents the paginated leaderboard response
type LeaderboardResponse struct {
	Data   []LeaderboardEntry `json:"data"`
	Offset int                `json:"offset"`
	Limit  int                `json:"limit"`
	Total  int64              `json:"total"`
}

// SearchResponse represents the response for user search
type SearchResponse struct {
	GlobalRank int    `json:"global_rank"`
	Username   string `json:"username"`
	Rating     int    `json:"rating"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
