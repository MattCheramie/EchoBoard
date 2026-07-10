package media_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/MattCheramie/echoboard/internal/media"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestLocalStorageRoundTrip(t *testing.T) {
	s, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := context.Background()

	n, err := s.Put(ctx, "hello.txt", strings.NewReader("hello world"))
	if err != nil || n != 11 {
		t.Fatalf("put = %d, %v", n, err)
	}
	rc, err := s.Open(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "hello world" {
		t.Errorf("read = %q", got)
	}
	if err := s.Delete(ctx, "hello.txt"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Open(ctx, "hello.txt"); err != media.ErrNotFound {
		t.Errorf("open after delete = %v, want ErrNotFound", err)
	}
	// Deleting a missing key is not an error.
	if err := s.Delete(ctx, "hello.txt"); err != nil {
		t.Errorf("delete missing = %v, want nil", err)
	}
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	s, _ := media.NewLocalStorage(t.TempDir())
	ctx := context.Background()
	for _, key := range []string{"../evil", "sub/x", "..", "a/../b", ""} {
		if _, err := s.Put(ctx, key, strings.NewReader("x")); err == nil {
			t.Errorf("Put(%q) succeeded; want rejection", key)
		}
		if _, err := s.Open(ctx, key); err == nil {
			t.Errorf("Open(%q) succeeded; want rejection", key)
		}
	}
}

func TestThumbnail(t *testing.T) {
	data := pngBytes(t, 200, 100)
	thumb, err := media.Thumbnail(bytes.NewReader(data), 64)
	if err != nil {
		t.Fatalf("thumbnail: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("thumbnail is not valid jpeg: %v", err)
	}
	b := img.Bounds()
	if b.Dx() > 64 || b.Dy() > 64 {
		t.Errorf("thumbnail %dx%d exceeds 64px box", b.Dx(), b.Dy())
	}
	// 200x100 scaled into 64 box preserves 2:1 aspect -> 64x32.
	if b.Dx() != 64 || b.Dy() != 32 {
		t.Errorf("thumbnail dims = %dx%d, want 64x32", b.Dx(), b.Dy())
	}
}

func TestThumbnailNotImage(t *testing.T) {
	if _, err := media.Thumbnail(strings.NewReader("not an image"), 64); err != media.ErrNotImage {
		t.Errorf("Thumbnail(non-image) = %v, want ErrNotImage", err)
	}
}
