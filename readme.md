# VOuTE — Backend Overview [YouTube: https://www.youtube.com/watch?v=TUmVkUhiWhA]

This README documents the server-side architecture, design decisions, runtime configuration, and operational guidance for the VOuTE backend.
It focuses exclusively on backend concerns (services, APIs, storage, auth, mailing, and deployment) — UI/frontend details are intentionally omitted.

**Project Structure (backend)**
- **Server entry:** [cmd/main.go](cmd/main.go) — HTTP server initialization, route registration, middleware and service wiring.
- **Database helpers:** [db/mongoDB.go](db/mongoDB.go), [db/redis.go](db/redis.go), [db/timescaleDB.go](db/timescaleDB.go)
- **Core packages:** `pkg/` contains domain logic and handlers:
  - [pkg/user](pkg/user) — user service, repository, and handler. Note: direct signup is disabled; OTP/Google flows are preferred.
  - [pkg/vote](pkg/vote) — poll/vote models, repository and service logic.
  - [pkg/bookmarks](pkg/bookmarks) — bookmark handlers and storage integration.
  - [pkg/comments](pkg/comments) — comment CRUD for polls.
  - [pkg/mailing](pkg/mailing) — email sender implementation using SMTP (go-mail) and OTP verification storage.
  - [pkg/middleware](pkg/middleware) — authentication middleware, token generation and OAuth handlers (including Google OAuth endpoints).
  - [pkg/response](pkg/response) — standardized API response helpers.
  - [pkg/ws](pkg/ws) — WebSocket hub and handlers used for poll updates.
  - [pkg/config](pkg/config) — environment parsing and provider configs.
  - [pkg/utils](pkg/utils) — helper utilities (env helpers, parse ID, snowflake IDs, password helpers).

**High-level architecture**
- HTTP server (Gin) serves REST API endpoints and registers WebSocket endpoint at `/ws/polls` for real-time updates.
- Data persistency:
  - MongoDB: primary data store for users, polls, options, comments, bookmarks.
  - Redis: ephemeral stores, OTP verification tokens, short-lived counters and rate-limiting, session-like data.
  - TimescaleDB (Postgres + Timescale): historical/time-series vote metrics and analytics.
- Authentication and sessions:
  - JWT access token (returned to SPA and stored per-tab in sessionStorage) and a secure httpOnly `refresh_token` cookie used to obtain new access tokens.
  - OTP-based signup/login flows for passwordless experiences and Google OAuth for federated sign-in.

**Auth flows (implementation highlights)**
- OTP flow:
  - Request OTP: `POST /mailing/otp` — mailing service creates OTP, stores verification token (Redis), sends code via SMTP.
  - Verify OTP: `POST /mailing/verify-otp` — validates OTP and returns a `verification_token` used for signup/login.
  - Signup with OTP: `POST /auth/signup-otp` — server validates `verification_token`, creates user, issues token pair (access + refresh).
  - Login with OTP (email/username): `POST /auth/login-otp` and `POST /auth/login-otp-username`.
  - Implementation in code: see [pkg/middleware/auth.go](pkg/middleware/auth.go) and [pkg/mailing](pkg/mailing).
- Google OAuth flow
  - `GET /auth/google/login` — server redirects to Google authorization endpoint.
  - `GET /auth/google/callback` — server exchanges code, reads profile, creates or fetches user, generates token pair and redirects to frontend with the access token.
  - OAuth config helper: `getGoogleOAuthConfig()` in [pkg/middleware/auth.go](pkg/middleware/auth.go).

**Mailing (SMTP) details**
- Mailing previously used SendGrid but has been migrated to SMTP using `go-mail` to avoid API quota bottlenecks.
- Key files: [pkg/mailing/handler.go](pkg/mailing/handler.go), [pkg/mailing/service.go](pkg/mailing/service.go).
- OTP codes and verification tokens are stored in Redis with TTL and are looked up by verification token when completing signup/login.

**Important API endpoints (backend-focused)**
- Authentication:
  - `POST /auth/login` — password login (email or username). See [pkg/middleware/auth.go](pkg/middleware/auth.go).
  - `POST /auth/logout` — invalidate session/refresh token.
  - `POST /auth/refresh` — refresh access token using `refresh_token` cookie.
  - `POST /auth/reset-password` — request/reset password flow.
  - `GET /auth/google/login`, `GET /auth/google/callback` — Google OAuth handlers.
- OTP / mailing:
  - `POST /mailing/otp` — send OTP.
  - `POST /mailing/verify-otp` — verify OTP and obtain `verification_token`.
- Users:
  - `GET /users/me` — get current authenticated user.
  - `GET /users/check?username=...` — check username availability.
  - NOTE: `POST /users/create` (direct signup) is intentionally disabled and returns 403 in `pkg/user/handler.go`.
- Polls & voting:
  - `GET /polls` — list polls; `GET /polls/:id` — poll details
  - `POST /polls/create` — create a poll (auth required)
  - `PUT /polls/update` — cast or adjust a vote
  - `PATCH /polls/:id` — close a poll
  - `GET /polls/creator` — list polls created by current user
  - `POST /polls/getHistoricData` — return time-series vote history (TimescaleDB)
- Bookmarks & Comments:
  - `GET /bookmarks`, `PUT /bookmarks/change`, `DELETE /bookmarks`
  - Comment creation/listing: `/comments/*`
- WebSocket:
  - `ws://<host>/ws/polls` — subscribe to live poll updates (see [pkg/ws](pkg/ws)).

**Key files to inspect**
- Server bootstrap and route registration: [cmd/main.go](cmd/main.go)
- Auth middleware and token logic: [pkg/middleware/auth.go](pkg/middleware/auth.go)
- Mailing implementation: [pkg/mailing/handler.go](pkg/mailing/handler.go)
- User handlers and create-user guard: [pkg/user/handler.go](pkg/user/handler.go)
- Database initialization helpers: [db/mongoDB.go](db/mongoDB.go), [db/redis.go](db/redis.go), [db/timescaleDB.go](db/timescaleDB.go)

**Environment variables (important)**
- Database & connections
  - `MONGO_URI` — MongoDB connection string
  - `REDIS_ADDR` / `REDIS_PASSWORD` — Redis address and credentials
  - `TIMESCALE_DSN` — Postgres/TimescaleDB DSN
- Mail / SMTP
  - `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- Auth / JWT / OAuth
  - `JWT_SECRET` — secret for signing tokens
  - `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL` — token lifetimes
  - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` — Google OAuth
  - `FRONTEND_URL` — used for OAuth redirect back to frontend
- Deployment / runtime
  - `FRONTEND_URL`, `VITE_API_BASE_URL`, `USE_SECURE_COOKIES` (set to `true` in production over HTTPS)

**Running locally (quick start)**
1. Start dependencies (recommended with docker-compose):

```bash
docker-compose --env-file .env up -d
```

2. Start the backend (from repo root):

```bash
# using go run
go run ./cmd

# or build and run
go build -o voute ./cmd && ./voute
```

3. Ensure env vars are set (example `.env` or environment injection). Confirm SMTP credentials and OAuth client secrets are present before testing OTP/OAuth flows.

**Database migrations / schema**
- Timeseries initialization script: [db/up.sql](db/up.sql) — run against the Timescale/Postgres instance.
- MongoDB collection indices and schema are created by the application on first run; confirm collections exist and set recommended indices for queries used by `pkg/vote` and `pkg/user`.

**Security & Operational notes**
- Direct signup is disabled to enforce OTP-first onboarding (see [pkg/user/handler.go](pkg/user/handler.go)).
- Passwords are hashed (bcrypt) before storage. See `pkg/utils/password.go`.
- Refresh tokens are set as httpOnly cookies and rotated on refresh operations; follow secure cookie settings in production (`USE_SECURE_COOKIES=true`, TLS).
- Rate-limit endpoints that send emails or OTP to prevent abuse. Redis is a good place for simple rate counters.
- Validate and rotate `JWT_SECRET` and OAuth client credentials regularly.

**Extending or troubleshooting**
- To change mailing provider, modify [pkg/mailing] to swap `go-mail` for another client. Keep the same `SendOTP` / `VerifyOTP` contract.
- To add providers (e.g., GitHub, Microsoft), follow the pattern in `getGoogleOAuthConfig` and add provider-specific callbacks that create/fetch users and return tokens.
- Useful local debug tips:
  - Tail backend logs for errors during signup/login.
  - Confirm the Redis key TTL for verification tokens to ensure the OTP window is long enough for users.
