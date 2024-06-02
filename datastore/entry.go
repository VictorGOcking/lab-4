package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

type entry struct {
	key, value string
}

func (e *entry) Encode() []byte {
	kl := len(e.key)
	vl := len(e.value)

	size := kl + vl + 12
	res := make([]byte, size)

	binary.LittleEndian.PutUint32(res, uint32(size))

	binary.LittleEndian.PutUint32(res[4:], uint32(kl))
	copy(res[8:], e.key)

	binary.LittleEndian.PutUint32(res[kl+8:], uint32(vl))
	copy(res[kl+12:], e.value)

	return res
}

func (e *entry) Length() int64 {
	return int64(len(e.key) + len(e.value) + 12)
}

func (e *entry) Decode(input []byte) {
	kl := binary.LittleEndian.Uint32(input[4:])
	key := make([]byte, kl)
	copy(key, input[8:kl+8])
	e.key = string(key)

	vl := binary.LittleEndian.Uint32(input[kl+8:])
	value := make([]byte, vl)
	copy(value, input[kl+12:kl+12+vl])
	e.value = string(value)
}

func readValue(in *bufio.Reader) (string, error) {
	header, err := in.Peek(8)
	if err != nil {
		return "", err
	}

	keySize := int(binary.LittleEndian.Uint32(header[4:]))
	_, err = in.Discard(keySize + 8)
	if err != nil {
		return "", err
	}

	header, err = in.Peek(4)
	if err != nil {
		return "", err
	}

	valSize := int(binary.LittleEndian.Uint32(header))
	_, err = in.Discard(4)
	if err != nil {
		return "", err
	}

	data := make([]byte, valSize)
	n, err := in.Read(data)
	if err != nil {
		return "", err
	}

	if n != valSize {
		return "", fmt.Errorf("can't read value bytes (read %d, expected %d)", n, valSize)
	}

	return string(data), nil
}
