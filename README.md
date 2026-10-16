# Event Booking Backend System

[![CI](https://github.com/FiraBro/Eventify/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/event-booking-backend/actions)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Framework-Gin-008ECF)](https://github.com/gin-gonic/gin)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL%2016-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Containerized-Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A modular event and service booking platform built with **Go, Gin, PostgreSQL, and Next.js**. The system is designed to support customer bookings, staff scheduling, service management, authentication, and administrative operations through a versioned REST API.

The project follows a layered backend architecture, uses PostgreSQL migrations for database schema management, and supports containerized development with Docker Compose.

> **Project status:** Under active development. Features and production-readiness guarantees should be considered implemented only when supported by the current codebase and tests.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Technology Stack](#technology-stack)
- [System Architecture](#system-architecture)
- [Project Structure](#project-structure)
- [Authentication and Authorization](#authentication-and-authorization)
- [API Overview](#api-overview)
- [Database and Migrations](#database-and-migrations)
- [Getting Started](#getting-started)
- [Environment Configuration](#environment-configuration)
- [Running the Application](#running-the-application)
- [Testing and Code Quality](#testing-and-code-quality)
- [Security Considerations](#security-considerations)
- [Development Roadmap](#development-roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

The Event Booking Backend System provides the API and business logic for a booking platform with three primary user groups:

- **Customers:** Browse services, check availability, create bookings, and manage their accounts.
- **Staff:** Manage assigned services, working schedules, holidays, and booking-related operations.
- **Administrators:** Manage users, roles, services, staff, bookings, and administrative reporting.

The backend is built as a modular monolith. Business logic is separated from HTTP handlers and database access, allowing individual modules to evolve without introducing the operational complexity of microservices.

The Next.js frontend provides the customer-facing application, staff workspace, and administrative dashboard.

## Key Features

### Authentication and Account Management

- User registration and login.
- JWT-based authentication.
- Access-token refresh and logout endpoints.
- Forgot-password and password-reset workflows.
- Authenticated profile retrieval and updates.
- Password changes.
- Role-based access control.
- Account deletion and restoration workflows, where implemented.
- Administrative user management.

### Service Management

- Create, retrieve, update, and delete services.
- Retrieve available service categories.
- Store service names, descriptions, categories, and prices.
- Restrict service-management operations to authorized administrators.

### Staff Management

- Create and manage staff records.
- Retrieve staff details and staff listings.
- Assign services to staff members.
- Configure working schedules.
- Record holidays and unavailable periods.
- Retrieve staff-specific service assignments and schedules.

### Availability and Booking

The intended booking workflow supports:

- Discovering services.
- Checking availability by service or staff member.
- Selecting a suitable appointment time.
- Creating and managing bookings.
- Supporting customer, staff, and administrator workflows.

Booking conflict prevention, transactional reservation handling, cancellation policies, and payment processing should be implemented and tested before being considered production-ready.

### Infrastructure and Development

- PostgreSQL-backed persistence.
- Version-controlled SQL migrations using `golang-migrate`.
- Docker and Docker Compose support.
- Environment-based configuration.
- Automated quality checks through GitHub Actions.
- A documented REST API under `/api/v1`.

---

## Technology Stack

| Component | Technology | Responsibility |
|---|---|---|
| Backend language | Go 1.22+ | Application logic |
| HTTP framework | Gin | Routing and HTTP middleware |
| Database | PostgreSQL 16 | Persistent application data |
| Database driver | `lib/pq` or `pgx`, depending on implementation | PostgreSQL connectivity |
| Migrations | golang-migrate | Versioned schema changes |
| Authentication | JWT | API authentication |
| Frontend | Next.js, React, TypeScript | Customer, staff, and admin interfaces |
| Validation | Go validation and frontend schemas | Request validation |
| Containerization | Docker, Docker Compose | Local development and deployment |
| CI/CD | GitHub Actions | Automated checks and delivery workflows |
| Configuration | Environment variables | Runtime configuration |

Use the PostgreSQL driver actually declared in `go.mod`; the table above describes the intended stack rather than asserting which driver is currently installed.

---

## System Architecture

The backend follows a layered architecture:

```text
Next.js Frontend
       |
       | HTTP / JSON
       v
Go API (Gin)
       |
       v
Middleware
(Authentication, Authorization)
       |
       v
HTTP Handlers
(Request validation, HTTP responses)
       |
       v
Services
(Business rules and use cases)
       |
       v
Repositories
(SQL queries and persistence)
       |
       v
PostgreSQL
```

### Architectural Responsibilities

**Handlers**

- Parse and validate incoming HTTP requests.
- Extract authenticated user information from request context.
- Call application services.
- Return appropriate HTTP status codes and response bodies.

**Services**

- Implement business rules and application use cases.
- Validate state transitions and permissions where appropriate.
- Coordinate repositories and database transactions.
- Prevent invalid operations from being exposed through HTTP handlers.

**Repositories**

- Encapsulate SQL queries and database access.
- Map database records to domain models.
- Support transaction-aware persistence where required.

**Middleware**

- Authenticate incoming requests.
- Enforce role-based access.
- Apply cross-cutting HTTP behavior such as request IDs and rate limits where configured.

**Database**

- Enforce data integrity through foreign keys, unique constraints, check constraints, and indexes.
- Store users, services, staff records, schedules, and booking-related data.

---

## Project Structure

### Go Backend

```text
.
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── migrate/
│       └── migration/
│           ├── 000001_user_table.up.sql
│           ├── 000001_user_table.down.sql
│           ├── 000002_event_table.up.sql
│           ├── 000002_event_table.down.sql
│           └── ...
├── internal/
│   ├── config/
│   ├── db/
│   ├── handlers/
│   ├── middlewares/
│   ├── models/
│   ├── repositories/
│   ├── routes/
│   └── services/
├── tests/
├── docs/
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

Directory responsibilities:

| Directory | Responsibility |
|---|---|
| `cmd/server/` | Application entry point |
| `cmd/migrate/migration/` | Versioned database migrations |
| `internal/config/` | Application configuration |
| `internal/db/` | Database initialization and connection management |
| `internal/handlers/` | HTTP request handlers |
| `internal/middlewares/` | Authentication and authorization middleware |
| `internal/models/` | Data and domain models |
| `internal/repositories/` | Database access |
| `internal/routes/` | API route registration |
| `internal/services/` | Business logic |
| `tests/` | Automated tests |
| `docs/` | API and development documentation |

### Next.js Frontend

The frontend is organized around Next.js App Router route groups and domain-oriented features.

```text
src/
├── middleware.ts
├── app/
│   ├── layout.tsx
│   ├── providers.tsx
│   ├── globals.css
│   ├── api/
│   │   └── auth/
│   ├── (public)/
│   ├── (auth)/
│   ├── (customer)/
│   ├── (staff)/
│   └── (admin)/
├── components/
│   ├── layout/
│   ├── shared/
│   ├── bookings/
│   ├── services/
│   ├── staff/
│   ├── payments/
│   └── reports/
├── features/
│   ├── auth/
│   ├── users/
│   ├── services/
│   ├── staff/
│   ├── bookings/
│   ├── payments/
│   ├── notifications/
│   ├── reports/
│   └── settings/
├── hooks/
├── lib/
│   ├── api/
│   ├── auth/
│   ├── utils/
│   └── validations/
├── types/
└── config/
```

The frontend separates route composition, reusable UI components, feature-specific API operations, shared utilities, and authentication helpers.

---

## Authentication and Authorization

The API uses JWT-based authentication for protected resources.

Authenticated requests should provide an access token using the Bearer scheme:

```http
Authorization: Bearer <access_token>
```

Public endpoints, such as registration and login, do not require authentication. Protected endpoints require a valid token and, where applicable, the necessary permissions.

### Roles

| Role | Intended access |
|---|---|
| `user` | Customer account and own bookings |
| `staff` | Assigned staff operations and bookings |
| `admin` | Administrative user, service, staff, and booking management |

Role names and permissions must remain consistent between database records, token claims, backend middleware, and frontend authorization logic.

**Security rule:** Frontend route protection is not a substitute for backend authorization. The API must validate permissions on every protected operation and enforce resource ownership independently.

---

## API Overview

The API uses the versioned base path:

```text
http://localhost:8080/api/v1
```

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| POST | `/auth/register` | Register an account |
| POST | `/auth/login` | Authenticate a user |
| POST | `/auth/refresh` | Refresh an access token |
| POST | `/auth/logout` | Log out and revoke the refresh session |
| POST | `/auth/forgot-password` | Start password recovery |
| POST | `/auth/reset-password` | Complete password recovery |
| GET | `/auth/profile` | Retrieve the authenticated profile |
| PATCH | `/auth/profile` | Update the authenticated profile |
| PATCH | `/auth/change-password` | Change the account password |

### Users

| Method | Endpoint | Description |
|---|---|---|
| GET | `/users` | List users, with pagination where implemented |
| POST | `/users` | Create a user, subject to authorization |
| GET | `/users/:id` | Retrieve a user |
| PATCH | `/users/:id` | Update a user |
| PATCH | `/users/:id/role` | Update a user's role |
| DELETE | `/users/:id` | Delete or deactivate a user according to the account lifecycle policy |

### Services

| Method | Endpoint | Description |
|---|---|---|
| GET | `/services` | List services |
| GET | `/services/:id` | Retrieve a service |
| GET | `/services/categories` | Retrieve service categories |
| POST | `/services` | Create a service |
| PATCH | `/services/:id` | Update a service |
| DELETE | `/services/:id` | Delete or deactivate a service |

Service-management operations require administrative authorization.

### Staff

| Method | Endpoint | Description |
|---|---|---|
| GET | `/staff` | List staff |
| POST | `/staff` | Create a staff record |
| GET | `/staff/:id` | Retrieve staff details |
| PATCH | `/staff/:id` | Update a staff record |
| DELETE | `/staff/:id` | Remove or deactivate a staff record |
| GET | `/staff/:id/services` | Retrieve assigned services |
| POST | `/staff/:id/services` | Assign services |
| GET | `/staff/:id/schedule` | Retrieve working hours |
| POST | `/staff/:id/schedule` | Configure working hours |
| POST | `/staff/:id/holidays` | Record a holiday or unavailable period |

### Availability

| Method | Endpoint | Description |
|---|---|---|
| GET | `/availability/services/:serviceId` | Find available slots for a service |
| GET | `/availability/staff/:staffId` | Find available slots for staff |

Availability requests should account for working schedules, holidays, service duration, buffers, existing bookings, and the requested date range.

### Bookings

The booking API should expose customer booking creation, retrieval, listing, and cancellation operations. Staff and administrators should have appropriately scoped access to assigned or administrative booking views.

The exact booking routes should match the implemented handlers and registered Gin routes. Update this section whenever the API contract changes.

---

## Database and Migrations

PostgreSQL is the primary persistence layer.

Database schema changes are maintained as versioned SQL migrations using `golang-migrate`.

### Migration Naming Convention

```text
000001_user_table.up.sql
000001_user_table.down.sql
000002_event_table.up.sql
000002_event_table.down.sql
```

- **Up migrations** apply schema changes.
- **Down migrations** reverse changes where a safe reversal is possible.
- Migrations should be reviewed and committed alongside the code that depends on them.
- Production migrations should be executed through a controlled deployment process.

### Run Migrations

Apply available migrations:

```bash
make migrate-up
```

Roll back the most recent migration, if supported by your Makefile:

```bash
make migrate-down
```

Check the Makefile for the exact available migration targets.

### Verify Database Tables

When using a local PostgreSQL installation:

```bash
psql "postgres://local_go_user:password@localhost:5432/local_go_db?sslmode=disable" -c "\dt"
```

When running inside Docker Compose, use the PostgreSQL service name from the container's network rather than assuming that the same hostname is resolvable from the host machine.

Do not edit an already-applied migration to introduce new schema changes. Create a new migration instead.

---

## Getting Started

### Prerequisites

Install or prepare:

- Go 1.22 or later.
- PostgreSQL 16, or Docker with Docker Compose.
- `golang-migrate` CLI.
- Git.
- Node.js and a package manager if running the Next.js frontend.

### 1. Clone the Repository

```bash
git clone https://github.com/your-username/event-booking-backend.git
cd event-booking-backend
```

Replace the repository URL with your actual GitHub repository.

### 2. Configure Environment Variables

Copy the example environment file:

```bash
cp .env.example .env
```

Set the database connection, JWT secret, server address, and any other variables required by the application.

Never commit `.env` or real production credentials.

### 3. Start PostgreSQL

If Docker Compose is configured to start the database:

```bash
docker compose up -d
```

Verify that the PostgreSQL container is healthy before running migrations.

### 4. Apply Database Migrations

```bash
make migrate-up
```

Confirm that the migrations were applied successfully before starting the API.

### 5. Install Go Dependencies

```bash
go mod download
```

### 6. Run the API

If the entry point is `cmd/server/main.go`:

```bash
go run ./cmd/server
```

The API is expected to listen on the configured address, typically:

```text
http://localhost:8080
```

The versioned API base path is:

```text
http://localhost:8080/api/v1
```

### 7. Verify the Application

Use an available health endpoint if one is implemented. Otherwise, test a known public endpoint, such as the service listing:

```bash
curl http://localhost:8080/api/v1/services
```

The endpoint must be registered and the database must contain the required tables for this request to succeed.

---

## Environment Configuration

Example `.env.example`:

```dotenv
# Application
APP_ENV=development
ADDR=:8080

# PostgreSQL
POSTGRES_DB=local_go_db
POSTGRES_USER=local_go_user
POSTGRES_PASSWORD=change-me
DB_ADDR=postgres://local_go_user:change-me@localhost:5432/local_go_db?sslmode=disable

# Authentication
JWT_SECRET=replace-with-a-long-random-secret

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASS=your-email-app-password
```

**Important configuration notes:**

- The example above assumes the API runs on the host and PostgreSQL is published on the local machine.
- When the API runs inside Docker Compose, `DB_ADDR` generally uses the Compose PostgreSQL service name as the hostname.
- The variable names must match what your configuration loader actually reads.
- Use a different database password and JWT secret in production.
- Never use placeholder credentials in a production environment.
- SMTP configuration is needed only if email delivery is implemented.

---

## Running with Docker Compose

Build and start the configured services:

```bash
docker compose up --build -d
```

Inspect running containers:

```bash
docker compose ps
```

View API logs:

```bash
docker compose logs -f
```

View only the API service logs:

```bash
docker compose logs -f api
```

The service name may differ depending on your `docker-compose.yml`.

Stop the stack:

```bash
docker compose down
```

Persistent database data should use a Docker volume. Avoid deleting volumes in environments where database data must be retained.

---

## Testing and Code Quality

Run the Go test suite:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

Run static analysis:

```bash
go vet ./...
```

Format the code:

```bash
gofmt -w .
```

For meaningful production confidence, add:

- Unit tests for business rules.
- Repository integration tests against PostgreSQL.
- HTTP handler tests for validation and error responses.
- Authorization tests for customer, staff, and administrator permissions.
- Concurrent booking tests to detect double-booking defects.
- Migration tests against a clean database.
- Frontend tests for forms, route guards, and API error handling.

A successful build alone does not demonstrate that the booking workflow is safe under concurrent requests.

---

## Security Considerations

The following practices should guide implementation and deployment:

1. Hash passwords with a suitable password-hashing algorithm.
2. Validate JWT signatures, expiration, and required claims.
3. Store refresh and reset tokens securely and support revocation.
4. Apply rate limits to authentication and password-recovery endpoints.
5. Validate all client input on the server.
6. Enforce authorization and resource ownership in the backend.
7. Use database constraints and transactions to protect data integrity.
8. Verify payment-provider webhook signatures before changing payment state.
9. Avoid logging passwords, raw tokens, secrets, or sensitive payment information.
10. Use HTTPS and securely configured cookies or authorization headers.
11. Use least-privilege database credentials.
12. Apply database migrations through a controlled release process.
13. Record important administrative actions in an audit log when auditability is required.

---

## Development Roadmap

The following roadmap describes suggested development priorities, not a claim that every item is already implemented.

- [x] Establish a Go/Gin backend with layered components.
- [x] Use PostgreSQL and versioned SQL migration files.
- [x] Establish authentication and service/staff management routes.
- [ ] Complete booking lifecycle and concurrency protection.
- [ ] Complete availability calculations and conflict handling.
- [ ] Add automated integration and end-to-end tests.
- [ ] Finalize API documentation with OpenAPI.
- [ ] Add structured logging, health checks, and operational metrics.
- [ ] Complete payment integration, webhook handling, and refund workflows if required.
- [ ] Add asynchronous notifications and booking reminders.
- [ ] Harden production deployment, secrets management, and backup procedures.

Adjust the checklist to reflect verified implementation status as the project evolves.

---

## Contributing

Contributions should preserve the project's architectural boundaries and existing API conventions.

1. Create a feature branch.
2. Keep business rules in services rather than HTTP handlers.
3. Use repositories for database access.
4. Include migrations for schema changes.
5. Add tests for new behavior and bug fixes.
6. Run formatting, tests, and static analysis.
7. Document API or environment configuration changes.
8. Submit a pull request describing the implementation and its impact.

Avoid committing secrets, generated credentials, local database files, or environment-specific configuration.

## License

This project is licensed under the MIT License, provided that the repository includes a corresponding `LICENSE` file. Otherwise, specify the actual project license before publishing.

---

**Maintainer:**FiraBro.

