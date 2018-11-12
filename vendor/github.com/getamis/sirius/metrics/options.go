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
	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

type Options struct {
	Namespace string
	Subsystem string
	Labels    map[string]string
}

func NewOptions(namespace, subsystem string, lbs map[string]string) *Options {
	opt := &Options{
		Namespace: namespace,
		Subsystem: subsystem,
		Labels:    make(map[string]string),
	}
	for k, v := range lbs {
		opt.Labels[k] = v
	}
	return opt
}

type Option func(opt *Options)

func Labels(lbs map[string]string) Option {
	return func(opt *Options) {
		for k, v := range lbs {
			opt.Labels[k] = v
		}
	}
}

func Namespace(namespace string) Option {
	return func(opt *Options) {
		opt.Namespace = namespace
	}
}

func Subsystem(subsystem string) Option {
	return func(opt *Options) {
		opt.Subsystem = subsystem
	}
}

// To support grpc prometheus options
func ToGRPCPromCounterOption(opts *Options) grpcProm.CounterOption {
	return func(promOpts *prom.CounterOpts) {
		promOpts.Namespace = opts.Namespace
		promOpts.Subsystem = opts.Subsystem
		promOpts.ConstLabels = prom.Labels(opts.Labels)
	}
}

func ToGRPCPromHistogramOption(opts *Options) grpcProm.HistogramOption {
	return func(promOpts *prom.HistogramOpts) {
		promOpts.Namespace = opts.Namespace
		promOpts.Subsystem = opts.Subsystem
		promOpts.ConstLabels = prom.Labels(opts.Labels)
	}
}

type counterOptions []grpcProm.CounterOption

func (co counterOptions) apply(o prom.CounterOpts) prom.CounterOpts {
	for _, f := range co {
		f(&o)
	}
	return o
}
