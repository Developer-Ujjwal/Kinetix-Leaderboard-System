// Avatar utility for leaderboard
// Uses a stable set of avatar images from a reliable CDN

// We use avatars from avatar.iran.liara.run (free, stable, no rate limits)
// Available avatars: 1-100 for each avatar set

const AVATAR_COUNT = 70; // Using 70 different avatars

// Simple hash function to map username to avatar index
const hashString = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return Math.abs(hash);
};

// Get consistent avatar URL for a username
export const getAvatarForUsername = (username: string): string => {
  const hash = hashString(username);
  const avatarNumber = (hash % AVATAR_COUNT) + 1;
  
  // Using pravatar.cc which provides consistent, high-quality avatar images
  return `https://i.pravatar.cc/48?img=${avatarNumber}`;
};
