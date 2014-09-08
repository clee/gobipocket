package mobipocket

import (
	"fmt"
	"bytes"
)

func palmdoc_unpack(data []byte) string {
	var buffer bytes.Buffer
	fmt.Printf("data: %d bytes\n", len(data))

	for position := 0; position < len(data); {
		currentByte := int(data[position])
		position += 1

		if currentByte > 0x00 && currentByte < 0x09 {
			buffer.WriteString(string(data[position:position+int(currentByte)]))
		} else if currentByte < 0x80 {
			buffer.WriteString(string(currentByte))
		} else if currentByte >= 0xC0 {
			buffer.WriteString(" " + string(currentByte ^ 128))
		} else {
			if position < len(data) {
				currentByte = ((int(currentByte) << 8) | int(data[position]))
				position += 1
				distance := (int(currentByte) >> 3) & 0x07FF
				length := (int(currentByte) & 7) + 3
				if distance > length {
					b := buffer.Bytes()
					o := string(b[len(b)-distance:len(b) + length - distance])
					buffer.WriteString(o)
				} else {
					// fmt.Printf("distance / length: %d / %d @%d\n", distance, length, position)
					for i := 0; i < length; i++ {
						o := buffer.String()
						buffer.WriteString(string(o[len(o) - distance]))
					}
				}
			}
		}
	}

	s := buffer.String()
	fmt.Printf("unpacked %d bytes\n", len(s))
	return s
}
