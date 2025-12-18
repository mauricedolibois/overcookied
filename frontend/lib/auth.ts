export interface UserSession {
  id: string;
  email: string;
  name: string;
  picture: string;
  token: string;
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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
      const response = await fetch(`${API_URL}/auth/verify`, {
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
        await fetch(`${API_URL}/auth/logout`, {
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

  // Initiate Google OAuth login
  loginWithGoogle() {
    window.location.href = `${API_URL}/auth/google/login`;
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
