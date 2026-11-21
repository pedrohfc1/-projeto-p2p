package profile

import (
	"bytes"
	"encoding/binary"
	"errors"

	"golang.org/x/image/vp8"
	"golang.org/x/image/vp8l"
)

// The extended WebP canvas can declare dimensions different from its encoded
// frame. Bound both before Decode can allocate pixels. The editor produces
// single-frame images, optionally with a bounded color profile.
func validateWebP(data []byte, side int) error {
	invalid := errors.New("invalid static WebP profile image")
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || binary.LittleEndian.Uint32(data[4:8]) != uint32(len(data)-8) {
		return invalid
	}
	var flags byte
	extended, colorProfile, alpha, frame := false, false, false, false
	for offset := 12; offset < len(data); {
		if len(data)-offset < 8 || frame {
			return invalid
		}
		kind := string(data[offset : offset+4])
		size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		padded := size + size%2
		if padded > uint64(len(data)-offset) {
			return invalid
		}
		chunk := data[offset : offset+int(size)]
		offset += int(padded)
		switch kind {
		case "VP8X":
			if extended || len(chunk) != 10 || chunk[0]&^byte(0x30) != 0 {
				return invalid
			}
			extended = true
			flags = chunk[0]
			width := 1 + int(chunk[4]) + (int(chunk[5]) << 8) + (int(chunk[6]) << 16)
			height := 1 + int(chunk[7]) + (int(chunk[8]) << 8) + (int(chunk[9]) << 16)
			if width != side || height != side {
				return invalid
			}
		case "ICCP":
			// Chromium canvas includes an sRGB profile even for freshly drawn
			// images. This is color data, not source EXIF or XMP.
			if flags&0x20 == 0 || colorProfile || alpha || len(chunk) < 128 || len(chunk) > 8*1024 ||
				binary.BigEndian.Uint32(chunk[:4]) != uint32(len(chunk)) || string(chunk[36:40]) != "acsp" {
				return invalid
			}
			colorProfile = true
		case "ALPH":
			if flags&0x10 == 0 || alpha {
				return invalid
			}
			alpha = true
		case "VP8 ":
			dec := vp8.NewDecoder()
			dec.Init(bytes.NewReader(chunk), len(chunk))
			header, err := dec.DecodeFrameHeader()
			if err != nil || header.Width != side || header.Height != side {
				return invalid
			}
			frame = true
		case "VP8L":
			if alpha {
				return invalid
			}
			config, err := vp8l.DecodeConfig(bytes.NewReader(chunk))
			if err != nil || config.Width != side || config.Height != side {
				return invalid
			}
			frame = true
		default:
			return invalid
		}
	}
	if !frame || (flags&0x20 != 0) != colorProfile {
		return invalid
	}
	return nil
}
