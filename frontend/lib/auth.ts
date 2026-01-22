export interface UserSession {
  id: string;
  email: string;
  name: string;
  picture: string;
  token: string;
}

// Standalone helper function for API URL (used by components)
// Returns empty string for production (relative URLs) or configured URL for development
export function getApiUrl(): string {
  if (typeof window === 'undefined') return '';
  return process.env.NEXT_PUBLIC_API_URL || '';
}

export const authService = {
  // Get current user from localStorage
  getCurrentUser(): UserSession | null {
    if (typeof window === 'undefined') return null;
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  // Save user to localStorage
  saveUser(user: UserSession) {
    if (typeof window === 'undefined') return;
    localStorage.setItem('user', JSON.stringify(user));
  },

  // Remove user from localStorage
  removeUser() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('user');
  },

  // Verify session with backend (JWT validation)
  async verifySession(token: string): Promise<UserSession | null> {
    try {
      // Use API URL - relative in production, absolute in development
      const apiUrl = this.getApiUrl();
      const response = await fetch(`${apiUrl}/auth/verify`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        return null;
      }

      return await response.json();
    } catch (error) {
      console.error('Session verification failed:', error);
      return null;
    }
  },

  // Logout
  async logout() {
    const user = this.getCurrentUser();
    if (user?.token) {
      try {
        // Use API URL - relative in production, absolute in development
        const apiUrl = this.getApiUrl();
        await fetch(`${apiUrl}/auth/logout`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${user.token}`,
            'Content-Type': 'application/json',
          },
          credentials: 'include',
        });
      } catch (error) {
        console.error('Logout failed:', error);
      }
    }
    this.removeUser();
  },

  // Get API URL - empty for production (relative URLs), configured for development
  getApiUrl(): string {
    if (typeof window === 'undefined') return '';
    // In production (same host), use relative URLs
    // In development, NEXT_PUBLIC_API_URL can be set to http://localhost:8080
    return process.env.NEXT_PUBLIC_API_URL || '';
  },

  // Initiate Google OAuth login
  loginWithGoogle() {
    // Use API URL for OAuth - important for local development
    const apiUrl = this.getApiUrl();
    window.location.href = `${apiUrl}/auth/google/login`;
  },

  // Check if user is authenticated (validates JWT locally)
  isAuthenticated(): boolean {
    const user = this.getCurrentUser();
    if (!user || !user.token) return false;
    
    try {
      // Decode JWT to check expiration
      const payload = JSON.parse(atob(user.token.split('.')[1]));
      const expiresAt = payload.exp * 1000; // Convert to milliseconds
      return expiresAt > Date.now();
    } catch (error) {
      console.error('JWT validation error:', error);
      return false;
    }
  }
};
