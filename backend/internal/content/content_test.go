package content_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/MattCheramie/echoboard/internal/content"
	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/media"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png: %v", err)
	}
	return buf.Bytes()
}

func TestContentCRUD(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	c, err := repo.CreateContent(ctx, content.CreateInput{
		AuthorID: "u1",
		Title:    "Launch post",
		Body:     "Hello world",
		Targets:  []string{"sandbox", "meta"},
		Tags:     []string{"launch", "promo"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.Status != content.StatusDraft {
		t.Errorf("status = %q, want draft", c.Status)
	}
	// Targets and tags come back sorted.
	if got := c.Targets; len(got) != 2 || got[0] != "meta" || got[1] != "sandbox" {
		t.Errorf("targets = %v, want [meta sandbox]", got)
	}
	if got := c.Tags; len(got) != 2 || got[0] != "launch" || got[1] != "promo" {
		t.Errorf("tags = %v, want [launch promo]", got)
	}

	// Partial update: change title and reduce tags; targets untouched.
	newTitle := "Updated post"
	updated, err := repo.UpdateContent(ctx, c.ID, content.UpdateInput{
		Title: &newTitle,
		Tags:  &[]string{"promo"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Updated post" {
		t.Errorf("title = %q", updated.Title)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "promo" {
		t.Errorf("tags after update = %v, want [promo]", updated.Tags)
	}
	if len(updated.Targets) != 2 {
		t.Errorf("targets should be unchanged, got %v", updated.Targets)
	}

	list, err := repo.ListContent(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %d, %v", len(list), err)
	}

	if err := repo.DeleteContent(ctx, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetContent(ctx, c.ID); err != content.ErrNotFound {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
}

func TestTagGetOrCreate(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()

	t1, err := repo.GetOrCreateTag(ctx, "marketing")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	t2, err := repo.GetOrCreateTag(ctx, "marketing")
	if err != nil {
		t.Fatalf("get tag: %v", err)
	}
	if t1.ID != t2.ID {
		t.Errorf("get-or-create returned different ids: %s vs %s", t1.ID, t2.ID)
	}
	tags, _ := repo.ListTags(ctx)
	if len(tags) != 1 {
		t.Errorf("tag count = %d, want 1", len(tags))
	}
}

func TestScheduledContent(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	ctx := context.Background()
	sched := time.Now().UTC().Add(time.Hour).Truncate(time.Second)

	c, err := repo.CreateContent(ctx, content.CreateInput{AuthorID: "u1", Title: "S", ScheduledAt: &sched})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ScheduledAt == nil || !c.ScheduledAt.Equal(sched) {
		t.Fatalf("scheduledAt = %v, want %v", c.ScheduledAt, sched)
	}
	// Clearing the schedule.
	cleared, err := repo.UpdateContent(ctx, c.ID, content.UpdateInput{SetSchedule: true, ScheduledAt: nil})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cleared.ScheduledAt != nil {
		t.Errorf("scheduledAt after clear = %v, want nil", cleared.ScheduledAt)
	}
}

func TestServiceUploadMedia(t *testing.T) {
	repo := content.NewRepository(dbtest.New(t))
	store, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	svc := content.NewService(repo, store)
	ctx := context.Background()

	data := pngBytes(t, 120, 80)
	m, err := svc.UploadMedia(ctx, "u1", "pic.png", "image/png", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if m.Size != int64(len(data)) {
		t.Errorf("size = %d, want %d", m.Size, len(data))
	}
	if !m.HasThumbnail() {
		t.Error("expected a thumbnail for a PNG upload")
	}

	// Original round-trips byte-for-byte.
	_, rc, ct, err := svc.OpenMedia(ctx, m.ID, false)
	if err != nil {
		t.Fatalf("open original: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Error("original bytes differ")
	}
	if ct != "image/png" {
		t.Errorf("original content type = %q", ct)
	}

	// Thumbnail is served as JPEG.
	_, trc, tct, err := svc.OpenMedia(ctx, m.ID, true)
	if err != nil {
		t.Fatalf("open thumb: %v", err)
	}
	trc.Close()
	if tct != "image/jpeg" {
		t.Errorf("thumb content type = %q, want image/jpeg", tct)
	}

	// Delete removes metadata and blobs.
	if err := svc.DeleteMedia(ctx, m.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetMedia(ctx, m.ID); err != content.ErrNotFound {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
	if _, err := store.Open(ctx, m.StorageKey); err != media.ErrNotFound {
		t.Errorf("blob after delete = %v, want ErrNotFound", err)
	}
}
