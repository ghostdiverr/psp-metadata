package iso

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ghostdiverr/psp-metadata/internal/cso"
)

// Result holds the opened image plus diagnostics.
type Result struct {
	Img    *Image
	Closer io.Closer
	CSO    *cso.Reader // nil for plain ISO
}

// Open opens a PSP ROM file (ISO or CSO).
func Open(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	var magic [4]byte
	if _, err := f.ReadAt(magic[:], 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("read magic: %w", err)
	}

	ext := strings.ToLower(path[max(0, len(path)-4):])
	isCSOByMagic := magic == [4]byte{'C', 'I', 'S', 'O'}
	isCSOByExt := ext == ".cso"

	if isCSOByMagic || isCSOByExt {
		cr, err := cso.NewReader(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("cso: %w", err)
		}
		img, err := OpenImage(cr)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("iso9660 (cso): %w", err)
		}
		return &Result{Img: img, Closer: f, CSO: cr}, nil
	}

	// Plain ISO.
	img, err := OpenImage(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("iso9660: %w", err)
	}
	return &Result{Img: img, Closer: f}, nil
}
