'use client';

import { useState, useEffect } from 'react';
import { getApiUrl } from '@/lib/auth';

interface GameRecord {
  id: string;
  score: number;
  opponentScore: number;
  won: boolean;
  winnerId: string;
  opponentId: string;
  playerName?: string;
  playerPicture?: string;
  opponentName?: string;
  opponentPicture?: string;
  timestamp: number;
}

interface GameHistoryProps {
  userId: string | number;
}

export default function GameHistory({ userId }: GameHistoryProps) {
  const [games, setGames] = useState<GameRecord[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function fetchHistory() {
      if (!userId) return;
      try {
        const res = await fetch(`${getApiUrl()}/api/history?userId=${userId}`);
        if (res.ok) {
          const data = await res.json();
          // API now returns { games: [...], totalCount: number }
          const gamesData = data.games || data;
          const mappedData = gamesData.map((g: any) => ({
            id: g.gameId,
            score: g.score,
            opponentScore: g.opponentScore,
            won: g.won,
            winnerId: g.winnerId,
            opponentId: g.opponent,
            playerName: g.playerName,
            playerPicture: g.playerPicture,
            opponentName: g.opponentName,
            opponentPicture: g.opponentPicture,
            timestamp: g.timestamp,
          }));
          setGames(mappedData);
          setTotalCount(data.totalCount || mappedData.length);
        }
      } catch (error) {
        console.error("Failed to fetch history", error);
      } finally {
        setIsLoading(false);
      }
    }
    fetchHistory();
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
        <div className="space-y-2 h-[600px] overflow-y-auto pr-2 custom-scrollbar">
          {games.map((game) => (
            <div
              key={game.id}
              className={`flex flex-col rounded-[16px] border-2 transition-all overflow-hidden ${game.winnerId === 'draw'
                ? 'bg-gray-50 border-gray-400 shadow-[0px_4px_0px_0px_#9CA3AF]'
                : game.won
                  ? 'bg-[#FFFAE6] border-[#FFD93D] shadow-[0px_4px_0px_0px_#FFD93D]'
                  : 'bg-white border-gray-200 shadow-sm'
                }`}
            >
              {/* ID & Date Header */}
              <div className={`px-4 py-2 flex justify-between items-center text-xs font-bold uppercase tracking-wider ${game.winnerId === 'draw'
                ? 'bg-gray-200 text-gray-700'
                : game.won
                  ? 'bg-[#FFD93D] text-gray-800'
                  : 'bg-gray-100 text-gray-500'
                }`}>
                <span>{game.winnerId === 'draw' ? '🤝 DRAW' : game.won ? '🏆 VICTORY' : '😔 DEFEAT'}</span>
                <span>{new Date(game.timestamp * 1000).toLocaleDateString()}</span>
              </div>

              <div className="p-4 flex items-center justify-between">
                {/* You */}
                <div className="flex flex-col items-center flex-1">
                  <img
                    src={game.playerPicture || `https://api.dicebear.com/7.x/avataaars/svg?seed=${game.winnerId}`}
                    alt="You"
                    className="w-12 h-12 rounded-full border-2 border-white shadow-md mb-2"
                  />
                  <div className="text-sm font-bold text-gray-800 mb-1 max-w-[100px] truncate">
                    {game.playerName || "You"}
                  </div>
                  <div className="text-2xl font-black text-gray-800">
                    {game.score.toLocaleString()}
                  </div>
                </div>

                {/* VS */}
                <div className="px-4">
                  <div className="text-gray-300 font-black text-2xl italic">VS</div>
                </div>

                {/* Opponent */}
                <div className="flex flex-col items-center flex-1">
                  <img
                    src={game.opponentPicture || `https://api.dicebear.com/7.x/avataaars/svg?seed=${game.opponentId}`}
                    alt="Opponent"
                    className="w-12 h-12 rounded-full border-2 border-white shadow-md mb-2 grayscale-[0.2]"
                  />
                  <div className="text-sm font-bold text-gray-600 mb-1 max-w-[100px] truncate">
                    {game.opponentName || "Opponent"}
                  </div>
                  <div className="text-xl font-bold text-gray-500">
                    {game.opponentScore.toLocaleString()}
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
                {totalCount}
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
