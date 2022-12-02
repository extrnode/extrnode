package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	MetricsNamespace = "io_extrnode"

	DefaultShutdownTimeout = 5 * time.Second

	// HTTP server defaults.
	DefaultReadTimeout       = 5 * time.Second
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultWriteTimeout      = 5 * time.Second
)

var (
	// Errors
	ErrorServerAlreadyRunning error = errors.New("HTTP server is already running")
)

// Metrics contains all set of methods to manage metrics collector instance behavior
type Metrics interface {
	StartHTTP(port int) error
	StopHTTP()

	gaugeMethods
	counterMethods
	histogramMethods
	summaryMethods
}

// metricsHandler implements Metrics interface.
type metricsHandler struct {
	sync.Mutex
	registry *prometheus.Registry
	server   *http.Server
}

// NewMetrics creates a new metrics handler and returns its interface.
func NewMetrics() Metrics {
	return &metricsHandler{
		registry: prometheus.NewRegistry(),
	}
}

// StartHTTP opens and listens HTTP route to let external Prometheus to collect metrics.
func (m *metricsHandler) StartHTTP(port int) error {
	m.Lock()
	defer m.Unlock()

	if port == 0 {
		return errors.New("bad param: port")
	}

	if m.server != nil {
		return ErrorServerAlreadyRunning
	}

	listen := fmt.Sprintf(":%d", port)
	m.server = &http.Server{
		Addr:              listen,
		ReadTimeout:       DefaultReadTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,

		Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
				}
			}),
	}

	go func(srv *http.Server) {
		for {
			log.Debugf("prometheus service starts listening '%s'", listen)
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				log.Errorf("prometheus service failed: %s", err)
			}

			time.Sleep(time.Second)
		}
	}(m.server)

	return nil
}

// StopHTTP stops running metrics provider HTTP server.
func (m *metricsHandler) StopHTTP() {
	m.Lock()
	defer m.Unlock()

	if m.server == nil {
		log.Debug("metrics HTTP server is not running")
		return
	}

	log.Debug("metrics HTTP server is stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer func() {
		m.server = nil
		cancel()
	}()

	if err := m.server.Shutdown(ctx); err != nil {
		log.Errorf("metrics HTTP server stop error: %s", err)
	} else {
		log.Debug("metrics HTTP server has been stopped")
	}
}
