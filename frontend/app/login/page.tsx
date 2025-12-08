'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

export default function LoginPage() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const router = useRouter();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    
    // Simulate login - replace with actual API call
    setTimeout(() => {
      if (username && password) {
        // Generate random emoji avatar
        const avatars = ['🍪', '🧁', '🍰', '🎂', '🍩', '🥐', '🥖', '🥨', '🥯', '🧇', '🥞', '🍞', '🥧', '🍮', '🍯', '🧈', '🥛', '🍫', '🍬', '🍭'];
        const randomAvatar = avatars[Math.floor(Math.random() * avatars.length)];
        
        // Store user session (replace with proper auth)
        localStorage.setItem('user', JSON.stringify({ username, id: Date.now(), avatar: randomAvatar }));
        router.push('/dashboard');
      }
      setIsLoading(false);
    }, 1000);
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 relative overflow-hidden">
      <div className="w-full max-w-md p-6 relative z-10">
        <div className="bg-white rounded-[32px] shadow-lg p-8">
          {/* Header */}
          <div className="text-center mb-8">
            <div className="inline-block text-7xl mb-4">🍪</div>
            <h2 className="text-4xl font-extrabold text-gray-800 mb-2">
              Overcookied
            </h2>
          </div>

          {/* Login Form */}
          <form onSubmit={handleLogin} className="space-y-6">
            <div>
              <label 
                htmlFor="username" 
                className="block text-sm font-bold text-[#5D4037] mb-2"
              >
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-6 py-4 rounded-[24px] border-2 border-[#E0E0E0] bg-white text-[#5D4037] font-medium focus:ring-2 focus:ring-[#FF6B4A] focus:border-[#FF6B4A] transition-all text-base"
                placeholder="Enter your username"
                required
              />
            </div>

            <div>
              <label 
                htmlFor="password" 
                className="block text-sm font-bold text-[#5D4037] mb-2"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-6 py-4 rounded-[24px] border-2 border-[#E0E0E0] bg-white text-[#5D4037] font-medium focus:ring-2 focus:ring-[#FF6B4A] focus:border-[#FF6B4A] transition-all text-base"
                placeholder="Enter your password"
                required
              />
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full px-6 py-4 bg-[#f6e58d] hover:bg-[#f9ca24] text-black font-extrabold rounded-[24px] shadow-[0_8px_0_0_#f9ca24] hover:shadow-[0_8px_0_0_#f0932b] active:shadow-[0_2px_0_0_#f0932b] active:translate-y-[6px] transition-all duration-75 text-lg disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Logging in...' : 'Login'}
            </button>
          </form>

          {/* Footer */}
          <div className="mt-6 text-center">
            <p className="text-sm text-gray-600 font-medium">
              Don&apos;t have an account?{' '}
              <a href="#" className="text-gray-800 hover:underline font-bold">
                Sign up
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
