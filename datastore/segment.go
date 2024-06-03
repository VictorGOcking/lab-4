package datastore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Segment struct {
	path   string
	offset int64

	index HashIndex
	mu    sync.Mutex
}

func (s *Segment) Read(pos int64) (string, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(pos, 0)
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		return "", err
	}
	return value, nil
}

type SegmentList struct {
	outDir string

	list   []*Segment
	length int
	size   int64
}

func NewSegmentList(size int64, outDir string) *SegmentList {
	return &SegmentList{
		outDir: outDir,
		list:   make([]*Segment, 0),
		length: 0,
		size:   size,
	}
}

func (sl *SegmentList) getPath() string {
	result := filepath.Join(
		sl.outDir,
		fmt.Sprintf("%s%d", outFileName, sl.length),
	)

	return result
}

func (sl *SegmentList) Add() (*os.File, error) {
	path := sl.getPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}

	segment := &Segment{
		path:  path,
		index: make(HashIndex),
	}

	sl.list = append(sl.list, segment)

	if len(sl.list) >= 3 {
		sl.length++
		sl.Compact()
	}

	sl.length++
	return f, nil
}

func (sl *SegmentList) GetLast() *Segment {
	return sl.list[len(sl.list)-1]
}

func (sl *SegmentList) Find(key string) (*Segment, int64, error) {
	for _, segment := range sl.list {
		pos, ok := segment.index[key]
		segment.mu.Lock()

		if ok {
			segment.mu.Unlock()
			return segment, pos, nil
		}

		segment.mu.Unlock()
	}

	return nil, 0, ErrNotFound
}

func contains(list []*Segment, key string) bool {
	for _, s := range list {
		s.mu.Lock()

		_, ok := s.index[key]

		if ok {
			s.mu.Unlock()
			return true
		}
		s.mu.Unlock()
	}
	return false
}

func (sl *SegmentList) Compact() {
	go func() {
		path := sl.getPath()

		segment := &Segment{
			path:  path,
			index: make(HashIndex),
		}

		var offset int64
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			return
		}

		last := len(sl.list) - 1
		for i := 0; i < last; i++ {
			s := sl.list[i]
			s.mu.Lock()

			for key, index := range s.index {
				if i < last-1 {
					isNew := contains(sl.list[i+1:last], key)
					if isNew {
						continue
					}
				}

				value, _ := s.Read(index)
				e := entry{
					key:   key,
					value: value,
				}

				n, err := f.Write(e.Encode())
				if err == nil {
					segment.index[key] = offset
					offset += int64(n)
				}
			}
			s.mu.Unlock()
		}
		sl.list = []*Segment{segment, sl.GetLast()}
	}()
}
