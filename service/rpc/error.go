package rpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidBlockNumber = status.Error(codes.InvalidArgument, "invalid block number")
)

// NewInternalServerError returns statusError for internal server error
func NewInternalServerError(err error) error {
	return status.Error(
		codes.Internal,
		err.Error(),
	)
}
