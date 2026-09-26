package exs

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestConstructorsAndStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		targetSentinel error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "NotFound constructor",
			err:            NotFound("nodo %s no encontrado", "node-1"),
			targetSentinel: ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "nodo node-1 no encontrado",
		},
		{
			name:           "BadRequest constructor",
			err:            BadRequest("parámetro inválido"),
			targetSentinel: ErrBadRequest,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "parámetro inválido",
		},
		{
			name:           "Unauthorized constructor",
			err:            Unauthorized("token expirado"),
			targetSentinel: ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "token expirado",
		},
		{
			name:           "Forbidden constructor",
			err:            Forbidden("acceso restringido a administradores"),
			targetSentinel: ErrForbidden,
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "acceso restringido a administradores",
		},
		{
			name:           "Conflict constructor",
			err:            Conflict("servicio duplicado"),
			targetSentinel: ErrConflict,
			expectedStatus: http.StatusConflict,
			expectedMsg:    "servicio duplicado",
		},
		{
			name:           "Internal constructor",
			err:            Internal("error de base de datos"),
			targetSentinel: ErrInternal,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    internalServerMessage,
		},
		{
			name:           "ServiceUnavailable constructor",
			err:            ServiceUnavailable("nodo ocupado"),
			targetSentinel: ErrServiceUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "nodo ocupado",
		},
		{
			name:           "Wrap generic error with sentinel",
			err:            Wrap(ErrBadRequest, errors.New("subyacente"), "error envuelto"),
			targetSentinel: ErrBadRequest,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "error envuelto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.targetSentinel) {
				t.Errorf("expected errors.Is match for sentinel %v", tt.targetSentinel)
			}

			status, msg := ResolveStatusAndMessage(tt.err)
			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}
			if msg != tt.expectedMsg {
				t.Errorf("expected message %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

func TestEchoHTTPErrorHandler_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "Echo standard HTTPError 404",
			err:            echo.NewHTTPError(http.StatusNotFound, "ruta inexistente"),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "ruta inexistente",
		},
		{
			name:           "Semantic NotFound error",
			err:            NotFound("volumen /opt/data no existe"),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "volumen /opt/data no existe",
		},
		{
			name:           "Semantic BadRequest error",
			err:            BadRequest("payload JSON malformado"),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "payload JSON malformado",
		},
		{
			name:           "Semantic Internal error returns safe message",
			err:            Internal("panic en driver sqlite"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    internalServerMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			EchoHTTPErrorHandler(tt.err, c)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var resp HTTPErrorResponse
			errDecode := json.NewDecoder(rec.Body).Decode(&resp)
			if errDecode != nil {
				t.Fatalf("failed to decode error JSON response: %v", errDecode)
			}
			if resp.Success {
				t.Errorf("expected Success=false, got true")
			}
			if resp.Message != tt.expectedMsg {
				t.Errorf("expected Message=%q, got %q", tt.expectedMsg, resp.Message)
			}
		})
	}
}
