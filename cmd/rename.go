package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghostdiverr/psp-metadata/internal/extractor"
	"github.com/ghostdiverr/psp-metadata/internal/iso"
	"github.com/ghostdiverr/psp-metadata/internal/sfo"
	"github.com/spf13/cobra"
)

var dryRun bool

var renameCmd = &cobra.Command{
	Use:   "rename <file.iso|file.cso> [...]",
	Short: "Rename ROM files to 'Title [DISC_ID].ext'",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRename,
}

func init() {
	renameCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be renamed without doing it")
	rootCmd.AddCommand(renameCmd)
}

// sanitize replaces characters that are problematic in filenames.
func sanitize(s string) string {
	r := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", " -",
		"*", "",
		"?", "",
		"\"", "'",
		"<", "",
		">", "",
		"|", "-",
		"™", "",
		"®", "",
		"©", "",
	)
	s = r.Replace(s)
	// Collapse multiple spaces.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

func runRename(cmd *cobra.Command, args []string) error {
	for _, path := range args {
		if err := renameOne(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
	}
	return nil
}

func renameOne(path string) error {
	t0 := time.Now()
	result, err := iso.Open(path)
	if err != nil {
		return err
	}
	defer result.Closer.Close()
	logf("opened %s in %s", filepath.Base(path), time.Since(t0).Round(time.Millisecond))

	sfoData, err := extractor.ReadSFO(result.Img)
	if err != nil {
		return fmt.Errorf("read PARAM.SFO: %w", err)
	}

	parsed, err := sfo.Parse(sfoData)
	if err != nil {
		return fmt.Errorf("parse PARAM.SFO: %w", err)
	}

	title, _ := parsed.Get("TITLE")
	discID, _ := parsed.Get("DISC_ID")
	if title == "" || discID == "" {
		return fmt.Errorf("missing TITLE or DISC_ID in PARAM.SFO")
	}

	ext := strings.ToLower(filepath.Ext(path))

	baseName := sanitize(title)

	newName := fmt.Sprintf("%s [%s]%s", baseName, discID, ext)
	newPath := filepath.Join(filepath.Dir(path), newName)

	if newPath == path {
		fmt.Printf("  ✓ %s (already correct)\n", filepath.Base(path))
		return nil
	}

	if dryRun {
		fmt.Printf("  %s\n  → %s\n\n", filepath.Base(path), newName)
		return nil
	}

	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target already exists: %s", newName)
	}

	if err := os.Rename(path, newPath); err != nil {
		return err
	}
	fmt.Printf("  %s\n  → %s\n\n", filepath.Base(path), newName)
	return nil
}
