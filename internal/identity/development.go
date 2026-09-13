package identity

import "context"

type DevelopmentAuthenticator struct {
	Subject Subject
}

func (a DevelopmentAuthenticator) Authenticate(context.Context, string) (Subject, error) {
	if a.Subject.Issuer == "" || a.Subject.Subject == "" {
		return Subject{}, ErrUnauthenticated
	}
	return a.Subject, nil
}
