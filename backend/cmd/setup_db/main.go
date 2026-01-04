package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env from parent directory
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: No .env file found in ../../, checking current dir")
		if err := godotenv.Load(".env"); err != nil {
			log.Println("Warning: No .env file found")
		}
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc := dynamodb.NewFromConfig(cfg)

	recreateTableUsers(svc)
	recreateTableGames(svc)
	log.Println("Database setup complete!")
}

func deleteTableIfExists(svc *dynamodb.Client, tableName string) {
	log.Printf("Deleting old table %s if it exists...", tableName)
	_, err := svc.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		// Ignore ResourceNotFoundException
		log.Printf("DeleteTable %s skipped (or error): %v", tableName, err)
		return
	}

	// Wait for deletion
	log.Printf("Waiting for table %s to be deleted...", tableName)
	for {
		_, err := svc.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{TableName: aws.String(tableName)})
		if err != nil {
			// If error is ResourceNotFoundException, we are good
			break
		}
		time.Sleep(2 * time.Second)
	}
	log.Printf("Table %s deleted.", tableName)
}

func recreateTableUsers(svc *dynamodb.Client) {
	tableName := "CookieUsers"
	deleteTableIfExists(svc, tableName)
	log.Printf("Creating table %s...", tableName)

	_, err := svc.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("UserID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("UserID"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		log.Printf("Could not create table %s: %v", tableName, err)
	} else {
		log.Printf("Table %s created successfully", tableName)
	}
}

func recreateTableGames(svc *dynamodb.Client) {
	tableName := "CookieGames"
	deleteTableIfExists(svc, tableName)
	log.Printf("Creating table %s...", tableName)

	_, err := svc.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("GameID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("PlayerID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("Timestamp"),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("GameID"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("PlayerID"),
				KeyType:       types.KeyTypeRange,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("PlayerHistoryIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("PlayerID"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("Timestamp"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		log.Printf("Could not create table %s: %v", tableName, err)
	} else {
		log.Printf("Table %s created successfully", tableName)
	}
}
