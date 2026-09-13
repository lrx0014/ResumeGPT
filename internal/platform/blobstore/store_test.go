package blobstore_test

import (
	"errors"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
)

func TestObjectKeyScopesObjectsToWorkspace(t *testing.T) {
	key, err := blobstore.ObjectKey("ws_example", "obj_example")
	if err != nil {
		t.Fatal(err)
	}
	if key != "ws_example/obj_example" {
		t.Fatalf("key = %q, want workspace-scoped key", key)
	}
}

func TestObjectKeyRejectsTraversal(t *testing.T) {
	for _, objectID := range []string{"../obj_example", "obj_example/other", "invalid"} {
		if _, err := blobstore.ObjectKey("ws_example", objectID); !errors.Is(err, blobstore.ErrInvalidObjectID) {
			t.Fatalf("object ID %q returned error %v", objectID, err)
		}
	}
}
