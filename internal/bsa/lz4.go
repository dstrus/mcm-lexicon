package bsa

import (
	"encoding/binary"
	"errors"
)

// lz4Frame unpacks LZ4 frames, as Skyrim SE's archives compress files:
// want is the size the archive says the file unpacks to, and output stops
// there. Checksums are skipped; the size check catches what matters here.
func lz4Frame(b []byte, want int) ([]byte, error) {
	out := make([]byte, 0, want)
	for len(b) > 0 {
		if len(b) < 7 {
			return nil, errors.New("LZ4: frame header cut short")
		}
		magic := binary.LittleEndian.Uint32(b)
		if magic&0xFFFFFFF0 == 0x184D2A50 { // a skippable frame
			if len(b) < 8 {
				return nil, errors.New("LZ4: skippable frame cut short")
			}
			n := int(binary.LittleEndian.Uint32(b[4:]))
			if n > len(b)-8 {
				return nil, errors.New("LZ4: skippable frame cut short")
			}
			b = b[8+n:]
			continue
		}
		if magic != 0x184D2204 {
			return nil, errors.New("LZ4: not a frame")
		}
		flg := b[4]
		if flg>>6 != 1 {
			return nil, errors.New("LZ4: unknown frame version")
		}
		blockSums, contentSize, contentSum, dictID := flg&0x10 != 0, flg&0x08 != 0, flg&0x04 != 0, flg&0x01 != 0
		if dictID {
			return nil, errors.New("LZ4: frames with a dictionary aren't supported")
		}
		at := 6 // magic, FLG, BD
		if contentSize {
			at += 8
		}
		at++ // header checksum
		if at > len(b) {
			return nil, errors.New("LZ4: frame header cut short")
		}
		b = b[at:]
		for {
			if len(b) < 4 {
				return nil, errors.New("LZ4: block cut short")
			}
			n := binary.LittleEndian.Uint32(b)
			b = b[4:]
			if n == 0 {
				break // the end mark
			}
			raw := n&0x80000000 != 0
			n &= 0x7FFFFFFF
			if int64(n) > int64(len(b)) {
				return nil, errors.New("LZ4: block cut short")
			}
			var err error
			if raw {
				if len(out)+int(n) > want {
					return nil, errors.New("LZ4: unpacks past the file's size")
				}
				out = append(out, b[:n]...)
			} else if out, err = lz4Block(b[:n], out, want); err != nil {
				return nil, err
			}
			b = b[n:]
			if blockSums {
				if len(b) < 4 {
					return nil, errors.New("LZ4: block checksum cut short")
				}
				b = b[4:]
			}
		}
		if contentSum {
			if len(b) < 4 {
				return nil, errors.New("LZ4: content checksum cut short")
			}
			b = b[4:]
		}
	}
	return out, nil
}

// lz4Block unpacks one LZ4 block onto out. A match may reach back into
// earlier blocks, which is what linked blocks need.
func lz4Block(src, out []byte, want int) ([]byte, error) {
	i := 0
	length := func(n int) (int, error) {
		if n != 15 {
			return n, nil
		}
		for {
			if i >= len(src) {
				return 0, errors.New("LZ4: length cut short")
			}
			c := src[i]
			i++
			n += int(c)
			if n > want {
				return 0, errors.New("LZ4: unpacks past the file's size")
			}
			if c != 255 {
				return n, nil
			}
		}
	}
	for i < len(src) {
		token := src[i]
		i++
		lit, err := length(int(token >> 4))
		if err != nil {
			return nil, err
		}
		if lit > len(src)-i {
			return nil, errors.New("LZ4: literals cut short")
		}
		if len(out)+lit > want {
			return nil, errors.New("LZ4: unpacks past the file's size")
		}
		out = append(out, src[i:i+lit]...)
		i += lit
		if i == len(src) {
			return out, nil // the last sequence has no match
		}
		if i+2 > len(src) {
			return nil, errors.New("LZ4: match offset cut short")
		}
		offset := int(src[i]) | int(src[i+1])<<8
		i += 2
		if offset == 0 || offset > len(out) {
			return nil, errors.New("LZ4: match reaches before the start")
		}
		match, err := length(int(token & 15))
		if err != nil {
			return nil, err
		}
		match += 4
		if len(out)+match > want {
			return nil, errors.New("LZ4: unpacks past the file's size")
		}
		from := len(out) - offset
		for k := range match { // byte by byte: a match may overlap what it writes
			out = append(out, out[from+k])
		}
	}
	return out, nil
}
