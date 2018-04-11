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

package rpc

import (
	"crypto/tls"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewServer creates a gRPC server with pre-configured services
func NewServer(opts ...ServerOption) *Server {
	server := &Server{}

	for _, opt := range opts {
		opt(server)
	}

	server.createGRPCServer()
	server.registerAPIs()

	return server
}

// API provides APIs for specific gRPC server
type API interface {
	Bind(server *grpc.Server)
}

// Server represents a gRPC server
type Server struct {
	grpcServer  *grpc.Server
	credentials *tls.Config

	apis []API
}

func (s *Server) Serve(l net.Listener) error {
	return s.grpcServer.Serve(l)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.grpcServer.ServeHTTP(w, r)
}

func (s *Server) Shutdown() {
	s.grpcServer.GracefulStop()
}

// ----------------------------------------------------------------------------

func (s *Server) createGRPCServer() {
	options := []grpc.ServerOption{}

	// credentials
	if s.credentials != nil {
		tls := credentials.NewTLS(s.credentials)
		options = append(options, grpc.Creds(tls))
	}

	s.grpcServer = grpc.NewServer(options...)
}

func (s *Server) registerAPIs() {
	for _, api := range s.apis {
		api.Bind(s.grpcServer)
	}
}
