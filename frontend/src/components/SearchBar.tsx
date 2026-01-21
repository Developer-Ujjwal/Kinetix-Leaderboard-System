import React from 'react';
import { View, TextInput, StyleSheet, Pressable } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { colors, typography, spacing, borderRadius } from '../theme';

interface SearchBarProps {
  value: string;
  onChangeText: (text: string) => void;
  placeholder?: string;
  isLoading?: boolean;
}

export const SearchBar: React.FC<SearchBarProps> = ({
  value,
  onChangeText,
  placeholder = 'Search by username...',
  isLoading = false,
}) => {
  const handleClear = () => {
    onChangeText('');
  };

  return (
    <View style={styles.container}>
      <Ionicons name="search" size={20} color={colors.textSecondary} style={styles.icon} />
      
      <TextInput
        style={styles.input}
        value={value}
        onChangeText={onChangeText}
        placeholder={placeholder}
        placeholderTextColor={colors.textTertiary}
        autoCapitalize="none"
        autoCorrect={false}
        returnKeyType="search"
      />

      {value.length > 0 && (
        <Pressable onPress={handleClear} style={styles.clearButton} hitSlop={8}>
          <Ionicons name="close-circle" size={20} color={colors.textSecondary} />
        </Pressable>
      )}

      {isLoading && (
        <View style={styles.loadingIndicator}>
          <Ionicons name="sync" size={20} color={colors.primary} />
        </View>
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.card,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    borderColor: colors.border,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    marginHorizontal: spacing.lg,
    maxWidth: 800,
    alignSelf: 'center',
    width: '100%',
  },
  icon: {
    marginRight: spacing.sm,
  },
  input: {
    flex: 1,
    ...typography.body,
    color: colors.text,
    padding: 0,
  },
  clearButton: {
    marginLeft: spacing.sm,
  },
  loadingIndicator: {
    marginLeft: spacing.sm,
  },
});
