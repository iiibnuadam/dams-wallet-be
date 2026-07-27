# Dams Wallet Backend

This is the backend service for the **Dams Wallet** application. It is built with **Go (Golang)**, utilizes the **go-chi** router for high-performance HTTP routing, and connects to a **MongoDB** database.

## Architecture

The project follows a standard layered architecture with clear separation of concerns, organized by domain:

- **`cmd/`**: Contains the main application entry point (`main.go`).
- **`config/`**: Configuration loading and environment variable management.
- **`internal/`**: The core business logic, split into distinct modules:
  - `auth`: Authentication and JWT generation/validation.
  - `budget`: Budget management and tracking.
  - `categories`: Transaction categories.
  - `dashboard`: Aggregated statistics and financial health metrics.
  - `debts`: Debt and loan tracking.
  - `goals`: Savings goals and payment tracking.
  - `routines`: Scheduled/recurring transactions.
  - `transactions`: Core income, expense, and transfer recording.
  - `wallets`: Financial accounts/wallets management.
- **`pkg/`**: Shared libraries and utilities:
  - `db`: Database connection and MongoDB setup.
  - `middleware`: HTTP middlewares (e.g., Auth, CORS).
  - `response`: Standardized API response formatters.

## Tech Stack

- **Language:** Go 1.25+
- **Router:** [go-chi/chi/v5](https://github.com/go-chi/chi)
- **Database:** MongoDB ([mongo-driver](https://go.mongodb.org/mongo-driver))
- **Auth:** JWT ([golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt))
- **Environment:** [godotenv](https://github.com/joho/godotenv)

## Setup & Running Locally

### Prerequisites

- Go 1.25 or higher installed
- A running MongoDB instance (e.g., MongoDB Atlas or local)

### 1. Environment Variables

Create a `.env` file in the root of the `backend/` directory with the following structure:

```env
MONGODB_URI=mongodb://<user>:<pass>@<host>/<database>?...
DB_NAME=dams-wallet
JWT_SECRET=your_super_secret_jwt_key
PORT=8080
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Run the Application

```bash
go run cmd/main.go
```

The server will start on the port specified in your `.env` (default is `8080`).

## API Structure

All endpoints expect and return JSON payloads. Authentication is handled via the `Authorization: Bearer <token>` header, verified by the JWT middleware in `pkg/middleware`.

Typical response structure:
```json
{
  "code": 200,
  "message": "Success message",
  "data": { ... }
}
```
