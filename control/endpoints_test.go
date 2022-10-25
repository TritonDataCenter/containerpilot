package control

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/events"
)

func TestPutEnviron(t *testing.T) {
	endpoints := &Endpoints{}
	testFunc := func(t *testing.T, body string) (int, string) {
		os.Setenv(t.Name(), "original")
		defer os.Unsetenv(t.Name())
		req, _ := http.NewRequest("POST", "/v3/environ", strings.NewReader(body))
		_, status := endpoints.PutEnviron(req)
		result := os.Getenv(t.Name())
		return status, result
	}

	t.Run("POST value", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": \"updated\"}\n", t.Name()))
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
		assert.Equal(t, "updated", result, "env var was not updated")
	})

	t.Run("POST empty", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": \"\"}\n", t.Name()))
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
		assert.Equal(t, "", result, "env var should be cleared")
	})

	t.Run("POST null", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": null}\n", t.Name()))
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
		assert.Equal(t, "", result, "env var should be cleared")
	})

	t.Run("POST string null", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": \"null\"}\n", t.Name()))
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
		assert.Equal(t, "null", result, "env var should not be cleared")
	})

	t.Run("POST bad JSON", func(t *testing.T) {
		status, result := testFunc(t, "{{\n")
		assert.Equal(t, http.StatusUnprocessableEntity, status, "status was not 422")
		assert.Equal(t, "original", result, "env var should not be updated")
	})
}

func TestPostHandler(t *testing.T) {

	testFunc := func(req *http.Request, mock PostHandler) (int, string) {
		ph := PostHandler(mock)
		w := httptest.NewRecorder()
		ph.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		status := resp.StatusCode
		return status, string(body)
	}

	t.Run("POST ok", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v3/foo", nil)
		status, result := testFunc(req,
			func(r *http.Request) (interface{}, int) {
				return nil, 200
			})
		assert.Equal(t, 200, status, "expected HTTP 200 OK")
		assert.Equal(t, "\n", result)
	})

	t.Run("POST JSON ok", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v3/foo", nil)
		status, result := testFunc(req, func(r *http.Request) (interface{}, int) {
			return map[string]string{"key": "val"}, 200
		})
		assert.Equal(t, 200, status, "expected HTTP 200 OK")
		assert.Equal(t, "{\"key\":\"val\"}\n", result,
			"expected JSON body in reply")
	})

	t.Run("GET bad method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v3/foo", nil)
		status, result := testFunc(req,
			func(r *http.Request) (interface{}, int) {
				return nil, 200
			})
		assert.Equal(t, 405, status, "expected HTTP405 method not allowed")
		assert.Equal(t, "Method Not Allowed\n", result)
	})
}

func TestPostMetric(t *testing.T) {
	testFunc := func(t *testing.T, expected map[events.Event]int, body string) int {
		_, cancel := context.WithCancel(context.Background())
		bus := events.NewEventBus()
		endpoints := &Endpoints{
			bus:    bus,
			cancel: cancel,
		}
		req, _ := http.NewRequest("POST", "/v3/metric", strings.NewReader(body))
		_, status := endpoints.PostMetric(req)
		got := map[events.Event]int{}
		results := bus.DebugEvents()
		for _, result := range results {
			if result != events.GlobalStartup {
				got[result]++
			}
		}
		assert.Equal(t, expected, got)
		return status
	}

	t.Run("POST bad JSON", func(t *testing.T) {
		body := "{{\n"
		expected := map[events.Event]int{}
		status := testFunc(t, expected, body)
		assert.Equal(t, http.StatusUnprocessableEntity, status, "status was not 422")
	})
	t.Run("POST value", func(t *testing.T) {
		body := "{\"mymetric\": 1.0}"
		expected := map[events.Event]int{{Code: events.Metric, Source: "mymetric|1"}: 1}
		status := testFunc(t, expected, body)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
	t.Run("POST multi-metric", func(t *testing.T) {
		body := "{\"mymetric\": 1.5, \"myothermetric\": 2}"
		status := testFunc(t, map[events.Event]int{
			{Code: events.Metric, Source: "mymetric|1.5"}:    1,
			{Code: events.Metric, Source: "myothermetric|2"}: 1,
		}, body)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
}

func TestPostEnableMaintenanceMode(t *testing.T) {
	testFunc := func(t *testing.T, expected map[events.Event]int, req *http.Request) int {
		_, cancel := context.WithCancel(context.Background())
		bus := events.NewEventBus()
		bus.Publish(events.GlobalStartup)
		endpoints := &Endpoints{
			bus:    bus,
			cancel: cancel,
		}
		_, status := endpoints.PostEnableMaintenanceMode(req)
		results := bus.DebugEvents()
		got := map[events.Event]int{}
		for _, result := range results {
			if result != events.GlobalStartup {
				got[result]++
			}
		}
		assert.Equal(t, expected, got)
		return status
	}

	t.Run("POST bad JSON", func(t *testing.T) {
		body := "{{\n"
		req, _ := http.NewRequest("POST", "/v3/maintenance/enable", strings.NewReader(body))
		expected := map[events.Event]int{events.GlobalEnterMaintenance: 1}
		status := testFunc(t, expected, req)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
	t.Run("POST disable", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/v3/maintenance/enable", nil)
		expected := map[events.Event]int{events.GlobalEnterMaintenance: 1}
		status := testFunc(t, expected, req)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
}

func TestPostDisableMaintenanceMode(t *testing.T) {
	testFunc := func(t *testing.T, expected map[events.Event]int, req *http.Request) int {
		_, cancel := context.WithCancel(context.Background())
		bus := events.NewEventBus()
		bus.Publish(events.GlobalStartup)
		endpoints := &Endpoints{
			bus:    bus,
			cancel: cancel,
		}
		_, status := endpoints.PostDisableMaintenanceMode(req)
		bus.Wait()
		results := bus.DebugEvents()
		got := map[events.Event]int{}
		for _, result := range results {
			if result != events.GlobalStartup {
				got[result]++
			}
		}
		assert.Equal(t, expected, got)
		return status
	}

	t.Run("POST bad JSON", func(t *testing.T) {
		body := "{{\n"
		req, _ := http.NewRequest("POST", "/v3/maintenance/disable", strings.NewReader(body))
		expected := map[events.Event]int{events.GlobalExitMaintenance: 1}
		status := testFunc(t, expected, req)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
	t.Run("POST disable", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/v3/maintenance/disable", nil)
		expected := map[events.Event]int{events.GlobalExitMaintenance: 1}
		status := testFunc(t, expected, req)
		assert.Equal(t, http.StatusOK, status, "status was not 200OK")
	})
}

func TestGetPing(t *testing.T) {
	req := httptest.NewRequest("GET", "/v3/ping", nil)
	w := httptest.NewRecorder()
	GetPing(w, req)
	resp := w.Result()
	defer resp.Body.Close()
	status := resp.StatusCode
	assert.Equal(t, 200, status, "expected HTTP 200 OK")
}
