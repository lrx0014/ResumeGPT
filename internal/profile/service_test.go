package profile_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/profile"
)

func TestProfileLifecycle(t *testing.T) {
	service := profile.NewService(memory.NewProfileRepository())
	ctx := context.Background()

	created, err := service.Create(ctx, "ws_personal", profile.CreateInput{
		Name: "  Backend profile  ", TargetRole: "  Senior Backend Engineer  ", Content: "# Experience\n\nBuilt APIs.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Backend profile" || created.TargetRole != "Senior Backend Engineer" || created.DefaultLanguage != "en-US" {
		t.Fatalf("unexpected created profile: %#v", created)
	}

	updated, err := service.Update(ctx, "ws_personal", created.ID, profile.UpdateInput{
		Name: "Platform profile", TargetRole: "Staff Engineer", DefaultLanguage: "de-DE",
		Content: "Updated text", AvatarObjectID: "obj_avatar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreatedAt != created.CreatedAt || updated.Content != "Updated text" || updated.AvatarObjectID != "obj_avatar" {
		t.Fatalf("unexpected updated profile: %#v", updated)
	}

	if err := service.Delete(ctx, "ws_personal", created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(ctx, "ws_personal", created.ID); !errors.Is(err, profile.ErrNotFound) {
		t.Fatalf("get deleted profile error = %v, want not found", err)
	}
}

func TestProfileValidation(t *testing.T) {
	service := profile.NewService(memory.NewProfileRepository())
	cases := []profile.CreateInput{
		{},
		{Name: "Valid", Content: "bad\x00text"},
		{Name: "Valid", Content: strings.Repeat("a", profile.MaxContentBytes+1)},
		{Name: "Valid", AvatarObjectID: "another-profile/avatar"},
	}
	for _, input := range cases {
		if _, err := service.Create(context.Background(), "ws_personal", input); !errors.Is(err, profile.ErrInvalid) {
			t.Fatalf("input %#v error = %v, want invalid", input, err)
		}
	}
}
