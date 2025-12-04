import { useRef, useEffect } from 'react';
import { useLeaderboard } from '../hooks/useLeaderboard';
import { LeaderboardEntry } from './LeaderboardEntry';
import type { LeaderboardEntry as EntryType } from '../types';

interface Props {
  leaderboardId: string;
}

export function Leaderboard({ leaderboardId }: Props) {
  const {
    entries,
    config,
    totalPlayers,
    isLoading,
    error,
    isConnected,
    lastUpdate,
  } = useLeaderboard(leaderboardId);

  // Track previous ranks for animation
  const previousEntriesRef = useRef<Map<string, number>>(new Map());
  
  useEffect(() => {
    const newMap = new Map<string, number>();
    entries.forEach((entry) => {
      newMap.set(entry.player_id, entry.rank);
    });
    previousEntriesRef.current = newMap;
  }, [entries]);

  const getPreviousRank = (playerId: string): number | undefined => {
    return previousEntriesRef.current.get(playerId);
  };

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-96 gap-4">
        <div className="w-16 h-16 border-4 border-primary-500/30 border-t-primary-500 rounded-full animate-spin" />
        <p className="text-dark-400">Loading leaderboard...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-96 gap-4 text-center">
        <div className="w-16 h-16 bg-red-500/10 rounded-full flex items-center justify-center">
          <span className="text-3xl">⚠️</span>
        </div>
        <div>
          <p className="text-red-400 font-semibold mb-1">Failed to load leaderboard</p>
          <p className="text-dark-500 text-sm">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold text-white mb-1">
            {config?.name || 'Leaderboard'}
          </h1>
          <div className="flex items-center gap-4 text-sm text-dark-400">
            <span>{totalPlayers.toLocaleString()} players</span>
            <span>•</span>
            <span className="capitalize">{config?.update_mode} mode</span>
            {config?.reset_period !== 'never' && (
              <>
                <span>•</span>
                <span className="capitalize">Resets {config?.reset_period}</span>
              </>
            )}
          </div>
        </div>
        
        {/* Live Status */}
        <div className="flex items-center gap-3">
          <div className={`flex items-center gap-2 px-4 py-2 rounded-xl ${isConnected ? 'bg-primary-500/10' : 'bg-red-500/10'}`}>
            <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-primary-500 live-indicator' : 'bg-red-500'}`} />
            <span className={`text-sm font-medium ${isConnected ? 'text-primary-400' : 'text-red-400'}`}>
              {isConnected ? 'Live' : 'Disconnected'}
            </span>
          </div>
          {lastUpdate && (
            <div className="text-xs text-dark-500">
              Last update: {lastUpdate.toLocaleTimeString()}
            </div>
          )}
        </div>
      </div>

      {/* Top 3 Podium */}
      {entries.length >= 3 && (
        <div className="grid grid-cols-3 gap-4 mb-8">
          {/* Second Place */}
          <div className="flex flex-col items-center pt-8">
            <PodiumCard entry={entries[1]} place={2} />
          </div>
          {/* First Place */}
          <div className="flex flex-col items-center">
            <PodiumCard entry={entries[0]} place={1} />
          </div>
          {/* Third Place */}
          <div className="flex flex-col items-center pt-12">
            <PodiumCard entry={entries[2]} place={3} />
          </div>
        </div>
      )}

      {/* Leaderboard List */}
      <div className="space-y-2">
        {entries.slice(3).map((entry, index) => (
          <LeaderboardEntry
            key={entry.player_id}
            entry={entry}
            index={index}
            previousRank={getPreviousRank(entry.player_id)}
          />
        ))}
      </div>

      {/* Empty State */}
      {entries.length === 0 && (
        <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
          <div className="w-20 h-20 bg-dark-800 rounded-full flex items-center justify-center">
            <span className="text-4xl">🏆</span>
          </div>
          <div>
            <p className="text-dark-300 font-semibold mb-1">No players yet</p>
            <p className="text-dark-500 text-sm">Be the first to join this leaderboard!</p>
          </div>
        </div>
      )}
    </div>
  );
}

interface PodiumCardProps {
  entry: EntryType;
  place: 1 | 2 | 3;
}

function PodiumCard({ entry, place }: PodiumCardProps) {
  const config = {
    1: {
      bgGradient: 'from-amber-500/20 to-yellow-600/20',
      borderColor: 'border-amber-500/30',
      badge: '👑',
      badgeBg: 'bg-amber-500',
      height: 'h-32',
    },
    2: {
      bgGradient: 'from-gray-400/20 to-gray-500/20',
      borderColor: 'border-gray-400/30',
      badge: '🥈',
      badgeBg: 'bg-gray-400',
      height: 'h-24',
    },
    3: {
      bgGradient: 'from-orange-500/20 to-orange-600/20',
      borderColor: 'border-orange-500/30',
      badge: '🥉',
      badgeBg: 'bg-orange-500',
      height: 'h-20',
    },
  }[place];

  const getPlayerAvatar = (playerId: string) => {
    const hash = playerId.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
    const hue = hash % 360;
    return `hsl(${hue}, 70%, 50%)`;
  };

  return (
    <div className={`w-full bg-gradient-to-b ${config.bgGradient} border ${config.borderColor} rounded-2xl p-4 text-center relative`}>
      {/* Badge */}
      <div className={`absolute -top-3 left-1/2 -translate-x-1/2 ${config.badgeBg} w-10 h-10 rounded-full flex items-center justify-center text-xl shadow-lg`}>
        {config.badge}
      </div>
      
      {/* Avatar */}
      <div 
        className="w-16 h-16 rounded-full mx-auto mt-4 mb-3 flex items-center justify-center text-white font-bold text-xl shadow-lg ring-4 ring-white/10"
        style={{ backgroundColor: getPlayerAvatar(entry.player_id) }}
      >
        {entry.player_id.slice(0, 2).toUpperCase()}
      </div>

      {/* Name */}
      <div className="font-semibold text-white truncate mb-1">
        {entry.username || entry.player_id}
      </div>

      {/* Score */}
      <div className="font-mono font-bold text-2xl text-white">
        {entry.score.toLocaleString()}
      </div>
      <div className="text-xs text-dark-400 uppercase tracking-wider">
        Points
      </div>

      {/* Podium Base */}
      <div className={`absolute -bottom-4 left-1/2 -translate-x-1/2 w-full ${config.height} bg-gradient-to-t from-dark-900 to-transparent -z-10 rounded-b-2xl`} />
    </div>
  );
}

