# AWS DynamoDB Setup Guide

Please create the following tables in your AWS Console (Region: `eu-central-1` or your preferred region).

## 1. Table: `CookieUsers`
This table stores user profiles and total stats (Global Leaderboard).

- **Partition Key**: `UserID` (String)
- **Sort Key**: None
- **Indexes**: None required for simple Scan (for Top 10), but for scale you might want a GSI on `Type` (PK) and `Score` (SK). For now, basic is fine.

## 2. Table: `CookieGames`
This table stores the history of played games.

- **Partition Key**: `GameID` (String)
- **Sort Key**: None
- **Global Secondary Index (GSI)**:
    - **Index Name**: `PlayerHistoryIndex`
    - **Partition Key**: `PlayerID` (String)
    - **Sort Key**: `Timestamp` (Number)
    - **Projection**: ALL

## 3. IAM Permissions
Ensure your IAM User (whose keys are in `.env`) has:
- `AmazonDynamoDBFullAccess` (or specific permissions for `PutItem`, `GetItem`, `Query`, `Scan`, `UpdateItem` on these tables).
