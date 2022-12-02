package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Histogram make an alias for non-decreasing numeric value.
type Histogram prometheus.Histogram

// HistogramVec interface to work with the Histogram labelled.
type HistogramVec interface {
	WithLabelValues(lvs ...string) prometheus.Observer
}

// histogramMethods
type histogramMethods interface {
	NewHistogram(subsystem, name, help string) Histogram
	NewHistogramVec(subsystem, name, help string, buckets []float64, labels ...string) HistogramVec
}

// NewHistogram creates and wraps out 'current_histogram_value_<of>' metrics.
func (m *metricsHandler) NewHistogram(subsystem, name, help string) Histogram {

	metric := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
	)
	m.registry.MustRegister(metric)

	return metric
}

// NewHistogramVec creates and wraps out 'current_Histogram_value_<of>' metrics with labels set.
func (m *metricsHandler) NewHistogramVec(subsystem, name, help string, buckets []float64, labels ...string) HistogramVec {

	b := make([]float64, len(buckets))
	copy(b, buckets)

	metric := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   b,
		},
		labels,
	)
	m.registry.MustRegister(metric)

	return metric
}
