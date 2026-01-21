import axios from 'axios';
import { Platform } from 'react-native';
import type {
  LeaderboardResponse,
  SearchResponse,
  ScoreUpdateRequest,
  ScoreUpdateResponse,
} from '../types';

// Configure base URL based on platform
const getBaseURL = () => {
  if (Platform.OS === 'android') {
    // Android emulator uses 10.0.2.2 to access host localhost
    return 'http://10.0.2.2:8000/api/v1';
  } else if (Platform.OS === 'ios') {
    // iOS simulator can use localhost
    return 'http://localhost:8000/api/v1';
  } else {
    // Web or other platforms
    return 'http://localhost:8000/api/v1';
  }
};

// Create axios instance
export const apiClient = axios.create({
  baseURL: getBaseURL(),
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for logging
apiClient.interceptors.request.use(
  (config) => {
    console.log(`[API] ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      console.error('[API Error]', error.response.data);
    } else if (error.request) {
      console.error('[Network Error]', error.message);
    }
    return Promise.reject(error);
  }
);

// API Service
export const leaderboardApi = {
  /**
   * Get paginated leaderboard
   */
  getLeaderboard: async (offset: number = 0, limit: number = 50): Promise<LeaderboardResponse> => {
    const response = await apiClient.get<LeaderboardResponse>('/leaderboard', {
      params: { offset, limit },
    });
    return response.data;
  },

  /**
   * Search for a specific user
   */
  searchUser: async (username: string): Promise<SearchResponse> => {
    const response = await apiClient.get<SearchResponse>(`/search/${username}`);
    return response.data;
  },

  /**
   * Update user score
   */
  updateScore: async (data: ScoreUpdateRequest): Promise<ScoreUpdateResponse> => {
    const response = await apiClient.post<ScoreUpdateResponse>('/scores', data);
    return response.data;
  },

  /**
   * Health check
   */
  healthCheck: async (): Promise<{ status: string; message: string }> => {
    const response = await apiClient.get('/health');
    return response.data;
  },

  /**
   * Trigger load simulation (stress test)
   */
  simulateLoad: async (): Promise<{ status: string; message: string; details: string }> => {
    const response = await apiClient.post('/debug/simulate');
    return response.data;
  },
};
