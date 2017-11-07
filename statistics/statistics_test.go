package statistics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollection(t *testing.T) {
	collector := new(Collector)
	request := Request{
		Server:   "Simple server",
		Endpoint: "/some_url",
		Method:   "POST",
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
		Server:   "Simple server",
		Endpoint: "/some_url",
		Method:   "POST",
	}
	collector.Chan <- request
	close(collector.Chan)

	collector.Run(done)

	assert.Equal(t, 1, collector.Get(request))
}