package cmd

import (
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

func startSpinner(desc string) func() {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionClearOnFinish(),
	)
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				bar.Add(1)
			}
		}
	}()
	return func() {
		close(stop)
		bar.Finish()
	}
}
