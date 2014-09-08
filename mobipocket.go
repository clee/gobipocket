package mobipocket

import (
	"io"
	"fmt"
	"os"
	"strings"
	"encoding/binary"
)

type Mobipocket struct {
	Metadata map[string][]string
	RawTextRecords [][]byte
}

// Open reads the file into memory and parses the headers to
// populate the Metadata
func Open(path string) (m *Mobipocket, e error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}

	m = new(Mobipocket)
	m.parse(f)
	fmt.Println("parsed A-OK")
	if m.Metadata["compression"][0] == "palmdoc" {
		rawRecords := make([]string, 0)
		for _, record := range m.RawTextRecords {
			rawRecords = append(rawRecords, palmdoc_unpack(record))
		}
		raw := strings.Join(rawRecords, "")
		fmt.Printf("raw length: %d, supposedly: %s\n", len(raw), m.Metadata["textLength"][0])
	}

	return m, nil
}

func (mobi *Mobipocket) parse(r io.ReaderAt) {
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

	compressionTypes := map[int]string {
		1: "none",
		2: "palmdoc",
		17480: "huffcdic",
	}

	drmTypes := map[int]string {
		0: "unencrypted",
		1: "deprecated",
		2: "encrypted",
	}

	m := make(map[string][]string)

	s := makeShortReader(r)
	l := makeLongReader(r)
	str := makeStringReader(r)

	// uint16 at 0x4C holds the number of records
	// first record's offset is a uint32 at 0x4E
	recordCount := int(s(0x4C));
	firstRecordOffset := int64(l(0x4E))
	headerLength := int64(l(firstRecordOffset + 0x14))

	mobiversion := int(l(firstRecordOffset + 0x24))
	fmt.Printf("MobiPocket file version: %d\n", mobiversion);

	m["compression"] = []string{compressionTypes[int(s(firstRecordOffset))]}
	m["textLength"] = []string{fmt.Sprintf("%d", l(firstRecordOffset + 0x04))}
	firstTextRecord := 1 // int(s(firstRecordOffset + 0xC0))
	numberTextRecords := int(s(firstRecordOffset + 0x08))

	fmt.Printf("PalmDB record count: %d (first text: %d, numTR: %d)\n", recordCount, firstTextRecord, numberTextRecords);
	/*
	for i := int(0); i < recordCount; i++ {
		fmt.Printf("\trecord %d @ 0x%04x\n", i, l(0x4E + int64(i * 8)))
	}
	*/
	mobi.RawTextRecords = make([][]byte, 0)
	for i := firstTextRecord; i < firstTextRecord + numberTextRecords; i++ {
		recordStart := l(0x4E + int64(i * 8))
		nextRecordStart:= l(0x4E + int64((i+1) * 8))
		record := make([]byte, nextRecordStart - recordStart)
		_, err := r.ReadAt(record, int64(recordStart))
		if err != nil {
			panic(err);
		}
		fmt.Printf("read record %d @ 0x%04x, length 0x%04x bytes\n", i, recordStart, len(record))
		mobi.RawTextRecords = append(mobi.RawTextRecords, record)
	}
	m["drm"] = []string{drmTypes[int(s(firstRecordOffset + 0x0C))]}

	fullTitlePos := int64(l(firstRecordOffset + 0x54))
	fullTitleLength := int64(l(firstRecordOffset + 0x58))
	m["title"] = []string{str(firstRecordOffset + fullTitlePos, fullTitleLength)}

	extendedFlags := int64(l(firstRecordOffset + 0x80))
	if extendedFlags & 0x40 != 0x40 {
		mobi.Metadata = m
		return
	}

	// extended header block should start with string EXTH
	if str(firstRecordOffset + headerLength + 16, 4) != "EXTH" {
		fmt.Printf("extended header is all wrong, man!\n")
		mobi.Metadata = m
		return
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

	mobi.Metadata = m
	return
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
