# Event Booking Backend System

[![CI](https://github.com/your-username/event-booking-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/event-booking-backend/actions)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-blue.svg)](https://www.docker.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue.svg)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A **production-ready Event Booking Backend System** built with **Go**, **PostgreSQL**, **Docker**, and a **CI/CD pipeline**.  
Designed using clean architecture principles and scalable for real-world usage.

---

## ✨ Features

- 🔐 JWT-based authentication (Register / Login)
- 👤 User management
- 📅 Event creation & management
- 🎟️ Event booking system
- 🔄 Password reset & refresh tokens
- 🗄️ PostgreSQL database with versioned migrations
- 🐳 Docker & Docker Compose support
- 🚀 CI/CD pipeline (GitHub Actions)
- 🧱 Clean Architecture (handlers, services, repositories)
- 📜 Structured logging & centralized error handling

---

## 🏗️ Tech Stack

- **Language:** Go (Gin framework)
- **Database:** PostgreSQL 16
- **Migrations:** golang-migrate
- **Auth:** JWT
- **Containerization:** Docker & Docker Compose
- **CI/CD:** GitHub Actions
- **Config:** Environment variables (`.env`)

---

## 📂 Project Structure

```text
.
├── cmd/
│   ├── server/           # Application entry point
│   └── migrate/
│       └── migration/    # SQL migration files
├── internal/
│   ├── config/           # App configuration
│   ├── db/               # Database connection
│   ├── handlers/         # HTTP handlers (Gin)
│   ├── repositories/     # DB access layer
│   ├── services/         # Business logic
│   └── routes/           # API routes
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
└── README.md
```

# Server

ADDR=:8080

# Database

POSTGRES_DB=local_go_db
POSTGRES_USER=local_go_user
POSTGRES_PASSWORD=change-me
DB_ADDR=postgres://local_go_user:change-me@db:5432/local_go_db?sslmode=disable

# Auth

JWT_SECRET=change-this-secret

# Email

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
