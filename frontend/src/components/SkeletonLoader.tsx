import React from 'react';
import { View, StyleSheet, Dimensions } from 'react-native';
import { colors, spacing, borderRadius } from '../theme';

const { width } = Dimensions.get('window');

interface SkeletonLoaderProps {
  count?: number;
}

export const SkeletonLoader: React.FC<SkeletonLoaderProps> = ({ count = 10 }) => {
  return (
    <View style={styles.container}>
      {Array.from({ length: count }).map((_, index) => (
        <SkeletonItem key={index} />
      ))}
    </View>
  );
};

const SkeletonItem: React.FC = () => {
  return (
    <View style={styles.item}>
      <View style={[styles.shimmer, styles.rankBadge]} />
      <View style={styles.content}>
        <View style={[styles.shimmer, styles.username]} />
        <View style={[styles.shimmer, styles.rating]} />
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    paddingVertical: spacing.sm,
  },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.card,
    padding: spacing.md,
    marginHorizontal: spacing.md,
    marginVertical: spacing.xs,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    borderColor: colors.border,
  },
  shimmer: {
    backgroundColor: colors.shimmer,
    borderRadius: borderRadius.sm,
  },
  rankBadge: {
    width: 48,
    height: 48,
    borderRadius: borderRadius.md,
  },
  content: {
    flex: 1,
    marginLeft: spacing.md,
  },
  username: {
    height: 16,
    width: '60%',
    marginBottom: spacing.xs,
  },
  rating: {
    height: 14,
    width: '30%',
  },
});
