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

## API Endpoints

### Authentication
- `POST /api/v1/auth/request-otp` - Request one-time password for authentication
- `POST /api/v1/auth/verify-otp` - Verify OTP and obtain access tokens
- `POST /api/v1/auth/refresh` - Refresh access token using refresh token

### Student Profile
- `GET /api/v1/students/profile` - Get current student profile information
- `POST /api/v1/students/profile` - Update student profile

### Exams
- `GET /api/v1/exams` - Get list of exams
- `POST /api/v1/exams` - Create new exam

### Mistakes and Analysis
- `POST /api/v1/mistakes` - Record student mistakes
- `GET /api/v1/mistakes` - Get student mistakes history

### Admin
- `GET /api/v1/admin/students` - Get all students (admin only)
- `POST /api/v1/admin/exams` - Manage exams (admin only)

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
- `JWT_ACCESS_TTL` - Adjust token lifetime as needed
- `JWT_REFRESH_TTL` - Adjust refresh token lifetime
- `OTP_PROVIDER=smsir` - `mock` is blocked in production
- `EXPOSE_MOCK_OTP=false` - startup should fail if true in production
- Database connection with production credentials
- API keys and external service credentials

### Mock OTP Safety

- `OTP_PROVIDER=mock` is for local/test only
- OTP codes are returned in API responses only when `EXPOSE_MOCK_OTP=true`
- `ENVIRONMENT=production` blocks both `OTP_PROVIDER=mock` and `EXPOSE_MOCK_OTP=true`

### Deploy with Docker Compose
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

## License

Proprietary
