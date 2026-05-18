package cmd

import (
	"fmt"
	"time"

	"github.com/ghostdiverr/psp-metadata/internal/extractor"
	"github.com/ghostdiverr/psp-metadata/internal/iso"
	"github.com/spf13/cobra"
)

var outputDir string

var extractCmd = &cobra.Command{
	Use:   "extract <file.iso|file.cso>",
	Short: "Extract artwork images from a PSP ROM",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtract,
}

func init() {
	extractCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Directory to write artwork files")
}

func runExtract(cmd *cobra.Command, args []string) error {
	path := args[0]

	stopSpinner := startSpinner("Extraction du ROM...")

	t0 := time.Now()
	logf("opening image...")
	result, err := iso.Open(path)
	logf("iso.Open done in %s", time.Since(t0).Round(time.Millisecond))
	if err != nil {
		stopSpinner()
		return err
	}
	defer result.Closer.Close()
	if result.CSO != nil {
		logf("CSO header: %s", result.CSO.Info())
	}

	t1 := time.Now()
	logf("extracting artworks...")
	files, err := extractor.ExtractArtworks(result.Img, outputDir)
	logf("ExtractArtworks done in %s", time.Since(t1).Round(time.Millisecond))
	stopSpinner()
	if err != nil {
		return fmt.Errorf("extract artworks: %w", err)
	}

	if result.CSO != nil {
		logf("CSO blocks decompressed (total): %d", result.CSO.BlocksRead)
	}

	if len(files) == 0 {
		fmt.Println("No artwork files found in ROM.")
		return nil
	}

	fmt.Printf("Extracted %d artwork(s) to %s:\n", len(files), outputDir)
	for _, f := range files {
		fmt.Printf("  %s  (%d bytes)\n", f.Name, f.Size)
	}
	return nil
}
