package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

// capturing output about server unavailability
func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			fmt.Println(err)
		}
		outC <- buf.String()
	}()

	f()

	// restoring previous stdout
	err := w.Close()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	os.Stdout = stdout
	out := <-outC

	return out
}

func TestBalancer(t *testing.T) {
	b := &Balancer{
		pool: []string{"server1", "server2", "server3", "server4"},
		checker: func(dst string) bool {
			return dst != "server2"
		},
		active: []string{},
	}

	// test index assigning
	findSrvIndex := func(url string) int {
		return int(b.Hash(url) % uint64(len(b.pool)))
	}

	serverIdx1 := findSrvIndex("/url/to/somewhere")

	serverIdx2 := findSrvIndex("/url/to/somewhere/else")
	sameIdx2 := findSrvIndex("/url/to/somewhere/else")

	serverIdx3 := findSrvIndex("/hey")
	sameIdx3 := findSrvIndex("/hey")

	serverIdx4 := findSrvIndex("/bye")

	assert.Equal(t, 1, serverIdx1)

	assert.Equal(t, 3, serverIdx2)
	assert.Equal(t, serverIdx2, sameIdx2)

	assert.Equal(t, 2, serverIdx3)
	assert.Equal(t, serverIdx3, sameIdx3)

	assert.Equal(t, 0, serverIdx4)

	// test health checking
	output := captureOutput(func() {
		b.Check()
	})

	assert.Equal(t, []string{"server1", "server3", "server4"}, b.active)
	assert.Contains(t, output, "Server server2 is unavailable")
}
