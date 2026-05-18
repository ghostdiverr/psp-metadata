package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "psp-metadata",
	Short: "Extract metadata and artworks from PSP game ROMs",
	Long:  "psp-metadata reads PSP ISO and CSO files to extract game info, artwork, and more.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print diagnostic info (timings, CSO stats)")
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(extractCmd)
}
