package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, 0, color.NRGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSaveImageDownscalesToJPEG(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, "/uploads/")
	if err != nil {
		t.Fatal(err)
	}

	url, err := store.SaveImage(bytes.NewReader(pngBytes(t, 3200, 1600)))
	if err != nil {
		t.Fatalf("SaveImage() error = %v", err)
	}
	if !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("url = %q", url)
	}

	f, err := os.Open(filepath.Join(dir, strings.TrimPrefix(url, "/uploads/")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		t.Fatalf("stored file is not a JPEG: %v", err)
	}
	if cfg.Width != MaxDimension || cfg.Height != MaxDimension/2 {
		t.Errorf("size = %dx%d, want %dx%d", cfg.Width, cfg.Height, MaxDimension, MaxDimension/2)
	}
}

func TestSaveImageKeepsSmallImages(t *testing.T) {
	store, _ := NewStore(t.TempDir(), "/uploads/")
	url, err := store.SaveImage(bytes.NewReader(pngBytes(t, 300, 400)))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(filepath.Join(store.Dir(), strings.TrimPrefix(url, "/uploads/")))
	defer func() { _ = f.Close() }()
	cfg, _ := jpeg.DecodeConfig(f)
	if cfg.Width != 300 || cfg.Height != 400 {
		t.Errorf("size = %dx%d, want 300x400", cfg.Width, cfg.Height)
	}
}

func TestSaveImageRejectsNonImages(t *testing.T) {
	store, _ := NewStore(t.TempDir(), "/uploads/")
	_, err := store.SaveImage(strings.NewReader("<svg onload=alert(1)>"))
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("SaveImage(svg) error = %v, want ErrUnsupported", err)
	}
}
