package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ghostdiverr/psp-metadata/internal/extractor"
	"github.com/ghostdiverr/psp-metadata/internal/iso"
	"github.com/ghostdiverr/psp-metadata/internal/metadata"
	"github.com/ghostdiverr/psp-metadata/internal/sfo"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var infoCmd = &cobra.Command{
	Use:   "info <file.iso|file.cso>",
	Short: "Display metadata for a PSP ROM",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	infoCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
}

func logf(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}

func runInfo(cmd *cobra.Command, args []string) error {
	path := args[0]

	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
	}
	fileSize := fi.Size()
	logf("file size on disk: %d bytes", fileSize)

	stopSpinner := startSpinner("Lecture du ROM...")

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
	logf("reading PARAM.SFO...")
	var readSFODone = make(chan struct{})
	if verbose && result.CSO != nil {
		go func() {
			tick := time.NewTicker(3 * time.Second)
			defer tick.Stop()
			for {
				select {
				case <-readSFODone:
					return
				case <-tick.C:
					logf("  still reading... blocks decompressed so far: %d", result.CSO.BlocksRead)
				}
			}
		}()
	}
	sfoData, err := extractor.ReadSFO(result.Img)
	close(readSFODone)
	if result.CSO != nil {
		logf("ReadSFO done in %s (blocks decompressed: %d)", time.Since(t1).Round(time.Millisecond), result.CSO.BlocksRead)
	} else {
		logf("ReadSFO done in %s", time.Since(t1).Round(time.Millisecond))
	}
	if err != nil {
		stopSpinner()
		return fmt.Errorf("read PARAM.SFO: %w", err)
	}

	t2 := time.Now()
	logf("parsing SFO...")
	parsed, err := sfo.Parse(sfoData)
	logf("Parse done in %s", time.Since(t2).Round(time.Millisecond))
	if err != nil {
		stopSpinner()
		return fmt.Errorf("parse PARAM.SFO: %w", err)
	}

	t3 := time.Now()
	logf("detecting artworks...")
	artworks := extractor.DetectArtworks(result.Img)
	logf("DetectArtworks done in %s", time.Since(t3).Round(time.Millisecond))

	info := metadata.FromSFO(parsed, fileSize)
	info.Artworks = artworks

	if result.CSO != nil {
		logf("CSO blocks decompressed (total): %d", result.CSO.BlocksRead)
	}

	stopSpinner()

	if jsonOutput {
		data, err := metadata.ToJSON(info)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	return metadata.PrintTable(info)
}
