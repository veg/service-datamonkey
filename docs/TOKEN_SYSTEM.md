# Datamonkey Token System

## Overview
The Datamonkey API uses JWT (JSON Web Tokens) for user authentication and authorization. Tokens identify users and control access to their resources (datasets, jobs, conversations).

## Token Structure

### JWT Claims
Tokens contain the following claims:

```json
{
  "sub": "user-id-123",           // Subject: User identifier
  "type": "user",                  // Token type
  "iat": 1697558400,              // Issued at (Unix timestamp)
  "exp": 1697644800               // Expires at (Unix timestamp)
}
```

### Key Components
- **`sub` (Subject)**: Unique user identifier - this is what identifies the user across the system
- **`type`**: Token type (always "user" for user tokens)
- **`iat` (Issued At)**: When the token was created
- **`exp` (Expiration)**: When the token expires (default: 24 hours)

## What Tokens Represent

### User Identity
The token's `sub` claim is the **user ID** that:
- Identifies who owns datasets
- Identifies who owns jobs
- Identifies who owns conversations
- Controls access to resources

**Important**: The user ID in the token is the source of truth for ownership. When you create a dataset, job, or conversation, it's associated with the user ID from your token.

### Resource Ownership
```
Token (sub: "alice") → Can access:
  ✅ Datasets created by "alice"
  ✅ Jobs submitted by "alice"
  ✅ Conversations created by "alice"
  ❌ Resources owned by "bob"
```

## Token Generation

### Configuration
Tokens are generated using:
- **Secret Key**: HMAC-SHA256 signing key (from `USER_JWT_KEY_PATH` or `JWT_KEY_PATH` in .env)
- **Expiration**: 24 hours (configurable via `ExpirationSecs`)
- **Algorithm**: HS256 (HMAC with SHA-256)

### Generating Tokens

#### Using the API (if endpoint exists)
```bash
# Generate token via API
curl -X POST http://localhost:9300/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "alice"}'
```

#### Using the CLI Script
```bash
# Generate a test token
./bin/generate-test-token.sh alice

# Output:
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTc2NDQ4MDAsImlhdCI6MTY5NzU1ODQwMCwic3ViIjoiYWxpY2UiLCJ0eXBlIjoidXNlciJ9.signature
```

#### Programmatically (Go)
```go
tokenService := NewTokenService(TokenConfig{
    KeyPath:        "/path/to/jwt.key",
    ExpirationSecs: 86400, // 24 hours
})

token, err := tokenService.GenerateUserToken("alice")
```

## Token Validation

### How Validation Works
1. **Extract token** from request (query param `user_token` or header `user_token` or `Authorization: Bearer`)
2. **Verify signature** using the secret key
3. **Check expiration** - reject if expired
4. **Extract user ID** from `sub` claim
5. **Verify resource ownership** (if accessing a specific resource)

### Validation Flow
```
Request → Extract Token → Verify Signature → Check Expiration
    ↓
Extract User ID (sub claim)
    ↓
Check Resource Ownership (if needed)
    ↓
Allow/Deny Access
```

## Using Tokens in Requests

### Method 1: Query Parameter
```bash
curl "http://localhost:9300/api/v1/datasets?user_token=YOUR_TOKEN"
```

### Method 2: Custom Header
```bash
curl http://localhost:9300/api/v1/datasets \
  -H "user_token: YOUR_TOKEN"
```

### Method 3: Authorization Header (Standard)
```bash
curl http://localhost:9300/api/v1/datasets \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Authorization Model

### Resource Access Control
The system enforces **user-based isolation**:

```go
// When creating a dataset
userID := extractFromToken(request) // e.g., "alice"
dataset.OwnerID = userID

// When accessing a dataset
requestUserID := extractFromToken(request)
datasetOwnerID := getDatasetOwner(datasetID)

if requestUserID != datasetOwnerID {
    return 403 Forbidden
}
```

### Access Checks
- **Datasets**: `CheckDatasetAccess()` - verifies user owns the dataset
- **Jobs**: `CheckJobAccess()` - verifies user owns the job
- **Conversations**: `CheckConversationAccess()` - verifies user owns the conversation

## Security Considerations

### Token Security
- ✅ Tokens are signed with HMAC-SHA256
- ✅ Tokens expire after 24 hours
- ✅ Secret key is stored securely (not in code)
- ✅ Signature prevents tampering

### Best Practices
1. **Never share tokens** - each user should have their own
2. **Store securely** - don't commit tokens to git
3. **Use HTTPS** - in production to prevent token interception
4. **Rotate keys** - periodically change the JWT secret key
5. **Short expiration** - 24 hours is reasonable for testing, consider shorter for production

### What Tokens DON'T Do
- ❌ Tokens don't encrypt data (they're signed, not encrypted)
- ❌ Tokens don't prevent replay attacks (within expiration window)
- ❌ Tokens don't provide rate limiting
- ❌ Tokens don't track sessions (stateless)

## Testing with Tokens

### Quick Test Flow
```bash
# 1. Generate a token for user "alice"
TOKEN=$(./bin/generate-test-token.sh alice)

# 2. Create a dataset as alice
curl -X POST http://localhost:9300/api/v1/datasets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "alice-data", "type": "fasta", "content": ">seq\nATGC"}'

# 3. List alice's datasets
curl http://localhost:9300/api/v1/datasets \
  -H "Authorization: Bearer $TOKEN"

# 4. Try to access as different user (should fail)
TOKEN_BOB=$(./bin/generate-test-token.sh bob)
curl http://localhost:9300/api/v1/datasets/alice-dataset-id \
  -H "Authorization: Bearer $TOKEN_BOB"
# Expected: 403 Forbidden
```

### Multi-User Testing
```bash
# Create tokens for different users
export ALICE_TOKEN=$(./bin/generate-test-token.sh alice)
export BOB_TOKEN=$(./bin/generate-test-token.sh bob)

# Run tests with different users
./bin/test-priority1.sh http://localhost:9300 $ALICE_TOKEN
./bin/test-priority1.sh http://localhost:9300 $BOB_TOKEN
```

## Token Lifecycle

```
┌─────────────────┐
│ Generate Token  │
│  (user_id: alice)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Token Created   │
│ exp: +24 hours  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Use Token       │
│ (make requests) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Token Expires   │
│ (after 24h)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Generate New    │
│ Token           │
└─────────────────┘
```

## Troubleshooting

### "Invalid token" Error
- Check token hasn't expired (24 hour limit)
- Verify correct JWT key is being used
- Ensure token wasn't modified/corrupted

### "Missing user token" Error
- Token not provided in request
- Check header/query parameter name
- Verify token is being passed correctly

### "User does not have access" Error
- Token user ID doesn't match resource owner
- Trying to access another user's resource
- Check token's `sub` claim matches expected user

### Debugging Tokens
```bash
# Decode token (without verification) to see claims
echo "YOUR_TOKEN" | cut -d'.' -f2 | base64 -d | jq .

# Example output:
# {
#   "sub": "alice",
#   "type": "user",
#   "iat": 1697558400,
#   "exp": 1697644800
# }
```

## Environment Configuration

### Required Environment Variables
```bash
# In .env file
USER_JWT_KEY_PATH=/path/to/jwt.key    # Path to JWT signing key
USER_TOKEN_ENABLED=true                # Enable token validation
```

### Key File Format
The JWT key file should contain a secret string (minimum 32 characters recommended):
```
your-secret-key-here-minimum-32-chars-recommended
```

## Summary

**Tokens are user identifiers** that:
- Contain a user ID in the `sub` claim
- Are signed to prevent tampering
- Expire after 24 hours
- Control access to user-owned resources
- Enable multi-user isolation in the API

The `sub` claim is the **most important part** - it's what identifies who you are and what you can access.
