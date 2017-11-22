package management

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/gorilla/mux"
)

// ReceivedRequest represents a request that was sent to a mock server
type ReceivedRequest struct {
	ServerName string `json:"server"`
	URL        string `json:"endpoint"`
	Method     string `json:"method"`
	StatusCode int    `json:"status_code"`
}

func (request ReceivedRequest) String() string {
	return fmt.Sprintf(
		"server: %s; url: %s; method: %s; response status: %d",
		request.ServerName,
		request.URL,
		request.Method,
		request.StatusCode,
	)
}

type statisticsRecord struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Count  int    `json:"count"`
}

type statisticsStorage struct {
	RequestsChannel chan ReceivedRequest
	requests        map[ReceivedRequest]int
}

func newStatisticsStorage() *statisticsStorage {
	storage := new(statisticsStorage)
	storage.requests = make(map[ReceivedRequest]int)
	storage.RequestsChannel = make(chan ReceivedRequest)
	return storage
}

func (storage *statisticsStorage) add(request ReceivedRequest) {
	storage.requests[request]++
}

func (storage statisticsStorage) get(request ReceivedRequest) int {
	return storage.requests[request]
}

func (storage *statisticsStorage) Run(wg *sync.WaitGroup) {
	log.Printf("[Statistics storage] Starting...")
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	defer close(signalChannel)
	defer signal.Stop(signalChannel)
	defer log.Printf("[Statistics storage] Stopped")
	defer wg.Done()

	for {
		select {
		case request, ok := <-storage.RequestsChannel:
			if !ok {
				return
			}
			storage.add(request)
		case <-signalChannel:
			return
		}
	}
}

func (storage *statisticsStorage) getRequestStatistics(request *ReceivedRequest) []statisticsRecord {
	var records []statisticsRecord

	for collectedRequest, count := range storage.requests {
		if request.ServerName != collectedRequest.ServerName {
			continue
		}
		if request.URL != "" && request.URL != collectedRequest.URL {
			continue
		}
		if request.Method != "" && request.Method != collectedRequest.Method {
			continue
		}

		records = append(
			records,
			statisticsRecord{
				URL:    collectedRequest.URL,
				Method: collectedRequest.Method,
				Count:  count,
			},
		)
	}
	return records
}

func (storage *statisticsStorage) HTTPHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	serverName := vars["serverName"]

	request := ReceivedRequest{
		ServerName: serverName,
	}

	urls, ok := req.URL.Query()["url"]
	if ok && len(urls) > 0 {
		request.URL = urls[0]
	}
	methods, ok := req.URL.Query()["method"]
	if ok && len(methods) > 0 {
		request.Method = methods[0]
	}

	statistics := storage.getRequestStatistics(&request)
	payload, err := json.Marshal(statistics)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("intervalServerError"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
