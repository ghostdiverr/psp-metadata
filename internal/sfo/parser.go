package sfo

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var sfoMagic = []byte{0x00, 'P', 'S', 'F'}

const (
	typeUTF8   uint16 = 0x0204
	typeUint32 uint16 = 0x0404
)

// Entry is a single key-value pair from PARAM.SFO.
type Entry struct {
	Key      string
	DataType uint16
	StrVal   string
	IntVal   uint32
}

// SFO holds all parsed entries from a PARAM.SFO file.
type SFO struct {
	Entries []Entry
	index   map[string]*Entry
}

type sfoHeader struct {
	Magic          [4]byte
	Version        uint32
	KeyTableOffset uint32
	DataTableOffset uint32
	EntryCount     uint32
}

type entryIndex struct {
	KeyOffset  uint16
	Alignment  uint8
	DataType   uint8 // low byte of the combined type field
	DataType2  uint8 // high byte — combined: uint16(DataType) | uint16(DataType2)<<8
	_          uint8 // padding
	DataSize   uint32
	MaxSize    uint32
	DataOffset uint32
}

// Parse parses a raw PARAM.SFO binary blob.
func Parse(data []byte) (*SFO, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("sfo: data too short")
	}
	if !bytes.HasPrefix(data, sfoMagic) {
		return nil, fmt.Errorf("sfo: invalid magic")
	}

	r := bytes.NewReader(data)

	var hdr sfoHeader
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("sfo: read header: %w", err)
	}

	if int(hdr.EntryCount) == 0 {
		return &SFO{index: make(map[string]*Entry)}, nil
	}

	s := &SFO{
		Entries: make([]Entry, 0, hdr.EntryCount),
		index:   make(map[string]*Entry, hdr.EntryCount),
	}

	// Entry index starts right after the 20-byte header.
	// Each entry is 16 bytes.
	for i := uint32(0); i < hdr.EntryCount; i++ {
		entryOff := 20 + int(i)*16
		if entryOff+16 > len(data) {
			return nil, fmt.Errorf("sfo: entry %d out of bounds", i)
		}
		e := data[entryOff : entryOff+16]

		keyOff := binary.LittleEndian.Uint16(e[0:2])
		// data_type is stored as two separate bytes:
		// e[2] = format/alignment, e[3] = type specifier
		// combined as uint16 little-endian: e[2] | e[3]<<8
		dataType := uint16(e[2]) | uint16(e[3])<<8
		dataSize := binary.LittleEndian.Uint32(e[4:8])
		// maxSize  := binary.LittleEndian.Uint32(e[8:12])  // not needed
		dataOffset := binary.LittleEndian.Uint32(e[12:16])

		// Decode key.
		keyStart := int(hdr.KeyTableOffset) + int(keyOff)
		if keyStart >= len(data) {
			return nil, fmt.Errorf("sfo: key offset out of bounds")
		}
		keyEnd := bytes.IndexByte(data[keyStart:], 0)
		if keyEnd < 0 {
			return nil, fmt.Errorf("sfo: key not null-terminated")
		}
		key := string(data[keyStart : keyStart+keyEnd])

		// Decode value.
		valStart := int(hdr.DataTableOffset) + int(dataOffset)
		if int(valStart)+int(dataSize) > len(data) {
			return nil, fmt.Errorf("sfo: value for %q out of bounds", key)
		}
		valData := data[valStart : valStart+int(dataSize)]

		entry := Entry{Key: key, DataType: dataType}
		switch dataType {
		case typeUTF8:
			// Strip null terminator(s).
			entry.StrVal = string(bytes.TrimRight(valData, "\x00"))
		case typeUint32:
			if len(valData) < 4 {
				return nil, fmt.Errorf("sfo: uint32 value for %q too short", key)
			}
			entry.IntVal = binary.LittleEndian.Uint32(valData[:4])
		default:
			// Unknown type — store raw as hex string for debug.
			entry.StrVal = fmt.Sprintf("(unknown type 0x%04X)", dataType)
		}

		s.Entries = append(s.Entries, entry)
		s.index[key] = &s.Entries[len(s.Entries)-1]
	}

	return s, nil
}

// Get returns the string value for key (converting uint32 to decimal if needed).
func (s *SFO) Get(key string) (string, bool) {
	e, ok := s.index[key]
	if !ok {
		return "", false
	}
	if e.DataType == typeUint32 {
		return fmt.Sprintf("%d", e.IntVal), true
	}
	return e.StrVal, true
}

// GetInt returns the uint32 value for key.
func (s *SFO) GetInt(key string) (uint32, bool) {
	e, ok := s.index[key]
	if !ok {
		return 0, false
	}
	if e.DataType != typeUint32 {
		return 0, false
	}
	return e.IntVal, true
}
