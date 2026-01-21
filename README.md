# 🏆 Kinetix Leaderboard System

A high-performance, real-time leaderboard system with tie-aware ranking, built with Go, Fiber, Redis, PostgreSQL, and React Native (Expo).

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![Node Version](https://img.shields.io/badge/Node-18+-339933?logo=node.js)
![React Native](https://img.shields.io/badge/React_Native-0.81-61DAFB?logo=react)

## ✨ Features

### Backend
- **🚀 High Performance**: Redis-based ranking with O(log N) search complexity
- **🎯 Tie-Aware Ranking**: Implements Standard Competition Ranking (1224 system)
- **💾 Write-Through Cache**: Synchronous Redis updates with asynchronous PostgreSQL persistence via worker pool
- **📡 Real-Time Updates**: WebSocket with version-based broadcasting (eliminates request storms)
- **🔄 Score Simulation**: Built-in simulator for testing with 2 updates/sec
- **🏗️ Clean Architecture**: Repository pattern with clear separation of concerns
- **⚡ Optimized**: Connection pooling for Redis and PostgreSQL
- **✅ Validation**: Request validation using go-playground/validator
- **🛡️ Resilient**: Graceful shutdown with proper resource cleanup

### Frontend
- **📱 Cross-Platform**: React Native with Expo (iOS, Android, Web)
- **⚡ Flash List**: Optimized rendering for 10,000+ users
- **♾️ Infinite Scroll**: Paginated loading (50 users per page)
- **🔍 Real-Time Search**: Debounced search with instant results
- **🌐 WebSocket Integration**: Real-time updates without polling
- **🎨 Modern UI**: Dark theme with electric blue accents and top 3 special styling
- **🔄 Pull to Refresh**: Manual data refresh capability
- **⏳ Skeleton Loaders**: Smooth loading states

## 🏗️ Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend (React Native + Expo)               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Mobile     │  │     Web      │  │   Tablet     │         │
│  │   (iOS/      │  │   Browser    │  │              │         │
│  │   Android)   │  │              │  │              │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
└─────────┼──────────────────┼──────────────────┼────────────────┘
          │                  │                  │
          └─────────┬────────┴──────────────────┘
                    │ HTTP/REST + WebSocket
                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Backend (Go + Fiber)                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  API Layer (Handlers)                                    │  │
│  │  • UpdateScore   • GetLeaderboard   • SearchUser        │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           ▼                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Service Layer (Business Logic)                         │  │
│  │  • Tie-Aware Ranking (1224)  • Score Validation        │  │
│  │  • Version Management        • Worker Pool             │  │
│  └──────────┬────────────────────────┬──────────────────────┘  │
└─────────────┼────────────────────────┼─────────────────────────┘
              │                        │
     ┌────────▼───────┐       ┌───────▼─────────┐
     │  Redis         │       │  PostgreSQL     │
     │  (Cache/Rank)  │       │  (Persistence)  │
     │  • Sorted Set  │       │  • Users Table  │
     │  • Metadata    │       │  • Bulk Ops     │
     │  • Version     │       │  • Indexes      │
     └────────────────┘       └─────────────────┘
```

### Data Flow

**Score Update Flow:**
1. Client → REST API → Service Layer
2. Service → Redis (synchronous, critical path) → Version++
3. Service → Worker Pool → PostgreSQL (async, non-blocking)
4. WebSocket Hub (polls version every 2s) → Broadcasts to clients
5. Clients → Invalidate cache → Refetch fresh data

**Ranking System:**
- **Composite Score**: `rating + (1.0 - timestamp/10^10)` for deterministic tie-breaking
- **Metadata Hash**: Stores actual ratings for display
- **Sorted Set**: Stores composite scores for ranking
- **Tie-Aware Logic**: Users with same rating get same rank

## 🚀 Quick Start

### Prerequisites

- **Docker** & **Docker Compose** (for databases)
- **Go** 1.21+ (for backend)
- **Node.js** 18+ & npm (for frontend)
- **Expo CLI** (for mobile development)

### 1. Clone & Setup

```bash
git clone <repository-url>
cd "Leaderboard System"
```

### 2. Configure Environment

Create `.env` file in the root directory:

```env
# Database
DATABASE_URL=postgres://postgres:your_password@localhost:5432/leaderboard
DB_PASSWORD=your_secure_password
DB_PORT=5432

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_USERNAME=admin
REDIS_PASSWORD=your_redis_password

# Backend
BACKEND_PORT=8000
```

### 3. Start Infrastructure

```bash
# Start PostgreSQL, Redis, and Backend
docker-compose up -d

# Verify containers are running
docker ps
```

You should see three containers:
- `leaderboard-postgres`
- `leaderboard-redis`
- `leaderboard-backend`

### 4. Seed Database

```bash
cd backend
go run cmd/seeder/main.go
```

This populates the database with 10,000 test users (ratings: 100-5000).

### 5. Start Frontend

```bash
cd frontend
npm install
npm start

# Or run on specific platform
npm run android  # Android
npm run ios      # iOS (Mac only)
npm run web      # Web browser
```

### 6. Access the Application

- **Frontend Web**: http://localhost:8081
- **Backend API**: http://localhost:8000
- **API Health**: http://localhost:8000/api/v1/health

## 📚 API Documentation

### Endpoints

#### Get Leaderboard
```http
GET /api/v1/leaderboard?offset=0&limit=50
```

**Response:**
```json
{
  "data": [
    {
      "rank": 1,
      "username": "user_1234",
      "rating": 5000
    }
  ],
  "offset": 0,
  "limit": 50,
  "total": 10000
}
```

#### Update Score
```http
POST /api/v1/scores
Content-Type: application/json

{
  "username": "user_1234",
  "rating": 4500
}
```

#### Search User
```http
GET /api/v1/search/user_1234
```

**Response:**
```json
{
  "username": "user_1234",
  "rating": 4500,
  "global_rank": 42
}
```

#### Health Check
```http
GET /api/v1/health
```

### WebSocket

**Endpoint:** `ws://localhost:8000/ws`

**Message Format:**
```json
{
  "type": "VERSION_UPDATE",
  "version": 12345
}
```

Frontend receives version updates every 2 seconds (when changed) and refetches data.


## 📊 Performance Benchmarks

- **Score Updates**: 2 updates/sec (configurable)
- **API Response Time**: < 50ms (p95)
- **WebSocket Latency**: < 100ms
- **Database**: Handles 10,000+ users efficiently
- **Frontend**: Smooth rendering with Flash List

## 🔧 Configuration

### Backend Configuration

Edit `backend/cmd/server/main.go`:

```go
simulatorConfig := jobs.SimulatorConfig{
    TickInterval:   500 * time.Millisecond, // Update frequency
    UpdatesPerTick: 1,                      // Batch size
    MinScoreChange: -50,                    // Min score delta
    MaxScoreChange: 50,                     // Max score delta
}
```

### Frontend Configuration

Edit `frontend/src/api/leaderboard.ts`:

```typescript
const API_BASE_URL = Platform.select({
  android: 'http://10.0.2.2:8000',        // Android emulator
  ios: 'http://localhost:8000',           // iOS simulator
  default: 'http://localhost:8000',       // Web
});
```

## 🐳 Docker Deployment

### Build Images

```bash
# Build backend
docker build -t leaderboard-backend:latest ./backend

# Or use docker-compose
docker-compose build
```

### Run with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f backend

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```


## 🛠️ Development

### Backend Development

```bash
cd backend

# Install dependencies
go mod download

# Run development server with hot reload (using air)
air

# Or run directly
go run cmd/server/main.go

# Format code
go fmt ./...

# Lint
golangci-lint run
```

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Start with hot reload
npm start

# Type checking
npm run type-check

# Lint
npm run lint
```

## 🐛 Troubleshooting

### Backend won't start
- Check if ports 8000, 5432, 6379 are available
- Verify `.env` file exists in root directory
- Check Docker containers are running: `docker ps`

### Frontend can't connect to backend
- For Android Emulator: Use `10.0.2.2` instead of `localhost`
- For physical device: Use your computer's local IP
- Check firewall settings

### Database connection errors
- Verify PostgreSQL is running: `docker ps`
- Check credentials in `.env` match docker-compose.yml
- Ensure DATABASE_URL is correct

### WebSocket not updating
- Check browser console for connection errors
- Verify backend WebSocket endpoint is accessible
- Ensure no-cache headers are set

## 📖 Additional Documentation

- [Backend Architecture](backend/ARCHITECTURE.md)
- [Backend Implementation Details](backend/IMPLEMENTATION.md)
- [Worker Pool Design](backend/WORKER_POOL.md)
- [Simulation Architecture](backend/SIMULATION_ARCHITECTURE.md)
- [Frontend Testing Guide](frontend/TESTING.md)
- [Frontend Quick Start](frontend/QUICKSTART.md)

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License.

## 👥 Authors

- **Your Name** - Initial work

## 🙏 Acknowledgments

- Go Fiber team for the excellent web framework
- Shopify for Flash List
- Redis and PostgreSQL communities
- React Native and Expo teams

---

**Built with ❤️ for high-performance leaderboards**
