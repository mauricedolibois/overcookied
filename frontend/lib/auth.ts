export interface UserSession {
  id: string;
  email: string;
  name: string;
  picture: string;
  token: string;
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
      // Use relative URL - works because frontend and backend share the same host
      const response = await fetch('/auth/verify', {
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
        // Use relative URL
        await fetch('/auth/logout', {
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
    // Use relative URL - the browser will use the current origin
    window.location.href = '/auth/google/login';
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
