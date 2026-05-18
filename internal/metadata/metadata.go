package metadata

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"os"

	"github.com/ghostdiverr/psp-metadata/internal/extractor"
	"github.com/ghostdiverr/psp-metadata/internal/sfo"
)

// GameInfo is the domain model for PSP game metadata.
type GameInfo struct {
	Title         string                    `json:"title"`
	DiscID        string                    `json:"disc_id"`
	Region        string                    `json:"region"`
	Category      string                    `json:"category"`
	DiscVersion   string                    `json:"disc_version"`
	SystemVersion string                    `json:"system_version"`
	ParentalLevel uint32                    `json:"parental_level"`
	FileSize      int64                     `json:"file_size_bytes"`
	Artworks      extractor.ArtworkPresence `json:"artworks"`
}

// FromSFO builds a GameInfo from a parsed SFO and file size.
func FromSFO(s *sfo.SFO, fileSize int64) GameInfo {
	g := GameInfo{FileSize: fileSize}

	if v, ok := s.Get("TITLE"); ok {
		g.Title = v
	}
	if v, ok := s.Get("DISC_ID"); ok {
		g.DiscID = v
	}
	if v, ok := s.Get("CATEGORY"); ok {
		g.Category = humanCategory(v)
	}
	if v, ok := s.Get("DISC_VERSION"); ok {
		g.DiscVersion = v
	}
	if v, ok := s.Get("PSP_SYSTEM_VER"); ok {
		g.SystemVersion = v
	}
	if v, ok := s.GetInt("PARENTAL_LEVEL"); ok {
		g.ParentalLevel = v
	}
	g.Region = RegionFromDiscID(g.DiscID)
	return g
}

// RegionFromDiscID derives a human-readable region string from the disc ID prefix.
func RegionFromDiscID(discID string) string {
	if len(discID) < 4 {
		return "Unknown"
	}
	prefix := strings.ToUpper(discID[:4])
	switch prefix {
	case "ULUS", "UCUS", "NPUH", "NPUG":
		return "North America"
	case "ULES", "UCES", "NPEH", "NPEG":
		return "Europe"
	case "ULJM", "UCJS", "NPJH", "NPJG":
		return "Japan"
	case "UCKS", "UCAS", "NPAH", "NPAG":
		return "Asia"
	case "ULKS":
		return "Korea"
	default:
		return "Unknown"
	}
}

func humanCategory(cat string) string {
	switch strings.ToUpper(cat) {
	case "UG":
		return "UMD Game"
	case "MS":
		return "Memory Stick"
	case "MG":
		return "Memory Stick Game"
	case "UC":
		return "UMD Video"
	default:
		return cat
	}
}

func yesNo(b bool) string {
	if b {
		return "[YES]"
	}
	return "[NO] "
}

func formatSize(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB (%d bytes)", float64(b)/GB, b)
	case b >= MB:
		return fmt.Sprintf("%.2f MB (%d bytes)", float64(b)/MB, b)
	case b >= KB:
		return fmt.Sprintf("%.2f KB (%d bytes)", float64(b)/KB, b)
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}

// PrintTable prints a formatted metadata table to stdout.
func PrintTable(g GameInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Field\tValue")
	fmt.Fprintln(w, "──────────────────\t──────────────────────────────────")
	fmt.Fprintf(w, "Title\t%s\n", g.Title)
	fmt.Fprintf(w, "Game ID\t%s\n", g.DiscID)
	fmt.Fprintf(w, "Region\t%s\n", g.Region)
	fmt.Fprintf(w, "Category\t%s\n", g.Category)
	fmt.Fprintf(w, "Disc Version\t%s\n", g.DiscVersion)
	fmt.Fprintf(w, "Min. Firmware\t%s\n", g.SystemVersion)
	if g.ParentalLevel > 0 {
		fmt.Fprintf(w, "Parental Level\t%d\n", g.ParentalLevel)
	}
	if g.FileSize > 0 {
		fmt.Fprintf(w, "File Size\t%s\n", formatSize(g.FileSize))
	}

	fmt.Fprintln(w, "\nArtwork\tStatus")
	fmt.Fprintln(w, "──────────────────\t──────")
	fmt.Fprintf(w, "ICON0.PNG (144x80)\t%s\n", yesNo(g.Artworks.Icon0))
	fmt.Fprintf(w, "PIC0.PNG  (480x272)\t%s\n", yesNo(g.Artworks.Pic0))
	fmt.Fprintf(w, "PIC1.PNG  (480x272)\t%s\n", yesNo(g.Artworks.Pic1))

	return w.Flush()
}

// ToJSON returns a pretty-printed JSON representation of g.
func ToJSON(g GameInfo) ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}
