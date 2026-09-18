package workqueue

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// ClaimQueue is the subset of queue operations a background processor needs
// to claim, retry, and fail jobs of a given kind.
type ClaimQueue interface {
	ClaimKind(ctx context.Context, workerID, kind string, lease time.Duration) (Job, error)
	Retry(ctx context.Context, job Job, workerID, errorClass, message string, delay time.Duration) error
	Fail(ctx context.Context, jobID, workerID, errorClass, message string) error
}

// RunLoop repeatedly claims jobs of kind and dispatches them to handle until
// ctx is done. It is shared by every background processor: each claimed job
// runs with its own handleTimeout, and the loop polls once per second when
// idle.
func RunLoop(ctx context.Context, queue ClaimQueue, workerID, kind string,
	claimLease, handleTimeout time.Duration, logger *slog.Logger, claimErrorMessage string,
	handle func(context.Context, Job)) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		jobValue, err := queue.ClaimKind(ctx, workerID, kind, claimLease)
		if err == nil {
			jobCtx, cancel := context.WithTimeout(ctx, handleTimeout)
			handle(jobCtx, jobValue)
			cancel()
		} else if !errors.Is(err, ErrEmpty) {
			logger.Error(claimErrorMessage, "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
