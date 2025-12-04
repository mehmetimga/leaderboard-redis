import { useLeaderboardList } from '../hooks/useLeaderboard';

interface Props {
  selectedId: string;
  onSelect: (id: string) => void;
}

export function LeaderboardSelector({ selectedId, onSelect }: Props) {
  const { leaderboards, isLoading, error } = useLeaderboardList();

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-dark-400">
        <div className="w-4 h-4 border-2 border-dark-600 border-t-dark-400 rounded-full animate-spin" />
        <span className="text-sm">Loading leaderboards...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-red-400 text-sm">
        Failed to load leaderboards
      </div>
    );
  }

  if (leaderboards.length === 0) {
    return (
      <div className="text-dark-500 text-sm">
        No leaderboards available
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-2">
      {leaderboards.map((lb) => (
        <button
          key={lb.id}
          onClick={() => onSelect(lb.id)}
          className={`
            px-4 py-2 rounded-xl text-sm font-medium transition-all duration-200
            ${selectedId === lb.id
              ? 'bg-primary-500 text-white shadow-lg shadow-primary-500/25'
              : 'bg-dark-800 text-dark-300 hover:bg-dark-700 hover:text-white'
            }
          `}
        >
          {lb.name}
        </button>
      ))}
    </div>
  );
}

