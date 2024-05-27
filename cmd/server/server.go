package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/VictorGOcking/lab-4/httptools"
	"github.com/VictorGOcking/lab-4/signal"
)

var port = flag.Int("port", 8080, "server port")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func addComplexHandlers(h *http.ServeMux, report Report, paths []string) {
	for _, path := range paths {
		h.HandleFunc(path, func(rw http.ResponseWriter, r *http.Request) {
			respDelayString := os.Getenv(confResponseDelaySec)
			if delaySec, parseErr := strconv.Atoi(respDelayString); parseErr == nil && delaySec > 0 && delaySec < 300 {
				time.Sleep(time.Duration(delaySec) * time.Second)
			}

			report.Process(r)

			rw.Header().Set("content-type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(rw).Encode([]string{
				"1", "2",
			})
		})
	}
}

func main() {
	h := new(http.ServeMux)

	h.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	addComplexHandlers(h, report, []string{
		"/api/v1/wow-data",
		"/api/v2/wtf/mad-data",
		"/really/good/end-point",
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}
