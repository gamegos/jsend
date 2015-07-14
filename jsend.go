package jsend

import (
	"encoding/json"
	"errors"
	"net/http"
)

// JSend status codes
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusFail    = "fail"
)

// Error types
var (
	ErrInvalidRawJSON = errors.New("jsend: given data is not a valid json.RawMessage")
	ErrJSONEncode     = errors.New("jsend: could not json encode given data")
)

// Success json encodes and writes data to specified response with "success" status.
func Success(w http.ResponseWriter, data interface{}, code int) (int, error) {
	return write(w, StatusSuccess, code, data, "")
}

// Error writes error string to specified response with "error" status.
func Error(w http.ResponseWriter, error string, code int) (int, error) {
	return write(w, StatusError, code, nil, error)
}

// Fail json encodes and writes data to specified response with "fail" status.
func Fail(w http.ResponseWriter, data interface{}, code int) (int, error) {
	return write(w, StatusFail, code, data, "")
}

type jsonResponse struct {
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

func writeJSONResponse(w http.ResponseWriter, response *jsonResponse) (int, error) {
	resJSON, err := json.Marshal(response)
	if err != nil {
		return 0, ErrInvalidRawJSON
	}

	return w.Write(resJSON)
}

func write(w http.ResponseWriter, status string, statusCode int, data interface{}, error string) (int, error) {
	res := &jsonResponse{Status: status}
	if data != nil {
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return 0, ErrJSONEncode
		}

		res.Data = dataJSON
	}

	if error != "" {
		res.Message = error
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(statusCode)

	return writeJSONResponse(w, res)
}
