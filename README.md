# Noshirvani Academy Smart Exam and Counseling Management System

A comprehensive exam management and student counseling platform with real-time performance tracking, OTP-based authentication, and an intuitive admin dashboard.

## Table of Contents
- [Getting Started](#getting-started)
- [Installation](#installation)
- [Running the Application](#running-the-application)
- [Development](#development)
- [Features](#features)
- [API Endpoints](#api-endpoints)
- [Configuration](#configuration)
- [License](#license)

## Getting Started

### Prerequisites
- Docker and Docker Compose
- OR: Go 1.19+, Node.js 16+, and pnpm

### Setup Environment Variables
1. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```
2. Update the values in `.env` as needed for your environment.

## Installation

### Using Docker Compose (Recommended)

#### Build and Start Services
```bash
docker compose up --build
```

This command will:
- Build all Docker images from their respective Dockerfiles
- Start all services (backend, frontend, database, etc.)
- Apply necessary configurations from `.env`

#### Start Without Rebuilding
If you've already built the images:
```bash
docker compose up
```

#### Run in Background
```bash
docker compose up -d --build
```

#### Stop Services
```bash
docker compose down
```

#### View Logs
```bash
docker compose logs -f
```

To follow logs from a specific service:
```bash
docker compose logs -f backend
docker compose logs -f frontend
```

### Manual Installation

#### Backend Setup
```bash
cd backend
go mod download
go run ./cmd/server
```

The backend will start on the configured port (typically `http://localhost:8080` or similar, check your `.env`).

#### Frontend Setup
```bash
cd frontend
pnpm install
pnpm dev
```

The frontend will start on `http://localhost:5173` (or the configured dev server port).

## Running the Application

### With Docker Compose
Once services are running:
- Web Application: http://localhost
- API Proxy: http://localhost/api
- Backend API: http://localhost:8000 (or configured port)

### Without Docker
- Frontend: http://localhost:5173 (or dev server port)
- Backend API: http://localhost:8080 (or configured port)

## Development

### Backend Development
```bash
cd backend
go run ./cmd/server
```

The server will auto-reload on file changes if you have a file watcher set up, or use a tool like `air`:
```bash
cd backend
air
```

### Frontend Development
```bash
cd frontend
pnpm install
pnpm dev
```

### Running Tests
```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
pnpm test
```

## Features

- Phone OTP authentication with secure token verification
- Student dashboard with exam history, mistake analysis, and performance metrics
- Admin panel with dynamic model fields and comprehensive management tools
- RTL-ready UI with Vazirmatn font for Persian language support
- Jalali calendar integration for Persian date handling and reporting
- Real-time performance tracking and analytics
- Responsive design for mobile and desktop devices
- Per-exam negative marking support (configurable per exam)
- Rate limiting with response headers
- Strict CORS configuration with development/production awareness
- Atomic exam updates for data consistency
- Blog management system with publish/unpublish functionality
- Comprehensive file upload system (single and multiple files)
- Environment-aware database logging

## API Endpoints

All endpoints are prefixed with `/api/v1/` when accessed through the Nginx proxy.

### Public Routes

#### Authentication
- `POST /auth/request-otp` - Request one-time password for authentication
- `POST /auth/verify-otp` - Verify OTP and obtain access tokens
- `POST /auth/refresh` - Refresh access token using refresh token

#### Blog
- `GET /blog` - Get all published blog posts
- `GET /blog/:slug` - Get a specific blog post by slug

#### Subjects and Majors
- `GET /subjects?major=<major_name>` - Get subjects by major
- `GET /majors` - Get all available majors

### Protected Routes (Require Authentication)

#### Student Profile
- `GET /students/profile` - Get current student profile information
- `POST /students/profile` - Update student profile

#### Exams
- `GET /exams` - Get list of exams
- `GET /exams/:id` - Get exam details
- `POST /exams` - Create new exam
- `PUT /exams/:id` - Update exam (supports per-exam negative marking)
- `DELETE /exams/:id` - Delete exam

#### Mistakes and Analysis
- `GET /mistakes` - Get student mistakes history
- `POST /mistakes` - Record student mistakes
- `PUT /mistakes/:id` - Update a mistake record
- `DELETE /mistakes/:id` - Delete a mistake record

#### Performance and Analytics
- `GET /students/performance` - Get student performance metrics
- `GET /students/statistics` - Get detailed student statistics
- `GET /students/dashboard` - Get dashboard summary

#### File Upload
- `POST /upload` - Upload single file
- `POST /upload/multiple` - Upload multiple files

### Admin Routes (Require Admin Role)

#### Student Management
- `GET /admin/students` - Get all students
- `GET /admin/students/with-stats` - Get all students with statistics
- `GET /admin/students/:id` - Get specific student details
- `GET /admin/students/:id/exams` - Get student's exams
- `GET /admin/students/:id/mistakes` - Get student's mistakes
- `GET /admin/students/:id/performance` - Get student's performance records
- `GET /admin/students/:id/statistics` - Get student's statistics
- `PUT /admin/students/:id` - Update student information
- `PUT /admin/students/:id/approve` - Approve/verify student
- `DELETE /admin/students/:id` - Delete student

#### Performance Management (Admin)
- `POST /admin/students/:id/performance` - Create performance record for student
- `PUT /admin/performance/:id` - Update performance record
- `DELETE /admin/performance/:id` - Delete performance record

#### Dynamic Fields
- `GET /admin/dynamic-fields` - Get all dynamic fields
- `POST /admin/dynamic-fields` - Create new dynamic field
- `PUT /admin/dynamic-fields/:id` - Update dynamic field
- `DELETE /admin/dynamic-fields/:id` - Delete dynamic field

#### Blog Management (Admin)
- `GET /admin/blog` - Get all blog posts (admin view)
- `POST /admin/blog` - Create new blog post
- `PUT /admin/blog/:id` - Update blog post
- `PUT /admin/blog/:id/publish` - Publish/unpublish blog post
- `DELETE /admin/blog/:id` - Delete blog post

## Configuration

### Environment Variables

Required variables in `.env`:

```
# JWT Configuration
JWT_SECRET=your_secret_key
JWT_REFRESH_SECRET=your_refresh_secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=7d

# Environment / OTP
ENVIRONMENT=development
OTP_PROVIDER=mock
EXPOSE_MOCK_OTP=true

# Database Configuration
DATABASE_URL=postgresql://user:password@localhost:5432/noshirvani

# Server Configuration
PORT=8080
ENVIRONMENT=development
```

`JWT_ACCESS_TTL` and `JWT_REFRESH_TTL` accept seconds or duration strings like `15m`, `24h`, and `7d`.

### Custom Fonts (Production)

For fully self-hosted fonts without external dependencies:

1. Place Vazirmatn font files under `frontend/public/fonts/`
2. Add @font-face rules in your CSS:
```css
@font-face {
  font-family: 'Vazirmatn';
  src: url('/fonts/Vazirmatn-Regular.woff2') format('woff2');
}
```

## Production Deployment

### Environment Setup
```bash
# Copy and update production environment variables
cp .env.example .env.production
```

Ensure these are set for production:
- `ENVIRONMENT=production`
- `JWT_SECRET` - Use a strong, random secret
- `JWT_REFRESH_SECRET` - Use a strong, random secret
- `JWT_ACCESS_TTL` - Adjust token lifetime as seconds or duration string like `15m`
- `JWT_REFRESH_TTL` - Adjust refresh token lifetime as seconds or duration string like `7d`
- `OTP_PROVIDER=smsir` - `mock` is blocked in production
- `EXPOSE_MOCK_OTP=false` - startup should fail if true in production
- `CORS_ORIGINS` - required outside development
- Database connection with production credentials
- API keys and external service credentials

### Mock OTP Safety

- `OTP_PROVIDER=mock` is for local/test only
- OTP codes are returned in API responses only when `EXPOSE_MOCK_OTP=true`
- `ENVIRONMENT=production` blocks both `OTP_PROVIDER=mock` and `EXPOSE_MOCK_OTP=true`

### Utility Routes

- `GET /health` - Health check endpoint (returns `{"status": "ok"}`)
- `GET /swagger-doc/doc.json` - OpenAPI/Swagger documentation in JSON format
- `GET /swagger/*` - Swagger UI interface

## Rate Limiting

The API implements rate limiting on all routes with response headers:
- `X-RateLimit-Limit` - Maximum requests allowed
- `X-RateLimit-Remaining` - Requests remaining in current window
- `X-RateLimit-Reset` - Unix timestamp when the limit resets

## Error Handling

- `400 Bad Request` - Invalid request parameters or validation errors
- `401 Unauthorized` - Missing or invalid authentication token
- `403 Forbidden` - Insufficient permissions (e.g., non-admin accessing admin routes)
- `404 Not Found` - Resource not found (for missing resources in write operations)
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server-side error

## CORS Configuration

CORS is automatically configured based on environment:
- **Development/Test**: All origins allowed
- **Production**: Only specified origins in `CORS_ORIGINS` environment variable are allowed

Separate multiple origins with commas: `CORS_ORIGINS=https://example.com,https://app.example.com`

## Authentication Flow

1. **Request OTP**: Call `POST /auth/request-otp` with phone number
   - Response includes OTP if `EXPOSE_MOCK_OTP=true` (development only)
   
2. **Verify OTP**: Call `POST /auth/verify-otp` with phone and OTP code
   - Response includes `access_token` (JWT) and `refresh_token`
   
3. **Use Token**: Include token in `Authorization: Bearer <token>` header for protected routes

4. **Refresh Token**: When access token expires, call `POST /auth/refresh` with `refresh_token`
   - Returns new `access_token`

## Token Configuration

Tokens are JWT-based with configurable TTL:
- `JWT_ACCESS_TTL` - How long access tokens are valid (default: 15m)
- `JWT_REFRESH_TTL` - How long refresh tokens are valid (default: 7d)
- Both accept duration strings: `15m`, `24h`, `7d`, or seconds as string

## API Response Format

All responses are JSON. Successful responses include data at root level:
```json
{
  "id": 1,
  "name": "John Doe"
}
```

Error responses include error message:
```json
{
  "error": "User not found"
}
```

### Route Prefixes

- Direct backend routes use paths like `/auth/request-otp` and `/students/profile`
- Nginx proxy exposes same routes under `/api/v1/*`
- Swagger UI is accessible at `/swagger/*` or through proxy at `/api/v1/swagger/*`

### Deploy with Docker Compose
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

## License

Proprietary
