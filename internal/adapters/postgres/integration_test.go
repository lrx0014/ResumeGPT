package postgres_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/adapters/postgres"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/knowledge"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

func TestRepositoriesPersistEventsAndEnforceWorkspaceScope(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	workspaceID := id.New("ws")
	otherWorkspaceID := id.New("ws")
	if _, err := pool.Exec(ctx, "INSERT INTO workspaces (id, name, kind) VALUES ($1, 'Integration', 'personal'), ($2, 'Other', 'personal')", workspaceID, otherWorkspaceID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id IN ($1, $2)", workspaceID, otherWorkspaceID)
	})

	requestCtx := requestcontext.WithActorID(ctx, "usr_integration")
	requestCtx = requestcontext.WithRequestID(requestCtx, "req_integration")
	profileRepository := postgres.NewProfileRepository(pool)
	jobRepository := postgres.NewJobRepository(pool)

	profileService := profile.NewService(profileRepository)
	createdProfile, err := profileService.Create(requestCtx, workspaceID, profile.CreateInput{Name: "Integration Profile"})
	if err != nil {
		t.Fatal(err)
	}
	jobService := job.NewService(jobRepository)
	createdJob, err := jobService.Create(requestCtx, workspaceID, job.CreateInput{Title: "Engineer", Company: "Example"})
	if err != nil {
		t.Fatal(err)
	}

	profiles, err := profileRepository.List(ctx, workspaceID)
	if err != nil || len(profiles) != 1 || profiles[0].ID != createdProfile.ID {
		t.Fatalf("unexpected profiles: %#v, error: %v", profiles, err)
	}
	otherProfiles, err := profileRepository.List(ctx, otherWorkspaceID)
	if err != nil || len(otherProfiles) != 0 {
		t.Fatalf("cross-workspace profiles: %#v, error: %v", otherProfiles, err)
	}

	var outboxCount, auditCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE workspace_id = $1 AND aggregate_id IN ($2, $3)", workspaceID, createdProfile.ID, createdJob.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND resource_id IN ($2, $3) AND request_id = 'req_integration'", workspaceID, createdProfile.ID, createdJob.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 2 || auditCount != 2 {
		t.Fatalf("outbox count = %d, audit count = %d, want 2 each", outboxCount, auditCount)
	}
	if _, err := pool.Exec(ctx, "UPDATE outbox_events SET occurred_at = '1900-01-01' WHERE workspace_id = $1 AND aggregate_id IN ($2, $3)", workspaceID, createdProfile.ID, createdJob.ID); err != nil {
		t.Fatal(err)
	}

	outboxRepository := postgres.NewOutboxRepository(pool)
	for range 2 {
		event, err := outboxRepository.Claim(ctx, "integration-worker", time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if event.WorkspaceID != workspaceID || event.Attempt != 1 {
			t.Fatalf("unexpected claimed event: %#v", event)
		}
		if err := outboxRepository.MarkPublished(ctx, event.ID, "integration-worker"); err != nil {
			t.Fatal(err)
		}
	}

	queue := postgres.NewWorkQueue(pool)
	queued := workqueue.Job{
		ID:             id.New("task"),
		WorkspaceID:    workspaceID,
		Kind:           "integration.test",
		IdempotencyKey: id.New("idem"),
		Payload:        []byte(`{"ok":true}`),
		MaxAttempts:    2,
		AvailableAt:    time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, queued); err != nil {
		t.Fatal(err)
	}
	claimed, err := queue.Claim(ctx, "integration-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != queued.ID || claimed.Attempt != 1 {
		t.Fatalf("unexpected claimed job: %#v", claimed)
	}
	if err := queue.Heartbeat(ctx, claimed.ID, "integration-worker", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := queue.Retry(ctx, claimed, "integration-worker", "transient", "retry requested", 0); err != nil {
		t.Fatal(err)
	}
	claimed, err = queue.Claim(ctx, "integration-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Attempt != 2 {
		t.Fatalf("claimed attempt = %d, want 2", claimed.Attempt)
	}
	if err := queue.Complete(ctx, claimed.ID, "integration-worker"); err != nil {
		t.Fatal(err)
	}

	failedJob := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: "integration.fail",
		IdempotencyKey: id.New("idem"), Payload: []byte(`{}`), MaxAttempts: 1, AvailableAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, failedJob); err != nil {
		t.Fatal(err)
	}
	failedJob, err = queue.Claim(ctx, "integration-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Retry(ctx, failedJob, "integration-worker", "permanent", "processing failed", 0); err != nil {
		t.Fatal(err)
	}
	assertJobState(t, ctx, pool, failedJob.ID, "failed")

	cancelledJob := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: "integration.cancel",
		IdempotencyKey: id.New("idem"), Payload: []byte(`{}`), MaxAttempts: 1, AvailableAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, cancelledJob); err != nil {
		t.Fatal(err)
	}
	if err := queue.Cancel(ctx, workspaceID, cancelledJob.ID); err != nil {
		t.Fatal(err)
	}
	assertJobState(t, ctx, pool, cancelledJob.ID, "cancelled")

	reclaimedJob := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: "integration.reclaim",
		IdempotencyKey: id.New("idem"), Payload: []byte(`{}`), MaxAttempts: 2, AvailableAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, reclaimedJob); err != nil {
		t.Fatal(err)
	}
	reclaimedJob, err = queue.Claim(ctx, "expired-worker", 0)
	if err != nil {
		t.Fatal(err)
	}
	reclaimedJob, err = queue.Claim(ctx, "replacement-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimedJob.Attempt != 2 || reclaimedJob.LeaseOwner != "replacement-worker" {
		t.Fatalf("unexpected reclaimed job: %#v", reclaimedJob)
	}
	if err := queue.Complete(ctx, reclaimedJob.ID, "replacement-worker"); err != nil {
		t.Fatal(err)
	}
}

func assertJobState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, jobID, expected string) {
	t.Helper()
	var actual string
	if err := pool.QueryRow(ctx, "SELECT state FROM durable_jobs WHERE id = $1", jobID).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("job %s state = %s, want %s", jobID, actual, expected)
	}
}

func TestKnowledgeLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := requestcontext.WithActorID(context.Background(), "usr_knowledge_test")
	ctx = requestcontext.WithRequestID(ctx, "req_knowledge_test")
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	workspaceID := id.New("ws")
	otherWorkspaceID := id.New("ws")
	profileID := id.New("prof")
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,kind) VALUES ($1,'Knowledge','personal'),($2,'Other','personal')`, workspaceID, otherWorkspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (id,workspace_id,name,created_at,updated_at) VALUES ($1,$2,'Knowledge Profile',now(),now())`, profileID, workspaceID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id IN ($1,$2)", workspaceID, otherWorkspaceID)
	})

	service := knowledge.NewService(postgres.NewKnowledgeRepository(pool))
	created, err := service.Import(ctx, workspaceID, profileID, knowledge.Import{Name: "Career history", Text: "Built Go services\nReduced latency by 30%"})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.Import(ctx, workspaceID, profileID, knowledge.Import{Name: "Duplicate name", Text: "Built Go services\nReduced latency by 30%"})
	if err != nil || duplicate.ID != created.ID {
		t.Fatalf("duplicate import = %#v, error = %v", duplicate, err)
	}

	snapshot, err := service.Snapshot(ctx, workspaceID, profileID)
	if err != nil || len(snapshot.Sources) != 1 || len(snapshot.Segments) != 2 || len(snapshot.Facts) != 2 {
		t.Fatalf("unexpected snapshot: %#v, error: %v", snapshot, err)
	}
	fact := snapshot.Facts[0]
	for _, candidate := range snapshot.Facts {
		if strings.Contains(candidate.Versions[0].Statement, "Go") {
			fact = candidate
			break
		}
	}
	current := fact.Versions[0]
	reviewed, err := service.Review(ctx, workspaceID, profileID, fact.ID, knowledge.Review{
		ExpectedVersionID: current.ID, Statement: current.Statement, Status: "user_confirmed", Sensitive: false,
	})
	if err != nil || reviewed.Number != 2 {
		t.Fatalf("unexpected review: %#v, error: %v", reviewed, err)
	}
	if _, err := service.Review(ctx, workspaceID, profileID, fact.ID, knowledge.Review{
		ExpectedVersionID: current.ID, Statement: current.Statement, Status: "rejected", Sensitive: true,
	}); !errors.Is(err, knowledge.ErrConflict) {
		t.Fatalf("stale review error = %v, want conflict", err)
	}

	results, err := service.Search(ctx, workspaceID, profileID, "go")
	if err != nil || len(results) != 1 || results[0].ID != reviewed.ID {
		t.Fatalf("unexpected search results: %#v, error: %v", results, err)
	}
	if _, err := service.Snapshot(ctx, otherWorkspaceID, profileID); !errors.Is(err, knowledge.ErrNotFound) {
		t.Fatalf("cross-workspace snapshot error = %v, want not found", err)
	}
	if err := service.DeleteSource(ctx, workspaceID, profileID, created.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, err = service.Snapshot(ctx, workspaceID, profileID)
	if err != nil || len(snapshot.Sources) != 0 || len(snapshot.Facts) != 0 {
		t.Fatalf("knowledge remained after deletion: %#v, error: %v", snapshot, err)
	}
}
