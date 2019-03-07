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

package postgresql

import (
	"fmt"
)

type Option func(*options)

type options struct {
	Address      string
	Port         string
	UserName     string
	Password     string
	DatabaseName string
	SSLMode      string
}

func Connector(address string, port string) Option {
	return func(o *options) {
		o.Address = address
		o.Port = port
	}
}

func UserInfo(username, password string) Option {
	return func(o *options) {
		o.UserName = username
		o.Password = password
	}
}

func Database(name string) Option {
	return func(o *options) {
		o.DatabaseName = name
	}
}

func SSLMode(sslMode string) Option {
	return func(o *options) {
		o.SSLMode = sslMode
	}
}

// [user[:password]@][netloc][:port][/dbname][?param1=value1&...]
func (o *options) String() string {
	return fmt.Sprintf(
		"%s:%s@%s:%s/%s?sslmode=%s",
		o.UserName,
		o.Password,
		o.Address,
		o.Port,
		o.DatabaseName,
		o.SSLMode)
}

func ToConnectionString(opts ...interface{}) (string, error) {
	o := defaultOptions()
	for _, opt := range opts {
		optFn, ok := opt.(Option)
		if ok {
			optFn(o)
		} else {
			return "", fmt.Errorf("Invalid option: %v", opt)
		}
	}

	return o.String(), nil
}

func ToConnectionArgs(opts ...interface{}) (string, error) {
	o := defaultOptions()
	for _, opt := range opts {
		optFn, ok := opt.(Option)
		if ok {
			optFn(o)
		} else {
			return "", fmt.Errorf("Invalid option: %v", opt)
		}
	}

	return fmt.Sprintf("host=%v port=%v user=%v dbname=%v password=%v sslmode=%v", o.Address, o.Port, o.UserName, o.DatabaseName, o.Password, o.SSLMode), nil
}

func defaultOptions() *options {
	return &options{
		Address:      "localhost",
		Port:         "5432",
		UserName:     "admin",
		Password:     "12345",
		DatabaseName: "postgres",
		SSLMode:      "disable",
	}
}
