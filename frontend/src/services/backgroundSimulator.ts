import { leaderboardApi } from '../api/leaderboard';

/**
 * Background Load Simulator
 * Automatically triggers load simulations at regular intervals without any UI
 */
class BackgroundSimulator {
  private intervalId: ReturnType<typeof setInterval> | null = null;
  private isRunning = false;
  private intervalSeconds = 5; // Default: 30 seconds

  /**
   * Start automatic background simulations
   * @param intervalSeconds Time between simulations (default: 30)
   */
  start(intervalSeconds: number = 30) {
    if (this.isRunning) {
      console.log('[BackgroundSimulator] Already running');
      return;
    }

    this.intervalSeconds = intervalSeconds;
    this.isRunning = true;

    console.log(`[BackgroundSimulator] Starting automatic simulations every ${intervalSeconds}s`);

    // Run first simulation immediately
    this.triggerSimulation();

    // Set up interval for subsequent simulations
    this.intervalId = setInterval(() => {
      this.triggerSimulation();
    }, intervalSeconds * 1000);
  }

  /**
   * Stop automatic background simulations
   */
  stop() {
    if (!this.isRunning) {
      return;
    }

    console.log('[BackgroundSimulator] Stopping automatic simulations');

    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }

    this.isRunning = false;
  }

  /**
   * Trigger a single simulation
   */
  private async triggerSimulation() {
    try {
      console.log('[BackgroundSimulator] Triggering simulation...');
      await leaderboardApi.simulateLoad();
      console.log('[BackgroundSimulator] Simulation started successfully');
    } catch (error: any) {
      console.error('[BackgroundSimulator] Simulation failed:', error.message);
    }
  }

  /**
   * Check if simulator is currently running
   */
  getStatus() {
    return {
      isRunning: this.isRunning,
      intervalSeconds: this.intervalSeconds,
    };
  }
}

// Export singleton instance
export const backgroundSimulator = new BackgroundSimulator();
