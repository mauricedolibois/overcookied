'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { authService } from '@/lib/auth';

export default function AuthCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      const token = searchParams.get('token');
      
      if (!token) {
        setError('No authentication token received');
        setTimeout(() => router.push('/login'), 2000);
        return;
      }

      // Verify the session with the backend
      const user = await authService.verifySession(token);
      
      if (user) {
        authService.saveUser(user);
        router.push('/dashboard');
      } else {
        setError('Session verification failed');
        setTimeout(() => router.push('/login'), 2000);
      }
    };

    handleCallback();
  }, [searchParams, router]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="text-center">
        {error ? (
          <>
            <div className="text-6xl mb-4">❌</div>
            <h2 className="text-2xl font-bold text-gray-800 mb-2">
              Authentication Failed
            </h2>
            <p className="text-gray-600">{error}</p>
            <p className="text-sm text-gray-500 mt-2">Redirecting to login...</p>
          </>
        ) : (
          <>
            <div className="text-6xl mb-4 animate-spin">🍪</div>
            <h2 className="text-2xl font-bold text-gray-800 mb-2">
              Authenticating...
            </h2>
            <p className="text-gray-600">Please wait while we log you in</p>
          </>
        )}
      </div>
    </div>
  );
}
