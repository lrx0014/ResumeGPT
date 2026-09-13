package requestcontext

import "context"

type key string

const (
	actorIDKey     key = "actor-id"
	requestIDKey   key = "request-id"
	workspaceIDKey key = "workspace-id"
	roleKey        key = "role"
)

func WithActorID(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, actorIDKey, actorID)
}

func ActorID(ctx context.Context) string {
	value, _ := ctx.Value(actorIDKey).(string)
	if value == "" {
		return "system"
	}
	return value
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func WithWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, workspaceIDKey, workspaceID)
}

func WorkspaceID(ctx context.Context) string {
	value, _ := ctx.Value(workspaceIDKey).(string)
	return value
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func Role(ctx context.Context) string {
	value, _ := ctx.Value(roleKey).(string)
	return value
}
