// Copyright 2017 AMIS Technologies
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

package database

import (
	"time"

	"github.com/getamis/sirius/log"
)

type Option func(*Options)

type Options struct {
	Driver        string
	TableName     string
	Logging       bool
	Logger        log.Logger
	RetryDelay    time.Duration
	RetryTimeout  time.Duration
	DriverOptions []interface{}

	// options to connection pool
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

func Retry(delay time.Duration, timeout time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = delay
		o.RetryTimeout = timeout
	}
}

func Table(name string) Option {
	return func(o *Options) {
		o.TableName = name
	}
}

func Logging(enabled bool) Option {
	return func(o *Options) {
		o.Logging = enabled
	}
}

func Logger(logger log.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func Driver(name string) Option {
	return func(o *Options) {
		o.Driver = name
	}
}

func DriverOption(opts ...interface{}) Option {
	return func(o *Options) {
		o.DriverOptions = opts
	}
}

func MaxIdleConns(n int) Option {
	return func(o *Options) {
		o.MaxIdleConns = n
	}
}

func MaxOpenConns(n int) Option {
	return func(o *Options) {
		o.MaxOpenConns = n
	}
}

func ConnMaxLifetime(d time.Duration) Option {
	return func(o *Options) {
		o.ConnMaxLifetime = d
	}
}
