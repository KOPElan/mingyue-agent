package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/scheduler"
)

type SchedulerHandlers struct {
	scheduler *scheduler.Scheduler
	audit     *audit.Logger
}

func NewSchedulerHandlers(sched *scheduler.Scheduler, auditLogger *audit.Logger) *SchedulerHandlers {
	return &SchedulerHandlers{
		scheduler: sched,
		audit:     auditLogger,
	}
}

func (h *SchedulerHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/scheduler/tasks", h.ListTasks)
	mux.HandleFunc("/api/v1/scheduler/tasks/get", h.GetTask)
	mux.HandleFunc("/api/v1/scheduler/tasks/add", h.AddTask)
	mux.HandleFunc("/api/v1/scheduler/tasks/update", h.UpdateTask)
	mux.HandleFunc("/api/v1/scheduler/tasks/delete", h.DeleteTask)
	mux.HandleFunc("/api/v1/scheduler/tasks/execute", h.ExecuteTask)
	mux.HandleFunc("/api/v1/scheduler/history", h.GetExecutionHistory)
}

// ListTasks godoc
// @Summary List all tasks
// @Description Returns all scheduled tasks
// @Tags scheduler
// @Produce json
// @Success 200 {object} Response{data=[]scheduler.Task}
// @Router /scheduler/tasks [get]
func (h *SchedulerHandlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	tasks := h.scheduler.ListTasks()
	writeJSON(w, http.StatusOK, Response{Success: true, Data: tasks})
}

// GetTask godoc
// @Summary Get task details
// @Description Returns details of a specific task
// @Tags scheduler
// @Produce json
// @Param id query string true "Task ID"
// @Success 200 {object} Response{data=scheduler.Task}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /scheduler/tasks/get [get]
func (h *SchedulerHandlers) GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "task ID required"})
		return
	}

	task, err := h.scheduler.GetTask(taskID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: task})
}

// AddTask godoc
// @Summary Add a new task
// @Description Creates a new scheduled task
// @Tags scheduler
// @Accept json
// @Produce json
// @Param body body scheduler.Task true "Task configuration"
// @Success 200 {object} Response{data=scheduler.Task}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /scheduler/tasks/add [post]
// @Security UserAuth
func (h *SchedulerHandlers) AddTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var task scheduler.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	if err := h.scheduler.AddTask(&task); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "add_task",
			Resource: task.ID,
			Result:   "success",
			SourceIP: r.RemoteAddr,
			Details:  map[string]interface{}{"task_name": task.Name, "task_type": task.Type},
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: task})
}

// UpdateTask godoc
// @Summary Update a task
// @Description Updates an existing scheduled task
// @Tags scheduler
// @Accept json
// @Produce json
// @Param body body scheduler.Task true "Task configuration"
// @Success 200 {object} Response{data=scheduler.Task}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /scheduler/tasks/update [put]
// @Security UserAuth
func (h *SchedulerHandlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var task scheduler.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	if err := h.scheduler.UpdateTask(&task); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "update_task",
			Resource: task.ID,
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: task})
}

// DeleteTask godoc
// @Summary Delete a task
// @Description Deletes a scheduled task
// @Tags scheduler
// @Produce json
// @Param id query string true "Task ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /scheduler/tasks/delete [delete]
// @Security UserAuth
func (h *SchedulerHandlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "task ID required"})
		return
	}

	if err := h.scheduler.DeleteTask(taskID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "delete_task",
			Resource: taskID,
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

// ExecuteTask godoc
// @Summary Execute a task manually
// @Description Manually triggers task execution
// @Tags scheduler
// @Produce json
// @Param id query string true "Task ID"
// @Success 200 {object} Response{data=scheduler.TaskExecution}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /scheduler/tasks/execute [post]
// @Security UserAuth
func (h *SchedulerHandlers) ExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "task ID required"})
		return
	}

	execution, err := h.scheduler.ExecuteTask(r.Context(), taskID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "execute_task",
			Resource: taskID,
			Result:   execution.Status,
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: execution})
}

// GetExecutionHistory godoc
// @Summary Get task execution history
// @Description Returns execution history for a task
// @Tags scheduler
// @Produce json
// @Param id query string true "Task ID"
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} Response{data=[]scheduler.TaskExecution}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /scheduler/history [get]
func (h *SchedulerHandlers) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "task ID required"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	history, err := h.scheduler.GetExecutionHistory(taskID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: history})
}
