package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/taskmonitor"
)

func (a *API) registerTaskMonitor(mux *http.ServeMux) {
	if a.tasks == nil {
		return
	}
	mux.Handle("GET /v1/system/tasks", a.requireRole(identity.RoleViewer, a.listTasks))
	mux.Handle("GET /v1/system/tasks/{taskID}", a.requireRole(identity.RoleViewer, a.getTask))
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	result, err := a.tasks.List(r.Context(), workspaceID(r), taskmonitor.Filter{
		State: r.URL.Query().Get("state"), Kind: r.URL.Query().Get("kind"), Search: r.URL.Query().Get("search"),
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		a.logger.Error("list background tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "tasks_list_failed", "Could not load background tasks.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	result, err := a.tasks.Get(r.Context(), workspaceID(r), r.PathValue("taskID"))
	switch {
	case errors.Is(err, taskmonitor.ErrNotFound):
		writeError(w, http.StatusNotFound, "task_not_found", "The requested background task does not exist.")
	case err != nil:
		a.logger.Error("get background task", "error", err)
		writeError(w, http.StatusInternalServerError, "task_read_failed", "Could not load the background task.")
	default:
		writeJSON(w, http.StatusOK, result)
	}
}
