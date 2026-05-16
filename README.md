# NVide Live Platform - Fase 1: Foundation & Authentication

Backend platform live streaming dengan Clean Architecture menggunakan Go.

## Tech Stack

- **Framework HTTP**: Gorilla Mux
- **Database**: PostgreSQL via Supabase (pgx/v5)
- **Cache**: Redis Cloud (go-redis/v9)
- **Auth**: JWT (golang-jwt/jwt/v5) + bcrypt
- **RBAC**: Permission matrix dengan 5 roles
- **Logging**: Uber Zap

## Project Structure

```
nvide-live/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── domain/                   # Domain entities & interfaces
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── token.go
│   │   ├── errors.go
│   │   └── uuid.go
│   ├── usecase/                  # Business logic
│   │   ├── auth.go
│   │   └── user.go
│   ├── repository/               # Data access layer
│   │   ├── user.go
│   │   ├── token.go
│   │   ├── role.go
│   │   └── permission.go
│   ├── delivery/                 # HTTP handlers & router
│   │   ├── handlers.go
│   │   └── router.go
│   └── middleware/               # HTTP middleware
│       ├── auth.go
│       ├── rbac.go
│       ├── rate_limit.go
│       └── common.go
├── pkg/
│   ├── auth/                     # JWT & bcrypt utilities
│   │   └── jwt.go
│   ├── rbac/                     # RBAC manager
│   │   └── manager.go
│   ├── config/                   # Configuration loader
│   │   └── config.go
│   ├── database/                 # Database connection
│   │   └── postgres.go
│   └── redis/                    # Redis client
│       └── client.go
├── migrations/
│   ├── 001_initial_schema.sql    # Database schema
│   └── 002_seed_data.sql         # Seed data (roles, permissions)
├── go.mod
├── go.sum
├── Makefile
├── .env.example
└── README.md
```

## Features (Fase 1)

### Authentication
- [x] User registration with validation
- [x] User login (email + password)
- [x] JWT access token (15 minutes)
- [x] Refresh token (7 days) with rotation
- [x] Token blacklist in Redis
- [x] Logout (invalidate tokens)
- [x] Get current user profile (/auth/me)

### RBAC (Role-Based Access Control)
- [x] 5 roles: guest, user, host, agency, admin
- [x] Comprehensive permission matrix
- [x] Permission-based middleware
- [x] Role-based middleware
- [x] Resource ownership checking

### Security
- [x] Bcrypt password hashing (cost 12)
- [x] JWT with HS256
- [x] Token blacklist in Redis
- [x] Rate limiting per IP
- [x] CORS support
- [x] Request logging

## Setup & Installation

### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- Make (optional)

### 1. Clone & Dependencies
```bash
cd f:/nvide
go mod download
go mod tidy
```

### 2. Database Setup
```bash
# Create database
createdb nvide_live

# Run migrations (using psql)
psql -U postgres -d nvide_live -f migrations/001_initial_schema.sql
psql -U postgres -d nvide_live -f migrations/002_seed_data.sql
```

### 3. Environment Configuration
```bash
cp .env.example .env
# Edit .env with your settings
```

Required environment variables:
```env
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=nvide_live

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=168h

# Bcrypt
BCRYPT_COST=12

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### 4. Run Server
```bash
# Using go run
go run cmd/server/main.go

# Or using Make
make run
```

Server akan berjalan di `http://localhost:8080`

### 5. Run Tests
```bash
# Test semua package
go test ./...

# Test specific package
go test ./internal/usecase/...

# With coverage
go test -cover ./...
```

## API Endpoints

### Public Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login user |
| POST | `/api/v1/auth/refresh` | Refresh tokens |
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

### Protected Endpoints (require auth)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/logout` | Logout user |
| GET | `/api/v1/auth/me` | Get current user profile |

## API Examples

### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Test123!"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test123!"
  }'
```

Response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "user": {
    "id": "uuid",
    "username": "testuser",
    "email": "test@example.com",
    "role": "user",
    "is_verified": false
  }
}
```

### Get Profile (with auth)
```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <access_token>"
```

### Refresh Token
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJ..."
  }'
```

### Logout
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJ..."
  }'
```

## Default Users (from seed)

| Email | Password | Role |
|-------|----------|------|
| admin@nvide.live | Admin123! | admin |
| test@nvide.live | Test123! | user |

## Error Responses

All errors follow this format:
```json
{
  "error": "ERROR_CODE",
  "message": "Human readable error message"
}
```

Common error codes:
- `UNAUTHORIZED` - Authentication required
- `FORBIDDEN` - Insufficient permissions
- `INVALID_TOKEN` - Token is invalid
- `TOKEN_EXPIRED` - Token has expired
- `TOKEN_REVOKED` - Token has been blacklisted
- `VALIDATION_ERROR` - Input validation failed
- `CONFLICT` - Resource already exists
- `NOT_FOUND` - Resource not found
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `INTERNAL_ERROR` - Server error

## RBAC Permission Matrix

### Guest Role
- `user:read:any`
- `stream:read`
- `vod:read`
- `story:read`
- `comment:read`
- `like:create`
- `auth:login`
- `auth:refresh`

### User Role (guest +)
- `user:create`
- `user:read:own`
- `user:update:own`
- `user:delete:own`
- `vod:upload`, `vod:update`, `vod:delete`
- `story:create`, `story:delete`
- `comment:create`, `comment:update:own`, `comment:delete:own`
- `like:delete`
- `gift:send`, `gift:read`
- `wallet:read:own`, `wallet:transaction`
- `chat:send`, `chat:read`
- `payment:create`, `payment:read`

### Host Role (user +)
- `stream:create`, `stream:update`, `stream:delete`
- `host_application:create`

### Agency Role (host +)
- `agency:read`, `agency:update`
- `agency:host:manage`
- `stream:manage`

### Admin Role
- **ALL permissions**

## Testing

### Unit Tests
```bash
# Run all tests
go test ./...

# Run with verbose
go test -v ./...

# Run specific test
go test -v ./internal/usecase -run TestAuthUseCase_Register

# Coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Manual Testing dengan Postman
Import collection dari `docs/postman_collection.json` (jika ada) atau gunakan curl examples di atas.

## Development Notes

### Clean Architecture Layers
1. **Domain** - Business entities & interfaces (no dependencies)
2. **Repository** - Data access implementations (depends on domain)
3. **Usecase** - Business logic (depends on domain & repository interfaces)
4. **Delivery** - HTTP handlers & router (depends on usecase)

### Dependency Injection
Manual DI di `main.go` - semua dependency diwire secara eksplisit.

### Graceful Shutdown
Server menangkap SIGINT/SIGTERM dan memberikan timeout 30 detik untuk cleanup.

### JWT Blacklist
Access token yang logout/disuspend disimpan di Redis dengan TTL = sisa expiry token.

### UUID
Menggunakan UUID v4 untuk saat ini (bisa diganti dengan UUID v7 library).

## Production Checklist

- [ ] Ganti JWT_SECRET dengan random string yang kuat
- [ ] Set DB_SSLMODE=require untuk production
- [ ] Configure Redis dengan password
- [ ] Set LOG_LEVEL=warn atau error
- [ ] Enable rate limiting
- [ ] Setup proper CORS origins (bukan "*")
- [ ] Configure backup database
- [ ] Setup monitoring & alerting
- [ ] Use environment-specific config
- [ ] Enable Redis persistence
- [ ] Setup log aggregation

## License

Proprietary - NVide Live Platform

## Support

Untuk pertanyaan, hubungi tim development.