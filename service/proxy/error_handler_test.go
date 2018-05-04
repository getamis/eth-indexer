package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorHandler(t *testing.T) {
	ctx := context.Background()

	for _, spec := range []struct {
		err    error
		status int
		msg    string
	}{
		{
			err:    status.Error(codes.InvalidArgument, "invalid argument"),
			status: http.StatusBadRequest,
			msg:    "invalid argument",
		},
		{
			err:    &strconv.NumError{Func: "", Num: "", Err: errors.New("test error")},
			status: http.StatusBadRequest,
			msg:    "invalid argument",
		},
		{
			err:    status.Error(codes.NotFound, "not found"),
			status: http.StatusNotFound,
			msg:    "not found",
		},
		{
			err:    status.Error(codes.Internal, "internal error"),
			status: http.StatusInternalServerError,
			msg:    "internal error",
		},
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("", "", nil)
		handleHTTPError(ctx, &runtime.ServeMux{}, &runtime.JSONBuiltin{}, w, req, spec.err)

		assert.Equal(t, w.Header().Get("Content-Type"), "application/json")

		assert.Equal(t, w.Code, spec.status)

		body := &errorBody{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Errorf("json.Unmarshal(%q, &body) failed with %v; want success", w.Body.Bytes(), err)
			continue
		}

		err := body.Error
		assert.Equal(t, err.Code, spec.status)
		assert.Equal(t, err.Message, spec.msg)
	}
}
