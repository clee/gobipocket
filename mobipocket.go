package mobipocket

import (
	"io"
	"log"
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
	// log.Println("parsed A-OK")
	if m.Metadata["compression"][0] == "palmdoc" {
		rawRecords := make([]string, 0)
		for _, record := range m.RawTextRecords {
			// log.Printf("attempting to unpack record #%d\n", recno)
			rawRecords = append(rawRecords, palmdoc_unpack(record))
		}
		raw := strings.Join(rawRecords, "")
		// log.Printf("raw length: %d, supposedly: %s\n", len(raw), m.Metadata["textLength"][0])
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
	// log.Printf("MobiPocket file version: %d\n", mobiversion);

	flags := int(s(firstRecordOffset + 0xF2))
	// log.Printf("MobiPocket flags: 0x%x\n", flags)

	multibyte := 0
	trailers := 0
	if headerLength > 0xE3 && mobiversion > 4 {
		for multibyte = flags & 1; flags > 1; flags = flags >> 1 {
			if flags & 2 == 2 {
				trailers += 1
			}
		}
		flags = int(s(firstRecordOffset + 0xF2))
	} else {
		flags = 0
	}

	// log.Printf("multibyte: %d\ntrailers: %d\n", multibyte, trailers)

	m["compression"] = []string{compressionTypes[int(s(firstRecordOffset))]}
	m["textLength"] = []string{fmt.Sprintf("%d", l(firstRecordOffset + 0x04))}
	firstTextRecord := 1 // int(s(firstRecordOffset + 0xC0))
	numberTextRecords := int(s(firstRecordOffset + 0x08))

	// log.Printf("PalmDB record count: %d (first text: %d, numTR: %d)\n", recordCount, firstTextRecord, numberTextRecords);
	/*
	for i := int(0); i < recordCount; i++ {
		log.Printf("\trecord %d @ %d\n", i, l(0x4E + int64(i * 8)))
	}
	*/
	mobi.RawTextRecords = make([][]byte, 0)
	for i := firstTextRecord; i < firstTextRecord + numberTextRecords; i++ {
		recordStart := l(0x4E + int64(i * 8))
		nextRecordStart:= l(0x4E + int64((i+1) * 8))

		record := make([]byte, nextRecordStart - recordStart)
		_, err := r.ReadAt(record, int64(recordStart))
		record = getTrimmedRecordData(record, flags)
		if err != nil {
			log.Printf("Tried to read record size %d starting at %d\n", len(record), recordStart)
			panic(err);
		}
		// log.Printf("read record %d @ 0x%04x, length %d bytes\n", i, recordStart, len(record))
		mobi.RawTextRecords = append(mobi.RawTextRecords, record)
	}
	m["drm"] = []string{drmTypes[int(s(firstRecordOffset + 0x0C))]}

	fullTitlePos := int64(l(firstRecordOffset + 0x54))
	fullTitleLength := int64(l(firstRecordOffset + 0x58))
	fullTitle := str(firstRecordOffset + fullTitlePos, fullTitleLength)
	if len(fullTitle) > 0 {
		m["title"] = []string{fullTitle}
	}

	defer func() {
		mobi.Metadata = m
	}()

	extendedFlags := int64(l(firstRecordOffset + 0x80))
	if extendedFlags & 0x40 != 0x40 {
		return
	}

	// extended header block should start with string EXTH
	if str(firstRecordOffset + headerLength + 16, 4) != "EXTH" {
		log.Printf("extended header is all wrong, man!\n")
		return
	}

	extBaseOffset := int64(firstRecordOffset + headerLength + 16)
	extCount := int(l(extBaseOffset + 8))

	pos := int64(12)
	for i := 0; i < extCount; i++ {
		recordType := int(l(extBaseOffset + pos))
		recordLength := int64(l(extBaseOffset + pos + 4))
		recordValue := str(extBaseOffset + pos + 8, recordLength - 8)

		key, valid := exth[recordType]
		pos += recordLength

		if !valid {
			continue
		}

		duplicateValue := false
		for _, v := range m[key] {
			if v == recordValue {
				duplicateValue = true
			}
		}

		if !duplicateValue {
			m[key] = append(m[key], recordValue)
		}
	}

	return
}

func getTrailingSize(b []byte) uint32 {
	val := uint32(0)
	for _, currentByte := range b[len(b) - 4:] {
		// log.Printf("currentByte: %x\n", currentByte)
		if currentByte & 0x80 == 0x80 {
			val = 0
		}
		val = (val << 7) | uint32(currentByte & 0x7F)
	}

	// log.Printf("getTrailingSize returning value %d\n", val)
	return val;
}

func getTrimmedRecordData(b []byte, flags int) []byte {
	c := b[:]
	for bit := uint8(15); bit > 0; bit-- {
		if (flags & (1 << bit)) == (1 << bit) {
			s := getTrailingSize(c)
			c = c[:uint32(len(c)) - s]
		}
	}
	if flags & 1 == 1 {
		s := uint32(c[len(c) - 1] & 0x03) + 1
		// log.Printf("Stealing %d bytes from the end of the record\n", s)
		c = c[:uint32(len(c)) - s]
	}
	return c
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
