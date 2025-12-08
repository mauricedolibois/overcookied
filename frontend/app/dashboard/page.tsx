'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Profile from '@/components/Profile';
import GameHistory from '@/components/GameHistory';
import Leaderboard from '@/components/Leaderboard';
import CookieBackground from '@/components/CookieBackground';

interface User {
  username: string;
  id: number;
  avatar: string;
}

export default function DashboardPage() {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [user, setUser] = useState<User | null>(() => {
    if (typeof window !== 'undefined') {
      const storedUser = localStorage.getItem('user');
      return storedUser ? JSON.parse(storedUser) : null;
    }
    return null;
  });
  const [sparkleActive, setSparkleActive] = useState(false);
  const router = useRouter();

  useEffect(() => {
    // Check if user is logged in
    if (!user) {
      router.push('/login');
    }
  }, [router, user]);

  const handleLogout = () => {
    localStorage.removeItem('user');
    router.push('/login');
  };

  const handleStartGame = () => {
    // Trigger sparkle animation
    setSparkleActive(true);
    setTimeout(() => setSparkleActive(false), 400);
    
    // Navigate to game after animation
    setTimeout(() => {
      router.push('/game');
    }, 300);
  };

  if (!user) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-xl">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-white relative">
      <CookieBackground />
      <div className="container mx-auto px-5 py-8 relative z-10">
        {/* Header with Profile */}
        <div className="mb-8">
          <Profile user={user} onLogout={handleLogout} />
        </div>

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          {/* Left Column - Start Game CTA */}
          <div className="lg:col-span-2">
            <div className="bg-[#fffef9] rounded-[24px] shadow-sm p-12 text-center border border-gray-100">
              <h2 className="text-5xl font-extrabold text-gray-800 mb-8">
                Are you ready?
              </h2>
              <button
                onClick={handleStartGame}
                className={`relative px-20 py-6 bg-[#f6e58d] hover:bg-[#f9ca24] text-black text-3xl font-extrabold rounded-[24px] shadow-[0_10px_0_0_#f9ca24] hover:shadow-[0_10px_0_0_#f0932b] active:shadow-[0_2px_0_0_#f0932b] active:translate-y-[8px] transition-all duration-75 inline-flex items-center gap-3 btn-sparkles ${sparkleActive ? 'active' : ''}`}
              >
                <span className="text-4xl">🍪</span>
                <span>Start Game</span>
                <span className="sparkle-container">
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                  <span className="sparkle"></span>
                </span>
              </button>
            </div>
          </div>

          {/* Game History */}
          <div className="bg-[#fffef9] rounded-[24px] shadow-sm p-6 border border-gray-100">
            <GameHistory userId={user.id} />
          </div>

          {/* Global Leaderboard */}
          <div className="bg-[#fffef9] rounded-[24px] shadow-sm p-6 border border-gray-100">
            <Leaderboard />
          </div>
        </div>
      </div>
    </div>
  );
}
