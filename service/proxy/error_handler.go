package proxy

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/getamis/sirius/log"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = log.New("ws", "proxy")

type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Error httpError `json:"error"`
}

func wrapError(err error) (*status.Status, bool) {
	st, ok := status.FromError(err)

	if _, ok := err.(*strconv.NumError); ok {
		return status.New(codes.InvalidArgument, "invalid argument"), true
	}

	return st, ok
}

func handleHTTPError(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	const fallback = `{"error": {"code": 500, "message": "failed to marshal error message"}}`

	w.Header().Set("Content-type", marshaler.ContentType())

	s, ok := wrapError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}

	st := runtime.HTTPStatusFromCode(s.Code())
	httpErr := &httpError{
		Message: s.Message(),
		Code:    st,
	}

	body := &errorBody{Error: *httpErr}

	buf, merr := marshaler.Marshal(body)
	if merr != nil {
		logger.Error("Failed to marshal error message", "body", body, "err", merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			logger.Error("Failed to write response", "err", err)
		}
		return
	}

	w.WriteHeader(st)

	if _, err := w.Write(buf); err != nil {
		logger.Error("Failed to write response", "err", err)
	}
}
