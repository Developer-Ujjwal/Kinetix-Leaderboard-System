import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { leaderboardApi } from '../api/leaderboard';
import type { LeaderboardPage, User } from '../types';

const ITEMS_PER_PAGE = 50;

/**
 * Hook for infinite scroll leaderboard
 * Fetches data in chunks of 50 users
 */
export const useLeaderboard = () => {
  return useInfiniteQuery<LeaderboardPage, Error>({
    queryKey: ['leaderboard'],
    queryFn: async ({ pageParam = 0 }) => {
      const offset = pageParam as number;
      const response = await leaderboardApi.getLeaderboard(offset, ITEMS_PER_PAGE);
      
      const hasMore = offset + ITEMS_PER_PAGE < response.total;
      const nextOffset = hasMore ? offset + ITEMS_PER_PAGE : undefined;

      return {
        data: response.data,
        nextOffset,
        hasMore,
        total: response.total,
      };
    },
    getNextPageParam: (lastPage) => lastPage.nextOffset,
    initialPageParam: 0,
    staleTime: 30000, // Consider data fresh for 30 seconds
    gcTime: 5 * 60 * 1000, // Keep unused data in cache for 5 minutes
  });
};

/**
 * Get flattened list of all users from infinite query
 */
export const useFlatLeaderboard = () => {
  const query = useLeaderboard();
  
  const users: User[] = query.data?.pages.flatMap((page) => page.data) ?? [];
  
  return {
    ...query,
    users,
  };
};

/**
 * Get total count of users
 */
export const useLeaderboardCount = () => {
  return useQuery({
    queryKey: ['leaderboard', 'count'],
    queryFn: async () => {
      const response = await leaderboardApi.getLeaderboard(0, 1);
      return response.total;
    },
    staleTime: 60000, // Fresh for 1 minute
  });
};
