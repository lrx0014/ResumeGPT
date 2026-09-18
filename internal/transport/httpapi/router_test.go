package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/transport/httpapi"
)

func TestCreateAndListProfile(t *testing.T) {
	handler := newHandler()
	body := bytes.NewBufferString(`{"name":"Backend Engineering","defaultLanguage":"en-US"}`)
	create := httptest.NewRequest(http.MethodPost, "/v1/profiles", body)
	create.Header.Set("Content-Type", "application/json")
	createResult := httptest.NewRecorder()
	handler.ServeHTTP(createResult, create)
	if createResult.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createResult.Code, http.StatusCreated)
	}

	list := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	listResult := httptest.NewRecorder()
	handler.ServeHTTP(listResult, list)
	if listResult.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResult.Code, http.StatusOK)
	}
	var response struct {
		Items []profile.Profile `json:"items"`
	}
	if err := json.NewDecoder(listResult.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Name != "Backend Engineering" {
		t.Fatalf("unexpected profiles: %#v", response.Items)
	}
}

func TestUpdateAndDeleteProfile(t *testing.T) {
	handler := newHandler()
	createResult := httptest.NewRecorder()
	handler.ServeHTTP(createResult, httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewBufferString(`{"name":"Original"}`)))
	var created profile.Profile
	if err := json.NewDecoder(createResult.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	updateBody := bytes.NewBufferString(`{"name":"Updated","targetRole":"Staff Engineer","defaultLanguage":"en-US","content":"# Experience\n\nBuilt APIs.","avatarObjectId":"obj_avatar"}`)
	updateResult := httptest.NewRecorder()
	handler.ServeHTTP(updateResult, httptest.NewRequest(http.MethodPut, "/v1/profiles/"+created.ID, updateBody))
	if updateResult.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updateResult.Code, http.StatusOK, updateResult.Body.String())
	}
	var updated profile.Profile
	if err := json.NewDecoder(updateResult.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated" || updated.TargetRole != "Staff Engineer" || updated.Content != "# Experience\n\nBuilt APIs." {
		t.Fatalf("unexpected updated profile: %#v", updated)
	}

	deleteResult := httptest.NewRecorder()
	handler.ServeHTTP(deleteResult, httptest.NewRequest(http.MethodDelete, "/v1/profiles/"+created.ID, nil))
	if deleteResult.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResult.Code, http.StatusNoContent)
	}
	getResult := httptest.NewRecorder()
	handler.ServeHTTP(getResult, httptest.NewRequest(http.MethodGet, "/v1/profiles/"+created.ID, nil))
	if getResult.Code != http.StatusNotFound {
		t.Fatalf("get deleted status = %d, want %d", getResult.Code, http.StatusNotFound)
	}
}

func TestCreatesProfileAvatarUpload(t *testing.T) {
	handler := newHandler()
	createResult := httptest.NewRecorder()
	handler.ServeHTTP(createResult, httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewBufferString(`{"name":"Avatar profile"}`)))
	var created profile.Profile
	if err := json.NewDecoder(createResult.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/profiles/"+created.ID+"/avatar-upload", bytes.NewBufferString(`{"contentType":"image/png"}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", result.Code, http.StatusCreated, result.Body.String())
	}
	var signed blobstore.SignedURL
	if err := json.NewDecoder(result.Body).Decode(&signed); err != nil {
		t.Fatal(err)
	}
	if signed.Headers["Content-Type"] != "image/png" || !strings.HasPrefix(signed.ObjectID, "obj_") {
		t.Fatalf("unexpected avatar upload: %#v", signed)
	}
}

func TestRejectsIncompleteJob(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewBufferString(`{"title":"Engineer"}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusUnprocessableEntity)
	}
}

func TestImportsLinkedInAndIndeedJobs(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/imports", bytes.NewBufferString(`{"urls":["https://www.linkedin.com/jobs/view/123","https://de.indeed.com/viewjob?jk=456"]}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d: %s", result.Code, http.StatusAccepted, result.Body.String())
	}
	var response struct {
		Items []job.Job `json:"items"`
	}
	if err := json.NewDecoder(result.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 2 || response.Items[0].ImportState != "queued" {
		t.Fatalf("unexpected imports: %#v", response.Items)
	}
}

func TestRejectsUnsupportedJobImportURL(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/imports", bytes.NewBufferString(`{"urls":["https://example.com/jobs/123"]}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusUnprocessableEntity)
	}
}

func TestAcceptsPublicHTTPSURLForAIAssistedImport(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/jobs/imports", bytes.NewBufferString(`{"urls":["https://careers.example.com/jobs/123"],"aiAssisted":true,"connectionId":"llm_test","model":"model"}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d: %s", result.Code, http.StatusAccepted, result.Body.String())
	}
}

func TestCreatesAndListsJobHunters(t *testing.T) {
	handler := newHandler()
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/v1/job-hunters", bytes.NewBufferString(`{
		"name":"Berlin backend roles","roleQuery":"Backend Engineer","location":"Berlin",
		"workMode":"Hybrid","employmentType":"Full-time","experienceYears":4,
		"keywords":"Go, PostgreSQL","connectionId":"llm_test","model":"test-model",
		"intervalMinutes":1440,"enabled":true
	}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create hunter status = %d: %s", create.Code, create.Body.String())
	}
	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/v1/job-hunters", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "Berlin backend roles") {
		t.Fatalf("unexpected hunter list: status=%d body=%s", list.Code, list.Body.String())
	}
}

func TestHealthAddsRequestID(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusOK)
	}
	if result.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestManagesSettingsWithoutReturningAPIToken(t *testing.T) {
	handler := newHandler()
	update := httptest.NewRecorder()
	handler.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/v1/settings", bytes.NewBufferString(`{"interfaceLanguage":"de","theme":"dark"}`)))
	if update.Code != http.StatusOK {
		t.Fatalf("settings update status = %d: %s", update.Code, update.Body.String())
	}
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/v1/settings/llm-connections", bytes.NewBufferString(`{"name":"Local","executionMode":"local","provider":"ollama","baseUrl":"http://host.docker.internal:11434","apiToken":"must-not-leak"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("connection create status = %d: %s", create.Code, create.Body.String())
	}
	if strings.Contains(create.Body.String(), "must-not-leak") || !strings.Contains(create.Body.String(), `"apiTokenConfigured":true`) {
		t.Fatalf("unsafe connection response: %s", create.Body.String())
	}
	var connection settings.LLMConnection
	if err := json.Unmarshal(create.Body.Bytes(), &connection); err != nil {
		t.Fatal(err)
	}
	defaults := httptest.NewRecorder()
	handler.ServeHTTP(defaults, httptest.NewRequest(http.MethodPut, "/v1/settings/agent-defaults", bytes.NewBufferString(`{"items":[{"agent":"writer","connectionId":"`+connection.ID+`","model":"model-a"}]}`)))
	if defaults.Code != http.StatusOK || !strings.Contains(defaults.Body.String(), `"agent":"writer"`) {
		t.Fatalf("Agent defaults update status = %d: %s", defaults.Code, defaults.Body.String())
	}
	listedDefaults := httptest.NewRecorder()
	handler.ServeHTTP(listedDefaults, httptest.NewRequest(http.MethodGet, "/v1/settings/agent-defaults", nil))
	if listedDefaults.Code != http.StatusOK || !strings.Contains(listedDefaults.Body.String(), `"model":"model-a"`) {
		t.Fatalf("Agent defaults list status = %d: %s", listedDefaults.Code, listedDefaults.Body.String())
	}
}

func TestSettingsAcceptsEverySupportedInterfaceLanguage(t *testing.T) {
	handler := newHandler()
	for _, language := range []string{"en", "de", "fr", "es", "ja", "zh-CN", "zh-TW"} {
		request := httptest.NewRequest(http.MethodPut, "/v1/settings", bytes.NewBufferString(fmt.Sprintf(`{"interfaceLanguage":%q,"theme":"system"}`, language)))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("language %q returned status %d: %s", language, response.Code, response.Body.String())
		}
	}
}

func TestViewerCannotCreateProfile(t *testing.T) {
	handler := newHandlerWithRole(identity.RoleViewer)
	request := httptest.NewRequest(http.MethodPost, "/v1/profiles", bytes.NewBufferString(`{"name":"Restricted"}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusForbidden)
	}
}

func TestRejectsUnknownWorkspace(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodGet, "/v1/profiles", nil)
	request.Header.Set("X-Workspace-ID", "ws_unknown")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusForbidden)
	}
}

func TestCreatesWorkspaceScopedSignedUpload(t *testing.T) {
	handler := newHandler()
	request := httptest.NewRequest(http.MethodPost, "/v1/storage/uploads", bytes.NewBufferString(`{"contentType":"application/pdf"}`))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", result.Code, http.StatusCreated)
	}
	var response blobstore.SignedURL
	if err := json.NewDecoder(result.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(response.ObjectID, "obj_") || !strings.Contains(response.URL, "ws_personal_dev") {
		t.Fatalf("unexpected signed upload: %#v", response)
	}
	if response.Headers["Content-Type"] != "application/pdf" {
		t.Fatalf("unexpected signed headers: %#v", response.Headers)
	}
}

func newHandler() http.Handler {
	return newHandlerWithRole(identity.RoleOwner)
}

func newHandlerWithRole(role identity.Role) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	subject := identity.Subject{Issuer: "development", Subject: "developer"}
	jobRepository := memory.NewJobRepository()
	settingsCipher, err := settings.NewAESGCMTokenCipher("test-settings-encryption-key-at-least-32-characters")
	if err != nil {
		panic(err)
	}
	return httpapi.New(httpapi.Dependencies{
		Profiles:      profile.NewService(memory.NewProfileRepository()),
		Jobs:          job.NewService(jobRepository),
		JobImports:    job.NewImportService(jobRepository),
		Logger:        logger,
		WebOrigin:     "http://localhost:5173",
		Authenticator: identity.DevelopmentAuthenticator{Subject: subject},
		Access: memory.AccessRepository{
			Subject:   subject,
			Principal: identity.Principal{UserID: "usr_dev", WorkspaceID: "ws_personal_dev", Role: role},
		},
		DefaultWorkspaceID: "ws_personal_dev",
		Blobs:              memory.BlobSigner{},
		Settings:           settings.NewService(memory.NewSettingsRepository(), settingsCipher, settings.NewHTTPModelDiscoverer()),
		Hunters:            hunter.NewService(memory.NewHunterRepository()),
	})
}
