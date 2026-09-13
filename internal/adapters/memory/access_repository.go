package memory

import (
	"context"

	"github.com/lrx0014/ResumeGPT/internal/identity"
)

type AccessRepository struct {
	Principal identity.Principal
	Subject   identity.Subject
}

func (r AccessRepository) Resolve(_ context.Context, subject identity.Subject, workspaceID string) (identity.Principal, error) {
	if subject.Issuer != r.Subject.Issuer || subject.Subject != r.Subject.Subject {
		return identity.Principal{}, identity.ErrForbidden
	}
	if workspaceID == "" {
		workspaceID = r.Principal.WorkspaceID
	}
	if workspaceID != r.Principal.WorkspaceID {
		return identity.Principal{}, identity.ErrForbidden
	}
	return r.Principal, nil
}
