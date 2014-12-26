package util

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/arjantop/saola"
	"github.com/arjantop/saola/httpservice"
	"golang.org/x/net/context"
	"gopkg.in/inconshreveable/log15.v2"
)

type Data map[string]interface{}

type ErrorResponse struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Err        error  `json:"error,omitempty"`
	Data       Data   `json:"-"`
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("[%d] %s (%s)", e.StatusCode, e.Message, e.Error)
}

func NewErrorResponse(message string, err error, data Data) ErrorResponse {
	return NewCustomError(http.StatusInternalServerError, message, err, data)
}

func NewCustomError(statusCode int, message string, err error, data Data) ErrorResponse {
	return ErrorResponse{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
		Data:       data,
	}
}

func NewErrorResponseFilter() saola.Filter {
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		req := httpservice.GetHttpRequest(ctx)
		err := s.Do(ctx)
		if er, ok := err.(ErrorResponse); ok {
			req.Writer.WriteHeader(er.StatusCode)
			req.Writer.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(req.Writer)
			result := struct {
				Message string `json:"message"`
				Err     string `json:"error,omitempty"`
			}{
				er.Message,
				er.Err.Error(),
			}
			encodeError := encoder.Encode(&result)
			if encodeError != nil {
				return fmt.Errorf("error encoding the error response: ", encodeError)
			}
		}
		return err
	})
}

func NewErrorLoggerFilter() saola.Filter {
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		logger := GetContextLogger(ctx)
		err := s.Do(ctx)
		if er, ok := err.(ErrorResponse); ok {
			er.Data["error"] = er.Err
			logger.Warn(er.Message, log15.Ctx(er.Data))
		} else if err != nil {
			logger.Warn("service error", "error", err)
		}
		return err
	})
}
