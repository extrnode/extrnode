package metrics

import "extrnode-be/internal/pkg/metrics"

type typedMetrics struct {
	Indexer indexer
}

type indexer struct {
	AccountsReceived metrics.Counter
}

var (
	mi      metrics.Metrics
	Metrics *typedMetrics
)

func init() {
	const subsystem = "scanner"

	mi = metrics.NewMetrics()

	Metrics = &typedMetrics{
		Indexer: indexer{
			AccountsReceived: mi.NewCounter(subsystem,
				"accounts_received", "accounts receiverd",
			),
		},
	}
}

func StartHTTP(port int) error {
	return mi.StartHTTP(port)
}
