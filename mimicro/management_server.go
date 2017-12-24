package mimicro

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// ManagementServer represents a server, responsible for statistics and administration
type ManagementServer struct {
	Port                int
	statisticsStorage   *statisticsStorage
	substitutionStorage *SubstitutionStorage
}

// NewManagementServer creates a new management server record
func NewManagementServer(port int, collectStatistics bool, substitutionStorage *SubstitutionStorage) *ManagementServer {
	server := ManagementServer{Port: port, substitutionStorage: substitutionStorage}

	if collectStatistics {
		server.statisticsStorage = newStatisticsStorage()
	}

	return &server
}

// WriteRequestLog is called by mock servers to write request into log and statistics into storage
func (server *ManagementServer) WriteRequestLog(serverName, URL, method string, statusCode int) {
	request := ReceivedRequest{
		ServerName: serverName,
		URL:        URL,
		Method:     method,
		StatusCode: statusCode,
	}

	log.Printf("Requested %s \n", request)

	if server.statisticsStorage != nil {
		server.statisticsStorage.RequestsChannel <- request
	}
}

func (server ManagementServer) startHTTPServer() *http.Server {
	router := mux.NewRouter()

	if server.statisticsStorage != nil {
		router.HandleFunc("/statistics/get", server.statisticsStorage.GetStatisticsHandler).Methods("GET")
		router.HandleFunc("/statistics/reset", server.statisticsStorage.DeleteStatisticsHandler).Methods("GET")
	}

	srv := &http.Server{
		Addr:           ":" + strconv.Itoa(server.Port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	return srv
}

// Serve method starts the server and does some operations after it stops
func (server ManagementServer) Serve(wg *sync.WaitGroup) {
	log.Printf("[Management] Starting...")

	if server.statisticsStorage != nil {
		server.statisticsStorage.Start()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer close(interrupt)
	defer signal.Stop(interrupt)

	srv := server.startHTTPServer()
	<-interrupt

	if server.statisticsStorage != nil {
		server.statisticsStorage.Stop()
	}

	log.Printf("[Management] Stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Management] Shutdown error: %s", err)
	}

	log.Printf("[Management] Stopped")

	wg.Done()
}
