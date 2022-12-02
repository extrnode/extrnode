package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Gauge make an alias for some numeric value.
type Gauge prometheus.Gauge

// GaugeVec interface to work with the LastTime metrics type.
type GaugeVec interface {
	WithLabelValues(lvs ...string) prometheus.Gauge
}

// gaugeMethods
type gaugeMethods interface {
	NewGauge(subsystem, name, help string) Gauge
	NewGaugeVec(subsystem, name, help string, labels ...string) GaugeVec
}

// NewGauge creates and wraps out 'current_amount_<of>' metrics.
func (m *metricsHandler) NewGauge(subsystem, name, help string) Gauge {

	metric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
	)
	m.registry.MustRegister(metric)

	return metric
}

// NewGaugeVec creates and wraps out 'last_time_<of event>' metrics metrics with labels set.
func (m *metricsHandler) NewGaugeVec(subsystem, name, help string, labels ...string) GaugeVec {
	metric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	m.registry.MustRegister(metric)

	return metric
}
