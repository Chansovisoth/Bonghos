package image

import (
	"bytes"
	stdimage "image"
	"image/color"
	"image/png"
	"testing"
)

func encodedTestIcon(t *testing.T) []byte {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 160, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 160; x++ {
			pixel := color.RGBA{R: 240, A: 255}
			if x >= 80 {
				pixel = color.RGBA{B: 240, A: 255}
			}
			img.SetRGBA(x, y, pixel)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestProcessAlwaysProduces64PixelPNG(t *testing.T) {
	processed, err := Process(encodedTestIcon(t), CropSpec{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := stdimage.DecodeConfig(bytes.NewReader(processed))
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" || cfg.Width != IconSize || cfg.Height != IconSize {
		t.Fatalf("got %s %dx%d, want png %dx%d", format, cfg.Width, cfg.Height, IconSize, IconSize)
	}
}

func TestProcessUsesExplicitCrop(t *testing.T) {
	processed, err := Process(encodedTestIcon(t), CropSpec{X: 80, Y: 0, Size: 80})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(processed))
	if err != nil {
		t.Fatal(err)
	}
	r, _, b, _ := decoded.At(IconSize/2, IconSize/2).RGBA()
	if b <= r {
		t.Fatalf("cropped center is not from the selected blue half: red=%d blue=%d", r, b)
	}
}
