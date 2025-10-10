package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestStatus_Error(t *testing.T) {
	status := New(codes.InvalidArgument, "test error message")
	assert.Equal(t, "test error message", status.Error())
}

func TestStatus_WriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		status         *Status
		expectedCode   int
		expectedStatus Code
		expectedMsg    string
	}{
		{
			name:           "bad request",
			status:         BadRequest("invalid parameter"),
			expectedCode:   http.StatusBadRequest,
			expectedStatus: codes.InvalidArgument,
			expectedMsg:    "invalid parameter",
		},
		{
			name:           "not found",
			status:         NotFound("resource not found"),
			expectedCode:   http.StatusNotFound,
			expectedStatus: codes.NotFound,
			expectedMsg:    "resource not found",
		},
		{
			name:           "internal error",
			status:         Internal("something went wrong"),
			expectedCode:   http.StatusInternalServerError,
			expectedStatus: codes.Internal,
			expectedMsg:    "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.status.WriteJSON(rec)

			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var result Status
			err := json.Unmarshal(rec.Body.Bytes(), &result)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, result.Code)
			assert.Equal(t, tt.expectedMsg, result.Message)
		})
	}
}

func TestStatus_WithDetail(t *testing.T) {
	status := BadRequest("test error").WithDetail(Detail{
		"@type":  "type.googleapis.com/ErrorInfo",
		"reason": "INVALID_PARAMETER",
	})

	require.Len(t, status.Details, 1)
	assert.Equal(t, "type.googleapis.com/ErrorInfo", status.Details[0]["@type"])
	assert.Equal(t, "INVALID_PARAMETER", status.Details[0]["reason"])
}

func TestStatus_WithMetadata(t *testing.T) {
	metadata := map[string]string{
		"parameter": "slot_gtw",
		"expected":  "slot_gte",
	}

	status := BadRequest("invalid parameter").WithMetadata(metadata)

	require.Len(t, status.Details, 1)
	assert.Equal(t, "type.googleapis.com/ErrorInfo", status.Details[0]["@type"])
	assert.Equal(t, metadata, status.Details[0]["metadata"])
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		code     Code
		expected int
	}{
		{codes.OK, http.StatusOK},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.NotFound, http.StatusNotFound},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.PermissionDenied, http.StatusForbidden},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, HTTPStatus(tt.code))
	}
}

func TestNewf(t *testing.T) {
	status := Newf(codes.InvalidArgument, "invalid parameter: %s", "slot_gtw")
	assert.Equal(t, "invalid parameter: slot_gtw", status.Message)
	assert.Equal(t, codes.InvalidArgument, status.Code)
}

func TestBadRequestf(t *testing.T) {
	status := BadRequestf("value %d is out of range", 999)
	assert.Equal(t, "value 999 is out of range", status.Message)
	assert.Equal(t, codes.InvalidArgument, status.Code)
}
