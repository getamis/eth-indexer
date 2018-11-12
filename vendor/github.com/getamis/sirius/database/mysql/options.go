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

package mysql

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/getamis/sirius/crypto/rand"
	"github.com/go-sql-driver/mysql"
)

type Option func(*options)

type options struct {
	Protocol             string
	Address              string
	Port                 string
	UserName             string
	Password             string
	DatabaseName         string
	TableName            string
	Location             string
	TLSConfigName        string
	AllowNativePasswords bool
}

const (
	DefaultProtocol = "tcp"
)

func Connector(protocol string, address string, port string) Option {
	return func(o *options) {
		o.Protocol = protocol
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

func Location(location string) Option {
	return func(o *options) {
		o.Location = location
	}
}

func EnableTLS(config *tls.Config) Option {
	TLSConfigName := rand.New().KeyEncoded()
	mysql.RegisterTLSConfig(TLSConfigName, config)
	return func(o *options) {
		o.TLSConfigName = TLSConfigName
	}
}

func AllowNativePasswords(allow bool) Option {
	return func(o *options) {
		o.AllowNativePasswords = allow
	}
}

func (o *options) String() string {
	loc := o.Location
	if loc == "" {
		loc = "Local"
	}
	tls := o.TLSConfigName
	if tls == "" {
		tls = "false"
	}
	return fmt.Sprintf(
		"%s:%s@%s(%s:%s)/%s?charset=utf8&parseTime=True&loc=%s&allowNativePasswords=%v&tls=%v",
		o.UserName,
		o.Password,
		o.Protocol,
		o.Address,
		o.Port,
		o.DatabaseName,
		loc,
		o.AllowNativePasswords,
		tls)
}

func DSNToOptions(dsn string) (Option, Option, Option) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, nil, nil
	}

	host, port, _ := net.SplitHostPort(config.Addr)

	return Connector(config.Net, host, port),
		UserInfo(config.User, config.Passwd),
		Database(config.DBName)
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

func defaultOptions() *options {
	return &options{
		Protocol:             DefaultProtocol,
		Address:              "localhost",
		Port:                 "3306",
		UserName:             "root",
		DatabaseName:         "db",
		AllowNativePasswords: true,
	}
}
