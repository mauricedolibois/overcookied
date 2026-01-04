'use client';

import { useEffect, useState, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useGameSocket, GameState } from '../hooks/useGameSocket';
import { authService, UserSession } from '@/lib/auth';

type Particle = {
  id: number;
  x: number;
  y: number;
  color: 'blue' | 'red' | 'gold';
  text: string;
};

export default function GamePage() {
  const router = useRouter();
  const [user, setUser] = useState<UserSession | null>(null);
  const {
    isConnected,
    gameState,
    gameStatus,
    opponentClick,
    goldenCookieInfo,
    powerUpExpiresAt,
    sendClick,
    claimGoldenCookie,
    quitGame
  } = useGameSocket(user);

  const [particles, setParticles] = useState<Particle[]>([]);
  const [showCountdown, setShowCountdown] = useState(false);
  const [countdownValue, setCountdownValue] = useState(5);
  const cookieRef = useRef<HTMLDivElement>(null);

  // Handle Game Start Countdown
  useEffect(() => {
    if (gameStatus === 'PLAYING') {
      setShowCountdown(true);
      setCountdownValue(5);

      const interval = setInterval(() => {
        setCountdownValue((prev) => {
          if (prev <= 1) {
            clearInterval(interval);
            setShowCountdown(false);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(interval);
    }
  }, [gameStatus]);

  useEffect(() => {
    // Auth Check
    const storedUser = localStorage.getItem('user');
    if (!storedUser) {
      router.push('/login');
    } else {
      setUser(JSON.parse(storedUser));
    }
  }, [router]);

  // Handle Opponent Clicks (Red Particles)
  useEffect(() => {
    if (opponentClick) {
      addParticle(
        50 + (Math.random() * 20 - 10), // Near center
        50 + (Math.random() * 20 - 10),
        'red',
        `+${opponentClick.count}`
      );
    }
  }, [opponentClick]);

  const addParticle = (x: number, y: number, color: 'blue' | 'red' | 'gold', text: string) => {
    const id = Date.now() + Math.random();
    setParticles(prev => [...prev, { id, x, y, color, text }]);
    setTimeout(() => {
      setParticles(prev => prev.filter(p => p.id !== id));
    }, 1000);
  };

  const handleCookieClick = (e: React.MouseEvent) => {
    if (gameStatus !== 'PLAYING' || showCountdown) return;

    // Calculate click position relative to container for particle
    const rect = e.currentTarget.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * 100;
    const y = ((e.clientY - rect.top) / rect.height) * 100;

    sendClick(); // Send to server (server handles double click logic)

    // Optimistic UI
    if (powerUpExpiresAt && Date.now() < powerUpExpiresAt) {
      addParticle(x, y, 'gold', '+2');
    } else {
      addParticle(x, y, 'blue', '+1');
    }
  };

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  if (!user) return null;

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#FFE082] via-[#FFD54F] to-[#FFEB99] overflow-hidden select-none">

      {/* MATCHMAKING SCREEN */}
      {gameStatus === 'MATCHMAKING' && (
        <div className="absolute inset-0 flex flex-col items-center justify-center z-50 bg-black/40 backdrop-blur-sm">
          <div className="bg-white p-10 rounded-2xl shadow-2xl text-center">
            <div className="text-6xl animate-bounce mb-4">🍪</div>
            <h2 className="text-3xl font-bold text-gray-800 mb-2">Finding Opponent...</h2>
            <p className="text-gray-600 mb-6">Prepare your fingers!</p>
            <button
              onClick={() => router.push('/dashboard')}
              className="px-6 py-2 bg-gray-200 hover:bg-gray-300 rounded-full font-bold text-gray-700 transition"
            >
              Cancel Search
            </button>
          </div>
        </div>
      )}

      {/* GAME OVER SCREEN */}
      {gameStatus === 'FINISHED' && gameState && (
        <div className="absolute inset-0 flex flex-col items-center justify-center z-50 bg-black/60 backdrop-blur-md">
          <div className="bg-white p-12 rounded-3xl shadow-2xl text-center border-4 border-[#FFD54F]">
            <h2 className="text-5xl font-extrabold text-[#FF6B4A] mb-4">
              {gameState.reason === 'opponent_disconnected' || gameState.reason === 'opponent_quit' ? 'OPPONENT LEFT' : (gameState.winner === user.id ? 'VICTORY!' : 'DEFEAT')}
            </h2>

            {(gameState.reason === 'opponent_disconnected' || gameState.reason === 'opponent_quit') && (
              <p className="text-xl text-gray-600 mb-6 font-bold">The opponent has fled the kitchen!</p>
            )}

            <div className="flex gap-8 justify-center mb-8 text-xl">
              <div className="flex flex-col items-center">
                <span className="font-bold text-gray-500">You</span>
                <span className="text-4xl font-black text-[#FF6B4A]">{gameState.p1Name === user.id ? gameState.p1Score : gameState.p2Score}</span>
              </div>
              <div className="flex flex-col items-center justify-center">
                <span className="text-2xl font-bold text-gray-300">VS</span>
              </div>
              <div className="flex flex-col items-center">
                <span className="font-bold text-gray-500">Opponent</span>
                <span className="text-4xl font-black text-gray-600">{gameState.p1Name === user.id ? gameState.p2Score : gameState.p1Score}</span>
              </div>
            </div>
            <button
              onClick={() => router.push('/dashboard')}
              className="px-8 py-3 bg-[#FF6B4A] text-white rounded-full font-bold text-lg hover:bg-[#ff8c73] transition shadow-lg"
            >
              Back to Dashboard
            </button>
          </div>
        </div>
      )}

      {/* GAME HUD */}
      {gameState && (
        <div className="container mx-auto px-4 py-6 h-screen flex flex-col">
          {/* Header */}
          <div className="flex justify-between items-center bg-white/90 backdrop-blur rounded-2xl p-4 shadow-lg mb-8 relative z-20">
            <div className="flex flex-col w-1/3">
              <span className="text-sm font-bold text-gray-400 uppercase tracking-wider">You</span>
              <span className="text-4xl font-black text-[#FF6B4A]">
                {gameState.p1Name === user.id ? gameState.p1Score : gameState.p2Score}
              </span>
            </div>

            <div className="flex flex-col items-center w-1/3">
              <div className="bg-gray-800 text-white px-6 py-2 rounded-full font-mono text-xl shadow-inner mb-2">
                {formatTime(gameState.timeRemaining)}
              </div>
              <button
                onClick={quitGame}
                className="text-xs text-red-500 hover:text-red-700 font-bold underline cursor-pointer"
              >
                QUIT GAME
              </button>
            </div>

            <div className="flex flex-col items-end w-1/3">
              <span className="text-sm font-bold text-gray-400 uppercase tracking-wider">Opponent</span>
              <span className="text-4xl font-black text-gray-600">
                {gameState.p1Name === user.id ? gameState.p2Score : gameState.p1Score}
              </span>
            </div>
          </div>

          {/* POWER UP TIMER */}
          {powerUpExpiresAt && (
            <div className="absolute top-24 left-1/2 transform -translate-x-1/2 w-64 h-2 bg-gray-200 rounded-full overflow-hidden border border-white/50 shadow-lg z-30">
              <div
                className="h-full bg-gradient-to-r from-yellow-400 to-yellow-600 animate-drain"
                style={{ width: '100%' }}
              ></div>
            </div>
          )}

          {/* COUNTDOWN OVERLAY */}
          {showCountdown && (
            <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm pointer-events-none">
              <div className="text-[15rem] font-black text-white animate-pulse drop-shadow-[0_10px_10px_rgba(0,0,0,0.5)]">
                {countdownValue}
              </div>
            </div>
          )}

          {/* MAIN GAME AREA */}
          <div className="flex-1 flex items-center justify-center relative">

            {/* GOLDEN COOKIE */}
            {goldenCookieInfo && (
              <div
                className="absolute w-20 h-20 cursor-pointer z-40 hover:scale-110 transition-transform animate-fly"
                style={{
                  left: `${goldenCookieInfo.x}%`,
                  top: `${goldenCookieInfo.y}%`,
                  background: 'radial-gradient(circle, #FFD700 0%, #B8860B 100%)',
                  borderRadius: '50%',
                  boxShadow: '0 0 20px #FFD700',
                  border: '2px solid #FFF',
                  animationDuration: '3s', // Fast fly
                  animationTimingFunction: 'linear',
                  animationIterationCount: 'infinite'
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  claimGoldenCookie();
                }}
              >
                <div className="sparkle-container absolute inset-0 flex items-center justify-center">
                  <span className="text-4xl animate-spin">✨</span>
                </div>
              </div>
            )}

            {/* BIG COOKIE */}
            <div
              className="relative w-64 h-64 md:w-96 md:h-96 cursor-pointer transition-transform active:scale-95 group"
              onClick={handleCookieClick}
              ref={cookieRef}
            >
              <div className="w-full h-full rounded-full bg-[#8D6E63] shadow-[0_10px_30px_rgba(0,0,0,0.2)] border-8 border-[#6D4C41] flex items-center justify-center relative overflow-hidden">
                {/* Cookie Texture */}
                <div className="absolute top-1/4 left-1/4 w-8 h-8 rounded-full bg-[#4E342E] opacity-60"></div>
                <div className="absolute top-3/4 left-1/3 w-10 h-10 rounded-full bg-[#4E342E] opacity-60"></div>
                <div className="absolute top-1/2 left-3/4 w-6 h-6 rounded-full bg-[#3E2723] opacity-60"></div>
                <div className="absolute top-1/3 left-2/3 w-9 h-9 rounded-full bg-[#3E2723] opacity-60"></div>

                {/* Shine */}
                <div className="absolute top-0 left-0 w-full h-full bg-gradient-to-br from-white/10 to-transparent pointer-events-none"></div>
              </div>

              {/* PARTICLES */}
              {particles.map(p => (
                <div
                  key={p.id}
                  className={`absolute pointer-events-none font-bold text-2xl animate-float-up ${p.color === 'blue' ? 'text-blue-500 text-shadow-blue' :
                    p.color === 'red' ? 'text-red-500 text-shadow-red' :
                      'text-yellow-400 text-shadow-gold'
                    }`}
                  style={{
                    left: `${p.x}%`,
                    top: `${p.y}%`,
                  }}
                >
                  {p.text}
                </div>
              ))}
            </div>

          </div>
        </div>
      )}

      <style jsx global>{`
        @keyframes float-up {
          0% { transform: translateY(0); opacity: 1; }
          100% { transform: translateY(-50px); opacity: 0; }
        }
        .animate-float-up {
          animation: float-up 0.8s ease-out forwards;
        }
        @keyframes fly {
          0% { transform: translate(0, 0) rotate(0deg); }
          25% { transform: translate(100px, -50px) rotate(90deg); }
          50% { transform: translate(0, -100px) rotate(180deg); }
          75% { transform: translate(-100px, -50px) rotate(270deg); }
          100% { transform: translate(0, 0) rotate(360deg); }
        }
        .animate-fly {
          animation-name: fly;
        }
        .text-shadow-blue {
          text-shadow: 0 2px 4px rgba(59, 130, 246, 0.5);
        }
        .text-shadow-red {
          text-shadow: 0 2px 4px rgba(239, 68, 68, 0.5);
        }
        .text-shadow-gold {
          text-shadow: 0 2px 4px rgba(234, 179, 8, 0.5);
        }
        @keyframes drain {
          from { width: 100%; }
          to { width: 0%; }
        }
        .animate-drain {
          animation: drain 5s linear forwards;
        }
      `}</style>
    </div>
  );
}
