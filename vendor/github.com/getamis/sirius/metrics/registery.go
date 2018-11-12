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
	"net/http"
	"os"
	"strings"

	"github.com/getamis/sirius/log"
)

const MetricsEnabledFlag = "metrics"

var DefaultRegistry Registry = NewDummyRegistry()

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.
func init() {
	for _, arg := range os.Args {
		if flag := strings.TrimLeft(arg, "-"); flag == MetricsEnabledFlag {
			log.Info("Enabling metrics collection")
			DefaultRegistry = NewPrometheusRegistry()
		}
	}
}

// Registry is a metrics gather
type Registry interface {
	NewHttpServerMetrics(opts ...Option) HttpServerMetrics
	NewServerMetrics(opts ...Option) ServerMetrics
	NewCounter(key string, opts ...Option) Counter
	NewGauge(key string, opts ...Option) Gauge
	NewHistogram(key string, opts ...Option) Histogram
	NewTimer(key string, opts ...Option) Timer
	NewWorker(key string, opts ...Option) Worker

	// ServeHTTP is used to display all metric values through http request
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func NewHttpServerMetrics(opts ...Option) HttpServerMetrics {
	return DefaultRegistry.NewHttpServerMetrics(opts...)
}

func NewServerMetrics(opts ...Option) ServerMetrics {
	return DefaultRegistry.NewServerMetrics(opts...)
}

func NewCounter(key string, opts ...Option) Counter {
	return DefaultRegistry.NewCounter(key, opts...)
}

func NewGauge(key string, opts ...Option) Gauge {
	return DefaultRegistry.NewGauge(key, opts...)
}

func NewHistogram(key string, opts ...Option) Histogram {
	return DefaultRegistry.NewHistogram(key, opts...)
}

func NewTimer(key string, opts ...Option) Timer {
	return DefaultRegistry.NewTimer(key, opts...)
}

func NewWorker(key string, opts ...Option) Worker {
	return DefaultRegistry.NewWorker(key, opts...)
}
