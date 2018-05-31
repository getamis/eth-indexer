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

package rpc

import (
	"net/http"

	"github.com/rs/cors"
)

type ProxyOption func(*proxy)

type Middleware interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request, next http.HandlerFunc)
}

// HTTPServer represents customized HTTP server to the proxy
func HTTPServer(s *http.Server) ProxyOption {
	return func(p *proxy) {
		p.httpServer = s
	}
}

// Proxies represents the proxy API to be registered to the http server
func Proxies(proxies ...Proxy) ProxyOption {
	return func(p *proxy) {
		p.apis = proxies
	}
}

// Middlewares represents the HTTP middlwares to be used in this proxy
func Middlewares(mws ...Middleware) ProxyOption {
	return func(p *proxy) {
		p.middlewares = append(p.middlewares, mws...)
	}
}

// AllowCORS represents the CORS origins to setup for RESTful API
func AllowCORS(origins []string) ProxyOption {
	return func(p *proxy) {
		c := cors.New(cors.Options{
			AllowedOrigins: origins,
		})
		p.middlewares = append(p.middlewares, c)
	}
}
