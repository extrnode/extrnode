package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Counter make an alias for non-decreasing numeric value.
type Counter prometheus.Counter

// CounterVec interface to work with the Counter labelled.
type CounterVec interface {
	WithLabelValues(lvs ...string) prometheus.Counter
}

// counterMethods
type counterMethods interface {
	NewCounter(subsystem, name, help string) Counter
	NewCounterVec(subsystem, name, help string, labels ...string) CounterVec
}

// NewCounter creates and wraps out 'current_counter_value_<of>' metrics.
func (m *metricsHandler) NewCounter(subsystem, name, help string) Counter {

	metric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
	)
	m.registry.MustRegister(metric)

	return metric
}

// NewCounterVec creates and wraps out 'current_counter_value_<of>' metrics with labels set.
func (m *metricsHandler) NewCounterVec(subsystem, name, help string, labels ...string) CounterVec {

	metric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
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
