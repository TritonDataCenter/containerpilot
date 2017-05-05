package control

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/tests/assert"
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
		assert.Equal(t, status, http.StatusOK, "status was not 200OK")
		assert.Equal(t, result, "updated", "env var should be '%s' but was '%s'")
	})

	t.Run("POST empty", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": \"\"}\n", t.Name()))
		assert.Equal(t, status, http.StatusOK, "status was not 200OK")
		assert.Equal(t, result, "", "env var should be '%s' (empty) but got '%s'")
	})

	t.Run("POST null", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": null}\n", t.Name()))
		assert.Equal(t, status, http.StatusOK, "status was not 200OK")
		assert.Equal(t, result, "", "env var should be '%s' (empty) but got '%s'")
	})

	t.Run("POST string null", func(t *testing.T) {
		status, result := testFunc(t, fmt.Sprintf("{\"%s\": \"null\"}\n", t.Name()))
		assert.Equal(t, status, http.StatusOK, "status was not 200OK")
		assert.Equal(t, result, "null", "env var should be '%s' but got '%s'")
	})

	t.Run("POST bad JSON", func(t *testing.T) {
		status, result := testFunc(t, "{{\n")
		assert.Equal(t, status, http.StatusUnprocessableEntity, "status was not 422")
		assert.Equal(t, result, "original", "env var should be '%s' but got '%s'")
	})
}

func TestPostHandler(t *testing.T) {

	testFunc := func(req *http.Request, mock PostHandler) (int, string) {
		ph := PostHandler(mock)
		w := httptest.NewRecorder()
		ph.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		status := resp.StatusCode
		return status, string(body)
	}

	t.Run("POST ok", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v3/foo", nil)
		status, result := testFunc(req,
			func(r *http.Request) (interface{}, int) {
				return nil, 200
			})
		assert.Equal(t, status, 200, "expected HTTP 200 OK")
		assert.Equal(t, result, "\n", "expected '%q' but got '%q'")
	})

	t.Run("POST JSON ok", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v3/foo", nil)
		status, result := testFunc(req, func(r *http.Request) (interface{}, int) {
			return map[string]string{"key": "val"}, 200
		})
		assert.Equal(t, status, 200, "expected HTTP 200 OK")
		assert.Equal(t, result, "{\"key\":\"val\"}\n",
			"expected JSON body '%q', but got '%q'")
	})

	t.Run("GET bad method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v3/foo", nil)
		status, result := testFunc(req,
			func(r *http.Request) (interface{}, int) {
				return nil, 200
			})
		assert.Equal(t, status, 501, "expected HTTP501 method not allowed")
		assert.Equal(t, result, "Not Implemented\n", "expected '%q' but got '%q'")
	})
}
