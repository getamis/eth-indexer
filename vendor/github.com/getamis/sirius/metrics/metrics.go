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
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

// Counter is a Metric that represents a single numerical value that only ever
// goes up.
// A Counter is typically used to count requests served, tasks completed, errors
// occurred, etc.
type Counter interface {
	Inc()
	Add(float64)
}

// Gauge is a Metric that represents a single numerical value that can
// arbitrarily go up and down.
// A Gauge is typically used for measured values like temperatures or current
// memory usage.
type Gauge interface {
	Set(float64)
}

// A Histogram counts individual observations from an event or sample stream in
// configurable buckets. Similar to a summary, it also provides a sum of
// observations and an observation count.
type Histogram interface {
	Observe(float64)
}

// Timer represents a Histogram Metrics to observe the time duration according to given begin time.
// Timer is usually used to time a function call in the
// following way:
//     func TimeMe() {
//         begin := time.Now()
//         defer Timer.Observe(begin)
//     }
type Timer interface {
	Observe(time.Time)
}

// Worker includes Timer Metrics to observe the time duration according to given begin time,
// and counter Metreic to gather the success count and fail count.
// Worker is usually used to measure a function call in the
// following way:
//     func MeasureMe()(err error) {
//         begin := time.Now()
//         defer Worker.Observe(begin, err)
//     }
type Worker interface {
	Observe(time.Time, error)
}

// ServerMetrics is an integrated metric collector to measure count of any kind of error
// and elapsed time of each grpc method.
type ServerMetrics interface {
	InitializeMetrics(*grpc.Server)
	StreamServerInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error
	UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error)
}

// HttpServerMetrics is an integrated metric collector to measure count of any kind of error
// and elapsed time of each http call.
type HttpServerMetrics interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request, next http.HandlerFunc)
}
