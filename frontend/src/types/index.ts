// User and Leaderboard Types

export interface User {
  rank: number;
  username: string;
  rating: number;
}

export interface LeaderboardResponse {
  data: User[];
  offset: number;
  limit: number;
  total: number;
}

export interface SearchResponse {
  global_rank: number;
  username: string;
  rating: number;
}

export interface ScoreUpdateRequest {
  username: string;
  rating: number;
}

export interface ScoreUpdateResponse {
  message: string;
  username: string;
  rating: number;
}

export interface ErrorResponse {
  error: string;
  message?: string;
}

// Pagination
export interface PaginationParams {
  offset: number;
  limit: number;
}

// Infinite Query Page
export interface LeaderboardPage {
  data: User[];
  nextOffset: number | undefined;
  hasMore: boolean;
  total: number;
}
