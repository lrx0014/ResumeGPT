package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type Dispatcher struct {
	Repository Repository
	Publisher  Publisher
	WorkerID   string
	Logger     *slog.Logger
}

func (d Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := d.dispatchOne(ctx); err != nil && !errors.Is(err, ErrEmpty) {
			d.Logger.Error("dispatch outbox event", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (d Dispatcher) dispatchOne(ctx context.Context) error {
	event, err := d.Repository.Claim(ctx, d.WorkerID, 30*time.Second)
	if err != nil {
		return err
	}
	if err := d.Publisher.Publish(ctx, event); err != nil {
		_ = d.Repository.MarkFailed(ctx, event.ID, d.WorkerID, err.Error())
		return err
	}
	return d.Repository.MarkPublished(ctx, event.ID, d.WorkerID)
}

type LogPublisher struct {
	Logger *slog.Logger
}

func (p LogPublisher) Publish(_ context.Context, event Event) error {
	p.Logger.Info("domain event published", "event_id", event.ID, "topic", event.Topic, "workspace_id", event.WorkspaceID)
	return nil
}
