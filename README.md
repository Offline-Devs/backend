# Noshirvani Academy Smart Exam and Counseling Management System

## Quick Start
1. Copy `.env.example` to `.env` and update values as needed.
2. Run `docker compose up -d`.
3. Visit `http://localhost` for the web app and `http://localhost/api` for API proxy.

## Development
- Backend: `cd backend` then `go run ./cmd/server`
- Frontend: `cd frontend` then `pnpm dev`

## Production Notes
- Place Vazirmatn font files under `frontend/public/fonts` and add @font-face rules if you want fully self-hosted fonts.
- Set `JWT_SECRET`, `JWT_REFRESH_SECRET`, `JWT_ACCESS_TTL`, and `JWT_REFRESH_TTL` in production.

## Features
- Phone OTP authentication
- Student dashboard with exams, mistakes, and performance history
- Admin panel with dynamic model fields and management tools
- RTL-ready UI with Vazirmatn font
- Jalali date helpers for reporting

## API Endpoints (Core)
- `POST /api/v1/auth/request-otp`
- `POST /api/v1/auth/verify-otp`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/students/profile`
- `POST /api/v1/students/profile`
- `POST /api/v1/exams`
- `GET /api/v1/exams`
- `POST /api/v1/mistakes`
- `GET /api/v1/admin/students`

## License
Proprietary
