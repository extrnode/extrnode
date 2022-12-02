package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Summary make an alias for non-decreasing numeric value.
type Summary prometheus.Summary

// SummaryVec interface to work with the Summary labelled.
type SummaryVec interface {
	WithLabelValues(lvs ...string) prometheus.Observer
}

// summaryMethods
type summaryMethods interface {
	NewSummary(subsystem, name, help string) Summary
	NewSummaryVec(subsystem, name, help string, objectives map[float64]float64, labels ...string) SummaryVec
}

// NewSummary creates and wraps out 'current_summary_value_<of>' metrics.
func (m *metricsHandler) NewSummary(subsystem, name, help string) Summary {

	metric := prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: MetricsNamespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
	)
	m.registry.MustRegister(metric)

	return metric
}

// NewSummaryVec creates and wraps out 'current_Summary_value_<of>' metrics with labels set.
func (m *metricsHandler) NewSummaryVec(subsystem, name, help string, objectives map[float64]float64, labels ...string) SummaryVec {

	o := make(map[float64]float64)
	for k, v := range objectives {
		o[k] = v
	}

	metric := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  MetricsNamespace,
			Subsystem:  subsystem,
			Name:       name,
			Help:       help,
			Objectives: o,
		},
		labels,
	)
	m.registry.MustRegister(metric)

	return metric
}
