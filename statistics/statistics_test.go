package statistics

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollection(t *testing.T) {
	collector := new(Collector)
	request := Request{
		ServerName: "Simple server",
		Url:        "/some_url",
		Method:     "POST",
	}
	collector.Add(request)
	collector.Add(request)
	assert.Equal(t, 2, collector.Get(request))
}

func TestCollectionFromChannel(t *testing.T) {

	collector := Collector{Chan: make(chan Request, 1)}
	done := make(chan bool, 1)
	defer close(done)

	request := Request{
		ServerName: "Simple server",
		Url:        "/some_url",
		Method:     "POST",
	}
	collector.Chan <- request
	close(collector.Chan)

	var wg sync.WaitGroup
	wg.Add(1)

	go collector.Run(&wg)
	wg.Wait()

	assert.Equal(t, 1, collector.Get(request))
}

func TestStringifyRequest(t *testing.T) {
	request := Request{
		ServerName: "Simple server",
		Url:        "/some_url",
		Method:     "POST",
		StatusCode: http.StatusCreated,
	}

	assert.Equal(
		t,
		"server: Simple server; url: /some_url; method: POST; response status: 201",
		fmt.Sprintf("%s", request),
	)
}
