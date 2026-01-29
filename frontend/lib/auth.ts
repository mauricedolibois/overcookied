export interface UserSession {
  id: string;
  email: string;
  name: string;
  picture: string;
  token: string;
}

// Returns empty string for production (relative URLs) or configured URL for development
export function getApiUrl(): string {
  if (typeof window === 'undefined') return '';
  return process.env.NEXT_PUBLIC_API_URL || '';
}

export const authService = {
  getCurrentUser(): UserSession | null {
    if (typeof window === 'undefined') return null;
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  saveUser(user: UserSession) {
    if (typeof window === 'undefined') return;
    localStorage.setItem('user', JSON.stringify(user));
  },

  removeUser() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('user');
  },

  async verifySession(token: string): Promise<UserSession | null> {
    try {
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

  async logout() {
    const user = this.getCurrentUser();
    if (user?.token) {
      try {
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

  getApiUrl(): string {
    if (typeof window === 'undefined') return '';
    return process.env.NEXT_PUBLIC_API_URL || '';
  },

  loginWithGoogle() {
    const apiUrl = this.getApiUrl();
    window.location.href = `${apiUrl}/auth/google/login`;
  },

  isAuthenticated(): boolean {
    const user = this.getCurrentUser();
    if (!user || !user.token) return false;
    
    try {
      const payload = JSON.parse(atob(user.token.split('.')[1]));
      const expiresAt = payload.exp * 1000; // Convert to milliseconds
      return expiresAt > Date.now();
    } catch (error) {
      console.error('JWT validation error:', error);
      return false;
    }
  }
};
