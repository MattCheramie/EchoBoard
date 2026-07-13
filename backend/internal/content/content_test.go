package content_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/echoboard/internal/content"
	"github.com/MattCheramie/echoboard/internal/dbtest"
)

func TestCreateAndGetRoundTrip(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	when := time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, content.CreateInput{
		AuthorID:    "author-1",
		Title:       "Launch announcement",
		Body:        "The default copy.",
		ScheduledAt: &when,
		Targets: []content.Target{
			{Platform: content.PlatformInstagram, Body: "IG-specific caption"},
			{Platform: content.PlatformX},
		},
		Tags:     []string{"Launch", "  launch  ", "Product"},
		Metadata: map[string]string{"campaign": "fall"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != content.StatusDraft {
		t.Errorf("default status = %q, want draft", created.Status)
	}
	if len(created.Tags) != 2 {
		t.Errorf("tags not de-duplicated: %v", created.Tags)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "Launch announcement" || got.Body != "The default copy." {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.ScheduledAt == nil || !got.ScheduledAt.Equal(when) {
		t.Errorf("scheduledAt = %v, want %v", got.ScheduledAt, when)
	}
	if len(got.Targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(got.Targets))
	}
	// Targets come back ordered by platform: instagram, x.
	if got.Targets[0].Platform != content.PlatformInstagram || got.Targets[0].Body != "IG-specific caption" {
		t.Errorf("target[0] = %+v", got.Targets[0])
	}
	if got.Metadata["campaign"] != "fall" {
		t.Errorf("metadata = %v", got.Metadata)
	}

	// EffectiveBody: override wins for IG, master body for X.
	if b := got.Targets[0].EffectiveBody(got); b != "IG-specific caption" {
		t.Errorf("IG EffectiveBody = %q", b)
	}
	if b := got.Targets[1].EffectiveBody(got); b != "The default copy." {
		t.Errorf("X EffectiveBody = %q", b)
	}
}

func TestCreateValidation(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	if _, err := repo.Create(ctx, content.CreateInput{Title: "no author"}); err == nil {
		t.Error("missing authorID should be rejected")
	}
	if _, err := repo.Create(ctx, content.CreateInput{AuthorID: "a"}); err == nil {
		t.Error("missing title should be rejected")
	}
	if _, err := repo.Create(ctx, content.CreateInput{
		AuthorID: "a", Title: "t",
		Targets: []content.Target{{Platform: "myspace"}},
	}); err == nil {
		t.Error("invalid platform should be rejected")
	}
	if _, err := repo.Create(ctx, content.CreateInput{
		AuthorID: "a", Title: "t",
		Targets: []content.Target{{Platform: content.PlatformX}, {Platform: content.PlatformX}},
	}); err == nil {
		t.Error("duplicate target platform should be rejected")
	}
}

func TestUpdateReplacesTargetsTagsAndStatus(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	c, err := repo.Create(ctx, content.CreateInput{
		AuthorID: "a", Title: "v1", Body: "b1",
		Targets:  []content.Target{{Platform: content.PlatformInstagram}},
		Tags:     []string{"one"},
		Metadata: map[string]string{"k": "v1"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := repo.Update(ctx, c.ID, content.UpdateInput{
		Title:    "v2",
		Body:     "b2",
		Status:   content.StatusApproved,
		Targets:  []content.Target{{Platform: content.PlatformTikTok}, {Platform: content.PlatformYouTube}},
		Tags:     []string{"two", "three"},
		Metadata: map[string]string{"k": "v2"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "v2" || updated.Status != content.StatusApproved {
		t.Errorf("update fields not applied: %+v", updated)
	}
	if len(updated.Targets) != 2 || updated.Targets[0].Platform != content.PlatformTikTok {
		t.Errorf("targets not replaced: %+v", updated.Targets)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "three" {
		// tags come back ordered alphabetically: three, two
		t.Errorf("tags not replaced/sorted: %v", updated.Tags)
	}
	if updated.Metadata["k"] != "v2" {
		t.Errorf("metadata not updated: %v", updated.Metadata)
	}
	// The old target/tag rows must be gone, not merely added to.
	if got, _ := repo.GetByID(ctx, c.ID); len(got.Targets) != 2 || len(got.Tags) != 2 {
		t.Errorf("stale targets/tags remain: targets=%d tags=%d", len(got.Targets), len(got.Tags))
	}

	if _, err := repo.Update(ctx, "missing", content.UpdateInput{Title: "x", Status: content.StatusDraft}); !errors.Is(err, content.ErrNotFound) {
		t.Errorf("Update(missing) err = %v, want ErrNotFound", err)
	}
}

func TestSetStatusAndDelete(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	c, err := repo.Create(ctx, content.CreateInput{AuthorID: "a", Title: "t"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.SetStatus(ctx, c.ID, content.StatusInReview); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	got, _ := repo.GetByID(ctx, c.ID)
	if got.Status != content.StatusInReview {
		t.Errorf("status = %q, want in_review", got.Status)
	}
	if err := repo.SetStatus(ctx, c.ID, "bogus"); err == nil {
		t.Error("invalid status should be rejected")
	}
	if err := repo.SetStatus(ctx, "missing", content.StatusDraft); !errors.Is(err, content.ErrNotFound) {
		t.Errorf("SetStatus(missing) err = %v, want ErrNotFound", err)
	}

	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, c.ID); !errors.Is(err, content.ErrNotFound) {
		t.Errorf("after Delete, GetByID err = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, c.ID); !errors.Is(err, content.ErrNotFound) {
		t.Errorf("second Delete err = %v, want ErrNotFound", err)
	}
}

func TestListByAuthor(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	for _, title := range []string{"a1", "a2"} {
		if _, err := repo.Create(ctx, content.CreateInput{AuthorID: "alice", Title: title}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	if _, err := repo.Create(ctx, content.CreateInput{AuthorID: "bob", Title: "b1"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	alice, err := repo.ListByAuthor(ctx, "alice")
	if err != nil {
		t.Fatalf("ListByAuthor: %v", err)
	}
	if len(alice) != 2 {
		t.Errorf("alice content = %d, want 2", len(alice))
	}
	if none, _ := repo.ListByAuthor(ctx, "nobody"); len(none) != 0 {
		t.Errorf("nobody content = %d, want 0", len(none))
	}
}
