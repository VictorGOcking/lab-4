package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/VictorGOcking/lab-4/httptools"
	"github.com/VictorGOcking/lab-4/signal"
)

var port = flag.Int("port", 8080, "server port")

const (
	databaseURL          = "http://db:8085/db"
	confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
	confHealthFailure    = "CONF_HEALTH_FAILURE"
)

type RequestStruct struct {
	Value string `json:"value"`
}

type ResponseStruct struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func main() {
	h := http.NewServeMux()
	client := http.DefaultClient

	h.HandleFunc("/health", healthHandler)
	report := make(Report)

	h.HandleFunc("/api/v1/some-data", someDataHandler(client, report))
	addComplexHandlers(h, report, []string{
		"/api/v1/wow-data",
		"/api/v2/wtf/mad-data",
		"/really/good/end-point",
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()

	storeCurrentDate(client)
	signal.WaitForTerminationSignal()
}

func healthHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("content-type", "text/plain")
	if os.Getenv(confHealthFailure) == "true" {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("FAILURE"))
	} else {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("OK"))
	}
}

func someDataHandler(client *http.Client, report Report) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(rw, "Bad Request: key is missing", http.StatusBadRequest)
			return
		}

		resp, err := client.Get(fmt.Sprintf("%s/%s", databaseURL, key))
		if err != nil {
			http.Error(rw, "Internal Server Error: failed to get data", http.StatusInternalServerError)
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)

		if resp.StatusCode == http.StatusNotFound {
			http.Error(rw, "Not Found", http.StatusNotFound)
			return
		} else if resp.StatusCode != http.StatusOK {
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if delaySec, err := strconv.Atoi(os.Getenv(confResponseDelaySec)); err == nil && delaySec > 0 && delaySec < 300 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}

		report.Process(r)

		var body ResponseStruct
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			http.Error(rw, "Internal Server Error: failed to decode response", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("content-type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(rw).Encode(body)
	}
}

func storeCurrentDate(client *http.Client) {
	buff := new(bytes.Buffer)
	body := RequestStruct{Value: time.Now().Format(time.RFC3339)}
	if err := json.NewEncoder(buff).Encode(body); err != nil {
		return
	}

	resp, err := client.Post(fmt.Sprintf("%s/victorgocking", databaseURL), "application/json", buff)
	if err != nil {
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
}

func addComplexHandlers(h *http.ServeMux, report Report, paths []string) {
	for _, path := range paths {
		h.HandleFunc(path, func(rw http.ResponseWriter, r *http.Request) {
			if delaySec, err := strconv.Atoi(os.Getenv(confResponseDelaySec)); err == nil && delaySec > 0 && delaySec < 300 {
				time.Sleep(time.Duration(delaySec) * time.Second)
			}

			report.Process(r)

			rw.Header().Set("content-type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(rw).Encode([]string{"1", "2"})
		})
	}
}
