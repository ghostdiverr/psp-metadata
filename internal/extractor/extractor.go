package extractor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghostdiverr/psp-metadata/internal/iso"
)

// ArtworkPresence records which standard PSP artwork files exist.
type ArtworkPresence struct {
	Icon0 bool // ICON0.PNG 144x80  — primary game icon
	Pic0  bool // PIC0.PNG  480x272 — overlay/frame artwork
	Pic1  bool // PIC1.PNG  480x272 — background artwork
}

// ExtractedFile records a file written to disk.
type ExtractedFile struct {
	Name string
	Path string
	Size int64
}

// ReadSFO reads /PSP_GAME/PARAM.SFO from img.
func ReadSFO(img *iso.Image) ([]byte, error) {
	return img.ReadFile("/PSP_GAME/PARAM.SFO")
}

// DetectArtworks checks which standard artwork files exist in img.
func DetectArtworks(img *iso.Image) ArtworkPresence {
	return ArtworkPresence{
		Icon0: img.FileExists("/PSP_GAME/ICON0.PNG"),
		Pic0:  img.FileExists("/PSP_GAME/PIC0.PNG"),
		Pic1:  img.FileExists("/PSP_GAME/PIC1.PNG"),
	}
}

// ExtractArtworks writes available artwork files to outputDir.
// Missing artworks are silently skipped.
func ExtractArtworks(img *iso.Image, outputDir string) ([]ExtractedFile, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	artworks := []string{"ICON0.PNG", "PIC0.PNG", "PIC1.PNG"}
	var extracted []ExtractedFile

	for _, name := range artworks {
		isoPath := "/PSP_GAME/" + name
		data, err := img.ReadFile(isoPath)
		if err != nil {
			continue
		}
		destPath := filepath.Join(outputDir, name)
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return extracted, fmt.Errorf("write %s: %w", name, err)
		}
		extracted = append(extracted, ExtractedFile{
			Name: name,
			Path: destPath,
			Size: int64(len(data)),
		})
	}

	return extracted, nil
}
