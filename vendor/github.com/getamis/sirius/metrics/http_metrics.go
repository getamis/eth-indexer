// Copyright 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"net/http"
	"time"

	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

// HttpMetrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a http server.
type HttpMetrics struct {
	serverStartedCounter          *prom.CounterVec
	serverHandledCounter          *prom.CounterVec
	serverHandledHistogramEnabled bool
	serverHandledHistogramOpts    prom.HistogramOpts
	serverHandledHistogram        *prom.HistogramVec
}

// NewHttpMetrics returns a HttpMetrics object. Use a new instance of
// HttpMetrics when not using the default Prometheus metrics registry, for
// example when wanting to control which metrics are added to a registry as
// opposed to automatically adding metrics via init functions.
func NewHttpMetrics(counterOpts ...grpcProm.CounterOption) *HttpMetrics {
	opts := counterOptions(counterOpts)
	return &HttpMetrics{
		serverStartedCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "http_server_started_total",
				Help: "Total number of http calls started on the server.",
			}), []string{"http_method", "http_path"}),
		serverHandledCounter: prom.NewCounterVec(
			opts.apply(prom.CounterOpts{
				Name: "http_server_handled_total",
				Help: "Total number of http calls completed on the server, regardless of success or failure.",
			}), append([]string{"http_method", "http_path", "http_code"})),
		serverHandledHistogramEnabled: false,
		serverHandledHistogramOpts: prom.HistogramOpts{
			Name:    "http_server_handling_seconds",
			Help:    "Histogram of response latency (seconds) of http call that had been application-level handled by the server.",
			Buckets: prom.DefBuckets,
		},
		serverHandledHistogram: nil,
	}
}

// EnableHandlingTimeHistogram enables histograms being registered when
// registering the HttpMetrics on a Prometheus registry. Histograms can be
// expensive on Prometheus servers. It takes options to configure histogram
// options such as the defined buckets.
func (m *HttpMetrics) EnableHandlingTimeHistogram(opts ...grpcProm.HistogramOption) {
	for _, o := range opts {
		o(&m.serverHandledHistogramOpts)
	}
	if !m.serverHandledHistogramEnabled {
		m.serverHandledHistogram = prom.NewHistogramVec(
			m.serverHandledHistogramOpts,
			append([]string{"http_method", "http_path"}),
		)
	}
	m.serverHandledHistogramEnabled = true
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *HttpMetrics) Describe(ch chan<- *prom.Desc) {
	m.serverStartedCounter.Describe(ch)
	m.serverHandledCounter.Describe(ch)
	if m.serverHandledHistogramEnabled {
		m.serverHandledHistogram.Describe(ch)
	}
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *HttpMetrics) Collect(ch chan<- prom.Metric) {
	m.serverStartedCounter.Collect(ch)
	m.serverHandledCounter.Collect(ch)
	if m.serverHandledHistogramEnabled {
		m.serverHandledHistogram.Collect(ch)
	}
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
func (m *HttpMetrics) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	var startTime time.Time
	if m.serverHandledHistogramEnabled {
		startTime = time.Now()
	}

	m.serverStartedCounter.WithLabelValues(req.Method, req.URL.Path).Inc()

	nrw := &responseWriter{ResponseWriter: rw}
	next(nrw, req)

	m.serverHandledCounter.WithLabelValues(req.Method, req.URL.Path, fmt.Sprintf("%d", nrw.Status())).Inc()
	if m.serverHandledHistogramEnabled {
		m.serverHandledHistogram.WithLabelValues(req.Method, req.URL.Path).Observe(time.Since(startTime).Seconds())
	}
}

// Register registers all server metrics in a given metrics registry. Depending
// on histogram options and whether they are enabled, histogram metrics are
// also registered.
//
// Deprecated: HttpMetrics implements Prometheus Collector interface. You can
// register an instance of HttpMetrics directly by using
// prometheus.Register(m).
func (m *HttpMetrics) Register(r prom.Registerer) error {
	return r.Register(m)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (r *responseWriter) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseWriter) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
