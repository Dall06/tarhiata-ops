package out

import "time"

// Response es el sobre estándar para respuestas JSON consistentes en Tarhiata-Ops.
type Response struct {
	Success   bool      `json:"success"`
	Data      any       `json:"data,omitempty"`
	Message   string    `json:"message,omitempty"`
	CalledAt  time.Time `json:"cat,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Meta      any       `json:"meta,omitempty"`
}

// OK crea una respuesta de éxito con datos y timestamp actual.
func OK(data any) Response {
	return Response{
		Success:  true,
		Data:     data,
		CalledAt: time.Now().UTC(),
	}
}

// Msg crea una respuesta de éxito con un mensaje descriptivo.
func Msg(message string) Response {
	return Response{
		Success:  true,
		Message:  message,
		CalledAt: time.Now().UTC(),
	}
}

// Fail crea una respuesta de error con un mensaje.
func Fail(message string) Response {
	return Response{
		Success:  false,
		Message:  message,
		CalledAt: time.Now().UTC(),
	}
}
