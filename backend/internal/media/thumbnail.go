package media

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif" // register GIF decoder
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"

	xdraw "golang.org/x/image/draw"
)

// ThumbnailContentType is the encoding of generated thumbnails.
const ThumbnailContentType = "image/jpeg"

// ErrNotImage is returned when the input cannot be decoded as an image.
var ErrNotImage = errors.New("media: not a decodable image")

// Thumbnail decodes an image and returns a JPEG thumbnail that fits within a
// maxDim×maxDim box, preserving aspect ratio. Inputs that are not decodable
// images (e.g. video) return ErrNotImage so callers can skip thumbnailing.
func Thumbnail(r io.Reader, maxDim int) ([]byte, error) {
	if maxDim <= 0 {
		maxDim = 256
	}
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, ErrNotImage
	}
	b := src.Bounds()
	tw, th := fit(b.Dx(), b.Dy(), maxDim)

	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fit returns the largest w×h that fits within maxDim in both dimensions while
// preserving the source aspect ratio. It never upscales.
func fit(w, h, maxDim int) (int, int) {
	if w <= 0 || h <= 0 {
		return 1, 1
	}
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w >= h {
		return maxDim, max1(h * maxDim / w)
	}
	return max1(w * maxDim / h), maxDim
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
