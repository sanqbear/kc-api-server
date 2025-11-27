# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

kc-api is a Go REST API server for Knowledge Center. It uses chi router with PostgreSQL database backend.

## Common Commands

```bash
make build      # Build the application (outputs ./main)
make run        # Run the application
make test       # Run all tests
make itest      # Run integration tests (database tests with testcontainers)
make watch      # Live reload development (requires air)
make docker-run # Start PostgreSQL container
make docker-down # Stop PostgreSQL container
make clean      # Remove built binary
make all        # Build and test
```

## Architecture

```
cmd/api/main.go          # Entry point with graceful shutdown handling, [DI Root] This is where all dependencies are assembled.
internal/
  server/
    server.go            # HTTP server setup, reads PORT from env
    routes.go            # Chi router with middleware (Logger, CORS)
  database/
    database.go          # PostgreSQL connection via pgx, singleton pattern
```

**Key patterns:**
- Database uses singleton pattern (`dbInstance`) to reuse connections
- Environment variables loaded via godotenv/autoload
- HTTP server configured with timeouts (read: 10s, write: 30s, idle: 1m)
- Integration tests use testcontainers-go for PostgreSQL
- It is a structure that uses the Domain-Driven Design style and complies with Dependency Injection (DI), and all dependencies must be injected in cmd/api/main.go
- Every domain handler is provided with test code in a file named handler_test.go

## Environment Variables

Copy `.env.sample` to `.env`:
- `PORT` - HTTP server port
- `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `DB_SCHEMA` - PostgreSQL connection
