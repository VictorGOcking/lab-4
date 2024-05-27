package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/VictorGOcking/lab-4/httptools"
	"github.com/VictorGOcking/lab-4/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout = time.Duration(*timeoutSec) * time.Second
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

type Balancer struct {
	pool   []string
	active []string

	checker func(server string) bool
	forward func(dst string, rw http.ResponseWriter, r *http.Request) error
}

func (b *Balancer) Hash(url string) uint32 {
	hasher := fnv.New32()
	_, _ = hasher.Write([]byte(url))
	return hasher.Sum32()
}

func (b *Balancer) Check() {
	b.active = []string{}

	for _, server := range b.pool {
		isFree := b.checker(server)
		if isFree {
			b.active = append(b.active, server)
		} else {
			fmt.Printf("Server %s is unavailable", server)
		}
	}
}

func (b *Balancer) Analyse() {
	b.Check()

	go func() {
		for range time.Tick(10 * time.Second) {
			b.Check()
		}
	}()
}

func (b *Balancer) Run() {
	flag.Parse()

	b.Analyse()

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if len(b.active) == 0 {
			log.Println("No free servers available")
			rw.WriteHeader(http.StatusBadGateway)
			return
		}

		serverIndex := b.Hash(r.URL.Path) % uint32(len(b.pool))
		err := b.forward(b.pool[serverIndex], rw, r)
		if err != nil {
			log.Fatal(err)
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

func main() {
	balancer := Balancer{
		pool: []string{
			"server1:8080",
			"server2:8080",
			"server3:8080",
		},
		active:  []string{},
		checker: health,
		forward: forward,
	}

	balancer.Run()
}
