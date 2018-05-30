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
	"context"
	"net"
	"net/http"

	"github.com/getamis/sirius/log"
	"github.com/getamis/sirius/rpc/pb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/urfave/negroni"
)

// NewProxy creates a RESTful proxy server that routes RESTful request to gRPC server
func NewProxy(opts ...ProxyOption) *proxy {
	p := &proxy{
		server: runtime.NewServeMux(
			runtime.WithMarshalerOption(runtime.MIMEWildcard, new(pb.JSONPb)),
		),
		router: negroni.New(),
	}

	for _, opt := range opts {
		opt(p)
	}

	if err := p.registerAPIs(); err != nil {
		log.Error("Failed to register API", "err", err)
		return nil
	}

	return p
}

// Proxy provides APIs for specific gRPC server
type Proxy interface {
	Bind(mux *runtime.ServeMux) error
}

// Server represents a gRPC server
type proxy struct {
	middlewares []Middleware
	router      *negroni.Negroni
	server      *runtime.ServeMux
	httpServer  *http.Server
	apis        []Proxy
}

func (p *proxy) Serve(l net.Listener) error {
	for _, mw := range p.middlewares {
		p.router.Use(mw.((negroni.Handler)))
	}

	// This should be added after all middlewares
	p.router.UseHandler(p.server)

	if p.httpServer == nil {
		p.httpServer = &http.Server{
			Handler: p.router,
		}
	}

	return p.httpServer.Serve(l)
}

func (p *proxy) Shutdown() {
	p.httpServer.Shutdown(context.TODO())
}

// ----------------------------------------------------------------------------

func (p *proxy) registerAPIs() error {
	for _, api := range p.apis {
		if err := api.Bind(p.server); err != nil {
			return err
		}
	}

	return nil
}
