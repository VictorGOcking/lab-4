package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

type ResponseBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	serverTests := []struct {
		url         string
		expectedLB  string
		description string
	}{
		{fmt.Sprintf("%s/api/v2/wtf/what-is-it", baseAddress), "server1:8080", "test server #1"},
		{fmt.Sprintf("%s/api/v1/some-data", baseAddress), "server2:8080", "test server #2"},
		{fmt.Sprintf("%s/really/good/endpoint", baseAddress), "server3:8080", "test server #3"},
	}

	for _, test := range serverTests {
		runServerTest(t, test.url, test.expectedLB, test.description)
	}

	// Test repeated request to server #3
	runServerTest(t, fmt.Sprintf("%s/really/good/endpoint", baseAddress), serverTests[2].expectedLB, "test repeated request to server #3")
}

func runServerTest(t *testing.T, url, expectedLB, description string) {
	resp, err := client.Get(url)
	assert.NoError(t, err, description)
	defer resp.Body.Close()

	lbHeader := resp.Header.Get("lb-from")
	assert.Equal(t, expectedLB, lbHeader, description)
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		assert.NoError(b, err, "benchmark request")
		resp.Body.Close()
	}
}
