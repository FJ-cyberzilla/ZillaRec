package omnicharge

import (
        "context"
        "fmt"
        "sync"
        "time"
)

// AppState represents the operational state of the OmniCharge Master Controller.
type AppState string

const (
        StateIdle        AppState = "IDLE"
        StateRunning     AppState = "RUNNING"
        StatePaused      AppState = "PAUSED"
        StateError       AppState = "ERROR"
        StateTerminating AppState = "TERMINATING"
)

// ControllerSnapshot holds a point-in-time snapshot of the master controller state.
type ControllerSnapshot struct {
	State      AppState     `json:"state"`
	Memory     MemoryStats  `json:"memory"`
	Health     HealthStatus `json:"health"`
	BaseDir    string       `json:"base_dir"`
	ByteOffset int64        `json:"byte_offset"`
	SourceURL  string       `json:"source_url"`
	UptimeSec  int64        `json:"uptime_sec"`
}

// MasterController orchestrates app lifecycle, health (via Catwatch), path security, resource governance, checksum verification, smart pause/resume, and dynamic link refreshing.
type MasterController struct {
	mu         sync.RWMutex
	state      AppState
	startTime  time.Time
	security   *SecurityManager
	memory     *MemoryGovernor
	catwatch   *HealthCatwatch
	checksum   *ChecksumManager
	lastHealth HealthStatus
	byteOffset int64
	sourceURL  string
	ctx        context.Context
	cancel     context.CancelFunc
	pauseChan  chan struct{}
	resumeChan chan struct{}
}

// NewMasterController initializes the OmniCharge Master Controller with the Health Catwatch and Checksum Manager.
func NewMasterController(baseDir string, maxRetries int) (*MasterController, error) {
        secMgr, err := NewSecurityManager(baseDir)
        if err != nil {
                return nil, fmt.Errorf("failed to init security manager: %w", err)
        }

        ctx, cancel := context.WithCancel(context.Background())

        initialHealth := HealthStatus{
                IsHealthy:    true,
                LastCategory: CategoryNone,
        }

        return &MasterController{
                state:      StateIdle,
                startTime:  time.Now(),
                security:   secMgr,
                memory:     NewMemoryGovernor(),
                catwatch:   NewHealthCatwatch(maxRetries),
                checksum:   NewChecksumManager(),
                lastHealth: initialHealth,
                ctx:        ctx,
                cancel:     cancel,
                pauseChan:  make(chan struct{}),
                resumeChan: make(chan struct{}),
        }, nil
}

// State returns the current application state safely.
func (mc *MasterController) State() AppState {
        mc.mu.RLock()
        defer mc.mu.RUnlock()
        return mc.state
}

// SetState transitions the controller state safely.
func (mc *MasterController) SetState(newState AppState) {
        mc.mu.Lock()
        defer mc.mu.Unlock()
        mc.state = newState
        if newState == StateError {
                mc.lastHealth.IsHealthy = false
        } else if newState == StateIdle || newState == StateRunning {
                if mc.lastHealth.LastCategory == CategoryNone {
                        mc.lastHealth.IsHealthy = true
                }
        }
}

// RecordError passes an error through the HealthCatwatch and updates controller health.
func (mc *MasterController) RecordError(err error) HealthStatus {
        mc.mu.Lock()
        defer mc.mu.Unlock()

        status := mc.catwatch.EvaluateError(err)
        mc.lastHealth = status
        if !status.IsHealthy {
                mc.state = StateError
        }
        return status
}

// ResetHealth clears health errors and resets the Catwatch counters.
func (mc *MasterController) ResetHealth() {
        mc.mu.Lock()
        defer mc.mu.Unlock()

        mc.catwatch.Reset()
        mc.lastHealth = HealthStatus{
                IsHealthy:    true,
                LastCategory: CategoryNone,
        }
        if mc.state == StateError {
                mc.state = StateIdle
        }
}

// Snapshot returns a comprehensive operational snapshot of the controller including Catwatch diagnostics.
func (mc *MasterController) Snapshot() ControllerSnapshot {
        mc.mu.RLock()
        state := mc.state
        startTime := mc.startTime
        health := mc.lastHealth
        mc.mu.RUnlock()

        // Ensure health syncs with state if an error occurred
        if state == StateError && health.IsHealthy {
                health.IsHealthy = false
                if health.LastCategory == CategoryNone {
                        health.LastCategory = CategoryGeneric
                        health.LastMessage = "System entered error state"
                }
        }

        return ControllerSnapshot{
			State:      state,
			Memory:     mc.memory.Stats(),
			Health:     health,
			BaseDir:    mc.security.BaseDir(),
			ByteOffset: mc.byteOffset,
			SourceURL:  mc.sourceURL,
			UptimeSec:  int64(time.Since(startTime).Seconds()),
		}
}

// SetByteOffset updates the current downloaded byte offset for smart pause/resume.
func (mc *MasterController) SetByteOffset(offset int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.byteOffset = offset
}

// ByteOffset returns the current downloaded byte offset.
func (mc *MasterController) ByteOffset() int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.byteOffset
}

// SetSourceURL updates the download source URL for dynamic link refreshing.
func (mc *MasterController) SetSourceURL(url string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.sourceURL = url
}

// SourceURL returns the current download source URL.
func (mc *MasterController) SourceURL() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.sourceURL
}

// Security returns the path security manager.
func (mc *MasterController) Security() *SecurityManager {
        return mc.security
}

// Memory returns the memory governor.
func (mc *MasterController) Memory() *MemoryGovernor {
        return mc.memory
}

// Checksum returns the checksum manager.
func (mc *MasterController) Checksum() *ChecksumManager {
        return mc.checksum
}

// Catwatch returns the health catwatch tracker.
func (mc *MasterController) Catwatch() *HealthCatwatch {
        return mc.catwatch
}

// Watchdog returns the health catwatch tracker.
func (mc *MasterController) Watchdog() *HealthCatwatch {
        return mc.catwatch
}

// Context returns the controller's root context.
func (mc *MasterController) Context() context.Context {
        return mc.ctx
}

// Pause pauses active operations.
func (mc *MasterController) Pause() {
        mc.mu.Lock()
        if mc.state == StateRunning {
                mc.state = StatePaused
        }
        mc.mu.Unlock()
}

// Resume resumes paused operations.
func (mc *MasterController) Resume() {
        mc.mu.Lock()
        if mc.state == StatePaused {
                mc.state = StateRunning
        }
        mc.mu.Unlock()
}

// Stop terminates the controller and cleans up resources.
func (mc *MasterController) Stop() {
        mc.SetState(StateTerminating)
        mc.cancel()
        mc.memory.FreeMemory()
        mc.SetState(StateIdle)
}
