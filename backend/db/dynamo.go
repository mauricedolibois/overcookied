package db

import (
	"context"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var svc *dynamodb.Client

func Init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc = dynamodb.NewFromConfig(cfg)
	log.Println("DynamoDB Session Initialized")

	// DIAGNOSTIC INFO
	stsSvc := sts.NewFromConfig(cfg)
	identity, err := stsSvc.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("DIAGNOSTIC ERROR: Could not get AWS identity: %v", err)
	} else {
		log.Printf("DIAGNOSTIC: Operating as Account: %s, ARN: %s", *identity.Account, *identity.Arn)
	}
	log.Printf("DIAGNOSTIC: Region: %s", cfg.Region)

	// Note: ListTables is optional diagnostic info, don't fail if not permitted
	tables, err := svc.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		log.Printf("DIAGNOSTIC: Could not list tables (permission may be restricted): %v", err)
	} else {
		log.Printf("DIAGNOSTIC: Found Tables: %v", tables.TableNames)
	}
}

// Model: CookieUser
type CookieUser struct {
	UserID  string `json:"userId" dynamodbav:"UserID"`
	Email   string `json:"email" dynamodbav:"Email"`
	Name    string `json:"name" dynamodbav:"Name"`
	Picture string `json:"picture" dynamodbav:"Picture"`
	Score   int    `json:"score" dynamodbav:"Score"` // Total Score
}

// Model: CookieGame
type CookieGame struct {
	GameID          string `json:"gameId" dynamodbav:"GameID"`
	PlayerID        string `json:"playerId" dynamodbav:"PlayerID"`   // Partition Key for GSI
	Timestamp       int64  `json:"timestamp" dynamodbav:"Timestamp"` // Sort Key for GSI
	Score           int    `json:"score" dynamodbav:"Score"`
	OpponentScore   int    `json:"opponentScore" dynamodbav:"OpponentScore"`
	Reason          string `json:"reason" dynamodbav:"Reason"`
	Won             bool   `json:"won" dynamodbav:"Won"`
	WinnerID        string `json:"winnerId" dynamodbav:"WinnerID"`
	Opponent        string `json:"opponent" dynamodbav:"Opponent"` // ID
	PlayerName      string `json:"playerName" dynamodbav:"PlayerName"`
	PlayerPicture   string `json:"playerPicture" dynamodbav:"PlayerPicture"`
	OpponentName    string `json:"opponentName" dynamodbav:"OpponentName"`
	OpponentPicture string `json:"opponentPicture" dynamodbav:"OpponentPicture"`
}

const TableUsers = "CookieUsers"
const TableGames = "CookieGames"

// --- User Operations ---

func SaveUser(user CookieUser) error {
	// Only put if not exists, or update mostly login non-stat fields
	// For simplicity, we PUT, but we must be careful not to overwrite score if we just logged in.
	// Actually auth logic handles creating session. We only want to create user if new.

	// Check if user exists
	existing, err := GetUser(user.UserID)
	if err == nil && existing != nil {
		// Update profile info only
		_, err = svc.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
			TableName: aws.String(TableUsers),
			Key: map[string]types.AttributeValue{
				"UserID": &types.AttributeValueMemberS{Value: user.UserID},
			},
			UpdateExpression: aws.String("set Picture = :p, #N = :n"),
			ExpressionAttributeNames: map[string]string{
				"#N": "Name", // Name is reserved
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":p": &types.AttributeValueMemberS{Value: user.Picture},
				":n": &types.AttributeValueMemberS{Value: user.Name},
			},
		})
		if err == nil {
			log.Printf("[DB] Updated user profile for: %s", user.Email)
		} else {
			log.Printf("[DB] Error updating user profile: %v", err)
		}
		return err
	}

	// New User
	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return err
	}
	_, err = svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(TableUsers),
		Item:      av,
	})
	if err == nil {
		log.Printf("[DB] Created new user: %s", user.Email)
	} else {
		log.Printf("[DB] Error creating user: %v", err)
	}
	return err
}

func GetUser(userID string) (*CookieUser, error) {
	out, err := svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(TableUsers),
		Key: map[string]types.AttributeValue{
			"UserID": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil // Not found
	}

	var user CookieUser
	err = attributevalue.UnmarshalMap(out.Item, &user)
	return &user, err
}

func UpdateUserStats(userID string, score int) error {
	_, err := svc.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(TableUsers),
		Key: map[string]types.AttributeValue{
			"UserID": &types.AttributeValueMemberS{Value: userID},
		},
		UpdateExpression: aws.String("set Score = if_not_exists(Score, :zero) + :s"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":    &types.AttributeValueMemberN{Value: strconv.Itoa(score)},
			":zero": &types.AttributeValueMemberN{Value: "0"},
		},
	})
	if err == nil {
		log.Printf("[DB] Updated stats for user %s: +%d score", userID, score)
	}
	return err
}

func GetLeaderboard(limit int) ([]CookieUser, error) {
	// Full Scan + Sort (Okay for < 10k users)
	out, err := svc.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(TableUsers),
	})
	if err != nil {
		return nil, err
	}

	var users []CookieUser
	err = attributevalue.UnmarshalListOfMaps(out.Items, &users)
	if err != nil {
		return nil, err
	}

	// Sort Descending by Score
	sort.Slice(users, func(i, j int) bool {
		return users[i].Score > users[j].Score
	})

	if len(users) > limit {
		users = users[:limit]
	}
	return users, nil
}

// --- Game History Operations ---

func SaveGame(game CookieGame) error {
	av, err := attributevalue.MarshalMap(game)
	if err != nil {
		return err
	}
	_, err = svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(TableGames),
		Item:      av,
	})
	if err == nil {
		log.Printf("[DB] Saved game record %s for player %s (Won: %v)", game.GameID, game.PlayerID, game.Won)
	} else {
		log.Printf("[DB] Error saving game: %v", err)
	}
	return err
}

func GetGameHistory(userID string, limit int32) ([]CookieGame, error) {
	out, err := svc.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(TableGames),
		IndexName:              aws.String("PlayerHistoryIndex"),
		KeyConditionExpression: aws.String("PlayerID = :pid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(false), // Descending timestamp
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var games []CookieGame
	err = attributevalue.UnmarshalListOfMaps(out.Items, &games)
	if err != nil {
		return nil, err
	}

	if len(games) == 0 {
		return games, nil
	}

	// 2. Enrich with User Data (BatchGetItem)
	// Collect all unique user IDs (players and opponents)
	userIDs := make(map[string]bool)
	for _, g := range games {
		if g.PlayerID != "" {
			userIDs[g.PlayerID] = true
		}
		if g.Opponent != "" {
			userIDs[g.Opponent] = true
		}
	}

	keys := []map[string]types.AttributeValue{}
	for uid := range userIDs {
		keys = append(keys, map[string]types.AttributeValue{
			"UserID": &types.AttributeValueMemberS{Value: uid},
		})
	}

	if len(keys) == 0 {
		return games, nil
	}

	// DynamoDB BatchGetItem (limit 100, we retrieve max 20 games * 2 users = 40 keys, so safe)
	batchOut, err := svc.BatchGetItem(context.TODO(), &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			TableUsers: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		log.Printf("[DB] Error batch fetching users for history: %v", err)
		// Return games anyway, just without enriched info (or with snapshot info)
		return games, nil
	}

	// Unmarshal Users
	var users []CookieUser
	err = attributevalue.UnmarshalListOfMaps(batchOut.Responses[TableUsers], &users)
	if err != nil {
		log.Printf("[DB] Error unmarshaling batch users: %v", err)
		return games, nil
	}

	// Create Map for lookup
	userMap := make(map[string]CookieUser)
	for _, u := range users {
		userMap[u.UserID] = u
	}

	// 3. Populate Game Structs
	for i := range games {
		if u, ok := userMap[games[i].PlayerID]; ok {
			games[i].PlayerName = u.Name
			games[i].PlayerPicture = u.Picture
		}
		if u, ok := userMap[games[i].Opponent]; ok {
			games[i].OpponentName = u.Name
			games[i].OpponentPicture = u.Picture
		}
	}

	return games, nil
}

// CountGamesByPlayer returns the total number of games for a player
func CountGamesByPlayer(userID string) (int, error) {
	out, err := svc.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(TableGames),
		IndexName:              aws.String("PlayerHistoryIndex"),
		KeyConditionExpression: aws.String("PlayerID = :pid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pid": &types.AttributeValueMemberS{Value: userID},
		},
		Select: types.SelectCount,
	})
	if err != nil {
		return 0, err
	}
	return int(out.Count), nil
}
