'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function GamePage() {
  const router = useRouter();

  useEffect(() => {
    // Check if user is logged in
    const storedUser = localStorage.getItem('user');
    if (!storedUser) {
      router.push('/login');
    }
  }, [router]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#FFE082] via-[#FFD54F] to-[#FFEB99]">
      <div className="container mx-auto px-4 py-8">
        <div className="text-center">
          <div className="text-8xl mb-6 animate-bounce inline-block">🍪</div>
          <h1 className="text-5xl font-extrabold text-[#FF6B4A] mb-4" style={{fontFamily: 'Nunito'}}>
            Game Coming Soon!
          </h1>
          <p className="text-[#5D4037] mb-10 text-xl font-semibold">
            The actual game implementation will go here.
          </p>
          <button
            onClick={() => router.push('/dashboard')}
            className="px-12 py-4 bg-gradient-to-r from-[#FF6B4A] to-[#FF7B5C] hover:from-[#FF7B5C] hover:to-[#FF6B4A] text-white font-bold rounded-[999px] shadow-[0px_4px_8px_rgba(255,107,74,0.3)] hover:shadow-[0px_6px_12px_rgba(255,107,74,0.4)] transform hover:scale-105 active:scale-95 transition-all text-lg"
          >
            ← Back to Dashboard
          </button>
        </div>
      </div>
    </div>
  );
}
