import { useState, useEffect, useCallback } from 'react';
import type { LeaderboardEntry, LeaderboardConfig, LeaderboardUpdate, APIResponse } from '../types';
import { useWebSocket } from './useWebSocket';

const API_BASE = 'http://localhost:8080';
const WS_URL = 'ws://localhost:8080/ws';

export function useLeaderboard(leaderboardId: string) {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [config, setConfig] = useState<LeaderboardConfig | null>(null);
  const [totalPlayers, setTotalPlayers] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  const handleUpdate = useCallback((data: LeaderboardUpdate) => {
    if (data.leaderboard_id === leaderboardId) {
      // Only show top 10
      setEntries(data.entries.slice(0, 10));
      setTotalPlayers(data.total_players);
      setLastUpdate(new Date());
    }
  }, [leaderboardId]);

  const { isConnected, subscribe, unsubscribe } = useWebSocket({
    url: WS_URL,
    leaderboardId,
    onUpdate: handleUpdate,
  });

  // Fetch initial data
  const fetchData = useCallback(async () => {
    if (!leaderboardId) return;

    setIsLoading(true);
    setError(null);

    try {
      // Fetch leaderboard config
      const configRes = await fetch(`${API_BASE}/api/v1/leaderboards/${leaderboardId}`);
      const configData: APIResponse<LeaderboardConfig> = await configRes.json();
      
      if (configData.success && configData.data) {
        setConfig(configData.data);
      }

      // Fetch top 10 entries
      const entriesRes = await fetch(`${API_BASE}/api/v1/leaderboards/${leaderboardId}/top?limit=10`);
      const entriesData: APIResponse<LeaderboardEntry[]> = await entriesRes.json();
      
      if (entriesData.success && entriesData.data) {
        setEntries(entriesData.data);
      }

      // Fetch stats
      const statsRes = await fetch(`${API_BASE}/api/v1/leaderboards/${leaderboardId}/stats`);
      const statsData: APIResponse<{ total_players: number }> = await statsRes.json();
      
      if (statsData.success && statsData.data) {
        setTotalPlayers(statsData.data.total_players);
      }

      setLastUpdate(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch leaderboard data');
    } finally {
      setIsLoading(false);
    }
  }, [leaderboardId]);

  // Change leaderboard subscription
  const changeLeaderboard = useCallback((newId: string, oldId?: string) => {
    if (oldId) {
      unsubscribe(oldId);
    }
    subscribe(newId);
  }, [subscribe, unsubscribe]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return {
    entries,
    config,
    totalPlayers,
    isLoading,
    error,
    isConnected,
    lastUpdate,
    refresh: fetchData,
    changeLeaderboard,
  };
}

export function useLeaderboardList() {
  const [leaderboards, setLeaderboards] = useState<LeaderboardConfig[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchLeaderboards = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const res = await fetch(`${API_BASE}/api/v1/leaderboards`);
      const data: APIResponse<LeaderboardConfig[]> = await res.json();
      
      if (data.success && data.data) {
        setLeaderboards(data.data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch leaderboards');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchLeaderboards();
  }, [fetchLeaderboards]);

  return {
    leaderboards,
    isLoading,
    error,
    refresh: fetchLeaderboards,
  };
}

