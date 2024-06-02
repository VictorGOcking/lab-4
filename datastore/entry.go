package datastore

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
)

type entry struct {
	key, value string
	checksum   []byte
}

func (e *entry) Encode() []byte {
	kl := len(e.key)
	vl := len(e.value)

	size := kl + vl + 32
	res := make([]byte, size)

	binary.LittleEndian.PutUint32(res, uint32(size))
	copy(res[12:], e.key)

	binary.LittleEndian.PutUint32(res[4:], uint32(kl))
	copy(res[kl+12:], e.value)
	data := make([]byte, size-20)

	binary.LittleEndian.PutUint32(res[8:], uint32(vl))

	copy(data, res[:size-19])
	sum := sha1.Sum(data)
	copy(res[size-20:], sum[:])

	return res
}

func (e *entry) Length() int64 {
	return int64(len(e.key) + len(e.value) + 12)
}

func (e *entry) Decode(input []byte) {
	kl := binary.LittleEndian.Uint32(input[4:])
	vl := binary.LittleEndian.Uint32(input[8:])

	key := make([]byte, kl)
	copy(key, input[12:kl+12])

	e.key = string(key)

	value := make([]byte, vl)
	copy(value, input[kl+12:kl+12+vl])

	e.value = string(value)
	e.checksum = make([]byte, 20)

	copy(e.checksum, input[kl+vl+12:])
}

func readValue(in *bufio.Reader) (string, error) {
	header, err := in.Peek(12)
	if err != nil {
		return "", err
	}

	keySize := int(binary.LittleEndian.Uint32(header[4:]))
	valSize := int(binary.LittleEndian.Uint32(header[8:]))

	data, err := in.Peek(12 + keySize + valSize)
	if err != nil {
		return "", err
	}

	_, err = in.Discard(12 + keySize)
	if err != nil {
		return "", err
	}

	value, err := in.Peek(valSize)
	if err != nil {
		return "", err
	}

	if len(value) != valSize {
		return "", fmt.Errorf("can't read value bytes (read %d, expected %d)", len(value), valSize)
	}

	_, err = in.Discard(valSize)
	if err != nil {
		return "", err
	}

	sum, err := in.Peek(20)
	if err != nil {
		return "", err
	}
	realSum := sha1.Sum(data)
	if bytes.Compare(sum, realSum[:]) != 0 {
		return "", errors.New("entry's checksum is wrong")
	}

	return string(value), nil
}
