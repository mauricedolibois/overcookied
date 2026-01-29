import { describe, it, expect } from 'vitest'
import { getWsUrl } from '../app/hooks/useGameSocket'

describe('getWsUrl', () => {
  describe('with API URL (development mode)', () => {
    it('should return ws:// URL for http API', () => {
      const result = getWsUrl('http://localhost:8080')
      expect(result).toBe('ws://localhost:8080/ws')
    })

    it('should return wss:// URL for https API', () => {
      const result = getWsUrl('https://api.example.com')
      expect(result).toBe('wss://api.example.com/ws')
    })

    it('should strip protocol from API URL', () => {
      const result = getWsUrl('http://dev-server:3000')
      expect(result).toBe('ws://dev-server:3000/ws')
    })

    it('should handle API URL with trailing path', () => {
      const result = getWsUrl('https://api.example.com:8443')
      expect(result).toBe('wss://api.example.com:8443/ws')
    })
  })

  describe('without API URL (production mode)', () => {
    it('should derive wss:// from https window location', () => {
      const mockLocation = {
        protocol: 'https:',
        host: 'overcookied.example.com',
      }
      const result = getWsUrl(undefined, mockLocation)
      expect(result).toBe('wss://overcookied.example.com/ws')
    })

    it('should derive ws:// from http window location', () => {
      const mockLocation = {
        protocol: 'http:',
        host: 'localhost:3000',
      }
      const result = getWsUrl(undefined, mockLocation)
      expect(result).toBe('ws://localhost:3000/ws')
    })

    it('should include port from window location', () => {
      const mockLocation = {
        protocol: 'https:',
        host: 'game.example.com:8080',
      }
      const result = getWsUrl(undefined, mockLocation)
      expect(result).toBe('wss://game.example.com:8080/ws')
    })
  })

  describe('edge cases', () => {
    it('should prefer API URL over window location when both provided', () => {
      const mockLocation = {
        protocol: 'https:',
        host: 'production.example.com',
      }
      const result = getWsUrl('http://localhost:8080', mockLocation)
      expect(result).toBe('ws://localhost:8080/ws')
    })

    it('should handle empty string API URL as falsy', () => {
      const mockLocation = {
        protocol: 'https:',
        host: 'production.example.com',
      }
      const result = getWsUrl('', mockLocation)
      expect(result).toBe('wss://production.example.com/ws')
    })
  })
})
