package proxy

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/maichain/eth-indexer/service/pb"
	"google.golang.org/grpc"
)

func NewProxy(targetEndpoint string, opts ...grpc.DialOption) *grpcProxy {
	// FIXME: this is a workaround since we have customized error handler in mapi
	// an individual proxy package is needed
	// see more: https://maicoin.atlassian.net/browse/ES-79
	runtime.HTTPError = handleHTTPError

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
