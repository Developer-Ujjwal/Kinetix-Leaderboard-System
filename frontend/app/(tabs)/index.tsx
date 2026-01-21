import React, { useState, useCallback, useMemo } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
  RefreshControl,
  StatusBar,
  Platform,
} from 'react-native';
import { FlashList } from '@shopify/flash-list';
import { SafeAreaView } from 'react-native-safe-area-context';
import FontAwesome from '@expo/vector-icons/FontAwesome';
import { useLeaderboard } from '../../src/hooks/useLeaderboard';
import { useUserSearch } from '../../src/hooks/useUserSearch';
import { SearchBar } from '../../src/components/SearchBar';
import { UserRow } from '../../src/components/UserRow';
import { SkeletonLoader } from '../../src/components/SkeletonLoader';
import { EmptyState } from '../../src/components/EmptyState';
import { User } from '../../src/types';
import { colors, spacing, typography } from '../../src/theme';

export default function LeaderboardScreen() {
  const [searchQuery, setSearchQuery] = useState('');

  // Leaderboard infinite scroll
  const {
    data: leaderboardData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading: isLeaderboardLoading,
    isError: isLeaderboardError,
    refetch,
  } = useLeaderboard();

  // Search functionality
  const {
    data: searchResults,
    isLoading: isSearchLoading,
    isError: isSearchError,
  } = useUserSearch(searchQuery);

  // Determine which data to display
  const isSearching = searchQuery.length >= 3;
  const displayData = useMemo(() => {
    if (isSearching && searchResults) {
      return [{ rank: searchResults.global_rank, username: searchResults.username, rating: searchResults.rating }];
    }
    if (leaderboardData) {
      return leaderboardData.pages.flatMap((page) => page.data);
    }
    return [];
  }, [isSearching, searchResults, leaderboardData]);

  // Get total user count
  const totalUsers = leaderboardData?.pages[0]?.total ?? 0;

  // Mock current user (in production, this would come from auth context)
  const currentUser: User = {
    rank: 42,
    username: 'you',
    rating: 9500,
  };

  // Refresh handler
  const [isRefreshing, setIsRefreshing] = useState(false);
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await refetch();
    setIsRefreshing(false);
  }, [refetch]);

  // Load more handler
  const handleLoadMore = useCallback(() => {
    if (!isSearching && hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [isSearching, hasNextPage, isFetchingNextPage, fetchNextPage]);

  // Render item
  const renderItem = useCallback(
    ({ item }: { item: User }) => <UserRow user={item} />,
    []
  );

  // Key extractor
  const keyExtractor = useCallback((item: User, index: number) => item?.username || `user-${index}`, []);

  // Separate header components to prevent SearchBar from losing focus
  const SearchHeader = React.useMemo(
    () => (
      <SearchBar
        value={searchQuery}
        onChangeText={setSearchQuery}
        isLoading={isSearchLoading}
      />
    ),
    [searchQuery, isSearchLoading]
  );

  const ListHeaderStats = React.useMemo(
    () => (
      <View style={styles.statsContainer}>
        <Text style={styles.statsLabel}>Total Participants</Text>
        {!isSearching && totalUsers > 0 && (
          <Text style={styles.statsValue}>
            {totalUsers.toLocaleString()} users
          </Text>
        )}
        {isSearching && (
          <Text style={styles.statsValue}>
            {displayData.length} {displayData.length === 1 ? 'result' : 'results'}
          </Text>
        )}
      </View>
    ),
    [isSearching, displayData.length, totalUsers]
  );

  // List footer (loading indicator)
  const ListFooterComponent = useCallback(() => {
    if (isFetchingNextPage) {
      return (
        <View style={styles.footer}>
          <ActivityIndicator size="small" color={colors.primary} />
        </View>
      );
    }
    return null;
  }, [isFetchingNextPage]);

  // Empty state
  const ListEmptyComponent = useCallback(() => {
    if (isLeaderboardLoading || isSearchLoading) {
      return <SkeletonLoader count={10} />;
    }
    if (isSearching && searchQuery.length < 3) {
      return (
        <EmptyState
          message="Type at least 3 characters to search"
          iconName="text-outline"
        />
      );
    }
    if (isLeaderboardError || isSearchError) {
      return (
        <EmptyState
          message="Failed to load data. Pull to refresh."
          iconName="alert-circle-outline"
        />
      );
    }
    return <EmptyState message="No users found" />;
  }, [
    isLeaderboardLoading,
    isSearchLoading,
    isSearching,
    searchQuery.length,
    isLeaderboardError,
    isSearchError,
  ]);

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <StatusBar barStyle="dark-content" backgroundColor={colors.background} />
      
      <View style={styles.header}>
        <View style={styles.headerTop}>
          <View style={styles.headerLeft}>
            <Text style={styles.headerTitle}>Leaderboard</Text>
            <Text style={styles.headerSubtitle}>The pinnacle of performance this season.</Text>
          </View>
          <View style={styles.headerRight}>
            <Text style={styles.statsLabel}>Total Participants</Text>
            {!isSearching && totalUsers > 0 && (
              <Text style={styles.statsValue}>
                {totalUsers.toLocaleString()} users
              </Text>
            )}
            {isSearching && (
              <Text style={styles.statsValue}>
                {displayData.length} {displayData.length === 1 ? 'result' : 'results'}
              </Text>
            )}
          </View>
        </View>
      </View>

      <View style={styles.searchContainer}>
        {SearchHeader}
      </View>

      <FlashList
        data={displayData}
        renderItem={renderItem}
        keyExtractor={keyExtractor}
        estimatedItemSize={80}
        ListFooterComponent={ListFooterComponent}
        ListEmptyComponent={ListEmptyComponent}
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.5}
        refreshControl={
          <RefreshControl
            refreshing={isRefreshing}
            onRefresh={handleRefresh}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.listContent}
      />
      
      {Platform.OS === 'web' && (
        <View style={styles.bottomBadgeContainer}>
          <View style={styles.bottomBadge}>
            <FontAwesome name="trophy" size={16} color="white" style={styles.badgeIcon} />
            <Text style={styles.bottomBadgeText}>Global Leaderboard</Text>
          </View>
        </View>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  searchContainer: {
    paddingTop: spacing.md,
    paddingBottom: spacing.md,
    backgroundColor: colors.background,
  },
  header: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing.md,
    backgroundColor: colors.background,
    maxWidth: 800,
    alignSelf: 'center',
    width: '100%',
  },
  headerTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  headerLeft: {
    flex: 1,
  },
  headerRight: {
    alignItems: 'center',
    marginLeft: spacing.md,
  },
  headerTitle: {
    ...typography.h1,
    color: colors.text,
    marginBottom: spacing.xs / 2,
  },
  headerSubtitle: {
    ...typography.caption,
    color: colors.textSecondary,
  },
  statsContainer: {
    alignItems: 'flex-end',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    backgroundColor: colors.background,
    maxWidth: 800,
    alignSelf: 'center',
    width: '100%',
  },
  statsLabel: {
    ...typography.caption,
    color: colors.textSecondary,
    fontSize: 12,
    marginBottom: 4,
  },
  statsValue: {
    ...typography.bodyBold,
    color: colors.text,
    fontSize: 18,
    fontWeight: '700',
  },
  listContent: {
    paddingBottom: spacing.lg,
  },
  footer: {
    paddingVertical: spacing.lg,
    alignItems: 'center',
  },
  bottomBadgeContainer: {
    position: 'absolute',
    bottom: 24,
    left: 0,
    right: 0,
    alignItems: 'center',
    justifyContent: 'center',
    pointerEvents: 'none',
  },
  bottomBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.primary,
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: 24,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.2,
    shadowRadius: 8,
    elevation: 5,
  },
  badgeIcon: {
    marginRight: 8,
  },
  bottomBadgeText: {
    color: 'white',
    fontSize: 14,
    fontWeight: '600',
    letterSpacing: -0.2,
  },
});
