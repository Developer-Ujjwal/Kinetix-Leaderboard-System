package jobs

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"backend/internal/models"
	"backend/internal/service"
)

// SimulationManager manages high-frequency score update simulations
// Bypasses HTTP layer for maximum performance
type SimulationManager struct {
	service       *service.LeaderboardService
	users         []models.User
	ticker        *time.Ticker
	stopCh        chan struct{}
	wg            sync.WaitGroup
	running       atomic.Bool
	
	// Metrics
	totalUpdates  atomic.Int64
	successCount  atomic.Int64
	errorCount    atomic.Int64
	startTime     time.Time
	
	// Configuration
	tickInterval  time.Duration // How often to update (e.g., 50ms)
	updatesPerTick int          // How many users to update per tick
	minScoreChange int          // Minimum score change (-50)
	maxScoreChange int          // Maximum score change (+50)
}

// SimulatorConfig holds configuration for the simulator
type SimulatorConfig struct {
	TickInterval   time.Duration // Default: 50ms (20 updates/sec base rate)
	UpdatesPerTick int           // Default: 1 (can batch multiple updates per tick)
	MinScoreChange int           // Default: -50
	MaxScoreChange int           // Default: +50
}

// NewSimulationManager creates a new simulation manager
func NewSimulationManager(service *service.LeaderboardService, config SimulatorConfig) *SimulationManager {
	// Apply defaults
	if config.TickInterval == 0 {
		config.TickInterval = 50 * time.Millisecond // 20 ticks/sec
	}
	if config.UpdatesPerTick == 0 {
		config.UpdatesPerTick = 1
	}
	if config.MinScoreChange == 0 {
		config.MinScoreChange = -50
	}
	if config.MaxScoreChange == 0 {
		config.MaxScoreChange = 50
	}

	return &SimulationManager{
		service:        service,
		stopCh:         make(chan struct{}),
		tickInterval:   config.TickInterval,
		updatesPerTick: config.UpdatesPerTick,
		minScoreChange: config.MinScoreChange,
		maxScoreChange: config.MaxScoreChange,
	}
}

// Start begins the simulation loop
func (sm *SimulationManager) Start(ctx context.Context) error {
	if sm.running.Load() {
		return fmt.Errorf("simulation already running")
	}

	// Load users from database
	users, err := sm.service.GetAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users available for simulation")
	}

	sm.users = users
	sm.startTime = time.Now()
	sm.running.Store(true)

	log.Printf("🚀 Simulation Manager Started")
	log.Printf("   - Users: %d", len(sm.users))
	log.Printf("   - Tick Interval: %v", sm.tickInterval)
	log.Printf("   - Updates per Tick: %d", sm.updatesPerTick)
	log.Printf("   - Score Change Range: [%d, %d]", sm.minScoreChange, sm.maxScoreChange)
	log.Printf("   - Theoretical Max Rate: %.0f updates/sec", float64(sm.updatesPerTick)*1000.0/float64(sm.tickInterval.Milliseconds()))

	// Start simulation goroutine
	sm.wg.Add(1)
	go sm.simulationLoop(ctx)

	// Start metrics reporter
	sm.wg.Add(1)
	go sm.metricsReporter(ctx)

	return nil
}

// Stop gracefully stops the simulation
func (sm *SimulationManager) Stop() {
	if !sm.running.Load() {
		log.Println("⚠️ Simulation not running")
		return
	}

	log.Println("⏹️ Stopping Simulation Manager...")
	sm.running.Store(false)
	close(sm.stopCh)
	sm.wg.Wait()

	elapsed := time.Since(sm.startTime)
	total := sm.totalUpdates.Load()
	success := sm.successCount.Load()
	errors := sm.errorCount.Load()
	rate := float64(total) / elapsed.Seconds()

	log.Println("✅ Simulation Manager Stopped")
	log.Printf("   - Total Updates: %d", total)
	log.Printf("   - Successful: %d", success)
	log.Printf("   - Errors: %d", errors)
	log.Printf("   - Duration: %v", elapsed.Round(time.Second))
	log.Printf("   - Average Rate: %.1f updates/sec", rate)
}

// IsRunning returns whether the simulation is currently running
func (sm *SimulationManager) IsRunning() bool {
	return sm.running.Load()
}

// GetMetrics returns current simulation metrics
func (sm *SimulationManager) GetMetrics() map[string]interface{} {
	elapsed := time.Since(sm.startTime)
	total := sm.totalUpdates.Load()
	rate := float64(total) / elapsed.Seconds()

	return map[string]interface{}{
		"running":       sm.running.Load(),
		"total_updates": total,
		"successful":    sm.successCount.Load(),
		"errors":        sm.errorCount.Load(),
		"duration_sec":  elapsed.Seconds(),
		"rate":          rate,
		"uptime":        elapsed.String(),
	}
}

// simulationLoop is the main event loop
func (sm *SimulationManager) simulationLoop(ctx context.Context) {
	defer sm.wg.Done()

	sm.ticker = time.NewTicker(sm.tickInterval)
	defer sm.ticker.Stop()

	rand.Seed(time.Now().UnixNano())
	userIndex := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Simulation context cancelled")
			return

		case <-sm.stopCh:
			return

		case <-sm.ticker.C:
			// Process batch of updates
			for i := 0; i < sm.updatesPerTick; i++ {
				// Wrap around user list
				if userIndex >= len(sm.users) {
					userIndex = 0
					// Reshuffle for variety
					rand.Shuffle(len(sm.users), func(i, j int) {
						sm.users[i], sm.users[j] = sm.users[j], sm.users[i]
					})
				}

				user := sm.users[userIndex]
				userIndex++

				// Generate score change in range [minScoreChange, maxScoreChange]
				scoreRange := sm.maxScoreChange - sm.minScoreChange + 1
				scoreChange := sm.minScoreChange + rand.Intn(scoreRange)
				newRating := user.Rating + scoreChange

				// Clamp to valid range (100-5000)
				if newRating < 100 {
					newRating = 100
				}
				if newRating > 5000 {
					newRating = 5000
				}

				// Direct service call (bypasses HTTP stack)
				sm.totalUpdates.Add(1)
				if err := sm.service.UpdateScore(context.Background(), user.Username, newRating); err != nil {
					sm.errorCount.Add(1)
					// Log only critical errors, not every failure
					if sm.errorCount.Load()%100 == 1 {
						log.Printf("⚠️ Simulation error (total: %d): %v", sm.errorCount.Load(), err)
					}
				} else {
					sm.successCount.Add(1)
					// Update local cache for next iteration (update the slice directly)
					sm.users[userIndex-1].Rating = newRating
				}
			}
		}
	}
}

// metricsReporter logs metrics periodically
func (sm *SimulationManager) metricsReporter(ctx context.Context) {
	defer sm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopCh:
			return
		case <-ticker.C:
			elapsed := time.Since(sm.startTime)
			total := sm.totalUpdates.Load()
			success := sm.successCount.Load()
			errors := sm.errorCount.Load()
			rate := float64(total) / elapsed.Seconds()

			log.Printf("📊 Simulation Metrics:")
			log.Printf("   - Updates: %d (%.1f/sec)", total, rate)
			log.Printf("   - Success: %d | Errors: %d", success, errors)
			log.Printf("   - Uptime: %v", elapsed.Round(time.Second))
		}
	}
}
