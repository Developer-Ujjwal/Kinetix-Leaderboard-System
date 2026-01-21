import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { colors, borderRadius } from '../theme';

interface RankBadgeProps {
  rank: number;
  size?: 'small' | 'medium' | 'large';
  showLabel?: boolean;
}

export const RankBadge: React.FC<RankBadgeProps> = ({ rank, size = 'medium', showLabel = false }) => {
  const getBadgeConfig = () => {
    switch (rank) {
      case 1:
        return { bg: colors.champion, label: 'CHAMPION', border: colors.championLight };
      case 2:
        return { bg: colors.silver, label: 'SILVER', border: '#D1D1D1' };
      case 3:
        return { bg: colors.bronze, label: 'BRONZE', border: '#F28B6B' };
      default:
        return null;
    }
  };

  const sizeStyles = {
    small: { container: 32, text: 9, padding: 6 },
    medium: { container: 40, text: 10, padding: 8 },
    large: { container: 48, text: 11, padding: 10 },
  };

  const currentSize = sizeStyles[size];
  const badgeConfig = getBadgeConfig();

  if (badgeConfig && showLabel) {
    return (
      <View style={[
        styles.badge,
        {
          backgroundColor: badgeConfig.bg,
          borderColor: badgeConfig.border,
          paddingHorizontal: currentSize.padding,
          paddingVertical: currentSize.padding / 2,
        },
      ]}>
        <Text style={[styles.badgeLabel, { fontSize: currentSize.text }]}>
          {badgeConfig.label}
        </Text>
      </View>
    );
  }

  return (
    <View style={[
      styles.rankContainer,
      { width: currentSize.container, height: currentSize.container },
    ]}>
      <Text style={[styles.rankText, { fontSize: currentSize.text + 4, color: colors.textSecondary }]}>
        #{rank}
      </Text>
    </View>
  );
};

const styles = StyleSheet.create({
  badge: {
    borderRadius: borderRadius.full,
    borderWidth: 2,
    justifyContent: 'center',
    alignItems: 'center',
  },
  badgeLabel: {
    color: '#FFFFFF',
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  rankContainer: {
    backgroundColor: colors.backgroundTertiary,
    borderRadius: borderRadius.md,
    justifyContent: 'center',
    alignItems: 'center',
  },
  rankText: {
    fontWeight: '600',
  },
});
