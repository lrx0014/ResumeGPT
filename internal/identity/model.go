package identity

import (
	"context"
	"errors"
)

var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrForbidden       = errors.New("workspace access denied")
)

type Subject struct {
	Issuer  string
	Subject string
	Email   string
}

type Principal struct {
	UserID      string
	WorkspaceID string
	Role        Role
}

type Role string

const (
	RoleViewer Role = "viewer"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
)

var roleRank = map[Role]int{RoleViewer: 1, RoleEditor: 2, RoleAdmin: 3, RoleOwner: 4}

func (r Role) Allows(required Role) bool {
	return roleRank[r] >= roleRank[required]
}

type Authenticator interface {
	Authenticate(ctx context.Context, authorizationHeader string) (Subject, error)
}

type AccessRepository interface {
	Resolve(ctx context.Context, subject Subject, workspaceID string) (Principal, error)
}
