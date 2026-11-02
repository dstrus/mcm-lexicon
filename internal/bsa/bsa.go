// Package bsa reads Bethesda archives as Skyrim uses them (versions 104
// and 105) far enough to list their files and extract the few MCM Lexicon
// reads. Archives are read in place, never unpacked to disk.
package bsa

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Versions: the original Skyrim's archives, and Skyrim SE's (which Skyrim
// VR also loads).
const (
	VersionSkyrim   = 104
	VersionSkyrimSE = 105
)

// Archive flags MCM Lexicon reads.
const (
	flagDirNames   = 0x1
	flagFileNames  = 0x2
	flagCompressed = 0x4   // files are compressed unless their own bit says otherwise
	flagEmbedNames = 0x100 // each file's data starts with its full path
)

// A file's size carries a bit that flips the archive's compression default.
const (
	sizeCompressionBit = 0x40000000
	sizeMask           = 0x3FFFFFFF
)

// maxFileSize bounds what one file can claim to unpack to, so a damaged
// archive can't ask for gigabytes.
const maxFileSize = 256 << 20

// Archive is an archive's table of contents.
type Archive struct {
	Path    string // where it was read from; "" when read from memory
	Version uint32
	Flags   uint32
	Files   []File // in the archive's order
}

// File is one file in an archive.
type File struct {
	Path       string // lowercase, with forward slashes, as the archive spells it
	Size       uint32 // as stored: compressed, and with any embedded name
	Compressed bool
	offset     uint32
}

// Open reads the table of contents of the archive at path.
func Open(path string) (*Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	a, err := Read(f, info.Size())
	if err != nil {
		return nil, err
	}
	a.Path = path
	return a, nil
}

// Read reads an archive's table of contents from r, which is size bytes.
func Read(r io.ReaderAt, size int64) (*Archive, error) {
	var h struct {
		Magic         [4]byte
		Version       uint32
		FolderOffset  uint32
		Flags         uint32
		FolderCount   uint32
		FileCount     uint32
		FolderNameLen uint32
		FileNameLen   uint32
		FileFlags     uint16
		_             uint16
	}
	if err := binary.Read(io.NewSectionReader(r, 0, size), binary.LittleEndian, &h); err != nil {
		return nil, errors.New("too short to be an archive")
	}
	if string(h.Magic[:]) != "BSA\x00" {
		return nil, errors.New("not a Bethesda archive")
	}
	var folderRecord int64
	switch h.Version {
	case VersionSkyrim:
		folderRecord = 16 // hash, count, offset
	case VersionSkyrimSE:
		folderRecord = 24 // hash, count, padding, 64-bit offset
	default:
		return nil, fmt.Errorf("archive version %d isn't one Skyrim uses", h.Version)
	}
	if h.Flags&flagDirNames == 0 || h.Flags&flagFileNames == 0 {
		return nil, errors.New("the archive doesn't store its file names")
	}
	// Every count must fit in the file before anything is allocated.
	if int64(h.FolderCount)*folderRecord > size || int64(h.FileCount)*16 > size || int64(h.FileNameLen) > size {
		return nil, errors.New("the archive's table of contents is larger than the archive")
	}

	sr := io.NewSectionReader(r, int64(h.FolderOffset), size-int64(h.FolderOffset))
	rd := &reader{r: sr}
	counts := make([]uint32, h.FolderCount)
	for i := range counts {
		rec := rd.bytes(int(folderRecord))
		if rd.err != nil {
			break
		}
		counts[i] = binary.LittleEndian.Uint32(rec[8:])
	}
	a := &Archive{Version: h.Version, Flags: h.Flags}
	var total uint64
	folders := make([]string, h.FolderCount)
	for i, n := range counts {
		total += uint64(n)
		if total > uint64(h.FileCount) {
			return nil, errors.New("the archive's folders hold more files than it says it has")
		}
		folders[i] = rd.bstringz()
		for range n {
			rec := rd.bytes(16)
			if rd.err != nil {
				break
			}
			size := binary.LittleEndian.Uint32(rec[8:])
			a.Files = append(a.Files, File{
				Path:       folders[i],
				Size:       size & sizeMask,
				Compressed: (h.Flags&flagCompressed != 0) != (size&sizeCompressionBit != 0),
				offset:     binary.LittleEndian.Uint32(rec[12:]),
			})
		}
	}
	names := rd.bytes(int(h.FileNameLen))
	if rd.err != nil {
		return nil, fmt.Errorf("reading the archive's table of contents: %w", rd.err)
	}
	for i := range a.Files {
		end := bytes.IndexByte(names, 0)
		if end < 0 {
			return nil, errors.New("the archive has fewer file names than files")
		}
		name := string(names[:end])
		names = names[end+1:]
		a.Files[i].Path = join(a.Files[i].Path, name)
	}
	return a, nil
}

// join is a file's path from its folder's and its own name: lowercase,
// with forward slashes, as MCM Lexicon's other paths are.
func join(folder, name string) string {
	folder = strings.Trim(strings.ReplaceAll(folder, `\`, "/"), "/")
	if folder == "" || folder == "." {
		return strings.ToLower(name)
	}
	return strings.ToLower(folder + "/" + name)
}

// ReadFile is f's contents, read from the archive's file.
func (a *Archive) ReadFile(f File) ([]byte, error) {
	if a.Path == "" {
		return nil, errors.New("the archive wasn't read from a file")
	}
	h, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer h.Close()
	return a.ReadFrom(h, f)
}

// ReadFrom is f's contents, read from r, the archive's bytes.
func (a *Archive) ReadFrom(r io.ReaderAt, f File) ([]byte, error) {
	raw := make([]byte, f.Size)
	if _, err := r.ReadAt(raw, int64(f.offset)); err != nil {
		return nil, fmt.Errorf("reading %s from the archive: %w", f.Path, err)
	}
	if a.Flags&flagEmbedNames != 0 {
		if len(raw) == 0 || int(raw[0])+1 > len(raw) {
			return nil, fmt.Errorf("%s: its embedded name runs past its data", f.Path)
		}
		raw = raw[1+int(raw[0]):]
	}
	if !f.Compressed {
		return raw, nil
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("%s: too short to be compressed", f.Path)
	}
	want := binary.LittleEndian.Uint32(raw)
	if want > maxFileSize {
		return nil, fmt.Errorf("%s: claims to unpack to %d bytes", f.Path, want)
	}
	var out []byte
	var err error
	if a.Version == VersionSkyrim {
		out, err = inflate(raw[4:], int(want))
	} else {
		out, err = lz4Frame(raw[4:], int(want))
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", f.Path, err)
	}
	if len(out) != int(want) {
		return nil, fmt.Errorf("%s: unpacked to %d bytes, not %d", f.Path, len(out), want)
	}
	return out, nil
}

func inflate(b []byte, want int) ([]byte, error) {
	z, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	out := make([]byte, 0, want)
	buf := bytes.NewBuffer(out)
	if _, err := io.Copy(buf, io.LimitReader(z, int64(want)+1)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type reader struct {
	r   io.Reader
	err error
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r.r, b); err != nil {
		r.err = err
		return nil
	}
	return b
}

// bstringz is a folder name: a length byte, then that many bytes ending
// in a zero.
func (r *reader) bstringz() string {
	n := r.bytes(1)
	if n == nil {
		return ""
	}
	b := r.bytes(int(n[0]))
	return string(bytes.TrimRight(b, "\x00"))
}
