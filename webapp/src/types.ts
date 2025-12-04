export interface LeaderboardEntry {
  rank: number;
  player_id: string;
  score: number;
  username?: string;
}

export interface LeaderboardConfig {
  id: string;
  name: string;
  sort_order: 'desc' | 'asc';
  reset_period: 'daily' | 'weekly' | 'monthly' | 'never';
  max_entries: number;
  update_mode: 'replace' | 'increment' | 'best';
  created_at: string;
  updated_at: string;
}

export interface LeaderboardUpdate {
  leaderboard_id: string;
  entries: LeaderboardEntry[];
  total_players: number;
}

export interface WebSocketMessage {
  type: string;
  leaderboard_id?: string;
  data?: LeaderboardUpdate | LeaderboardEntry | { status: string; error?: string };
  timestamp: string;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

