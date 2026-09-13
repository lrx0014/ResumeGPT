package identity

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCAuthenticator struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCAuthenticator(ctx context.Context, issuer, clientID string) (*OIDCAuthenticator, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	return &OIDCAuthenticator{verifier: provider.Verifier(&oidc.Config{ClientID: clientID})}, nil
}

func (a *OIDCAuthenticator) Authenticate(ctx context.Context, authorizationHeader string) (Subject, error) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(authorizationHeader), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return Subject{}, ErrUnauthenticated
	}
	verified, err := a.verifier.Verify(ctx, strings.TrimSpace(token))
	if err != nil {
		return Subject{}, ErrUnauthenticated
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := verified.Claims(&claims); err != nil {
		return Subject{}, ErrUnauthenticated
	}
	return Subject{Issuer: verified.Issuer, Subject: verified.Subject, Email: claims.Email}, nil
}
