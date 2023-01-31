package metrics

import (
	"fmt"
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	httpCodeMetricArg   = "http_code"
	methodMetricArg     = "method"
	serverMetricArg     = "server"
	rpcErrCodeMetricArg = "code"
	successArg          = "success"
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
		bytesReadTotal        *prometheus.Metric
		httpResponsesTotal    *prometheus.Metric
		rpcError              *prometheus.Metric
		userFailedRequestsCnt *prometheus.Metric

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

	initMetric(&metrics.bytesReadTotal, newCounter(
		"bytesReadTotal",
		"bytes_read_total",
		"",
		[]string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
	))

	initMetric(&metrics.httpResponsesTotal, newCounter(
		"httpResponsesTotal",
		"http_responses_total",
		"",
		[]string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
	))

	initMetric(&metrics.rpcError, newCounter(
		"rpcError",
		"rpc_error",
		"inner blockchain rpc error",
		[]string{rpcErrCodeMetricArg, httpCodeMetricArg, methodMetricArg, serverMetricArg},
	))

	initMetric(&metrics.userFailedRequestsCnt, newCounter(
		"userFailedRequestsCnt",
		"user_failed_requests",
		"processing error due to user",
		[]string{rpcErrCodeMetricArg, httpCodeMetricArg, methodMetricArg, serverMetricArg},
	))

	initMetric(&metrics.executionTime, newHistogram(
		"executionTime",
		"execution_time",
		"total request execution time",
		[]string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
		[]float64{10, 50, 100, 200, 400, 600, 800, 1000, 1500, 2000, 4000, 6000, 8000, 10000, 15000, 20000, 25000, 30000},
	))

	initMetric(&metrics.nodeResponseTime, newHistogram(
		"nodeResponseTime",
		"node_response_time",
		"the time it took to fetch data from node",
		[]string{methodMetricArg, serverMetricArg},
		[]float64{10, 50, 100, 200, 400, 600, 800, 1000, 1500, 2000, 4000, 6000},
	))

	initMetric(&metrics.nodeAttempts, newHistogram(
		"nodeAttempts",
		"node_attempts",
		"attempts to fetch data from node",
		[]string{methodMetricArg, serverMetricArg, successArg},
		[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
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

// TODO: use after adding error types
func IncUserFailedRequestsCnt(rpcErrCode, httpCode, method, server string) {
	l := prom.Labels{rpcErrCodeMetricArg: rpcErrCode, httpCodeMetricArg: httpCode, methodMetricArg: method, serverMetricArg: server}
	metrics.userFailedRequestsCnt.MetricCollector.(*prom.CounterVec).With(l).Inc()
}

func ObserveExecutionTime(httpCode, method, server string, d time.Duration) {
	l := prom.Labels{httpCodeMetricArg: httpCode, methodMetricArg: method, serverMetricArg: server}
	metrics.executionTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d.Milliseconds()))
}
func ObserveNodeResponseTime(method, server string, d int64) {
	l := prom.Labels{methodMetricArg: method, serverMetricArg: server}
	metrics.nodeResponseTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d))
}
func ObserveNodeAttempts(method, server string, attempts int, success bool) {
	l := prom.Labels{methodMetricArg: method, serverMetricArg: server, successArg: fmt.Sprintf("%t", success)}
	metrics.nodeAttempts.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(attempts))
}

func AddBytesReadTotalCnt(httpCode, method, server string, bytes float64) {
	l := prom.Labels{httpCodeMetricArg: httpCode, methodMetricArg: method, serverMetricArg: server}
	metrics.bytesReadTotal.MetricCollector.(*prom.CounterVec).With(l).Add(bytes)
}
func IncHttpResponsesTotalCnt(httpCode, method, server string) {
	l := prom.Labels{httpCodeMetricArg: httpCode, methodMetricArg: method, serverMetricArg: server}
	metrics.httpResponsesTotal.MetricCollector.(*prom.CounterVec).With(l).Inc()
}
func IncRpcErrorCnt(rpcErrCode, httpCode, method, server string) {
	l := prom.Labels{rpcErrCodeMetricArg: rpcErrCode, httpCodeMetricArg: httpCode, methodMetricArg: method, serverMetricArg: server}
	metrics.rpcError.MetricCollector.(*prom.CounterVec).With(l).Inc()
}

func ObserveAvailableEndpoints(amount int) {
	metrics.availableEndpoints.MetricCollector.(prom.Gauge).Set(float64(amount))
}
