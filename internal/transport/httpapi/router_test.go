package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/profile"
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
	return httpapi.New(httpapi.Dependencies{
		Profiles:      profile.NewService(memory.NewProfileRepository()),
		Jobs:          job.NewService(memory.NewJobRepository()),
		Logger:        logger,
		WebOrigin:     "http://localhost:5173",
		Authenticator: identity.DevelopmentAuthenticator{Subject: subject},
		Access: memory.AccessRepository{
			Subject:   subject,
			Principal: identity.Principal{UserID: "usr_dev", WorkspaceID: "ws_personal_dev", Role: role},
		},
		DefaultWorkspaceID: "ws_personal_dev",
		Blobs:              memory.BlobSigner{},
	})
}
