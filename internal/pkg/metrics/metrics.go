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
)

// See the NewMetrics func for proper descriptions and prometheus names!
// In case you add a metric here later, make sure to include it in the
// MetricsList method or you'll going to have a bad time.
var (
	metrics struct {
		startTime              *prometheus.Metric
		userFailedRequestsCnt  *prometheus.Metric
		nodeFailedRequestsCnt  *prometheus.Metric
		successRequestsCnt     *prometheus.Metric
		processingTime         *prometheus.Metric
		nodeResponseTime       *prometheus.Metric
		nodeAttemptsPerRequest *prometheus.Metric
	}

	metricList []*prometheus.Metric
)

// Needed by echo-contrib so echo can register and collect these metrics
func MetricList() []*prometheus.Metric {
	return metricList
}

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
		Args:        []string{"chain"},
	})
	initMetric(&metrics.nodeFailedRequestsCnt, &prometheus.Metric{
		ID:          "nodeFailedRequestsCnt",
		Name:        "node_failed_requests",
		Description: "processing error due to node",
		Type:        counterVecMetricType,
		Args:        []string{"chain"},
	})
	initMetric(&metrics.successRequestsCnt, &prometheus.Metric{
		ID:          "successRequestsCnt",
		Name:        "success_requests",
		Description: "successfully handled requests by node",
		Type:        counterVecMetricType,
		Args:        []string{"chain"},
	})
	initMetric(&metrics.processingTime, &prometheus.Metric{
		ID:          "processingTime",
		Name:        "processing_time",
		Description: "the time it took to process the request",
		Type:        histogramVecMetricType,
		Args:        []string{"chain"},
	})
	initMetric(&metrics.nodeResponseTime, &prometheus.Metric{
		ID:          "nodeResponseTime",
		Name:        "node_response_time",
		Description: "the time it took to fetch data from node",
		Type:        histogramVecMetricType,
		Args:        []string{"chain", "node"},
	})
	initMetric(&metrics.nodeAttemptsPerRequest, &prometheus.Metric{
		ID:          "nodeAttemptsPerRequest",
		Name:        "node_attempts_per_request",
		Description: "attempts to fetch data from node",
		Type:        histogramVecMetricType,
		Args:        []string{"chain"},
	})
}

func initMetric(dest **prometheus.Metric, metric *prometheus.Metric) {
	*dest = metric
	metricList = append(metricList, metric)
}

func InitStartTime() {
	metrics.startTime.MetricCollector.(prom.Gauge).Set(float64(time.Now().UTC().Unix()))
}

// TODO: use after adding error types
func IncUserFailedRequestsCnt(chain string) {
	metrics.userFailedRequestsCnt.MetricCollector.(*prom.CounterVec).WithLabelValues(chain).Inc()
}
func IncNodeFailedRequestsCnt(chain string) {
	metrics.nodeFailedRequestsCnt.MetricCollector.(*prom.CounterVec).WithLabelValues(chain).Inc()
}
func IncSuccessRequestsCnt(chain string) {
	metrics.successRequestsCnt.MetricCollector.(*prom.CounterVec).WithLabelValues(chain).Inc()
}

func ObserveProcessingTime(chain string, d time.Duration) {
	metrics.processingTime.MetricCollector.(*prom.HistogramVec).WithLabelValues(chain).Observe(float64(d.Milliseconds()))
}
func ObserveNodeResponseTime(chain, node string, d time.Duration) {
	metrics.nodeResponseTime.MetricCollector.(*prom.HistogramVec).WithLabelValues(chain, node).Observe(float64(d.Milliseconds()))
}
func ObserveNodeAttemptsPerRequest(chain string, attempts int) {
	metrics.nodeAttemptsPerRequest.MetricCollector.(*prom.HistogramVec).WithLabelValues(chain).Observe(float64(attempts))
}
