import { useEffect, useState } from 'react';
import type { LeaderboardEntry as EntryType } from '../types';

interface Props {
  entry: EntryType;
  index: number;
  previousRank?: number;
}

export function LeaderboardEntry({ entry, index, previousRank }: Props) {
  const [isHighlighted, setIsHighlighted] = useState(false);
  const [rankChange, setRankChange] = useState<'up' | 'down' | null>(null);

  useEffect(() => {
    if (previousRank !== undefined && previousRank !== entry.rank) {
      setIsHighlighted(true);
      setRankChange(entry.rank < previousRank ? 'up' : 'down');
      const timer = setTimeout(() => {
        setIsHighlighted(false);
        setRankChange(null);
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [entry.rank, previousRank]);

  const getRankBadge = () => {
    if (entry.rank === 1) {
      return (
        <div className="rank-gold w-10 h-10 rounded-xl flex items-center justify-center font-bold text-lg shadow-lg">
          👑
        </div>
      );
    }
    if (entry.rank === 2) {
      return (
        <div className="rank-silver w-10 h-10 rounded-xl flex items-center justify-center font-bold text-lg shadow-lg">
          2
        </div>
      );
    }
    if (entry.rank === 3) {
      return (
        <div className="rank-bronze w-10 h-10 rounded-xl flex items-center justify-center font-bold text-lg shadow-lg">
          3
        </div>
      );
    }
    return (
      <div className="bg-dark-800 w-10 h-10 rounded-xl flex items-center justify-center font-semibold text-dark-300">
        {entry.rank}
      </div>
    );
  };

  const formatScore = (score: number) => {
    return score.toLocaleString();
  };

  const getPlayerAvatar = (playerId: string) => {
    // Generate a deterministic color from player ID
    const hash = playerId.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
    const hue = hash % 360;
    return `hsl(${hue}, 70%, 50%)`;
  };

  return (
    <div
      className={`
        flex items-center gap-4 p-4 rounded-2xl transition-all duration-300
        ${isHighlighted ? 'animate-rank-change bg-primary-500/10' : 'bg-dark-900/50 hover:bg-dark-800/80'}
        ${entry.rank <= 3 ? 'border border-dark-700/50' : ''}
      `}
      style={{
        animationDelay: `${index * 50}ms`,
      }}
    >
      {/* Rank Badge */}
      <div className="relative">
        {getRankBadge()}
        {rankChange && (
          <div className={`absolute -right-1 -top-1 text-xs ${rankChange === 'up' ? 'text-green-400' : 'text-red-400'}`}>
            {rankChange === 'up' ? '▲' : '▼'}
          </div>
        )}
      </div>

      {/* Player Avatar */}
      <div 
        className="w-12 h-12 rounded-full flex items-center justify-center text-white font-bold text-lg shadow-lg"
        style={{ backgroundColor: getPlayerAvatar(entry.player_id) }}
      >
        {entry.player_id.slice(0, 2).toUpperCase()}
      </div>

      {/* Player Info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-white truncate">
            {entry.username || entry.player_id}
          </span>
          {entry.rank === 1 && (
            <span className="text-xs bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded-full">
              Champion
            </span>
          )}
        </div>
        <div className="text-sm text-dark-400">
          Player ID: {entry.player_id}
        </div>
      </div>

      {/* Score */}
      <div className="text-right">
        <div className={`font-mono font-bold text-xl ${isHighlighted ? 'text-primary-400 animate-count' : 'text-white'}`}>
          {formatScore(entry.score)}
        </div>
        <div className="text-xs text-dark-500 uppercase tracking-wider">
          Points
        </div>
      </div>
    </div>
  );
}

