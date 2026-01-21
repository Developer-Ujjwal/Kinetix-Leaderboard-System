import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { colors, typography, spacing } from '../theme';

interface EmptyStateProps {
  message?: string;
  iconName?: keyof typeof Ionicons.glyphMap;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  message = 'No users found',
  iconName = 'search-outline',
}) => {
  return (
    <View style={styles.container}>
      <Ionicons name={iconName} size={64} color={colors.textSecondary} />
      <Text style={styles.message}>{message}</Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: spacing.xl * 2,
  },
  message: {
    ...typography.body,
    color: colors.textSecondary,
    marginTop: spacing.lg,
  },
});
