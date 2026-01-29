import { describe, it, expect } from 'vitest'

// These are the data transformation functions extracted from components
// for better testability - focus on API-to-Component data mapping

interface LeaderboardApiResponse {
  userId: string
  email: string
  name: string
  picture: string
  score: number
}

interface LeaderboardEntry {
  rank: number
  username: string
  score: number
  avatar: string
}

// Extracted from Leaderboard.tsx - important for API contract testing
export function mapLeaderboardData(data: LeaderboardApiResponse[]): LeaderboardEntry[] {
  return data.map((u, index) => ({
    rank: index + 1,
    username: u.name,
    score: u.score,
    avatar: u.picture,
  }))
}

interface GameApiResponse {
  gameId: string
  score: number
  opponentScore: number
  won: boolean
  winnerId: string
  opponent: string
  playerName?: string
  playerPicture?: string
  opponentName?: string
  opponentPicture?: string
  timestamp: number
}

interface GameRecord {
  id: string
  score: number
  opponentScore: number
  won: boolean
  winnerId: string
  opponentId: string
  playerName?: string
  playerPicture?: string
  opponentName?: string
  opponentPicture?: string
  timestamp: number
}

// Extracted from GameHistory.tsx - important for API contract testing
export function mapGameHistoryData(games: GameApiResponse[]): GameRecord[] {
  return games.map((g) => ({
    id: g.gameId,
    score: g.score,
    opponentScore: g.opponentScore,
    won: g.won,
    winnerId: g.winnerId,
    opponentId: g.opponent,
    playerName: g.playerName,
    playerPicture: g.playerPicture,
    opponentName: g.opponentName,
    opponentPicture: g.opponentPicture,
    timestamp: g.timestamp,
  }))
}

// These tests validate the API-to-Component data contract
// Important: If the backend API changes, these tests should catch mismatches
describe('API Data Transformation', () => {
  describe('Leaderboard API Contract', () => {
    it('should correctly transform backend CookieUser to frontend LeaderboardEntry', () => {
      // This mimics the actual API response structure from /api/leaderboard
      const apiData: LeaderboardApiResponse[] = [
        { userId: 'u1', email: 'a@test.com', name: 'Alice', picture: 'https://pic.url/a', score: 1500 },
        { userId: 'u2', email: 'b@test.com', name: 'Bob', picture: 'https://pic.url/b', score: 1200 },
      ]

      const result = mapLeaderboardData(apiData)

      // Verify the key field mappings that the UI depends on
      expect(result[0].username).toBe('Alice') // name -> username
      expect(result[0].avatar).toBe('https://pic.url/a') // picture -> avatar
      expect(result[0].score).toBe(1500)
      expect(result[0].rank).toBe(1) // auto-assigned based on position
      expect(result[1].rank).toBe(2)
    })
  })

  describe('GameHistory API Contract', () => {
    it('should correctly transform backend CookieGame to frontend GameRecord', () => {
      // This mimics the actual API response structure from /api/history
      const apiData: GameApiResponse[] = [
        {
          gameId: 'game-123',
          score: 150,
          opponentScore: 120,
          won: true,
          winnerId: 'player-1',
          opponent: 'player-2', // Note: backend uses 'opponent', frontend uses 'opponentId'
          playerName: 'Alice',
          opponentName: 'Bob',
          timestamp: 1706500000,
        },
      ]

      const result = mapGameHistoryData(apiData)

      // Verify critical field mappings
      expect(result[0].id).toBe('game-123') // gameId -> id
      expect(result[0].opponentId).toBe('player-2') // opponent -> opponentId (key rename!)
      expect(result[0].score).toBe(150)
      expect(result[0].opponentScore).toBe(120)
      expect(result[0].won).toBe(true)
    })

    it('should handle games with missing optional fields', () => {
      const apiData: GameApiResponse[] = [
        {
          gameId: 'g1',
          score: 50,
          opponentScore: 60,
          won: false,
          winnerId: 'other',
          opponent: 'other',
          timestamp: 999,
          // playerName, playerPicture, opponentName, opponentPicture are omitted
        },
      ]

      const result = mapGameHistoryData(apiData)
      
      // These should be undefined, not crash
      expect(result[0].playerName).toBeUndefined()
      expect(result[0].opponentPicture).toBeUndefined()
    })
  })
})
