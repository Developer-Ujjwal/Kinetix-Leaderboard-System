import { useQuery } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import { leaderboardApi } from '../api/leaderboard';
import type { SearchResponse } from '../types';

/**
 * Hook for searching users with debouncing
 * @param searchTerm - Username to search for
 * @param debounceMs - Debounce delay in milliseconds (default: 300ms)
 */
export const useUserSearch = (searchTerm: string, debounceMs: number = 300) => {
  const [debouncedTerm, setDebouncedTerm] = useState(searchTerm);

  // Debounce logic
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedTerm(searchTerm);
    }, debounceMs);

    return () => {
      clearTimeout(timer);
    };
  }, [searchTerm, debounceMs]);

  // Only query if debounced term is not empty and at least 3 characters
  const shouldSearch = debouncedTerm.trim().length >= 3;

  return useQuery<SearchResponse, Error>({
    queryKey: ['user-search', debouncedTerm],
    queryFn: () => leaderboardApi.searchUser(debouncedTerm),
    enabled: shouldSearch,
    staleTime: 60000, // Fresh for 1 minute
    retry: 1, // Only retry once on failure
    retryDelay: 500,
  });
};

/**
 * Hook to check if search is actively debouncing
 */
export const useSearchDebounce = (value: string, delay: number = 300) => {
  const [debouncedValue, setDebouncedValue] = useState(value);
  const [isDebouncing, setIsDebouncing] = useState(false);

  useEffect(() => {
    setIsDebouncing(true);
    const timer = setTimeout(() => {
      setDebouncedValue(value);
      setIsDebouncing(false);
    }, delay);

    return () => {
      clearTimeout(timer);
      setIsDebouncing(false);
    };
  }, [value, delay]);

  return { debouncedValue, isDebouncing };
};
