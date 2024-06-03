package integration

import (
	"encoding/json"
	"fmt"
	"io"
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
		{fmt.Sprintf("%s/api/v2/wtf/mad-data", baseAddress), "server1:8080", "test server #1"},
		{fmt.Sprintf("%s/api/v1/wow-data", baseAddress), "server2:8080", "test server #2"},
		{fmt.Sprintf("%s/really/good/end-point", baseAddress), "server3:8080", "test server #3"},
	}

	for _, test := range serverTests {
		runServerTest(t, test.url, test.expectedLB, test.description)
	}

	// Test repeated request to server #3
	runServerTest(t, fmt.Sprintf("%s/really/good/end-point", baseAddress), serverTests[2].expectedLB, "test repeated request to server #3")

	testDatabaseRequest(t, "victorgocking")
}

func runServerTest(t *testing.T, url, expectedLB, description string) {
	resp, err := client.Get(url)
	assert.NoError(t, err, description)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	lbHeader := resp.Header.Get("lb-from")
	assert.Equal(t, expectedLB, lbHeader, description)
}

func testDatabaseRequest(t *testing.T, expectedKey string) {
	dbResp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, expectedKey))
	assert.NoError(t, err, "test request to check database")
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(dbResp.Body)

	var body ResponseBody
	err = json.NewDecoder(dbResp.Body).Decode(&body)
	assert.NoError(t, err, "decode response body")

	assert.Equal(t, expectedKey, body.Key, "check response key")
	assert.NotEmpty(t, body.Value, "check response value")
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/wow-data", baseAddress))
		assert.NoError(b, err, "benchmark request")
		err = resp.Body.Close()
		if err != nil {
			return
		}
	}
}
