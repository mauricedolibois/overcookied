package db

import (
	"log"

	"github.com/mauricedolibois/overcookied/backend/mocks"
)

// useMocks indicates whether to use mock implementations
var useMocks bool

// InitWithMocks initializes the database layer with mock support
func InitWithMocks() {
	useMocks = mocks.IsMockMode()

	if useMocks {
		log.Println("[DB] Running in MOCK MODE - using in-memory database")
		// Initialize mock
		mocks.GetMockDynamoDB()
	} else {
		// Initialize real DynamoDB
		Init()
	}
}

// SaveUserWithMock saves a user (mock or real)
func SaveUserWithMock(user CookieUser) error {
	if useMocks {
		mockUser := mocks.CookieUser{
			UserID:  user.UserID,
			Email:   user.Email,
			Name:    user.Name,
			Picture: user.Picture,
			Score:   user.Score,
		}
		return mocks.GetMockDynamoDB().SaveUser(mockUser)
	}
	return SaveUser(user)
}

// GetUserWithMock retrieves a user (mock or real)
func GetUserWithMock(userID string) (*CookieUser, error) {
	if useMocks {
		mockUser, err := mocks.GetMockDynamoDB().GetUser(userID)
		if err != nil || mockUser == nil {
			return nil, err
		}
		return &CookieUser{
			UserID:  mockUser.UserID,
			Email:   mockUser.Email,
			Name:    mockUser.Name,
			Picture: mockUser.Picture,
			Score:   mockUser.Score,
		}, nil
	}
	return GetUser(userID)
}

// GetLeaderboardWithMock retrieves the leaderboard (mock or real)
func GetLeaderboardWithMock(limit int) ([]CookieUser, error) {
	if useMocks {
		mockUsers, err := mocks.GetMockDynamoDB().GetTopUsers(limit)
		if err != nil {
			return nil, err
		}
		users := make([]CookieUser, len(mockUsers))
		for i, mu := range mockUsers {
			users[i] = CookieUser{
				UserID:  mu.UserID,
				Email:   mu.Email,
				Name:    mu.Name,
				Picture: mu.Picture,
				Score:   mu.Score,
			}
		}
		return users, nil
	}
	return GetLeaderboard(limit)
}

// UpdateUserStatsWithMock updates user stats (mock or real)
func UpdateUserStatsWithMock(userID string, score int) error {
	if useMocks {
		return mocks.GetMockDynamoDB().IncrementUserScore(userID, score)
	}
	return UpdateUserStats(userID, score)
}

// SaveGameWithMock saves a game record (mock or real)
func SaveGameWithMock(game CookieGame) error {
	if useMocks {
		mockGame := mocks.CookieGame{
			GameID:          game.GameID,
			PlayerID:        game.PlayerID,
			Timestamp:       game.Timestamp,
			Score:           game.Score,
			OpponentScore:   game.OpponentScore,
			Reason:          game.Reason,
			Won:             game.Won,
			WinnerID:        game.WinnerID,
			Opponent:        game.Opponent,
			PlayerName:      game.PlayerName,
			PlayerPicture:   game.PlayerPicture,
			OpponentName:    game.OpponentName,
			OpponentPicture: game.OpponentPicture,
		}
		return mocks.GetMockDynamoDB().SaveGame(mockGame)
	}
	return SaveGame(game)
}

// GetGameHistoryWithMock retrieves game history (mock or real)
func GetGameHistoryWithMock(userID string, limit int32) ([]CookieGame, error) {
	if useMocks {
		mockGames, err := mocks.GetMockDynamoDB().GetGamesByPlayer(userID, int(limit))
		if err != nil {
			return nil, err
		}
		games := make([]CookieGame, len(mockGames))
		for i, mg := range mockGames {
			games[i] = CookieGame{
				GameID:          mg.GameID,
				PlayerID:        mg.PlayerID,
				Timestamp:       mg.Timestamp,
				Score:           mg.Score,
				OpponentScore:   mg.OpponentScore,
				Reason:          mg.Reason,
				Won:             mg.Won,
				WinnerID:        mg.WinnerID,
				Opponent:        mg.Opponent,
				PlayerName:      mg.PlayerName,
				PlayerPicture:   mg.PlayerPicture,
				OpponentName:    mg.OpponentName,
				OpponentPicture: mg.OpponentPicture,
			}
		}
		return games, nil
	}
	return GetGameHistory(userID, limit)
}

// CountGamesByPlayerWithMock returns the total number of games for a player (mock or real)
func CountGamesByPlayerWithMock(userID string) (int, error) {
	if useMocks {
		return mocks.GetMockDynamoDB().CountGamesByPlayer(userID), nil
	}
	return CountGamesByPlayer(userID)
}

// IsMockMode returns whether mock mode is enabled
func IsMockMode() bool {
	return useMocks
}
