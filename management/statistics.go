package management

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// ReceivedRequest represents a request that was sent to a mock server
type ReceivedRequest struct {
	ServerName string
	URL        string
	Method     string
	StatusCode int
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

func (request ReceivedRequest) matches(anotherRequest ReceivedRequest) bool {
	if request.ServerName != "*" && request.ServerName != anotherRequest.ServerName {
		return false
	}
	if request.URL != "*" && request.URL != anotherRequest.URL {
		return false
	}
	if request.Method != "*" && request.Method != anotherRequest.Method {
		return false
	}

	return true
}

type requestsCounter map[ReceivedRequest]int

func (counter requestsCounter) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("[")
	length := len(counter)
	count := 0
	for request, requestsCount := range counter {
		buffer.WriteString("{")
		buffer.WriteString(fmt.Sprintf("\"server\":\"%s\",", request.ServerName))
		buffer.WriteString(fmt.Sprintf("\"url\":\"%s\",", request.URL))
		buffer.WriteString(fmt.Sprintf("\"method\":\"%s\",", request.Method))
		buffer.WriteString(fmt.Sprintf("\"count\":%s", strconv.Itoa(requestsCount)))
		buffer.WriteString("}")
		count++
		if count < length {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

type statisticsStorage struct {
	mutex           sync.RWMutex
	RequestsChannel chan ReceivedRequest
	requests        requestsCounter
}

func newStatisticsStorage() *statisticsStorage {
	storage := new(statisticsStorage)
	storage.requests = make(requestsCounter)
	storage.RequestsChannel = make(chan ReceivedRequest, 100)
	return storage
}

func (storage *statisticsStorage) add(request ReceivedRequest) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	storage.requests[request]++
}

func (storage *statisticsStorage) del(request ReceivedRequest) {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	for collectedRequest := range storage.requests {
		if request.matches(collectedRequest) {
			delete(storage.requests, collectedRequest)
		}
	}
}

func (storage *statisticsStorage) get(request ReceivedRequest) int {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	return storage.requests[request]
}

func (storage *statisticsStorage) iter(f func(request ReceivedRequest, count int) bool) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	for request, count := range storage.requests {
		if !f(request, count) {
			return
		}
	}
}

func (storage *statisticsStorage) run() {
	log.Printf("[Statistics storage] Starting...")

	defer log.Printf("[Statistics storage] Stopped")

	for request := range storage.RequestsChannel {
		storage.add(request)
	}
}

func (storage *statisticsStorage) Start() {
	go storage.run()
}

func (storage *statisticsStorage) Stop() {
	close(storage.RequestsChannel)
}

func (storage *statisticsStorage) getRequestStatistics(request *ReceivedRequest) requestsCounter {
	records := make(requestsCounter)

	storage.iter(func(collectedRequest ReceivedRequest, count int) bool {
		if !request.matches(collectedRequest) {
			return true
		}

		records[collectedRequest] = count

		return true
	})
	return records
}

func (storage *statisticsStorage) GetStatisticsHandler(w http.ResponseWriter, req *http.Request) {
	var request ReceivedRequest

	servers, ok := req.URL.Query()["server"]
	if ok && len(servers) > 0 {
		request.ServerName = servers[0]
	} else {
		request.ServerName = "*"
	}

	urls, ok := req.URL.Query()["url"]
	if ok && len(urls) > 0 {
		request.URL = urls[0]
	} else {
		request.URL = "*"
	}

	methods, ok := req.URL.Query()["method"]
	if ok && len(methods) > 0 {
		request.Method = strings.ToUpper(methods[0])
	} else {
		request.Method = "*"
	}

	statistics := storage.getRequestStatistics(&request)
	payload, err := json.Marshal(statistics)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
