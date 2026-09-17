package hunter_test

import (
	"context"
	"testing"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/adapters/memory"
	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/profile"
)

func TestServiceCreatesAndUpdatesHunter(t *testing.T) {
	repository := memory.NewHunterRepository()
	profileRepository := memory.NewProfileRepository()
	profileService := profile.NewService(profileRepository)
	selectedProfile, err := profileService.Create(context.Background(), "ws_test", profile.CreateInput{Name: "Experienced backend engineer", Content: "Go and distributed systems experience."})
	if err != nil {
		t.Fatal(err)
	}
	service := hunter.NewService(repository, profileRepository)
	before := time.Now().UTC()
	years := 4
	created, err := service.Create(context.Background(), "ws_test", hunter.SaveInput{
		Name: "Backend roles", RoleQuery: "Backend Engineer", Location: "Berlin", ExperienceYears: &years,
		ProfileID: selectedProfile.ID, ConnectionID: "llm_test", Model: "test-model", IntervalMinutes: 1440, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.LastState != "never" || created.NextRunAt.Before(before) || created.ExperienceYears == nil || *created.ExperienceYears != 4 || created.ProfileID != selectedProfile.ID || created.MaxResults != 10 {
		t.Fatalf("unexpected created hunter: %#v", created)
	}
	updated, err := service.Update(context.Background(), "ws_test", created.ID, hunter.SaveInput{
		Name: created.Name, RoleQuery: created.RoleQuery, ConnectionID: created.ConnectionID, Model: created.Model,
		ProfileID: created.ProfileID, MaxResults: 4, IntervalMinutes: 10080, Enabled: false,
	})
	if err != nil || updated.Enabled || updated.IntervalMinutes != 10080 || updated.MaxResults != 4 {
		t.Fatalf("unexpected updated hunter: %#v, error: %v", updated, err)
	}
}

func TestServiceRejectsInvalidHunter(t *testing.T) {
	service := hunter.NewService(memory.NewHunterRepository())
	_, err := service.Create(context.Background(), "ws_test", hunter.SaveInput{Name: "Invalid", RoleQuery: "Engineer", ConnectionID: "llm", Model: "model", IntervalMinutes: 30})
	if err != hunter.ErrInvalid {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestServiceRejectsTooManyResultsAndUnknownProfile(t *testing.T) {
	profiles := memory.NewProfileRepository()
	service := hunter.NewService(memory.NewHunterRepository(), profiles)
	base := hunter.SaveInput{Name: "Invalid", RoleQuery: "Engineer", ConnectionID: "llm", Model: "model", IntervalMinutes: 1440, MaxResults: 11}
	if _, err := service.Create(context.Background(), "ws_test", base); err != hunter.ErrInvalid {
		t.Fatalf("too many results error = %v, want ErrInvalid", err)
	}
	base.MaxResults, base.ProfileID = 5, "profile_missing"
	if _, err := service.Create(context.Background(), "ws_test", base); err != hunter.ErrInvalid {
		t.Fatalf("unknown profile error = %v, want ErrInvalid", err)
	}
}
