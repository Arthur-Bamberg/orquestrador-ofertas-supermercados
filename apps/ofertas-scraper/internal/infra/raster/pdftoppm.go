package raster

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/image/draw"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// Pdftoppm rasterizes PDF pages via poppler's pdftoppm, then downscales (ADR 0002).
type Pdftoppm struct {
	MaxEdgePx   int
	JPEGQuality int
	Bin         string // default "pdftoppm"
}

func NewPdftoppm(maxEdgePx, jpegQuality int) *Pdftoppm {
	if maxEdgePx <= 0 {
		maxEdgePx = 1280
	}
	if jpegQuality <= 0 {
		jpegQuality = 80
	}
	return &Pdftoppm{MaxEdgePx: maxEdgePx, JPEGQuality: jpegQuality, Bin: "pdftoppm"}
}

func (r *Pdftoppm) Rasterize(ctx context.Context, pdf []byte) ([]domain.PageImage, error) {
	dir, err := os.MkdirTemp("", "ofertas-raster-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(pdfPath, pdf, 0o644); err != nil {
		return nil, err
	}
	prefix := filepath.Join(dir, "page")
	bin := r.Bin
	if bin == "" {
		bin = "pdftoppm"
	}
	cmd := exec.CommandContext(ctx, bin, "-png", pdfPath, prefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, string(out))
	}

	var pages []domain.PageImage
	for i := 1; ; i++ {
		path := fmt.Sprintf("%s-%d.png", prefix, i)
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return nil, err
		}
		img, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		scaled := downscale(img, r.MaxEdgePx)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: r.JPEGQuality}); err != nil {
			return nil, err
		}
		pages = append(pages, domain.PageImage{Page: i, JPEG: buf.Bytes()})
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no pages")
	}
	return pages, nil
}

func downscale(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := float64(maxEdge) / float64(w)
	if h > w {
		scale = float64(maxEdge) / float64(h)
	}
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// Fixed returns predetermined page images (tests / stub raster).
type Fixed struct {
	Pages []domain.PageImage
	Err   error
}

func (f Fixed) Rasterize(context.Context, []byte) ([]domain.PageImage, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Pages, nil
}
