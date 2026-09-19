package httpapi

import (
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
	"golang.org/x/sync/errgroup"
)

type llmConnectionCounts struct {
	Total int `json:"total"`
}

type overviewResponse struct {
	Profiles       profile.Counts        `json:"profiles"`
	Jobs           job.Counts            `json:"jobs"`
	Templates      resumetemplate.Counts `json:"templates"`
	LLMConnections llmConnectionCounts   `json:"llmConnections"`
	Generations    generation.Counts     `json:"generations"`
}

// overview returns lightweight aggregate counts for the dashboard's metric
// cards, avoiding a full list fetch per domain. The counts are fetched
// concurrently since they're independent read-only queries. The dashboard
// still calls /v1/jobs directly on top of this for the full job list it
// needs to show "recent job opportunities".
func (a *API) overview(w http.ResponseWriter, r *http.Request) {
	ctx, ws := r.Context(), workspaceID(r)

	var (
		profiles       profile.Counts
		jobs           job.Counts
		templates      resumetemplate.Counts
		llmConnections int
		generations    generation.Counts
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		profiles, err = a.profiles.Count(groupCtx, ws)
		return err
	})
	group.Go(func() (err error) {
		jobs, err = a.jobs.Count(groupCtx, ws)
		return err
	})
	if a.templates != nil {
		group.Go(func() (err error) {
			templates, err = a.templates.Count(groupCtx, ws)
			return err
		})
	}
	if a.settings != nil {
		group.Go(func() (err error) {
			llmConnections, err = a.settings.CountConnections(groupCtx, ws)
			return err
		})
	}
	if a.generations != nil {
		group.Go(func() (err error) {
			generations, err = a.generations.Count(groupCtx, ws)
			return err
		})
	}

	if err := group.Wait(); err != nil {
		writeError(w, http.StatusInternalServerError, "overview_failed", "Could not load workspace overview.")
		return
	}

	writeJSON(w, http.StatusOK, overviewResponse{
		Profiles:       profiles,
		Jobs:           jobs,
		Templates:      templates,
		LLMConnections: llmConnectionCounts{Total: llmConnections},
		Generations:    generations,
	})
}

func (a *API) registerOverview(mux *http.ServeMux) {
	mux.Handle("GET /v1/overview", a.requireRole(identity.RoleViewer, a.overview))
}
