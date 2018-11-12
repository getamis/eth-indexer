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

type DummyRegistry struct {
}

func NewDummyRegistry() *DummyRegistry {
	return &DummyRegistry{}
}

func (d *DummyRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Not support metrics.\n"))
}

func (d *DummyRegistry) NewHttpServerMetrics(opts ...Option) HttpServerMetrics {
	return &dummyHttpMetrics{}
}

func (d *DummyRegistry) NewServerMetrics(opts ...Option) ServerMetrics {
	return &dummyServerMetrics{}
}

func (d *DummyRegistry) NewCounter(key string, opts ...Option) Counter {
	return &dummyCounter{}
}

func (d *DummyRegistry) NewGauge(key string, opts ...Option) Gauge {
	return &dummyGauge{}
}

func (d *DummyRegistry) NewHistogram(key string, opts ...Option) Histogram {
	return &dummyHistogram{}
}

func (d *DummyRegistry) NewTimer(key string, opts ...Option) Timer {
	return &dummyTimer{}
}

func (d *DummyRegistry) NewWorker(key string, opts ...Option) Worker {
	return &dummyWorker{}
}

type dummyServerMetrics struct{}

func (d *dummyServerMetrics) InitializeMetrics(*grpc.Server) {}
func (d *dummyServerMetrics) StreamServerInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}
func (d *dummyServerMetrics) UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}

type dummyCounter struct{}

func (d *dummyCounter) Inc()        {}
func (d *dummyCounter) Add(float64) {}

type dummyGauge struct{}

func (d *dummyGauge) Set(float64) {}

type dummyHistogram struct{}

func (d *dummyHistogram) Observe(float64) {}

type dummyTimer struct{}

func (d *dummyTimer) Observe(time.Time) {}

type dummyWorker struct{}

func (d *dummyWorker) Observe(time.Time, error) {}

type dummyHttpMetrics struct {
}

func (*dummyHttpMetrics) ServeHTTP(rw http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	next(rw, req)
}
