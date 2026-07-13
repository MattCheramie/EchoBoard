package media

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"

	// Register the decoders image.Decode dispatches to.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// DefaultThumbMaxDim is the longest-edge size, in pixels, of generated
// thumbnails.
const DefaultThumbMaxDim = 320

// DecodeDimensions reads only the image header and returns its dimensions. It
// returns ok=false for data that is not a decodable image, without erroring, so
// callers can treat non-images as "no dimensions / no thumbnail".
func DecodeDimensions(r io.Reader) (w, h int, ok bool) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// Thumbnail decodes an image and returns a PNG thumbnail whose longest edge is
// at most maxDim, together with the source image's dimensions. Images already
// within maxDim are re-encoded at their original size (never upscaled). It
// returns ok=false (no error) when the data is not a decodable image.
func Thumbnail(r io.Reader, maxDim int) (data []byte, srcW, srcH int, ok bool, err error) {
	if maxDim <= 0 {
		maxDim = DefaultThumbMaxDim
	}
	src, _, derr := image.Decode(r)
	if derr != nil {
		return nil, 0, 0, false, nil
	}
	b := src.Bounds()
	srcW, srcH = b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return nil, srcW, srcH, false, nil
	}

	dstW, dstH := fitDimensions(srcW, srcH, maxDim)
	thumb := resample(src, dstW, dstH)

	var buf bytes.Buffer
	if err := png.Encode(&buf, thumb); err != nil {
		return nil, srcW, srcH, false, fmt.Errorf("media: encode thumbnail: %w", err)
	}
	return buf.Bytes(), srcW, srcH, true, nil
}

// fitDimensions scales (w,h) down so the longest edge is at most maxDim,
// preserving aspect ratio. Dimensions already within maxDim are unchanged.
func fitDimensions(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w >= h {
		nh := h * maxDim / w
		if nh < 1 {
			nh = 1
		}
		return maxDim, nh
	}
	nw := w * maxDim / h
	if nw < 1 {
		nw = 1
	}
	return nw, maxDim
}

// resample downscales src to dstW x dstH by averaging each destination pixel's
// source box (area sampling). It works in premultiplied-alpha space so
// transparent regions don't bleed color, and returns an NRGBA image.
func resample(src image.Image, dstW, dstH int) *image.NRGBA {
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))

	for dy := 0; dy < dstH; dy++ {
		sy0 := b.Min.Y + dy*srcH/dstH
		sy1 := b.Min.Y + (dy+1)*srcH/dstH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for dx := 0; dx < dstW; dx++ {
			sx0 := b.Min.X + dx*srcW/dstW
			sx1 := b.Min.X + (dx+1)*srcW/dstW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}

			var rSum, gSum, bSum, aSum, count uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					// RGBA() returns 16-bit premultiplied components.
					r, g, bl, a := src.At(sx, sy).RGBA()
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(bl)
					aSum += uint64(a)
					count++
				}
			}
			if count == 0 {
				continue
			}
			pr := rSum / count
			pg := gSum / count
			pb := bSum / count
			pa := aSum / count

			// Un-premultiply back to NRGBA (0-255).
			var nr, ng, nb uint8
			if pa > 0 {
				nr = uint8(clamp16(pr*0xffff/pa) >> 8)
				ng = uint8(clamp16(pg*0xffff/pa) >> 8)
				nb = uint8(clamp16(pb*0xffff/pa) >> 8)
			}
			i := dst.PixOffset(dx, dy)
			dst.Pix[i+0] = nr
			dst.Pix[i+1] = ng
			dst.Pix[i+2] = nb
			dst.Pix[i+3] = uint8(pa >> 8)
		}
	}
	return dst
}

func clamp16(v uint64) uint64 {
	if v > 0xffff {
		return 0xffff
	}
	return v
}
