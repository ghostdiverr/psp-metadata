package cso

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

var magic = [4]byte{'C', 'I', 'S', 'O'}

type header struct {
	Magic      [4]byte
	HeaderSize uint32
	TotalBytes uint64
	BlockSize  uint32
	Version    uint8
	Align      uint8
	Reserved   [2]uint8
}

const maxCacheBlocks = 256

// Reader wraps an *os.File and exposes the decompressed ISO data
// as an io.ReaderAt and io.ReadSeeker.
type Reader struct {
	f            *os.File
	hdr          header
	blockIndex   []uint32
	cache        map[int][]byte
	numBlocks    int
	pos          int64
	BlocksRead   int // total decompressBlock calls (for diagnostics)
}

// Info returns a human-readable summary of the CSO header (for diagnostics).
func (r *Reader) Info() string {
	return fmt.Sprintf("blockSize=%d align=%d numBlocks=%d totalBytes=%d",
		r.hdr.BlockSize, r.hdr.Align, r.numBlocks, r.hdr.TotalBytes)
}

// NewReader parses the CSO header and block index from f.
func NewReader(f *os.File) (*Reader, error) {
	var hdr header
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("cso: read header: %w", err)
	}
	if hdr.Magic != magic {
		return nil, fmt.Errorf("cso: invalid magic %q", hdr.Magic)
	}
	if hdr.BlockSize == 0 {
		return nil, fmt.Errorf("cso: block size is zero")
	}

	numBlocks := int((hdr.TotalBytes + uint64(hdr.BlockSize) - 1) / uint64(hdr.BlockSize))

	// Read numBlocks+1 index entries (the extra entry gives the end offset of the last block).
	blockIndex := make([]uint32, numBlocks+1)
	if err := binary.Read(f, binary.LittleEndian, &blockIndex); err != nil {
		return nil, fmt.Errorf("cso: read block index: %w", err)
	}

	return &Reader{
		f:          f,
		hdr:        hdr,
		blockIndex: blockIndex,
		cache:      make(map[int][]byte),
		numBlocks:  numBlocks,
	}, nil
}

// Size returns the total uncompressed size of the ISO.
func (r *Reader) Size() int64 {
	return int64(r.hdr.TotalBytes)
}

// ReadAt implements io.ReaderAt.
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(r.hdr.TotalBytes) {
		return 0, io.EOF
	}

	blockSize := int64(r.hdr.BlockSize)
	startBlock := int(off / blockSize)
	endBlock := int((off + int64(len(p)) - 1) / blockSize)
	if endBlock >= r.numBlocks {
		endBlock = r.numBlocks - 1
	}

	written := 0
	for b := startBlock; b <= endBlock; b++ {
		block, err := r.decompressBlock(b)
		if err != nil {
			return written, fmt.Errorf("cso: block %d: %w", b, err)
		}

		// Byte range within this block that we need.
		blockStart := int64(b) * blockSize
		inBlockOff := int(off + int64(written) - blockStart)
		avail := len(block) - inBlockOff
		need := len(p) - written
		if avail <= 0 {
			break
		}
		n := avail
		if n > need {
			n = need
		}
		copy(p[written:written+n], block[inBlockOff:inBlockOff+n])
		written += n
	}

	if written < len(p) {
		return written, io.EOF
	}
	return written, nil
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.pos)
	r.pos += int64(n)
	return n, err
}

// Seek implements io.Seeker.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.pos + offset
	case io.SeekEnd:
		abs = int64(r.hdr.TotalBytes) + offset
	default:
		return 0, fmt.Errorf("cso: invalid whence %d", whence)
	}
	if abs < 0 {
		return 0, fmt.Errorf("cso: negative seek position")
	}
	r.pos = abs
	return abs, nil
}

// decompressBlock auto-detects zlib (0x78 CMF byte) vs raw deflate.
// Modern tools like maxcso write raw deflate; older ciso tools write zlib-wrapped deflate.
// limit caps the decompressed output to prevent deflate bombs from exhausting memory.
func decompressBlock(data []byte, limit int64) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty compressed block")
	}
	// zlib streams always start with a CMF byte where bits 0-3 == 8 (deflate)
	// and the two-byte header (CMF, FLG) must satisfy (CMF*256 + FLG) % 31 == 0.
	// The most common values are 0x78 0x01, 0x78 0x9C, 0x78 0xDA.
	// We validate the checksum before trying zlib to avoid false positives on
	// raw deflate blocks that happen to start with 0x78.
	if len(data) >= 2 && data[0] == 0x78 && (uint(data[0])*256+uint(data[1]))%31 == 0 {
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err == nil {
			out, err := io.ReadAll(io.LimitReader(zr, limit))
			zr.Close()
			if err == nil {
				return out, nil
			}
		}
	}
	// Fall back to raw deflate (no zlib framing).
	fr := flate.NewReader(bytes.NewReader(data))
	out, err := io.ReadAll(io.LimitReader(fr, limit))
	fr.Close()
	if err != nil {
		return nil, fmt.Errorf("deflate decompress: %w", err)
	}
	return out, nil
}

func (r *Reader) decompressBlock(blockIdx int) ([]byte, error) {
	if cached, ok := r.cache[blockIdx]; ok {
		return cached, nil
	}
	r.BlocksRead++

	indexEntry := r.blockIndex[blockIdx]
	nextEntry := r.blockIndex[blockIdx+1]

	isRaw := (indexEntry >> 31) & 1
	dataOffset := int64(indexEntry&0x7FFFFFFF) << r.hdr.Align
	nextOffset := int64(nextEntry&0x7FFFFFFF) << r.hdr.Align
	compressedSize := nextOffset - dataOffset

	// A compressed block can't be larger than 2× the uncompressed block size.
	// Guard against a corrupt index that would trigger a multi-GB allocation.
	if compressedSize <= 0 || compressedSize > int64(r.hdr.BlockSize)*2 {
		return nil, fmt.Errorf("invalid compressed block size %d for block %d", compressedSize, blockIdx)
	}

	compressed := make([]byte, compressedSize)
	if _, err := r.f.ReadAt(compressed, dataOffset); err != nil {
		return nil, fmt.Errorf("read compressed data: %w", err)
	}

	var block []byte
	if isRaw == 1 {
		block = compressed
	} else {
		decompressed, err := decompressBlock(compressed, int64(r.hdr.BlockSize))
		if err != nil {
			return nil, err
		}
		block = decompressed
	}

	if len(r.cache) >= maxCacheBlocks {
		for k := range r.cache {
			delete(r.cache, k)
			break
		}
	}
	r.cache[blockIdx] = block
	return block, nil
}
