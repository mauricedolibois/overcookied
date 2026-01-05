'use client';

import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { authService } from '@/lib/auth';

export default function LoginPage() {
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    // Check if user is already authenticated
    if (authService.isAuthenticated()) {
      router.push('/dashboard');
    }

    // Check for OAuth errors
    const oauthError = searchParams.get('error');
    if (oauthError) {
      setError('Authentication failed. Please try again.');
    }
  }, [router, searchParams]);

  const handleGoogleLogin = () => {
    authService.loginWithGoogle();
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

          {/* Error Message */}
          {error && (
            <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded-[16px]">
              {error}
            </div>
          )}

          {/* Google Login Button */}
          <button
            type="button"
            onClick={handleGoogleLogin}
            className="w-full px-6 py-4 bg-white hover:bg-gray-50 text-gray-700 font-bold rounded-[24px] border-2 border-gray-300 shadow-md hover:shadow-lg transition-all duration-150 text-lg mb-6 flex items-center justify-center gap-3"
          >
            <svg className="w-6 h-6" viewBox="0 0 24 24">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
            Continue with Google
          </button>
        </div>
      </div>
    </div>
  );
}
