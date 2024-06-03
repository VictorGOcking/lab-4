package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/VictorGOcking/lab-4/datastore"
	"github.com/VictorGOcking/lab-4/httptools"
	"github.com/VictorGOcking/lab-4/signal"
)

var port = flag.Int("port", 8085, "server port")

type ResponseStruct struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequestStruct struct {
	Value string `json:"value"`
}

func main() {
	flag.Parse()

	dir, err := ioutil.TempDir("", "temp-dir")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	db, err := datastore.NewDb(dir, 150)
	if err != nil {
		log.Fatalf("Failed to create datastore: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/db/", func(rw http.ResponseWriter, req *http.Request) {
		handleDBRequest(rw, req, db)
	})

	server := httptools.CreateServer(*port, mux)
	go func() {
		server.Start()
	}()

	signal.WaitForTerminationSignal()
}

func handleDBRequest(rw http.ResponseWriter, req *http.Request, db *datastore.Db) {
	key := req.URL.Path[len("/db/"):]

	switch req.Method {
	case http.MethodGet:
		handleGetRequest(rw, key, db)
	case http.MethodPost:
		handlePostRequest(rw, req, key, db)
	default:
		http.Error(rw, "Bad request method", http.StatusBadRequest)
	}
}

func handleGetRequest(rw http.ResponseWriter, key string, db *datastore.Db) {
	value, err := db.Get(key)
	if err != nil {
		http.Error(rw, fmt.Sprintf("Key not found: %v", err), http.StatusNotFound)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(ResponseStruct{Key: key, Value: value}); err != nil {
		http.Error(rw, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func handlePostRequest(rw http.ResponseWriter, req *http.Request, key string, db *datastore.Db) {
	var body RequestStruct
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(rw, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := db.Put(key, body.Value); err != nil {
		http.Error(rw, fmt.Sprintf("Failed to store value: %v", err), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusCreated)
}
