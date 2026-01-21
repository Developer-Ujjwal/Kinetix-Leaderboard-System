// Light Mode Theme Configuration (Arena Style)

export const colors = {
  // Background
  background: '#F5F7FA',
  backgroundSecondary: '#FFFFFF',
  backgroundTertiary: '#E8EDF2',
  
  // Cards & Surfaces
  card: '#FFFFFF',
  cardHighlight: '#F8FAFC',
  
  // Primary Colors
  primary: '#4A90E2', // Blue
  primaryDark: '#357ABD',
  primaryLight: '#6BA3E8',
  
  // Accent
  accent: '#10B981', // Green
  accentDark: '#059669',
  
  // Text
  text: '#1E293B',
  textSecondary: '#64748B',
  textTertiary: '#94A3B8',
  textMuted: '#CBD5E1',
  
  // Rankings
  gold: '#FF9947', // Orange/Gold
  silver: '#B8B8B8', // Gray
  bronze: '#E17055', // Bronze/Orange
  
  // Champion badge
  champion: '#FF9947',
  championLight: '#FFB366',
  
  // Status
  success: '#10B981',
  error: '#EF4444',
  warning: '#F59E0B',
  boost: '#10B981',
  
  // Borders & Dividers
  border: '#E2E8F0',
  divider: '#F1F5F9',
  
  // Overlay
  overlay: 'rgba(0, 0, 0, 0.1)',
  shimmer: 'rgba(0, 0, 0, 0.05)',
};

export const spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
};

export const borderRadius = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  full: 9999,
};

export const typography = {
  h1: {
    fontSize: 32,
    fontWeight: '700' as const,
    lineHeight: 40,
    letterSpacing: -0.5,
  },
  h2: {
    fontSize: 24,
    fontWeight: '700' as const,
    lineHeight: 32,
    letterSpacing: -0.3,
  },
  h3: {
    fontSize: 20,
    fontWeight: '600' as const,
    lineHeight: 28,
  },
  body: {
    fontSize: 16,
    fontWeight: '400' as const,
    lineHeight: 24,
  },
  bodyBold: {
    fontSize: 16,
    fontWeight: '600' as const,
    lineHeight: 24,
  },
  caption: {
    fontSize: 14,
    fontWeight: '400' as const,
    lineHeight: 20,
  },
  captionBold: {
    fontSize: 14,
    fontWeight: '600' as const,
    lineHeight: 20,
  },
  small: {
    fontSize: 12,
    fontWeight: '400' as const,
    lineHeight: 16,
  },
  scoreText: {
    fontSize: 28,
    fontWeight: '700' as const,
    lineHeight: 36,
    letterSpacing: -0.5,
  },
};

export const shadows = {
  sm: {
    shadowColor: '#1E293B',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 1,
  },
  md: {
    shadowColor: '#1E293B',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.08,
    shadowRadius: 4,
    elevation: 2,
  },
  lg: {
    shadowColor: '#1E293B',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 8,
    elevation: 4,
  },
  xl: {
    shadowColor: '#1E293B',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.12,
    shadowRadius: 16,
    elevation: 6,
  },
};

export const theme = {
  colors,
  spacing,
  borderRadius,
  typography,
  shadows,
};

export type Theme = typeof theme;
