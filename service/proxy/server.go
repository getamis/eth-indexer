package proxy

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/maichain/eth-indexer/service/pb"
)

func NewProxy(targetEndpoint string, opts ...grpc.DialOption) *grpcProxy {
	return &grpcProxy{
		endpoint:    targetEndpoint,
		grpcOptions: opts,
	}
}

type grpcProxy struct {
	grpcOptions []grpc.DialOption
	endpoint    string
}

func (proxy *grpcProxy) Bind(mux *runtime.ServeMux) error {
	err := pb.RegisterBlockServiceHandlerFromEndpoint(context.Background(), mux, proxy.endpoint, proxy.grpcOptions)
	if err != nil {
		return err
	}
	err = pb.RegisterTransactionServiceHandlerFromEndpoint(context.Background(), mux, proxy.endpoint, proxy.grpcOptions)
	if err != nil {
		return err
	}
	err = pb.RegisterAccountServiceHandlerFromEndpoint(context.Background(), mux, proxy.endpoint, proxy.grpcOptions)
	if err != nil {
		return err
	}
	return nil
}
