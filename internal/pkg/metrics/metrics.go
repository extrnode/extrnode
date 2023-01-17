package metrics

import (
	"time"

	"github.com/labstack/echo-contrib/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	gaugeMetricType        = "gauge"
	counterVecMetricType   = "counter_vec"
	histogramVecMetricType = "histogram_vec"

	httpCodeMetricArg   = "http_code"
	methodMetricArg     = "method"
	serverMetricArg     = "server"
	rpcErrCodeMetricArg = "code"
)

// See the NewMetrics func for proper descriptions and prometheus names!
// In case you add a metric here later, make sure to include it in the
// MetricsList method or you'll going to have a bad time.
var (
	metrics struct {
		startTime              *prometheus.Metric
		userFailedRequestsCnt  *prometheus.Metric
		processingTime         *prometheus.Metric
		nodeResponseTime       *prometheus.Metric
		nodeAttemptsPerRequest *prometheus.Metric
		bytesReadTotal         *prometheus.Metric
		httpResponsesTotal     *prometheus.Metric
		rpcError               *prometheus.Metric
	}

	metricList []*prometheus.Metric
)

// Creates and populates a new Metrics struct
// This is where all the prometheus metrics, names and labels are specified
func init() {
	initMetric(&metrics.startTime, &prometheus.Metric{
		ID:          "startTime",
		Name:        "start_time",
		Description: "api start time",
		Type:        gaugeMetricType,
	})
	initMetric(&metrics.userFailedRequestsCnt, &prometheus.Metric{
		ID:          "userFailedRequestsCnt",
		Name:        "user_failed_requests",
		Description: "processing error due to user",
		Type:        counterVecMetricType,
		Args:        []string{rpcErrCodeMetricArg, httpCodeMetricArg, methodMetricArg, serverMetricArg},
	})
	initMetric(&metrics.processingTime, &prometheus.Metric{
		ID:          "processingTime",
		Name:        "processing_time",
		Description: "the time it took to process the request",
		Type:        histogramVecMetricType,
		Args:        []string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
	})
	initMetric(&metrics.nodeResponseTime, &prometheus.Metric{
		ID:          "nodeResponseTime",
		Name:        "node_response_time",
		Description: "the time it took to fetch data from node",
		Type:        histogramVecMetricType,
		Args:        []string{serverMetricArg},
	})
	initMetric(&metrics.nodeAttemptsPerRequest, &prometheus.Metric{
		ID:          "nodeAttemptsPerRequest",
		Name:        "node_attempts_per_request",
		Description: "attempts to fetch data from node",
		Type:        histogramVecMetricType,
		Args:        []string{serverMetricArg},
	})
	initMetric(&metrics.bytesReadTotal, &prometheus.Metric{
		ID:   "bytesReadTotal",
		Name: "bytes_read_total",
		Type: counterVecMetricType,
		Args: []string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
	})
	initMetric(&metrics.httpResponsesTotal, &prometheus.Metric{
		ID:   "httpResponsesTotal",
		Name: "http_responses_total",
		Type: counterVecMetricType,
		Args: []string{httpCodeMetricArg, methodMetricArg, serverMetricArg},
	})
	initMetric(&metrics.rpcError, &prometheus.Metric{
		ID:          "rpcError",
		Name:        "rpc_error",
		Description: "inner blockchain rpc error",
		Type:        counterVecMetricType,
		Args:        []string{rpcErrCodeMetricArg, httpCodeMetricArg, methodMetricArg, serverMetricArg},
	})
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

func ObserveProcessingTime(d time.Duration) {
	l := prom.Labels{httpCodeMetricArg: "httpCode", methodMetricArg: "method", serverMetricArg: "server"}
	metrics.processingTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d.Milliseconds()))
}
func ObserveNodeResponseTime(server string, d time.Duration) {
	l := prom.Labels{serverMetricArg: server}
	metrics.nodeResponseTime.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(d.Milliseconds()))
}
func ObserveNodeAttemptsPerRequest(server string, attempts int) {
	l := prom.Labels{serverMetricArg: server}
	metrics.nodeAttemptsPerRequest.MetricCollector.(*prom.HistogramVec).With(l).Observe(float64(attempts))
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
