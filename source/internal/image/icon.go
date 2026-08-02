// Package image safely processes server project icons. Any accepted PNG,
// JPEG or WebP is decoded with limits, EXIF-stripped (metadata is never
// copied), square-cropped, resized to exactly 64x64 and re-encoded as PNG.
// SVG is never accepted.
package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	_ "golang.org/x/image/webp"
)

const (
	MaxUploadBytes = 10 << 20 // 10 MiB
	MaxDimension   = 8192
	MaxPixels      = 40_000_000
	IconSize       = 64
	IconFilename   = "server-icon.png"
)

var (
	ErrTooLarge   = errors.New("icon upload too large")
	ErrBadImage   = errors.New("invalid or unsupported image (PNG, JPEG or WebP only; SVG is not accepted)")
	ErrDimensions = errors.New("image dimensions exceed limits")
)

// CropSpec selects a square region (source pixel coordinates). Zero value
// means automatic center crop.
type CropSpec struct {
	X, Y, Size int
}

// Process decodes, validates, crops, resizes and re-encodes an icon,
// returning the finished 64x64 PNG bytes.
func Process(data []byte, crop CropSpec) ([]byte, error) {
	if len(data) > MaxUploadBytes {
		return nil, ErrTooLarge
	}
	// decode config first: enforce dimension limits before full decode
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrBadImage
	}
	switch format {
	case "png", "jpeg", "webp":
	default:
		return nil, ErrBadImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxDimension || cfg.Height > MaxDimension ||
		cfg.Width*cfg.Height > MaxPixels {
		return nil, ErrDimensions
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrBadImage
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// crop to square (center by default; never stretch non-square images)
	size := crop.Size
	x, y := crop.X, crop.Y
	if size <= 0 {
		size = w
		if h < size {
			size = h
		}
		x = b.Min.X + (w-size)/2
		y = b.Min.Y + (h-size)/2
	} else {
		x += b.Min.X
		y += b.Min.Y
		if x < b.Min.X || y < b.Min.Y || x+size > b.Max.X || y+size > b.Max.Y {
			return nil, errors.New("crop region outside the image")
		}
	}

	// high-quality box resampling to exactly 64x64 (RGBA output)
	out := image.NewRGBA(image.Rect(0, 0, IconSize, IconSize))
	scale := float64(size) / IconSize
	for oy := 0; oy < IconSize; oy++ {
		for ox := 0; ox < IconSize; ox++ {
			// average the source box mapping onto this output pixel
			sx0 := float64(ox) * scale
			sy0 := float64(oy) * scale
			sx1 := sx0 + scale
			sy1 := sy0 + scale
			var r, g, bl, a, n float64
			for sy := int(sy0); sy < int(sy1+0.9999) && sy < size; sy++ {
				for sx := int(sx0); sx < int(sx1+0.9999) && sx < size; sx++ {
					pr, pg, pb, pa := img.At(x+sx, y+sy).RGBA()
					r += float64(pr)
					g += float64(pg)
					bl += float64(pb)
					a += float64(pa)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			out.SetRGBA(ox, oy, color.RGBA{
				R: uint8(r / n / 257), G: uint8(g / n / 257),
				B: uint8(bl / n / 257), A: uint8(a / n / 257),
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	// re-encoding drops all metadata by construction
	return buf.Bytes(), nil
}

// ValidateExisting checks a manually placed server-icon.png. Returns nil when
// it is a valid 64x64 PNG, or a descriptive error (the caller shows a warning
// and offers safe conversion; the file is never silently replaced).
func ValidateExisting(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("not a decodable image: %w", ErrBadImage)
	}
	if format != "png" {
		return fmt.Errorf("server-icon must be PNG, found %s", format)
	}
	if cfg.Width != IconSize || cfg.Height != IconSize {
		return fmt.Errorf("server-icon must be exactly 64x64, found %dx%d", cfg.Width, cfg.Height)
	}
	return nil
}

// InstallIcon atomically replaces serverDir/server-icon.png, backing up any
// previous icon beside it.
func InstallIcon(serverDir string, pngBytes []byte) error {
	dest := filepath.Join(serverDir, IconFilename)
	if _, err := os.Stat(dest); err == nil {
		prev, err := os.ReadFile(dest)
		if err == nil {
			os.WriteFile(dest+".previous", prev, 0o644)
		}
	}
	tmp, err := os.CreateTemp(serverDir, ".bonghos-icon-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(pngBytes); err != nil {
		tmp.Close()
		return err
	}
	tmp.Chmod(0o644)
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, dest)
}

var _ = draw.Draw
