package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	outFileName = "current-data"
	bufferSize  = 8192 // 1 K = 8192 B
)

var ErrNotFound = fmt.Errorf("record does not exist")

type HashIndex map[string]int64

type Db struct {
	// Output options
	out    *os.File
	offset int64

	// Segmentation
	segments *SegmentList

	// Goroutines handlers
	operator HashOperator
	ops      chan EntryElement

	// Hash indexing
	index HashIndex
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments: NewSegmentList(segmentSize, dir),
		operator: HashOperator{
			queries: make(chan HashOperation),
			answers: make(chan *SegmentPosition),
		},
		ops: make(chan EntryElement),
	}

	err := db.addSegment()

	if err != nil {
		return nil, err
	}

	err = db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Start goroutines handlers
	db.handleInput()
	db.handleOperations()

	return db, nil
}

func (db *Db) addSegment() error {
	f, err := db.segments.Add()
	db.out = f
	db.offset = 0

	return err
}

func (db *Db) find(key string) *SegmentPosition {
	op := HashOperation{
		put: false,
		key: key,
	}

	db.operator.queries <- op
	return <-db.operator.answers
}

func (db *Db) Get(key string) (string, error) {
	keyPos := db.find(key)
	if keyPos == nil {
		return "", ErrNotFound
	}

	value, err := keyPos.segment.Read(keyPos.position)
	if err != nil {
		return "", err
	}

	return value, nil
}

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
	}

	ee := EntryElement{
		ent: e,
		err: make(chan error),
	}

	db.ops <- ee
	return <-ee.err
}

func (db *Db) recover() error {
	var err error
	var buf [bufferSize]byte

	in := bufio.NewReaderSize(db.out, bufferSize)
	for err == nil {
		var (
			header []byte
			data   []byte
			n      int
		)
		header, err = in.Peek(bufferSize)

		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}

		size := binary.LittleEndian.Uint32(header)

		if size < bufferSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}

		n, err = in.Read(data)
		if err != nil {
			return err
		}

		if n != int(size) {
			return fmt.Errorf("corrupted file")
		}

		var e entry
		e.Decode(data)

		db.segments.GetLast().index[e.key] = db.offset
		db.offset += int64(n)
	}
	return err
}

func (db *Db) Close() error {
	return db.out.Close()
}
