# JWT Authentication Implementation

## Overview

The authentication system has been updated to use **JWT (JSON Web Tokens)** for secure, stateless authentication.

## What Changed

### Backend Changes

1. **JWT Library**: Added `github.com/golang-jwt/jwt/v5` for token generation and validation

2. **Token Generation**: Instead of random session tokens, the system now generates JWT tokens containing:
   - User ID
   - Email
   - Name
   - Profile picture URL
   - Standard claims (expiration, issued at, issuer)

3. **Stateless Authentication**: 
   - No more in-memory session storage
   - Tokens are self-contained and validated cryptographically
   - Each token is signed with a secret key (JWT_SECRET)

4. **Token Expiration**: Tokens expire after 24 hours

### Frontend Changes

1. **JWT Validation**: Client-side validation checks token expiration before making requests
2. **Token Storage**: Unchanged - still uses localStorage (consider httpOnly cookies for production)
3. **Token Format**: Updated to handle JWT structure (header.payload.signature)

## JWT Structure

A JWT token consists of three parts separated by dots:
```
header.payload.signature
```

**Example Payload**:
```json
{
  "user_id": "123456789",
  "email": "user@example.com",
  "name": "John Doe",
  "picture": "https://example.com/photo.jpg",
  "exp": 1733961600,
  "iat": 1733875200,
  "nbf": 1733875200,
  "iss": "overcookied"
}
```

## Benefits

1. **Stateless**: Server doesn't need to store session data
2. **Scalable**: Works across multiple servers without shared storage
3. **Secure**: Cryptographically signed tokens prevent tampering
4. **Self-contained**: All user information is in the token
5. **Standard**: Industry-standard authentication method

## Security Considerations

### Current Implementation
- ✅ Tokens signed with HS256 (HMAC-SHA256)
- ✅ 24-hour expiration
- ✅ Secure secret generation
- ✅ CORS protection

### Production Recommendations
1. **Use httpOnly Cookies**: Store JWT in httpOnly cookies instead of localStorage to prevent XSS attacks
2. **Implement Refresh Tokens**: Allow users to get new tokens without re-authenticating
3. **Token Blacklisting**: For logout functionality, consider maintaining a blacklist of revoked tokens
4. **Short Expiration**: Consider shorter expiration times (e.g., 15 minutes) with refresh tokens
5. **HTTPS Only**: Always use HTTPS in production
6. **Strong Secret**: Use a cryptographically secure random string (at least 32 characters)

## API Usage

### Authentication Flow

1. User clicks "Continue with Google"
2. User authenticates with Google OAuth
3. Backend receives user info and generates JWT
4. Frontend receives JWT and stores it
5. Frontend includes JWT in Authorization header for all requests

### Making Authenticated Requests

```typescript
const response = await fetch(`${API_URL}/api/endpoint`, {
  headers: {
    'Authorization': `Bearer ${user.token}`,
    'Content-Type': 'application/json',
  },
});
```

### Verifying Token

```go
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
