import React, { memo } from 'react';
import { View, Text, StyleSheet, Image } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { colors, typography, spacing, borderRadius } from '../theme';
import { getAvatarForUsername } from '../utils/avatarUtils';
import type { User } from '../types';

interface UserRowProps {
  user: User;
}

const getUserTier = (rank: number): string => {
  if (rank === 1) return 'Elite Performer';
  if (rank === 2) return 'Rising Star';
  if (rank === 3) return 'Competitive';
  if (rank <= 10) return 'Pro Tier';
  if (rank <= 50) return 'Intermediate';
  return 'Intermediate';
};

const getAvatarColor = (rank: number): string => {
  if (rank === 1) return colors.champion;
  if (rank === 2) return colors.silver;
  if (rank === 3) return colors.bronze;
  return colors.primary;
};

export const UserRow: React.FC<UserRowProps> = memo(({ user }) => {
  const tier = getUserTier(user.rank);
  const avatarColor = getAvatarColor(user.rank);
  const isVerified = user.rank <= 3;
  const avatarUrl = getAvatarForUsername(user.username);

  return (
    <View style={styles.container}>
      <View style={styles.rankContainer}>
        <Text style={styles.rankText}>#{user.rank}</Text>
      </View>

      <View style={[styles.avatarContainer, { backgroundColor: avatarColor + '20' }]}>
        <Image 
          source={{ uri: avatarUrl }}
          style={styles.avatar}
        />
      </View>

      <View style={styles.content}>
        <View style={styles.userInfo}>
          <Text style={styles.username} numberOfLines={1}>
            {user.username}
          </Text>
          {isVerified && (
            <Ionicons name="checkmark-circle" size={16} color={colors.primary} style={styles.verifiedIcon} />
          )}
        </View>
        <Text style={styles.tier}>{tier}</Text>
      </View>

      <View style={styles.scoreContainer}>
        <Text style={styles.points}>{user.rating.toLocaleString()}</Text>
        <Text style={styles.pointsLabel}>POINTS</Text>
      </View>
    </View>
  );
});

UserRow.displayName = 'UserRow';

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.card,
    paddingVertical: spacing.lg,
    paddingHorizontal: spacing.lg,
    marginHorizontal: spacing.lg,
    marginVertical: spacing.xs / 2,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    borderColor: colors.border,
    maxWidth: 800,
    alignSelf: 'center',
    width: '100%',
  },
  rankContainer: {
    width: 40,
    alignItems: 'flex-start',
  },
  rankText: {
    ...typography.bodyBold,
    color: colors.textSecondary,
    fontSize: 16,
  },
  avatarContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    justifyContent: 'center',
    alignItems: 'center',
    marginHorizontal: spacing.md,
    overflow: 'hidden',
  },
  avatar: {
    width: 48,
    height: 48,
  },
  content: {
    flex: 0,
    minWidth: 200,
    justifyContent: 'center',
  },
  userInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  username: {
    ...typography.bodyBold,
    color: colors.text,
    fontSize: 16,
  },
  verifiedIcon: {
    marginLeft: 4,
  },
  tier: {
    ...typography.caption,
    color: colors.textSecondary,
    fontSize: 13,
  },
  scoreContainer: {
    alignItems: 'flex-end',
    marginLeft: 'auto',
    flex: 0,
  },
  points: {
    ...typography.h3,
    color: colors.primary,
    fontSize: 20,
    fontWeight: '700',
  },
  pointsLabel: {
    ...typography.small,
    color: colors.textTertiary,
    fontSize: 11,
    marginTop: 2,
    letterSpacing: 0.5,
  },
});
