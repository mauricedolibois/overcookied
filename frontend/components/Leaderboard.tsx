'use client';

import { useState, useEffect } from 'react';

interface LeaderboardEntry {
  rank: number;
  username: string;
  score: number;
  cookies: number;
  avatar: string;
}

export default function Leaderboard() {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Simulate fetching leaderboard - replace with actual API call
    setTimeout(() => {
      const mockEntries: LeaderboardEntry[] = [
        { rank: 1, username: 'CookieMaster', score: 125000, cookies: 12500, avatar: '👑' },
        { rank: 2, username: 'BakingQueen', score: 98500, cookies: 9850, avatar: '👸' },
        { rank: 3, username: 'SweetTooth', score: 87200, cookies: 8720, avatar: '🍬' },
        { rank: 4, username: 'ChipChamp', score: 76800, cookies: 7680, avatar: '🏆' },
        { rank: 5, username: 'OvenKing', score: 65400, cookies: 6540, avatar: '🔥' },
        { rank: 6, username: 'DoughPro', score: 54300, cookies: 5430, avatar: '🥖' },
        { rank: 7, username: 'SugarRush', score: 48900, cookies: 4890, avatar: '⚡' },
        { rank: 8, username: 'CrumbCrusher', score: 42100, cookies: 4210, avatar: '💪' },
        { rank: 9, username: 'BatchBoss', score: 38700, cookies: 3870, avatar: '🎯' },
        { rank: 10, username: 'FlourPower', score: 32500, cookies: 3250, avatar: '⭐' },
      ];
      setEntries(mockEntries);
      setIsLoading(false);
    }, 500);
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
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {entries.map((entry) => (
            <div
              key={entry.rank}
              className={`flex items-center justify-between p-4 rounded-[16px] transition-all ${
                entry.rank <= 3
                  ? 'bg-gradient-to-r from-[#FFD93D] to-[#FFEB99] border-2 border-[#FF6B4A] shadow-[0px_4px_8px_rgba(255,107,74,0.2)]'
                  : 'bg-[#FFF4E6] hover:bg-[#FFEB99] border-2 border-[#FFD93D]'
              }`}
            >
              <div className="flex items-center space-x-3 flex-1">
                {/* Rank */}
                <div className={`text-xl font-extrabold min-w-[3.5rem] ${
                  entry.rank <= 3 
                    ? 'text-gray-800' 
                    : 'text-gray-800'
                }`}>
                  {getMedalEmoji(entry.rank)}
                </div>

                {/* Avatar */}
                <div className="text-2xl">
                  {entry.avatar}
                </div>

                {/* User Info */}
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-gray-800 truncate">
                    {entry.username}
                  </div>
                  <div className="text-sm text-gray-600 font-semibold">
                    🍪 {entry.cookies.toLocaleString()}
                  </div>
                </div>

                {/* Score */}
                <div className="text-right">
                  <div className="font-extrabold text-gray-800 text-lg">
                    {entry.score.toLocaleString()}
                  </div>
                  <div className="text-xs text-gray-600 font-bold">
                    points
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
