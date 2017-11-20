package mockServer

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/pokidovea/mimicro/statistics"
	"github.com/stretchr/testify/assert"
)

func createEndpoint() Endpoint {
	str := `
url: /simple_url
GET:
    template: "{}"
    headers:
        content-type: application/json
POST:
    template: OK
    status_code: 201
`

	var endpoint Endpoint
	err := yaml.Unmarshal([]byte(str), &endpoint)

	if err != nil {
		panic(err)
	}

	return endpoint
}

func TestHandleGETResponse(t *testing.T) {
	endpoint := createEndpoint()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/simple_url", nil)

	handler := endpoint.GetHandler()
	handler(w, r)

	resp := w.Result()

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "{}", string(body))
}

func TestHandlePOSTResponse(t *testing.T) {
	endpoint := createEndpoint()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/simple_url", nil)

	handler := endpoint.GetHandler()
	handler(w, r)

	resp := w.Result()

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "OK", string(body))
}

func TestHandleNonexistingResponses(t *testing.T) {
	endpoint := createEndpoint()

	handler := endpoint.GetHandler()

	methods := [...]string{"PATCH", "PUT", "DELETE"}

	for _, method := range methods {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/simple_url", nil)
		handler(w, r)

		resp := w.Result()

		body, _ := ioutil.ReadAll(resp.Body)
		assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, "404 page not found\n", string(body))
	}
}

func TestWritesStatistics(t *testing.T) {
	endpoint := createEndpoint()
	statisticsChannel := make(chan statistics.Request, 1)
	defer close(statisticsChannel)

	endpoint.CollectStatistics(statisticsChannel, "simple_test_server")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/simple_url", nil)

	handler := endpoint.GetHandler()
	handler(w, r)

	expectedRequest := statistics.Request{
		ServerName: "simple_test_server",
		Url:        "/simple_url",
		Method:     "GET",
		StatusCode: endpoint.GET.StatusCode,
	}
	assert.Equal(t, expectedRequest, <-statisticsChannel)

}