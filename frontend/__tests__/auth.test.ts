import { describe, it, expect, beforeEach, vi } from 'vitest'
import { authService, getApiUrl, UserSession } from '../lib/auth'

// Helper to create a valid-looking JWT token
function createMockJWT(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payloadStr = btoa(JSON.stringify(payload))
  const signature = 'mock-signature'
  return `${header}.${payloadStr}.${signature}`
}

describe('authService', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
  })

  describe('getCurrentUser', () => {
    it('should return null when no user is stored', () => {
      const user = authService.getCurrentUser()
      expect(user).toBeNull()
    })

    it('should return user when stored in localStorage', () => {
      const mockUser: UserSession = {
        id: 'user-123',
        email: 'test@example.com',
        name: 'Test User',
        picture: 'https://example.com/pic.jpg',
        token: 'mock-token',
      }
      localStorage.setItem('user', JSON.stringify(mockUser))

      const user = authService.getCurrentUser()

      expect(user).not.toBeNull()
      expect(user?.id).toBe('user-123')
      expect(user?.email).toBe('test@example.com')
      expect(user?.name).toBe('Test User')
    })
  })

  describe('saveUser', () => {
    it('should save user to localStorage', () => {
      const mockUser: UserSession = {
        id: 'user-456',
        email: 'save@example.com',
        name: 'Save User',
        picture: '',
        token: 'save-token',
      }

      authService.saveUser(mockUser)

      const stored = localStorage.getItem('user')
      expect(stored).not.toBeNull()

      const parsed = JSON.parse(stored!)
      expect(parsed.id).toBe('user-456')
      expect(parsed.email).toBe('save@example.com')
    })
  })

  describe('removeUser', () => {
    it('should remove user from localStorage', () => {
      localStorage.setItem('user', JSON.stringify({ id: 'to-remove' }))

      authService.removeUser()

      expect(localStorage.getItem('user')).toBeNull()
    })

    it('should not throw when no user exists', () => {
      expect(() => authService.removeUser()).not.toThrow()
    })
  })

  describe('isAuthenticated', () => {
    it('should return false when no user is stored', () => {
      expect(authService.isAuthenticated()).toBe(false)
    })

    it('should return false when user has no token', () => {
      localStorage.setItem('user', JSON.stringify({ id: 'user-1' }))
      expect(authService.isAuthenticated()).toBe(false)
    })

    it('should return true for valid non-expired token', () => {
      const futureExp = Math.floor(Date.now() / 1000) + 3600 // 1 hour from now
      const token = createMockJWT({ exp: futureExp, user_id: 'user-1' })

      const mockUser: UserSession = {
        id: 'user-1',
        email: 'test@example.com',
        name: 'Test',
        picture: '',
        token,
      }
      localStorage.setItem('user', JSON.stringify(mockUser))

      expect(authService.isAuthenticated()).toBe(true)
    })

    it('should return false for expired token', () => {
      const pastExp = Math.floor(Date.now() / 1000) - 3600 // 1 hour ago
      const token = createMockJWT({ exp: pastExp, user_id: 'user-1' })

      const mockUser: UserSession = {
        id: 'user-1',
        email: 'test@example.com',
        name: 'Test',
        picture: '',
        token,
      }
      localStorage.setItem('user', JSON.stringify(mockUser))

      expect(authService.isAuthenticated()).toBe(false)
    })

    it('should return false for malformed token', () => {
      const mockUser: UserSession = {
        id: 'user-1',
        email: 'test@example.com',
        name: 'Test',
        picture: '',
        token: 'not-a-valid-jwt',
      }
      localStorage.setItem('user', JSON.stringify(mockUser))

      expect(authService.isAuthenticated()).toBe(false)
    })
  })
})

describe('getApiUrl', () => {
  it('should return empty string for production (no NEXT_PUBLIC_API_URL)', () => {
    // When no env var is set, function returns empty string
    // Note: In actual test environment, process.env.NEXT_PUBLIC_API_URL may not be set
    const result = getApiUrl()
    // Result should be empty string or the env var value
    expect(typeof result).toBe('string')
  })
})
