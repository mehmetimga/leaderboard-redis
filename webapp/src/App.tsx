import { useState, useEffect } from 'react';
import { Leaderboard } from './components/Leaderboard';
import { LeaderboardSelector } from './components/LeaderboardSelector';
import { useLeaderboardList } from './hooks/useLeaderboard';

function App() {
  const { leaderboards } = useLeaderboardList();
  const [selectedLeaderboard, setSelectedLeaderboard] = useState<string>('');

  // Set initial leaderboard when list loads
  useEffect(() => {
    if (leaderboards.length > 0 && !selectedLeaderboard) {
      setSelectedLeaderboard(leaderboards[0].id);
    }
  }, [leaderboards, selectedLeaderboard]);

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Background Effects */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 left-1/4 w-96 h-96 bg-primary-500/5 rounded-full blur-3xl" />
        <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-gradient-radial from-dark-900/50 to-transparent rounded-full" />
      </div>

      {/* Content */}
      <div className="relative max-w-4xl mx-auto px-4 py-8">
        {/* Header */}
        <header className="mb-8">
          <div className="flex items-center gap-3 mb-6">
            <div className="w-12 h-12 bg-gradient-to-br from-primary-400 to-primary-600 rounded-2xl flex items-center justify-center shadow-lg shadow-primary-500/25">
              <span className="text-2xl">🏆</span>
            </div>
            <div>
              <h1 className="text-2xl font-bold text-white">Leaderboard</h1>
              <p className="text-dark-400 text-sm">Real-time rankings</p>
            </div>
          </div>

          {/* Leaderboard Selector */}
          <LeaderboardSelector
            selectedId={selectedLeaderboard}
            onSelect={setSelectedLeaderboard}
          />
        </header>

        {/* Main Content */}
        <main>
          {selectedLeaderboard ? (
            <Leaderboard leaderboardId={selectedLeaderboard} />
          ) : (
            <div className="flex flex-col items-center justify-center h-96 gap-4 text-center">
              <div className="w-20 h-20 bg-dark-800 rounded-full flex items-center justify-center">
                <span className="text-4xl">📊</span>
              </div>
              <div>
                <p className="text-dark-300 font-semibold mb-1">No leaderboard selected</p>
                <p className="text-dark-500 text-sm">Create a leaderboard to get started</p>
              </div>
            </div>
          )}
        </main>

        {/* Footer */}
        <footer className="mt-12 pt-8 border-t border-dark-800">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4 text-sm text-dark-500">
            <div className="flex items-center gap-2">
              <span>Powered by</span>
              <span className="text-dark-400 font-medium">Redis + PostgreSQL</span>
            </div>
            <div className="flex items-center gap-4">
              <span>WebSocket Real-time Updates</span>
              <span>•</span>
              <span>30-min Batch Sync</span>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}

export default App;
