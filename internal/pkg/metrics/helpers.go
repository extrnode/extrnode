package metrics

import "github.com/labstack/echo-contrib/prometheus"

const (
	gaugeMetricType        = "gauge"
	counterVecMetricType   = "counter_vec"
	histogramVecMetricType = "histogram_vec"
)

func newGauge(id, name, description string) *prometheus.Metric {
	return &prometheus.Metric{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        gaugeMetricType,
	}
}

func newCounter(id, name, description string, labels []string) *prometheus.Metric {
	return &prometheus.Metric{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        counterVecMetricType,
		Args:        labels,
	}
}

func newHistogram(id, name, description string, labels []string, buckets []float64) *prometheus.Metric {
	return &prometheus.Metric{
		ID:          id,
		Name:        name,
		Description: description,
		Type:        histogramVecMetricType,
		Args:        labels,
		Buckets:     buckets,
	}
}
