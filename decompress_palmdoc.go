package mobipocket

import (
	"bytes"
)

func palmdoc_unpack(data []byte) string {
	var buffer bytes.Buffer
	datalen := len(data)

	for position := 0; position < datalen; {
		c := int(data[position])
		position += 1
		if c >= 0xC0 {
			buffer.WriteByte(' ')
			buffer.WriteByte(byte(c) ^ 0x80)
		} else if c >= 0x80 {
			next := int(data[position])
			position += 1
			distance := ((((c << 8) | next) >> 3) & 0x7FF)
			length := (next & 0x7) + 3;
			for j := 0; j < length; j++ {
				b := buffer.Bytes()
				buffer.WriteByte(b[len(b) - distance])
			}
		} else if c >= 0x09 {
			buffer.WriteByte(byte(c))
		} else if c >= 0x01 {
			for j := 0; j < c; j++ {
				buffer.WriteByte(data[position + j])
			}
			position += c
		} else {
			buffer.WriteByte(byte(c))
		}
	}

	s := buffer.String()
	return s
}
