package mobipocket

import (
	"io"
	"fmt"
	"os"
	"encoding/binary"
)

type Mobipocket struct {
	Metadata map[string][]string
}

// Open reads the file into memory and parses the headers to
// populate the Metadata
func Open(path string) (m *Mobipocket, e error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}

	m = new(Mobipocket)
	m.Metadata = parse(f)

	return m, nil
}

func parse(r io.ReaderAt) map[string][]string {
	exth := map[int]string{
		100: "author",
		101: "publisher",
		103: "description",
		104: "isbn",
		105: "subject",
		106: "pubdate",
		113: "asin",
		503: "title",
		504: "asin",
	}

	m := make(map[string][]string)

	// s := makeShortReader(r)
	l := makeLongReader(r)
	str := makeStringReader(r)

	// uint16 at 0x4C holds the number of records
	// first record's offset is a uint32 at 0x4E
	firstRecordOffset := int64(l(0x4E))
	headerLength := int64(l(firstRecordOffset + 0x14))

	// compressionType := int64(s(firstRecordOffset))
	fullTitlePos := int64(l(firstRecordOffset + 0x54))
	fullTitleLength := int64(l(firstRecordOffset + 0x58))
	m["title"] = []string{str(firstRecordOffset + fullTitlePos, fullTitleLength)}

	extendedFlags := int64(l(firstRecordOffset + 0x80))
	if extendedFlags & 0x40 != 0x40 {
		return m
	}

	// extended header block should start with string EXTH
	if str(firstRecordOffset + headerLength + 16, 4) != "EXTH" {
		fmt.Printf("extended header is all wrong, man!\n")
		return m
	}

	extBaseOffset := int64(firstRecordOffset + headerLength + 16)
	extCount := int(l(extBaseOffset + 8))

	pos := int64(12)
	for i := 0; i < extCount; i++ {
		recordType := int(l(extBaseOffset + pos))
		recordLength := int64(l(extBaseOffset + pos + 4))

		key, valid := exth[recordType]
		if valid {
			m[key] = append(m[key], str(extBaseOffset + pos + 8, recordLength - 8))
		}

		pos += recordLength
	}

	return m
}

func makeShortReader(r io.ReaderAt) (func(int64) uint16) {
	return func(o int64) uint16 {
		s := make([]byte, 2)
		_, err := r.ReadAt(s, o)
		if err != nil {
			panic(err)
		}
		return binary.BigEndian.Uint16(s)
	}
}

func makeLongReader(r io.ReaderAt) (func(int64) uint32) {
	return func(o int64) uint32 {
		l := make([]byte, 4)
		_, err := r.ReadAt(l, o)
		if err != nil {
			panic(err)
		}
		return binary.BigEndian.Uint32(l)
	}
}

func makeStringReader(r io.ReaderAt) (func(int64, int64) string) {
	return func(o, l int64) string {
		s := make([]byte, l)
		_, err := r.ReadAt(s, o)
		if err != nil {
			panic(err)
		}
		return string(s)
	}
}
