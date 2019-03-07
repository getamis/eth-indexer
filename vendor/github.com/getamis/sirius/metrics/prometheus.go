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
	"os"
	"time"

	"github.com/getamis/sirius/log"
	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusRegistry struct {
	namespace   string
	labels      map[string]string
	registry    *prom.Registry
	httpHandler http.Handler
}

func NewPrometheusRegistry() *PrometheusRegistry {
	defaultLabels := map[string]string{
		"bin": os.Args[0],
	}
	registry := prom.NewRegistry()
	registry.MustRegister(prom.NewGoCollector())
	return &PrometheusRegistry{
		registry:    registry,
		labels:      defaultLabels,
		httpHandler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}
}

func (p *PrometheusRegistry) SetNamespace(namespace string) {
	p.namespace = namespace
}

func (p *PrometheusRegistry) AppendLabels(labels map[string]string) {
	if p.labels == nil {
		p.labels = make(map[string]string)
	}
	for k, v := range labels {
		p.labels[k] = v
	}
}

func (p *PrometheusRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.httpHandler.ServeHTTP(w, r)
}

func (p *PrometheusRegistry) NewHttpServerMetrics(opts ...Option) HttpServerMetrics {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	httpMetrics := NewHttpMetrics(ToGRPCPromCounterOption(options))
	httpMetrics.EnableHandlingTimeHistogram(ToGRPCPromHistogramOption(options))
	err := p.registry.Register(httpMetrics)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return reg.ExistingCollector.(*HttpMetrics)
		}
		log.Warn("Failed to register a http server metrics", "err", err)
	}
	return httpMetrics
}

func (p *PrometheusRegistry) NewServerMetrics(opts ...Option) ServerMetrics {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	grpcMetrics := grpcProm.NewServerMetrics(ToGRPCPromCounterOption(options))
	grpcMetrics.EnableHandlingTimeHistogram(ToGRPCPromHistogramOption(options))
	err := p.registry.Register(grpcMetrics)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return reg.ExistingCollector.(*grpcProm.ServerMetrics)
		}
		log.Warn("Failed to register a server metrics", "err", err)
	}
	return grpcMetrics
}

func (p *PrometheusRegistry) NewCounter(key string, opts ...Option) Counter {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	cnt := prom.NewCounter(prom.CounterOpts{
		Namespace:   options.Namespace,
		Subsystem:   options.Subsystem,
		Name:        key,
		Help:        key,
		ConstLabels: prom.Labels(options.Labels),
	})
	err := p.registry.Register(cnt)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return reg.ExistingCollector.(prom.Counter)
		}
		log.Warn("Failed to register a counter", "key", key, "err", err)
	}
	return cnt
}

func (p *PrometheusRegistry) NewGauge(key string, opts ...Option) Gauge {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	g := prom.NewGauge(prom.GaugeOpts{
		Namespace:   options.Namespace,
		Subsystem:   options.Subsystem,
		Name:        key,
		Help:        key,
		ConstLabels: prom.Labels(options.Labels),
	})
	err := p.registry.Register(g)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return reg.ExistingCollector.(prom.Gauge)
		}
		log.Warn("Failed to register a gauge", "key", key, "err", err)
	}
	return g
}

func (p *PrometheusRegistry) NewHistogram(key string, opts ...Option) Histogram {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	h := prom.NewHistogram(prom.HistogramOpts{
		Namespace:   options.Namespace,
		Subsystem:   options.Subsystem,
		Name:        key,
		Help:        key,
		ConstLabels: prom.Labels(options.Labels),
	})
	err := p.registry.Register(h)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return reg.ExistingCollector.(prom.Histogram)
		}
		log.Warn("Failed to register a histogram", "key", key, "err", err)
	}
	return h
}

func (p *PrometheusRegistry) NewHistogramVec(key string, labels []string, opts ...Option) HistogramVec {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	hv := prom.NewHistogramVec(prom.HistogramOpts{
		Namespace:   options.Namespace,
		Subsystem:   options.Subsystem,
		Name:        key,
		Help:        key,
		ConstLabels: prom.Labels(options.Labels),
	}, labels)
	err := p.registry.Register(hv)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			return &histogramVec{reg.ExistingCollector.(*prom.HistogramVec)}
		}
		log.Warn("Failed to register a histogram vector", "key", key, "err", err)
	}
	return &histogramVec{hv}
}

func (p *PrometheusRegistry) NewTimer(key string, opts ...Option) Timer {
	return &timer{
		elapsedTime: p.NewHistogram(fmt.Sprintf("%s_elapsedtime", key), opts...),
	}
}

func (p *PrometheusRegistry) NewWorker(key string, opts ...Option) Worker {
	counterVec := p.NewCounterVec(key, []string{"result"}, opts...)
	// These are just references (no increments),
	// as just referencing will create the labels but not set values.
	success, _ := counterVec.GetMetricWithLabelValues("success")
	fail, _ := counterVec.GetMetricWithLabelValues("fail")
	return &worker{
		duration: p.NewTimer(key, opts...),
		success:  success,
		fail:     fail,
	}
}

func (p *PrometheusRegistry) NewCounterVec(key string, labelNames []string, opts ...Option) CounterVec {
	options := NewOptions(p.namespace, "", p.labels)
	for _, fn := range opts {
		fn(options)
	}
	cnt := prom.NewCounterVec(prom.CounterOpts{
		Namespace:   options.Namespace,
		Subsystem:   options.Subsystem,
		Name:        key,
		Help:        key,
		ConstLabels: prom.Labels(options.Labels),
	}, labelNames)
	err := p.registry.Register(cnt)
	if err != nil {
		reg, ok := err.(prom.AlreadyRegisteredError)
		if ok {
			existingCV := reg.ExistingCollector.(*prom.CounterVec)
			return &counterVec{existingCV}
		}
		log.Warn("Failed to register a counter vec", "key", key, "err", err)
	}
	return &counterVec{cnt}
}

// -----------------------------------------------------------------------------
// timer
type timer struct {
	elapsedTime Histogram
}

func (t *timer) Observe(begin time.Time) {
	t.elapsedTime.Observe(time.Since(begin).Seconds())
}

// -----------------------------------------------------------------------------
// worker
type worker struct {
	duration Timer
	success  Counter
	fail     Counter
}

func (w *worker) Observe(begin time.Time, err error) {
	w.duration.Observe(begin)
	if err != nil {
		w.fail.Inc()
		return
	}
	w.success.Inc()
}

// -----------------------------------------------------------------------------
// counterVec
type counterVec struct {
	*prom.CounterVec
}

func (c *counterVec) GetMetricWith(labels MetricsLabels) (Counter, error) {
	return c.CounterVec.GetMetricWith(labels)
}
func (c *counterVec) GetMetricWithLabelValues(lvs ...string) (Counter, error) {
	return c.CounterVec.GetMetricWithLabelValues(lvs...)
}

// -----------------------------------------------------------------------------
// histogramVec
type histogramVec struct {
	*prom.HistogramVec
}

func (h *histogramVec) GetMetricWith(labels MetricsLabels) (Histogram, error) {
	return h.HistogramVec.GetMetricWith(labels)
}
func (h *histogramVec) GetMetricWithLabelValues(lvs ...string) (Histogram, error) {
	return h.HistogramVec.GetMetricWithLabelValues(lvs...)
}
