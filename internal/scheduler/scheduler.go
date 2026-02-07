package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Task represents a scheduled task
type Task struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // e.g., "scan", "cleanup", "backup"
	Schedule    string                 `json:"schedule"` // cron-like format
	Params      map[string]interface{} `json:"params"`
	Enabled     bool                   `json:"enabled"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	NextRun     *time.Time             `json:"next_run,omitempty"`
	Status      string                 `json:"status"` // idle, running, failed
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TaskExecution represents a task execution record
type TaskExecution struct {
	ID          int64                  `json:"id"`
	TaskID      string                 `json:"task_id"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      string                 `json:"status"` // running, success, failed
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// TaskHandler is a function that executes a task
type TaskHandler func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error)

// Scheduler manages task scheduling and execution
type Scheduler struct {
	db       *sql.DB
	mu       sync.RWMutex
	handlers map[string]TaskHandler
	tasks    map[string]*Task
	running  map[string]context.CancelFunc
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// Config holds scheduler configuration
type Config struct {
	DBPath           string
	SyncInterval     time.Duration // How often to sync tasks from WebUI
	PersistenceFile  string
	OfflineTolerance bool
}

// New creates a new scheduler
func New(config Config) (*Scheduler, error) {
	if config.DBPath == "" {
		config.DBPath = "/var/lib/mingyue-agent/scheduler.db"
	}
	if config.SyncInterval == 0 {
		config.SyncInterval = 5 * time.Minute
	}

	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Scheduler{
		db:       db,
		handlers: make(map[string]TaskHandler),
		tasks:    make(map[string]*Task),
		running:  make(map[string]context.CancelFunc),
		stopCh:   make(chan struct{}),
	}

	if err := s.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	// Load persisted tasks
	if err := s.loadTasks(); err != nil {
		db.Close()
		return nil, fmt.Errorf("load tasks: %w", err)
	}

	return s, nil
}

func (s *Scheduler) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		schedule TEXT,
		params TEXT,
		enabled INTEGER DEFAULT 1,
		last_run INTEGER,
		next_run INTEGER,
		status TEXT DEFAULT 'idle',
		created_at INTEGER,
		updated_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS task_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		started_at INTEGER NOT NULL,
		completed_at INTEGER,
		status TEXT NOT NULL,
		result TEXT,
		error TEXT,
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);

	CREATE INDEX IF NOT EXISTS idx_task_id ON task_executions(task_id);
	CREATE INDEX IF NOT EXISTS idx_started_at ON task_executions(started_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Scheduler) loadTasks() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT id, name, type, schedule, params, enabled, last_run, next_run, status, created_at, updated_at
		FROM tasks
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		var paramsJSON string
		var enabled int
		var lastRun, nextRun, createdAt, updatedAt int64

		err := rows.Scan(&task.ID, &task.Name, &task.Type, &task.Schedule, &paramsJSON,
			&enabled, &lastRun, &nextRun, &task.Status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		task.Enabled = enabled != 0
		if lastRun > 0 {
			t := time.Unix(lastRun, 0)
			task.LastRun = &t
		}
		if nextRun > 0 {
			t := time.Unix(nextRun, 0)
			task.NextRun = &t
		}
		task.CreatedAt = time.Unix(createdAt, 0)
		task.UpdatedAt = time.Unix(updatedAt, 0)

		if err := json.Unmarshal([]byte(paramsJSON), &task.Params); err == nil {
			s.tasks[task.ID] = &task
		}
	}

	return rows.Err()
}

// RegisterHandler registers a task handler for a specific task type
func (s *Scheduler) RegisterHandler(taskType string, handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[taskType] = handler
}

// AddTask adds a new task
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = "idle"

	// Calculate next run based on schedule
	if task.Schedule != "" {
		nextRun := s.calculateNextRun(task.Schedule)
		task.NextRun = &nextRun
	}

	paramsJSON, err := json.Marshal(task.Params)
	if err != nil {
		return err
	}

	var nextRunUnix int64
	if task.NextRun != nil {
		nextRunUnix = task.NextRun.Unix()
	}

	_, err = s.db.Exec(`
		INSERT INTO tasks (id, name, type, schedule, params, enabled, next_run, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Name, task.Type, task.Schedule, string(paramsJSON),
		boolToInt(task.Enabled), nextRunUnix, task.Status, task.CreatedAt.Unix(), task.UpdatedAt.Unix())
	if err != nil {
		return err
	}

	s.tasks[task.ID] = task
	return nil
}

// UpdateTask updates an existing task
func (s *Scheduler) UpdateTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task.UpdatedAt = time.Now()

	paramsJSON, err := json.Marshal(task.Params)
	if err != nil {
		return err
	}

	var nextRunUnix int64
	if task.NextRun != nil {
		nextRunUnix = task.NextRun.Unix()
	}

	_, err = s.db.Exec(`
		UPDATE tasks
		SET name = ?, type = ?, schedule = ?, params = ?, enabled = ?, next_run = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, task.Name, task.Type, task.Schedule, string(paramsJSON),
		boolToInt(task.Enabled), nextRunUnix, task.Status, task.UpdatedAt.Unix(), task.ID)
	if err != nil {
		return err
	}

	s.tasks[task.ID] = task
	return nil
}

// DeleteTask deletes a task
func (s *Scheduler) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel if running
	if cancel, ok := s.running[taskID]; ok {
		cancel()
		delete(s.running, taskID)
	}

	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	if err != nil {
		return err
	}

	delete(s.tasks, taskID)
	return nil
}

// GetTask retrieves a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// ListTasks returns all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// ExecuteTask manually executes a task
func (s *Scheduler) ExecuteTask(ctx context.Context, taskID string) (*TaskExecution, error) {
	task, err := s.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	return s.executeTask(ctx, task)
}

func (s *Scheduler) executeTask(ctx context.Context, task *Task) (*TaskExecution, error) {
	s.mu.RLock()
	handler, ok := s.handlers[task.Type]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no handler registered for task type: %s", task.Type)
	}

	execution := &TaskExecution{
		TaskID:    task.ID,
		StartedAt: time.Now(),
		Status:    "running",
	}

	// Record execution start
	result, err := s.db.Exec(`
		INSERT INTO task_executions (task_id, started_at, status)
		VALUES (?, ?, ?)
	`, task.ID, execution.StartedAt.Unix(), "running")
	if err != nil {
		return nil, err
	}

	execID, _ := result.LastInsertId()
	execution.ID = execID

	// Update task status
	s.mu.Lock()
	task.Status = "running"
	task.LastRun = &execution.StartedAt
	s.mu.Unlock()

	// Execute the task
	taskResult, execErr := handler(ctx, task.Params)

	// Update execution record
	completedAt := time.Now()
	execution.CompletedAt = &completedAt
	execution.Result = taskResult

	if execErr != nil {
		execution.Status = "failed"
		execution.Error = execErr.Error()
	} else {
		execution.Status = "success"
	}

	resultJSON, _ := json.Marshal(taskResult)

	_, err = s.db.Exec(`
		UPDATE task_executions
		SET completed_at = ?, status = ?, result = ?, error = ?
		WHERE id = ?
	`, completedAt.Unix(), execution.Status, string(resultJSON), execution.Error, execID)

	// Update task status and schedule next run
	s.mu.Lock()
	task.Status = execution.Status
	if task.Schedule != "" {
		nextRun := s.calculateNextRun(task.Schedule)
		task.NextRun = &nextRun
	}
	s.mu.Unlock()

	s.UpdateTask(task)

	return execution, execErr
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.wg.Add(1)
	go s.run(ctx)
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndExecuteTasks(ctx)
		}
	}
}

func (s *Scheduler) checkAndExecuteTasks(ctx context.Context) {
	now := time.Now()

	s.mu.RLock()
	var tasksToRun []*Task
	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}
		if task.Status == "running" {
			continue
		}
		if task.NextRun != nil && task.NextRun.Before(now) {
			tasksToRun = append(tasksToRun, task)
		}
	}
	s.mu.RUnlock()

	// Execute tasks concurrently
	for _, task := range tasksToRun {
		taskCtx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.running[task.ID] = cancel
		s.mu.Unlock()

		go func(t *Task) {
			defer func() {
				s.mu.Lock()
				delete(s.running, t.ID)
				s.mu.Unlock()
			}()

			s.executeTask(taskCtx, t)
		}(task)
	}
}

func (s *Scheduler) calculateNextRun(schedule string) time.Time {
	// Simplified cron-like parsing
	// For now, support simple intervals like "every 1h", "every 30m", "daily", etc.

	// Parse simple formats
	var duration time.Duration
	switch schedule {
	case "daily":
		duration = 24 * time.Hour
	case "hourly":
		duration = 1 * time.Hour
	case "every 30m":
		duration = 30 * time.Minute
	case "every 1h":
		duration = 1 * time.Hour
	case "every 6h":
		duration = 6 * time.Hour
	default:
		duration = 1 * time.Hour
	}

	return time.Now().Add(duration)
}

// Stop stops the scheduler
func (s *Scheduler) Stop(ctx context.Context) error {
	close(s.stopCh)

	// Cancel all running tasks
	s.mu.Lock()
	for _, cancel := range s.running {
		cancel()
	}
	s.mu.Unlock()

	// Wait for all tasks to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	return s.db.Close()
}

// GetExecutionHistory returns execution history for a task
func (s *Scheduler) GetExecutionHistory(taskID string, limit int) ([]*TaskExecution, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, started_at, completed_at, status, result, error
		FROM task_executions
		WHERE task_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*TaskExecution
	for rows.Next() {
		var exec TaskExecution
		var startedAt, completedAt int64
		var resultJSON string

		err := rows.Scan(&exec.ID, &exec.TaskID, &startedAt, &completedAt,
			&exec.Status, &resultJSON, &exec.Error)
		if err != nil {
			continue
		}

		exec.StartedAt = time.Unix(startedAt, 0)
		if completedAt > 0 {
			t := time.Unix(completedAt, 0)
			exec.CompletedAt = &t
		}

		if resultJSON != "" {
			json.Unmarshal([]byte(resultJSON), &exec.Result)
		}

		executions = append(executions, &exec)
	}

	return executions, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
