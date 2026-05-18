package iso

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const sectorSize = 2048

// Image provides read access to an ISO 9660 filesystem.
type Image struct {
	ra         io.ReaderAt
	rootDirLoc uint32 // sector number of root directory
	rootDirLen uint32 // byte length of root directory data
}

// DirEntry is a parsed ISO 9660 directory record.
type DirEntry struct {
	ExtentLoc uint32 // starting sector of data
	ExtentLen uint32 // data length in bytes
	IsDir     bool
	Name      string
}

// OpenImage reads the Primary Volume Descriptor and returns an Image.
func OpenImage(ra io.ReaderAt) (*Image, error) {
	buf := make([]byte, sectorSize)
	if _, err := ra.ReadAt(buf, 16*sectorSize); err != nil {
		return nil, fmt.Errorf("read PVD sector: %w", err)
	}
	if buf[0] != 1 || string(buf[1:6]) != "CD001" {
		return nil, fmt.Errorf("invalid PVD: type=%d id=%q", buf[0], buf[1:6])
	}

	// Root directory record starts at offset 156 and is 34 bytes.
	root := buf[156:]
	loc := binary.LittleEndian.Uint32(root[2:6])
	length := binary.LittleEndian.Uint32(root[10:14])

	return &Image{ra: ra, rootDirLoc: loc, rootDirLen: length}, nil
}

// readDirectory reads all directory entries from a directory extent.
func (img *Image) readDirectory(loc uint32, length uint32) ([]DirEntry, error) {
	if length > 1*1024*1024 {
		return nil, fmt.Errorf("directory too large: %d bytes", length)
	}
	data := make([]byte, length)
	if _, err := img.ra.ReadAt(data, int64(loc)*sectorSize); err != nil {
		return nil, fmt.Errorf("read directory at sector %d: %w", loc, err)
	}

	var entries []DirEntry
	off := 0
	for off < len(data) {
		recLen := int(data[off])
		if recLen == 0 {
			// Zero padding to end of sector — skip to next.
			next := ((off / sectorSize) + 1) * sectorSize
			if next >= len(data) {
				break
			}
			off = next
			continue
		}
		if off+recLen > len(data) {
			break
		}

		idLen := int(data[off+32])
		if off+33+idLen > off+recLen {
			break
		}

		name := string(data[off+33 : off+33+idLen])
		flags := data[off+25]

		// Skip "." and ".." (identifiers 0x00 and 0x01).
		if idLen == 1 && (name[0] == 0 || name[0] == 1) {
			off += recLen
			continue
		}

		// Strip ISO 9660 ";1" version suffix.
		if idx := strings.Index(name, ";"); idx >= 0 {
			name = name[:idx]
		}
		// Strip trailing ".".
		name = strings.TrimRight(name, ".")

		entries = append(entries, DirEntry{
			ExtentLoc: binary.LittleEndian.Uint32(data[off+2 : off+6]),
			ExtentLen: binary.LittleEndian.Uint32(data[off+10 : off+14]),
			IsDir:     flags&0x02 != 0,
			Name:      name,
		})
		off += recLen
	}
	return entries, nil
}

// FindFile navigates the directory tree to locate the given absolute path
// (e.g. "/PSP_GAME/PARAM.SFO"). Matching is case-insensitive.
func (img *Image) FindFile(path string) (*DirEntry, error) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	dirLoc := img.rootDirLoc
	dirLen := img.rootDirLen

	for i, part := range parts {
		entries, err := img.readDirectory(dirLoc, dirLen)
		if err != nil {
			return nil, err
		}

		var found *DirEntry
		for j := range entries {
			if strings.EqualFold(entries[j].Name, part) {
				found = &entries[j]
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("%q not found in ISO", path)
		}
		if i == len(parts)-1 {
			return found, nil
		}
		if !found.IsDir {
			return nil, fmt.Errorf("%q is not a directory", strings.Join(parts[:i+1], "/"))
		}
		dirLoc = found.ExtentLoc
		dirLen = found.ExtentLen
	}
	return nil, fmt.Errorf("%q not found in ISO", path)
}

// ReadFile reads the entire file at the given ISO path into memory.
func (img *Image) ReadFile(path string) ([]byte, error) {
	entry, err := img.FindFile(path)
	if err != nil {
		return nil, err
	}
	if entry.ExtentLen > 16*1024*1024 {
		return nil, fmt.Errorf("%q: file too large (%d bytes)", path, entry.ExtentLen)
	}
	buf := make([]byte, entry.ExtentLen)
	if _, err := img.ra.ReadAt(buf, int64(entry.ExtentLoc)*sectorSize); err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return buf, nil
}

// FileExists reports whether the given ISO path exists.
func (img *Image) FileExists(path string) bool {
	_, err := img.FindFile(path)
	return err == nil
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}
