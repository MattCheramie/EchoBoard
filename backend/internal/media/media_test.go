package media_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/MattCheramie/echoboard/internal/dbtest"
	"github.com/MattCheramie/echoboard/internal/media"
)

func newService(t *testing.T) *media.Service {
	t.Helper()
	store, err := media.NewFSStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	return media.NewService(store, media.NewRepository(dbtest.New(t)), 64)
}

// pngBytes builds a solid-color PNG of the given size.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestFSStoreRoundTripAndTraversal(t *testing.T) {
	store, err := media.NewFSStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	ctx := context.Background()

	n, err := store.Put(ctx, "originals/a/b.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if n != 5 {
		t.Errorf("Put wrote %d bytes, want 5", n)
	}
	rc, err := store.Open(ctx, "originals/a/b.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "hello" {
		t.Errorf("read %q, want hello", got)
	}

	if err := store.Delete(ctx, "originals/a/b.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := store.Delete(ctx, "originals/a/b.txt"); err != nil {
		t.Errorf("deleting a missing key should be a no-op, got %v", err)
	}

	if _, err := store.Put(ctx, "../escape.txt", strings.NewReader("x")); err == nil {
		t.Error("path-traversal key should be rejected")
	}
}

func TestUploadImageGeneratesThumbnail(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	m, err := svc.Upload(ctx, media.UploadInput{
		OwnerID:  "owner-1",
		Filename: "banner.png",
		MimeType: "image/png",
		Data:     bytes.NewReader(pngBytes(t, 200, 100)),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if m.Width != 200 || m.Height != 100 {
		t.Errorf("dimensions = %dx%d, want 200x100", m.Width, m.Height)
	}
	if !m.HasThumbnail() {
		t.Fatal("expected a thumbnail for an image upload")
	}

	rc, err := svc.OpenThumbnail(ctx, m)
	if err != nil {
		t.Fatalf("OpenThumbnail: %v", err)
	}
	defer rc.Close()
	tw, th, ok := media.DecodeDimensions(rc)
	if !ok {
		t.Fatal("thumbnail is not a decodable image")
	}
	// maxDim is 64; the longest edge (was 200) should be scaled to 64.
	if tw != 64 || th != 32 {
		t.Errorf("thumbnail = %dx%d, want 64x32", tw, th)
	}
}

func TestUploadNonImageHasNoThumbnail(t *testing.T) {
	svc := newService(t)
	ctx := context.Background()

	m, err := svc.Upload(ctx, media.UploadInput{
		OwnerID:  "owner-1",
		Filename: "notes.txt",
		MimeType: "text/plain",
		Data:     strings.NewReader("just text"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if m.HasThumbnail() {
		t.Error("text upload should not have a thumbnail")
	}
	if m.Size != int64(len("just text")) {
		t.Errorf("size = %d, want %d", m.Size, len("just text"))
	}

	rc, err := svc.Open(ctx, m)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != "just text" {
		t.Errorf("stored bytes = %q", got)
	}
}

func TestAttachListAndDelete(t *testing.T) {
	store, err := media.NewFSStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	repo := media.NewRepository(dbtest.New(t))
	svc := media.NewService(store, repo, 64)
	ctx := context.Background()

	m, err := svc.Upload(ctx, media.UploadInput{
		OwnerID: "owner-1", Filename: "a.png", MimeType: "image/png",
		Data: bytes.NewReader(pngBytes(t, 20, 20)),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if err := repo.Attach(ctx, m.ID, "content-9"); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	list, err := repo.ListByContent(ctx, "content-9")
	if err != nil {
		t.Fatalf("ListByContent: %v", err)
	}
	if len(list) != 1 || list[0].ID != m.ID {
		t.Fatalf("ListByContent = %+v", list)
	}
	if list[0].ContentID == nil || *list[0].ContentID != "content-9" {
		t.Errorf("content_id not persisted: %+v", list[0])
	}

	if err := svc.Delete(ctx, m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, m.ID); !errors.Is(err, media.ErrNotFound) {
		t.Errorf("after Delete, GetByID err = %v, want ErrNotFound", err)
	}
	// The stored bytes should be gone too.
	if _, err := store.Open(ctx, m.StorageKey); err == nil {
		t.Error("expected the original blob to be deleted")
	}
}

func TestThumbnailRejectsCorruptImage(t *testing.T) {
	// Bytes that claim to be an image but are not decodable: no thumbnail,
	// no error.
	_, _, _, ok, err := media.Thumbnail(strings.NewReader("not really a png"), 64)
	if err != nil {
		t.Fatalf("Thumbnail error on junk: %v", err)
	}
	if ok {
		t.Error("junk data should not produce a thumbnail")
	}
}
