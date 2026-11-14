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
	State        AppState       `json:"state"`
	Memory       MemoryStats    `json:"memory"`
	Health       HealthStatus   `json:"health"`
	BaseDir      string         `json:"base_dir"`
	UptimeSec    int64          `json:"uptime_sec"`
}

// MasterController orchestrates app lifecycle, health, path security, and resource governance.
type MasterController struct {
	mu           sync.RWMutex
	state        AppState
	startTime    time.Time
	security     *SecurityManager
	memory       *MemoryGovernor
	watchdog     *HealthWatchdog
	ctx          context.Context
	cancel       context.CancelFunc
	pauseChan    chan struct{}
	resumeChan   chan struct{}
}

// NewMasterController initializes the OmniCharge Master Controller.
func NewMasterController(baseDir string, maxRetries int) (*MasterController, error) {
	secMgr, err := NewSecurityManager(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to init security manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MasterController{
		state:      StateIdle,
		startTime:  time.Now(),
		security:   secMgr,
		memory:     NewMemoryGovernor(),
		watchdog:   NewHealthWatchdog(maxRetries),
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
}

// Snapshot returns a comprehensive operational snapshot of the controller.
func (mc *MasterController) Snapshot() ControllerSnapshot {
	mc.mu.RLock()
	state := mc.state
	startTime := mc.startTime
	mc.mu.RUnlock()

	return ControllerSnapshot{
		State:     state,
		Memory:    mc.memory.Stats(),
		Health:    HealthStatus{IsHealthy: state != StateError},
		BaseDir:   mc.security.BaseDir(),
		UptimeSec: int64(time.Since(startTime).Seconds()),
	}
}

// Security returns the path security manager.
func (mc *MasterController) Security() *SecurityManager {
	return mc.security
}

// Memory returns the memory governor.
func (mc *MasterController) Memory() *MemoryGovernor {
	return mc.memory
}

// Watchdog returns the health watchdog.
func (mc *MasterController) Watchdog() *HealthWatchdog {
	return mc.watchdog
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
