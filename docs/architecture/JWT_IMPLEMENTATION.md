# JWT Authentication Implementation

## Overview

The authentication system uses **JWT (JSON Web Tokens)** for secure, stateless authentication. This is the primary authentication mechanism for WebSocket connections and API requests.

## Current Implementation Details

### Backend Stack

1. **JWT Library**: `github.com/golang-jwt/jwt/v5` v5.2.1 for token generation and validation
2. **Token Generation**: Occurs after successful Google OAuth callback
3. **Signing Algorithm**: HS256 (HMAC-SHA256)
4. **Token Format**: Standard JWT (header.payload.signature)

### Token Payload

```json
{
  "user_id": "123456789",
  "email": "user@example.com", 
  "name": "John Doe",
  "picture": "https://lh3.googleusercontent.com/...",
  "exp": 1734051200,
  "iat": 1733964800,
  "nbf": 1733964800,
  "iss": "overcookied"
}
```

**Token Expiration**: 24 hours (86,400 seconds) from issue time

### Authentication Flow

1. **Step 1 - User Initiates Login**: User clicks "Login with Google" on frontend
2. **Step 2 - OAuth Redirect**: Frontend redirects to `/auth/google/login` (backend)
3. **Step 3 - Google OAuth**: Backend redirects to Google OAuth provider
4. **Step 4 - User Consent**: User grants permissions to access profile
5. **Step 5 - OAuth Callback**: Google redirects back to `/auth/google/callback?code=...&state=...`
6. **Step 6 - Token Exchange**: Backend exchanges authorization code for user profile
7. **Step 7 - JWT Creation**: Backend creates JWT with user info and secret
8. **Step 8 - Redirect to Frontend**: Backend redirects to `/dashboard?token=JWT&user=BASE64`
9. **Step 9 - Token Storage**: Frontend stores JWT in localStorage as part of UserSession
10. **Step 10 - Authenticated Requests**: All subsequent requests include `Authorization: Bearer JWT`

### Frontend Storage

```typescript
type UserSession = {
  id: string;           // Google user ID
  email: string;
  name: string;
  picture: string;
  token: string;        // JWT token
};

// Stored in localStorage
localStorage.setItem('user', JSON.stringify(userSession));
```

### WebSocket Authentication

The WebSocket connection uses JWT tokens for authentication:

```typescript
// Frontend
const ws = new WebSocket(`ws://localhost:8080/ws?token=${encodeURIComponent(user.token)}`);

// Backend
// Token is extracted from query params and validated before upgrading connection
```

### API Endpoints Protected by JWT

- `GET /api/leaderboard` - Public (no auth required)
- `GET /api/history?userId=...` - Public user history
- `POST /ws` - Requires valid JWT in query parameter
- `GET /auth/verify` - Validates and refreshes JWT tokengo
claims, err := verifyJWT(tokenString)
if err != nil {
    // Token is invalid or expired
    return
}
// Use claims.UserID, claims.Email, etc.
```

## Environment Variables

Add to your `.env` file:
```bash
JWT_SECRET=your_secure_random_string_at_least_32_characters
```

Generate a secure secret:
```bash
openssl rand -base64 32
```

## Future Enhancements

1. **Refresh Token Flow**: Implement refresh tokens for seamless re-authentication
2. **Token Revocation**: Add ability to revoke specific tokens
3. **Multiple Device Management**: Track active sessions per user
4. **Rate Limiting**: Add rate limiting for token refresh endpoints
5. **Token Encryption**: Consider encrypting sensitive data in JWT payload
