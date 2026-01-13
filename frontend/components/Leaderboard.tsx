'use client';

import { useState, useEffect } from 'react';

interface LeaderboardEntry {
  rank: number;
  username: string;
  score: number;
  avatar: string;
}

export default function Leaderboard() {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function fetchLeaderboard() {
      try {
        const res = await fetch(`/api/leaderboard`);
        if (res.ok) {
          const data = await res.json();
          console.log("Leaderboard Data:", data);
          // Map API response to Component format (if needed, but structure matches closely)
          // API returns CookieUser: { userId, email, name, picture, score, cookies }
          // Component expects: { rank, username, score, avatar }
          const mappedData = data.map((u: any, index: number) => ({
            rank: index + 1,
            username: u.name,
            score: u.score,
            avatar: u.picture // Assuming picture is a URL or emoji
          }));
          setEntries(mappedData);
        }
      } catch (error) {
        console.error("Failed to fetch leaderboard", error);
      } finally {
        setIsLoading(false);
      }
    }
    fetchLeaderboard();
  }, []);

  const getMedalEmoji = (rank: number) => {
    switch (rank) {
      case 1: return '🥇';
      case 2: return '🥈';
      case 3: return '🥉';
      default: return `#${rank}`;
    }
  };

  return (
    <div>
      <div className="flex items-center mb-6">
        <h2 className="text-2xl font-extrabold text-gray-800 flex items-center">
          🏆 Global Leaderboard
        </h2>
      </div>

      {isLoading ? (
        <div className="text-center py-8">
          <div className="inline-block animate-spin rounded-full h-10 w-10 border-4 border-[#f6e58d] border-t-transparent"></div>
        </div>
      ) : (
        <div className="space-y-2 h-[600px] overflow-y-auto pr-2 custom-scrollbar">
          {entries.map((entry) => (
            <div
              key={entry.rank}
              className={`flex items-center justify-between p-4 rounded-[16px] transition-all ${entry.rank <= 3
                ? 'bg-gradient-to-r from-[#FFD93D] to-[#FFEB99] border-2 border-[#FF6B4A] shadow-[0px_4px_8px_rgba(255,107,74,0.2)]'
                : 'bg-[#FFF4E6] hover:bg-[#FFEB99] border-2 border-[#FFD93D]'
                }`}
            >
              <div className="flex items-center space-x-3 flex-1">
                {/* Rank */}
                <div className={`text-xl font-extrabold min-w-[3.5rem] ${entry.rank <= 3
                  ? 'text-gray-800'
                  : 'text-gray-800'
                  }`}>
                  {getMedalEmoji(entry.rank)}
                </div>

                {/* Avatar */}
                <div className="text-2xl flex-shrink-0">
                  {entry.avatar && entry.avatar.startsWith('http') ? (
                    <img
                      src={entry.avatar}
                      alt={entry.username}
                      className="w-8 h-8 rounded-full object-cover"
                      referrerPolicy="no-referrer"
                      onError={(e) => {
                        const target = e.target as HTMLImageElement;
                        target.style.display = 'none';
                        const parent = target.parentElement;
                        if (parent) {
                          parent.innerHTML = '🍪';
                        }
                      }}
                    />
                  ) : (
                    <span>{entry.avatar || '🍪'}</span>
                  )}
                </div>

                {/* User Info */}
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-gray-800 truncate">
                    {entry.username}
                  </div>
                </div>

                {/* Score */}
                <div className="text-right">
                  <div className="font-extrabold text-gray-800 text-lg">
                    {entry.score.toLocaleString()}
                  </div>
                  <div className="text-xs text-gray-600 font-bold">
                    Total Cookies
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {entries.length > 0 && (
        <div className="mt-6 pt-4 border-t-2 border-[#FFD93D] text-center">
          <p className="text-sm text-gray-600 font-semibold">
            🎮 Play more games to climb the leaderboard!
          </p>
        </div>
      )}
    </div>
  );
}
