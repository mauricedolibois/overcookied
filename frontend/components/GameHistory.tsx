'use client';

import { useState, useEffect } from 'react';

interface GameRecord {
  id: number;
  score: number;
  cookies: number;
}

interface GameHistoryProps {
  userId: number;
}

export default function GameHistory({ userId }: GameHistoryProps) {
  const [games, setGames] = useState<GameRecord[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Simulate fetching game history - replace with actual API call
    setTimeout(() => {
      const mockGames: GameRecord[] = [
        {
          id: 1,
          score: 15420,
          cookies: 1542,
        },
        {
          id: 2,
          score: 23100,
          cookies: 2310,
        },
        {
          id: 3,
          score: 8750,
          cookies: 875,
        },
        {
          id: 4,
          score: 31500,
          cookies: 3150,
        },
      ];
      setGames(mockGames);
      setIsLoading(false);
    }, 500);
  }, [userId]);

  return (
    <div>
      <div className="flex items-center mb-6">
        <h2 className="text-2xl font-extrabold text-gray-800 flex items-center">
          📊 Game History
        </h2>
      </div>

      {isLoading ? (
        <div className="text-center py-8">
          <div className="inline-block animate-spin rounded-full h-10 w-10 border-4 border-[#f6e58d] border-t-transparent"></div>
        </div>
      ) : games.length === 0 ? (
        <div className="text-center py-8 text-gray-600">
          <p className="text-4xl mb-2">🍪</p>
          <p>No games played yet!</p>
          <p className="text-sm mt-2">Start your first game to see your history here.</p>
        </div>
      ) : (
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {games.map((game, index) => (
            <div
              key={game.id}
              className="flex items-center justify-between p-4 rounded-[16px] transition-all bg-[#FFF4E6] hover:bg-[#FFEB99] border-2 border-[#FFD93D]"
            >
              <div className="flex items-center space-x-3 flex-1">
                {/* Rank/Number */}
                <div className="text-xl font-extrabold min-w-[3.5rem] text-gray-800">
                  #{index + 1}
                </div>

                {/* Cookie Icon */}
                <div className="text-2xl">
                  🍪
                </div>

                {/* Game Info */}
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-gray-800 truncate">
                    {game.cookies.toLocaleString()} cookies
                  </div>
                </div>

                {/* Score */}
                <div className="text-right">
                  <div className="font-extrabold text-gray-800 text-lg">
                    {game.score.toLocaleString()}
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

      {games.length > 0 && (
        <div className="mt-6 pt-4 border-t-2 border-[#FFD93D]">
          <div className="grid grid-cols-2 gap-4 text-center">
            <div>
              <div className="text-3xl font-extrabold text-gray-800">
                {games.length}
              </div>
              <div className="text-sm text-gray-600 font-bold">
                Games Played
              </div>
            </div>
            <div>
              <div className="text-3xl font-extrabold text-gray-800">
                {Math.max(...games.map(g => g.score)).toLocaleString()}
              </div>
              <div className="text-sm text-gray-600 font-bold">
                Personal Best
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
