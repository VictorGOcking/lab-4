package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const baseAddress = "http://balancer:8090"

type IntegrationTestSuite struct {
	suite.Suite
	client http.Client
}

type ResponseBody struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.client = http.Client{
		Timeout: 3 * time.Second,
	}
}

func (s *IntegrationTestSuite) TestBalancer() {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		s.T().Skip("Integration test is not enabled")
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
		s.runServerTest(test.url, test.expectedLB, test.description)
	}

	// Test repeated request to server #3
	s.runServerTest(fmt.Sprintf("%s/really/good/endpoint", baseAddress), serverTests[2].expectedLB, "test repeated request to server #3")

	// Test request to check database
	s.testDatabaseRequest("beshenayaP4olka")
}

func (s *IntegrationTestSuite) runServerTest(url, expectedLB, description string) {
	resp, err := s.client.Get(url)
	assert.NoError(s.T(), err, description)
	defer resp.Body.Close()

	lbHeader := resp.Header.Get("lb-from")
	assert.Equal(s.T(), expectedLB, lbHeader, description)
}

func (s *IntegrationTestSuite) testDatabaseRequest(expectedKey string) {
	dbResp, err := s.client.Get(fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, expectedKey))
	assert.NoError(s.T(), err, "test request to check database")
	defer dbResp.Body.Close()

	var body ResponseBody
	err = json.NewDecoder(dbResp.Body).Decode(&body)
	assert.NoError(s.T(), err, "decode response body")

	assert.Equal(s.T(), expectedKey, body.Key, "check response key")
	assert.NotEmpty(s.T(), body.Value, "check response value")
}

func (s *IntegrationTestSuite) BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	for i := 0; i < b.N; i++ {
		resp, err := s.client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		assert.NoError(b, err, "benchmark request")
		resp.Body.Close()
	}
}
