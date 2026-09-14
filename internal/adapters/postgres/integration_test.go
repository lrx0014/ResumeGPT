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
	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/database"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
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
	jobKindPrefix := id.New("integration")
	queued := workqueue.Job{
		ID:             id.New("task"),
		WorkspaceID:    workspaceID,
		Kind:           jobKindPrefix + ".test",
		IdempotencyKey: id.New("idem"),
		Payload:        []byte(`{"ok":true}`),
		MaxAttempts:    2,
		AvailableAt:    time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, queued); err != nil {
		t.Fatal(err)
	}
	claimed, err := queue.ClaimKind(ctx, "integration-worker", queued.Kind, time.Minute)
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
	claimed, err = queue.ClaimKind(ctx, "integration-worker", queued.Kind, time.Minute)
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
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: jobKindPrefix + ".fail",
		IdempotencyKey: id.New("idem"), Payload: []byte(`{}`), MaxAttempts: 1, AvailableAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, failedJob); err != nil {
		t.Fatal(err)
	}
	failedJob, err = queue.ClaimKind(ctx, "integration-worker", failedJob.Kind, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Retry(ctx, failedJob, "integration-worker", "permanent", "processing failed", 0); err != nil {
		t.Fatal(err)
	}
	assertJobState(t, ctx, pool, failedJob.ID, "failed")

	cancelledJob := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: jobKindPrefix + ".cancel",
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
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: jobKindPrefix + ".reclaim",
		IdempotencyKey: id.New("idem"), Payload: []byte(`{}`), MaxAttempts: 2, AvailableAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := queue.Enqueue(ctx, reclaimedJob); err != nil {
		t.Fatal(err)
	}
	reclaimedJob, err = queue.ClaimKind(ctx, "expired-worker", reclaimedJob.Kind, 0)
	if err != nil {
		t.Fatal(err)
	}
	reclaimedJob, err = queue.ClaimKind(ctx, "replacement-worker", reclaimedJob.Kind, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimedJob.Attempt != 2 || reclaimedJob.LeaseOwner != "replacement-worker" {
		t.Fatalf("unexpected reclaimed job: %#v", reclaimedJob)
	}
	if err := queue.Complete(ctx, reclaimedJob.ID, "replacement-worker"); err != nil {
		t.Fatal(err)
	}

	updatedProfile, err := profileService.Update(requestCtx, workspaceID, createdProfile.ID, profile.UpdateInput{
		Name: "Updated integration profile", TargetRole: "Staff Engineer", DefaultLanguage: "en-US",
		Content: "# Experience\n\nBuilt reliable services.", AvatarObjectID: "obj_integration_avatar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedProfile.TargetRole != "Staff Engineer" || updatedProfile.Content == "" || !updatedProfile.CreatedAt.Equal(createdProfile.CreatedAt) {
		t.Fatalf("unexpected updated profile: %#v", updatedProfile)
	}
	var updateEventPayload string
	if err := pool.QueryRow(ctx, `SELECT payload::text FROM outbox_events
		WHERE workspace_id=$1 AND aggregate_id=$2 AND topic='profile.updated.v1' ORDER BY occurred_at DESC LIMIT 1`,
		workspaceID, createdProfile.ID).Scan(&updateEventPayload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(updateEventPayload, updatedProfile.Content) {
		t.Fatalf("profile content leaked into outbox payload: %s", updateEventPayload)
	}
	if _, err := profileService.Update(requestCtx, otherWorkspaceID, createdProfile.ID, profile.UpdateInput{Name: "Cross-workspace update"}); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("cross-workspace update error = %v, want not found", err)
	}
	if err := profileService.Delete(requestCtx, workspaceID, createdProfile.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Get(requestCtx, workspaceID, createdProfile.ID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("deleted profile read error = %v, want not found", err)
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

func TestDocumentExtractionCompletionIsTransactionallyDeduplicated(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := requestcontext.WithActorID(context.Background(), "usr_document_test")
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	workspaceID, profileID := id.New("ws"), id.New("prof")
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,kind) VALUES ($1,'Documents','personal')`, workspaceID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profiles (id,workspace_id,name,created_at,updated_at) VALUES ($1,$2,'Documents',now(),now())`, profileID, workspaceID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id=$1", workspaceID) })

	repository := postgres.NewDocumentRepository(pool)
	upload := document.Upload{
		ID: id.New("upl"), WorkspaceID: workspaceID, ProfileID: profileID, ObjectID: id.New("obj"),
		Name: "resume.pdf", DeclaredMediaType: "application/pdf", State: "staged",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repository.Stage(ctx, upload); err != nil {
		t.Fatal(err)
	}
	if allowed, err := repository.DownloadAllowed(ctx, workspaceID, upload.ObjectID); err != nil || allowed {
		t.Fatalf("quarantined download allowed = %v, error = %v", allowed, err)
	}
	job := workqueue.Job{
		ID: id.New("task"), WorkspaceID: workspaceID, Kind: document.ExtractJobKind,
		IdempotencyKey: upload.ID, MaxAttempts: 2, AvailableAt: time.Now().UTC(),
	}
	queued, err := repository.Queue(ctx, workspaceID, profileID, upload.ID, job)
	if err != nil {
		t.Fatal(err)
	}
	queue := postgres.NewWorkQueue(pool)
	claimed, err := queue.ClaimKind(ctx, "document-integration-worker", document.ExtractJobKind, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	processed, err := repository.StoreExtraction(ctx, claimed, upload.ID, "Extracted evidence")
	if err != nil || !processed {
		t.Fatalf("first processing result = %v, error = %v", processed, err)
	}
	processed, err = repository.StoreExtraction(ctx, claimed, upload.ID, "Extracted evidence")
	if err != nil || processed {
		t.Fatalf("duplicate processing result = %v, error = %v", processed, err)
	}
	stored, err := repository.Get(ctx, workspaceID, profileID, upload.ID)
	if err != nil || stored.State != "ready" || stored.JobID != queued.JobID || stored.ExtractedText != "Extracted evidence" {
		t.Fatalf("unexpected completed upload: %#v, error = %v", stored, err)
	}
	if allowed, err := repository.DownloadAllowed(ctx, workspaceID, upload.ObjectID); err != nil || !allowed {
		t.Fatalf("ready download allowed = %v, error = %v", allowed, err)
	}
	var inboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_messages WHERE consumer='profile-document-extractor-v1' AND message_id=$1`, claimed.ID).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if inboxCount != 1 {
		t.Fatalf("inbox count = %d, want 1", inboxCount)
	}
	assertJobState(t, ctx, pool, claimed.ID, "succeeded")
}

func TestJobImportCompletionIsTransactionallyDeduplicated(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := requestcontext.WithActorID(context.Background(), "usr_job_import_test")
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	workspaceID := id.New("ws")
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,kind) VALUES ($1,'Job imports','personal')`, workspaceID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id=$1", workspaceID) })

	repository := postgres.NewJobRepository(pool)
	now := time.Now().UTC()
	value := job.Job{ID: id.New("job"), WorkspaceID: workspaceID, SourceURL: "https://www.linkedin.com/jobs/view/123",
		Status: "interested", ImportState: "queued", CreatedAt: now, UpdatedAt: now}
	task := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: id.New("integration_job_import"),
		IdempotencyKey: value.ID, MaxAttempts: 2, AvailableAt: now}
	queued, err := repository.QueueImport(ctx, value, task)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := repository.QueueImport(ctx, job.Job{ID: id.New("job"), WorkspaceID: workspaceID, SourceURL: value.SourceURL}, workqueue.Job{})
	if err != nil || duplicate.ID != queued.ID {
		t.Fatalf("duplicate import = %#v, error = %v", duplicate, err)
	}

	queue := postgres.NewWorkQueue(pool)
	claimed, err := queue.ClaimKind(ctx, "job-import-integration-worker", task.Kind, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.SetImportState(ctx, claimed, "fetching", ""); err != nil {
		t.Fatal(err)
	}
	processed, err := repository.StoreImport(ctx, claimed, job.ParsedJob{Title: "Staff Engineer", Company: "Example",
		City: "Berlin", Country: "Germany", Location: "Berlin, Germany", Description: "Build APIs."})
	if err != nil || !processed {
		t.Fatalf("first processing result = %v, error = %v", processed, err)
	}
	processed, err = repository.StoreImport(ctx, claimed, job.ParsedJob{Title: "Changed"})
	if err != nil || processed {
		t.Fatalf("duplicate processing result = %v, error = %v", processed, err)
	}
	stored, err := repository.Get(ctx, workspaceID, value.ID)
	if err != nil || stored.Title != "Staff Engineer" || stored.Company != "Example" || stored.ImportState != "ready" {
		t.Fatalf("unexpected imported job: %#v, error = %v", stored, err)
	}
	var inboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbox_messages WHERE consumer='job-page-importer-v1' AND message_id=$1`, claimed.ID).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if inboxCount != 1 {
		t.Fatalf("inbox count = %d, want 1", inboxCount)
	}
	assertJobState(t, ctx, pool, claimed.ID, "succeeded")

	if err := repository.Delete(ctx, workspaceID, value.ID); err != nil {
		t.Fatal(err)
	}
	replacement := job.Job{ID: id.New("job"), WorkspaceID: workspaceID, SourceURL: value.SourceURL,
		Status: "interested", ImportState: "queued", CreatedAt: now, UpdatedAt: now}
	replacementTask := workqueue.Job{ID: id.New("task"), WorkspaceID: workspaceID, Kind: task.Kind,
		IdempotencyKey: replacement.ID, MaxAttempts: 2, AvailableAt: now}
	if recreated, err := repository.QueueImport(ctx, replacement, replacementTask); err != nil || recreated.ID != replacement.ID {
		t.Fatalf("reimport after deletion = %#v, error = %v", recreated, err)
	}
}

func TestSettingsRepositoryPersistsPreferencesAndEncryptedConnections(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := requestcontext.WithActorID(context.Background(), "usr_settings_test")
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	workspaceID, otherWorkspaceID := id.New("ws"), id.New("ws")
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,kind) VALUES ($1,'Settings','personal'),($2,'Other settings','personal')`, workspaceID, otherWorkspaceID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM workspaces WHERE id IN ($1,$2)", workspaceID, otherWorkspaceID)
	})

	repository := postgres.NewSettingsRepository(pool)
	now := time.Now().UTC()
	preferences, err := repository.SavePreferences(ctx, settings.Preferences{
		WorkspaceID: workspaceID, InterfaceLanguage: "de", Theme: "dark", UpdatedAt: now,
	})
	if err != nil || preferences.Theme != "dark" {
		t.Fatalf("preferences = %#v, error = %v", preferences, err)
	}
	connection := settings.StoredConnection{Connection: settings.LLMConnection{
		ID: id.New("llm"), WorkspaceID: workspaceID, Name: "Local Ollama", ExecutionMode: "local",
		Provider: "ollama", BaseURL: "http://host.docker.internal:11434", CreatedAt: now, UpdatedAt: now,
	}, TokenCiphertext: []byte("encrypted-token")}
	created, err := repository.CreateConnection(ctx, connection)
	if err != nil || string(created.TokenCiphertext) != "encrypted-token" {
		t.Fatalf("connection = %#v, error = %v", created, err)
	}
	if _, err := repository.GetConnection(ctx, otherWorkspaceID, connection.Connection.ID); !errors.Is(err, settings.ErrNotFound) {
		t.Fatalf("cross-workspace connection error = %v, want not found", err)
	}
	if err := repository.DeleteConnection(ctx, workspaceID, connection.Connection.ID); err != nil {
		t.Fatal(err)
	}
}
