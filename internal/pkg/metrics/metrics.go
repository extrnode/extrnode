package metrics

import (
	"fmt"
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	methodMetricArg = "method"
	successArg      = "success"
)

// See the NewMetrics func for proper descriptions and prometheus names!
// In case you add a metric here later, make sure to include it in the
// MetricsList method or you'll going to have a bad time.
var (
	metrics struct {
		// Gauge
		startTime          *prometheus.Metric
		availableEndpoints *prometheus.Metric

		// Counter
		httpResponsesTotal *prometheus.Metric

		// Histogram
		executionTime    *prometheus.Metric
		nodeResponseTime *prometheus.Metric
		nodeAttempts     *prometheus.Metric
	}

	metricList []*prometheus.Metric
)

// Creates and populates a new Metrics struct
// This is where all the prometheus metrics, names and labels are specified
func init() {
	initMetric(&metrics.startTime, newGauge(
		"startTime",
		"start_time",
		"api start time",
	))

	initMetric(&metrics.availableEndpoints, newGauge(
		"availableEndpoints",
		"available_endpoints",
		"amount of available endpoints (without partners)",
	))

	basicArgs := []string{methodMetricArg, successArg}

	initMetric(&metrics.httpResponsesTotal, newCounter(
		"httpResponsesTotal",
		"http_responses_total",
		"",
		basicArgs,
	))

	initMetric(&metrics.executionTime, newHistogram(
		"executionTime",
		"execution_time",
		"total request execution time",
		basicArgs,
		[]float64{50, 100, 500, 800, 1000, 2000, 4000, 8000, 10000, 15000, 20000, 30000},
	))

	initMetric(&metrics.nodeResponseTime, newHistogram(
		"nodeResponseTime",
		"node_response_time",
		"the time it took to fetch data from node",
		basicArgs,
		[]float64{50, 100, 500, 800, 1000, 2000, 4000, 8000, 10000, 15000, 20000, 30000},
	))

	initMetric(&metrics.nodeAttempts, newHistogram(
		"nodeAttempts",
		"node_attempts",
		"attempts to fetch data from node",
		basicArgs,
		[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	))
}

func initMetric(dest **prometheus.Metric, metric *prometheus.Metric) {
	*dest = metric
	metricList = append(metricList, metric)
}

// Needed by echo-contrib so echo can register and collect these metrics
func MetricList() []*prometheus.Metric {
	return metricList
}

func InitStartTime() {
	metrics.startTime.MetricCollector.(prom.Gauge).Set(float64(time.Now().UTC().Unix()))
}

func ObserveExecutionTime(method string, success bool, d time.Duration) {
	l := prom.Labels{methodMetricArg: method, successArg: fmt.Sprintf("%t", success)}
	metrics.executionTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d.Milliseconds()))
}

func ObserveNodeResponseTime(method string, success bool, d int64) {
	l := prom.Labels{methodMetricArg: method, successArg: fmt.Sprintf("%t", success)}
	metrics.nodeResponseTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d))
}

func ObserveNodeAttempts(method string, success bool, attempts int) {
	l := prom.Labels{methodMetricArg: method, successArg: fmt.Sprintf("%t", success)}
	metrics.nodeAttempts.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(attempts))
}

func IncHttpResponsesTotalCnt(method string, success bool) {
	l := prom.Labels{methodMetricArg: method, successArg: fmt.Sprintf("%t", success)}
	metrics.httpResponsesTotal.MetricCollector.(*prom.CounterVec).With(l).Inc()
}

func ObserveAvailableEndpoints(amount int) {
	metrics.availableEndpoints.MetricCollector.(prom.Gauge).Set(float64(amount))
}
