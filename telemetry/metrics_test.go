package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/events"
)

/*
The prometheus client library doesn't expose any of the internals of the
collectors, so we can't ask them directly to find out if we've recorded metrics
in our tests. So for those tests we'll stand up a test HTTP server and give it
the prometheus handler and then check the results of a GET.
*/

func TestMetricRun(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()
	cfg := &MetricConfig{
		Namespace: "telemetry",
		Subsystem: "metrics",
		Name:      "TestMetricObserve",
		Help:      "help",
		Type:      "counter",
	}
	cfg.Validate()
	metric := NewMetric(cfg)

	bus := events.NewEventBus()
	ctx := context.Background()
	metric.Run(ctx, bus)

	record := events.Event{Code: events.Metric, Source: fmt.Sprintf("%s|84", metric.Name)}
	bus.Publish(record)

	metric.Receive(events.QuitByTest)
	bus.Wait()
	results := bus.DebugEvents()

	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	resp := getFromTestServer(t, testServer)
	assert.Equal(t,
		1, strings.Count(resp, "telemetry_metrics_TestMetricObserve 84"),
		"failed to get match for metric in response")
}

// TestMetricProcessMetric covers the same ground as the 4 collector-
// specific tests below, but checks the unhappy path.
func TestMetricProcessMetric(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()
	cfg := &MetricConfig{
		Namespace: "telemetry",
		Subsystem: "metrics",
		Name:      "TestMetricProcessMetric",
		Help:      "help",
		Type:      "gauge",
	}
	cfg.Validate()
	metric := NewMetric(cfg)
	testFunc := func(input, expected string) bool {
		metric.processMetric(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}

	t.Run("record Ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_metrics_TestMetricProcessMetric|30.0",
			"telemetry_metrics_TestMetricProcessMetric 30",
		), "failed to get match for metric in response")
	})
	t.Run("record wrong name", func(t *testing.T) {
		assert.True(t, testFunc(
			"TestMetricProcessMetric|20.0",
			"telemetry_metrics_TestMetricProcessMetric 30",
		), "should not have updated metric value")
	})
	t.Run("invalid record", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_metrics_TestMetricProcessMetric",
			"telemetry_metrics_TestMetricProcessMetric 30",
		), "should not have updated metric value")
	})
	t.Run("non-numeric record", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_metrics_TestMetricProcessMetric|xxx",
			"telemetry_metrics_TestMetricProcessMetric 30",
		), "should not have updated metric value")
	})

}

func TestMetricRecordCounter(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()
	metric := &Metric{
		Type: Counter,
		collector: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "telemetry",
			Subsystem: "metrics",
			Name:      "TestMetricRecordCounter",
			Help:      "help",
		})}
	prometheus.MustRegister(metric.collector)
	testFunc := func(input, expected string) bool {
		metric.record(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"1", "telemetry_metrics_TestMetricRecordCounter 1"),
			"failed to update metric")
	})
	t.Run("record update", func(t *testing.T) {
		assert.True(t, testFunc(
			"2", "telemetry_metrics_TestMetricRecordCounter 3"),
			"failed to update metric")
	})
}

func TestMetricRecordGauge(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()
	metric := &Metric{
		Type: Gauge,
		collector: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "telemetry",
			Subsystem: "metrics",
			Name:      "TestMetricRecordGauge",
			Help:      "help",
		})}
	prometheus.MustRegister(metric.collector)

	testFunc := func(input, expected string) bool {
		metric.record(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"1.2", "telemetry_metrics_TestMetricRecordGauge 1.2"),
			"failed to update metric")
	})
	t.Run("record update", func(t *testing.T) {
		assert.True(t, testFunc(
			"2.3", "telemetry_metrics_TestMetricRecordGauge 2.3"),
			"failed to update metric")
	})
}

func TestMetricRecordHistogram(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()

	metric := &Metric{
		Type: Histogram,
		collector: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "telemetry",
			Subsystem: "metrics",
			Name:      "TestMetricRecordHistogram",
			Help:      "help",
		})}
	prometheus.MustRegister(metric.collector)
	patt := `telemetry_metrics_TestMetricRecordHistogram_bucket{le="([\.0-9|\+Inf]*)"} ([1-9])`

	testFunc := func(input string, expected [][]string) bool {
		metric.record(input)
		resp := getFromTestServer(t, testServer)
		return checkBuckets(resp, patt, expected)
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc("1.2",
			[][]string{{"2.5", "1"}, {"5", "1"}, {"10", "1"}, {"+Inf", "1"}}),
			"failed to update metric")
	})
	t.Run("record add", func(t *testing.T) {
		assert.True(t, testFunc("1.2",
			[][]string{{"2.5", "2"}, {"5", "2"}, {"10", "2"}, {"+Inf", "2"}}),
			"failed to update metric")
	})
	t.Run("record overlap", func(t *testing.T) {
		assert.True(t, testFunc("4.5",
			[][]string{{"2.5", "2"}, {"5", "3"}, {"10", "3"}, {"+Inf", "3"}}),
			"failed to update metric")
	})
}

func TestMetricRecordSummary(t *testing.T) {
	testServer := httptest.NewServer(promhttp.Handler())
	defer testServer.Close()
	objMap := map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}

	metric := &Metric{
		Type: Summary,
		collector: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  "telemetry",
			Subsystem:  "metrics",
			Name:       "TestMetricRecordSummary",
			Help:       "help",
			Objectives: objMap,
		})}
	prometheus.MustRegister(metric.collector)
	patt := `telemetry_metrics_TestMetricRecordSummary{quantile="([\.0-9]*)"} ([0-9\.]*)`

	t.Run("record ok", func(t *testing.T) {
		// need a bunch of metrics to make quantiles make any sense
		for i := 1; i <= 10; i++ {
			metric.record(fmt.Sprintf("%v", i))
		}
		resp := getFromTestServer(t, testServer)
		expected := [][]string{{"0.5", "5"}, {"0.9", "9"}, {"0.99", "10"}}
		assert.True(t, checkBuckets(resp, patt, expected),
			"failed to get match for metric in response")
	})
	t.Run("record update", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			// add a new record for each one in the bottom half
			metric.record(fmt.Sprintf("%v", i))
		}
		resp := getFromTestServer(t, testServer)
		expected := [][]string{{"0.5", "4"}, {"0.9", "9"}, {"0.99", "10"}}
		assert.True(t, checkBuckets(resp, patt, expected),
			"failed to get match for metric in response")
	})
}

// test helpers

func checkBuckets(resp, patt string, expected [][]string) bool {
	re := regexp.MustCompile(patt)
	matches := re.FindAllStringSubmatch(resp, -1)
	var buckets [][]string
	if len(matches) != len(expected) {
		fmt.Printf("%v matches vs %v expected\n", len(matches), len(expected))
		return false
	}
	for _, m := range matches {
		buckets = append(buckets, []string{m[1], m[2]})
	}
	var ok bool
	if ok = reflect.DeepEqual(buckets, expected); !ok {
		fmt.Println("Match content:")
		for _, m := range matches {
			fmt.Println(m)
		}
		fmt.Printf("Expected:\n%v\n-----Actual:\n%v\n", expected, buckets)
	}
	return ok
}

func getFromTestServer(t *testing.T, testServer *httptest.Server) string {
	if res, err := http.Get(testServer.URL); err != nil {
		t.Fatal(err)
	} else {
		defer res.Body.Close()
		if resp, err := io.ReadAll(res.Body); err != nil {
			t.Fatal(err)
		} else {
			response := string(resp)
			return response
		}
	}
	return ""
}
