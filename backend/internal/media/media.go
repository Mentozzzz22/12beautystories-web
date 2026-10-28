// Package media stores uploaded images, downscaled and re-encoded as JPEG.
package media

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// Limits for uploads.
const (
	MaxUploadBytes = 15 << 20
	MaxDimension   = 1600
	jpegQuality    = 85
	// maxPixels guards against decompression bombs (tiny files with huge dimensions).
	maxPixels = 50_000_000
)

// ErrUnsupported is returned for files that are not JPEG, PNG or WebP images.
var ErrUnsupported = errors.New("unsupported image")

// Store writes images to a directory served under URLPrefix.
type Store struct {
	dir       string
	urlPrefix string
}

// NewStore creates the directory if needed.
func NewStore(dir, urlPrefix string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Store{dir: dir, urlPrefix: urlPrefix}, nil
}

// Dir is the storage directory.
func (s *Store) Dir() string { return s.dir }

// SaveImage decodes, downsizes to MaxDimension and stores the image. It returns its public URL.
// Re-encoding also strips EXIF metadata such as GPS location from phone photos.
func (s *Store) SaveImage(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	if len(data) > MaxUploadBytes {
		return "", fmt.Errorf("%w: file too large", ErrUnsupported)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrUnsupported
	}
	if cfg.Width*cfg.Height > maxPixels {
		return "", fmt.Errorf("%w: image too large", ErrUnsupported)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", ErrUnsupported
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resize(src, MaxDimension), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return "", fmt.Errorf("encode jpeg: %w", err)
	}

	name, err := randomName()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(s.dir, name), buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}
	return s.urlPrefix + name, nil
}

// resize scales the image so its longer side is at most limit, flattening transparency onto white.
func resize(src image.Image, limit int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > limit || h > limit {
		if w >= h {
			h = h * limit / w
			w = limit
		} else {
			w = w * limit / h
			h = limit
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random name: %w", err)
	}
	return hex.EncodeToString(b) + ".jpg", nil
}
