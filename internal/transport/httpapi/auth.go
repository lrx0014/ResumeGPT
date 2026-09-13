package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
)

func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject, err := a.authenticator.Authenticate(r.Context(), r.Header.Get("Authorization"))
		if errors.Is(err, identity.ErrUnauthenticated) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
			return
		}
		if err != nil {
			a.logger.Error("authenticate request", "error", err)
			writeError(w, http.StatusInternalServerError, "authentication_failed", "Authentication could not be completed.")
			return
		}
		workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
		if workspaceID == "" {
			workspaceID = a.defaultWorkspaceID
		}
		principal, err := a.access.Resolve(r.Context(), subject, workspaceID)
		if errors.Is(err, identity.ErrForbidden) {
			writeError(w, http.StatusForbidden, "workspace_access_denied", "Access to the workspace is denied.")
			return
		}
		if err != nil {
			a.logger.Error("authorize workspace", "error", err)
			writeError(w, http.StatusInternalServerError, "authorization_failed", "Authorization could not be completed.")
			return
		}
		ctx := requestcontext.WithActorID(r.Context(), principal.UserID)
		ctx = requestcontext.WithWorkspaceID(ctx, principal.WorkspaceID)
		ctx = requestcontext.WithRole(ctx, string(principal.Role))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *API) requireRole(role identity.Role, handler http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual := identity.Role(requestcontext.Role(r.Context()))
		if !actual.Allows(role) {
			writeError(w, http.StatusForbidden, "insufficient_role", "The workspace role does not permit this action.")
			return
		}
		handler.ServeHTTP(w, r)
	})
}
