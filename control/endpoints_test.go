package control

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/tests/assert"
)

func TestPutEnviron(t *testing.T) {
	// NOTE: 'control' means nothing here, connection string must use
	// protocol/url format.
	var envURL = "http://control/v3/environ"

	tempSocketPath := tempSocketPath()
	defer os.Remove(tempSocketPath)

	s := SetupHTTPServer(t, fmt.Sprintf(`{ "socket": %q}`, tempSocketPath))
	defer s.Stop()
	s.Start()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: socketDialer(tempSocketPath),
		},
	}

	// NOTE: Surely these can be cleaned up by iterating over unique test input...
	//
	t.Run("POST value", func(t *testing.T) {
		os.Setenv("TESTVAR1", "original")
		defer os.Unsetenv("TESTVAR1")
		assert.Equal(t, os.Getenv("TESTVAR1"), "original", "Test env var should be original value")

		r := strings.NewReader("{\"TESTVAR1\": \"updated\"}\n")
		resp, err := client.Post(envURL, "application/json", r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
		}

		assert.Equal(t, os.Getenv("TESTVAR1"), "updated", "Test env var should be original value")
	})

	t.Run("POST empty", func(t *testing.T) {
		os.Setenv("TESTVAR1", "original")
		defer os.Unsetenv("TESTVAR1")
		assert.Equal(t, os.Getenv("TESTVAR1"), "original", "Test env var should be original value")

		r := strings.NewReader("{\"TESTVAR1\": \"\"}\n")
		resp, err := client.Post(envURL, "application/json", r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
		}

		assert.Equal(t, os.Getenv("TESTVAR1"), "", "Test env var should be empty")
	})

	t.Run("POST null", func(t *testing.T) {
		os.Setenv("TESTVAR1", "original")
		defer os.Unsetenv("TESTVAR1")
		assert.Equal(t, os.Getenv("TESTVAR1"), "original", "Test env var should be original value")

		r := strings.NewReader("{\"TESTVAR1\": null}\n")
		resp, err := client.Post(envURL, "application/json", r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
		}

		assert.Equal(t, os.Getenv("TESTVAR1"), "", "Test env var should be empty")
	})

	t.Run("POST string null", func(t *testing.T) {
		os.Setenv("TESTVAR1", "original")
		defer os.Unsetenv("TESTVAR1")
		assert.Equal(t, os.Getenv("TESTVAR1"), "original", "Test env var should be original value")

		r := strings.NewReader("{\"TESTVAR1\": \"null\"}\n")
		resp, err := client.Post(envURL, "application/json", r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
		}

		assert.Equal(t, os.Getenv("TESTVAR1"), "null", "Test env var should be empty")
	})

	t.Run("GET", func(t *testing.T) {
		resp, err := client.Get(envURL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
		}
	})
}
