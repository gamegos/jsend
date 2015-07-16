/*
Package jsend implements JSend* specification.

Use by wrapping existing ResponseWriter

You can wrap your ResponseWriter:

	w := jsend.Wrap(w)

Returning object is also implements http.ResponseWriter. You can "Write" your json blob to it.

	w.Write(`{"foo": "bar"}`)

	// body:
	{
		"status": "success",
		"data": {
			"foo": "bar"
		}
	}

Status field in response body is derived from http status code. Status is "fail"
if code is 4XX, "error" if code is 5XX and "success" otherwise.


Fail:
	w.WriteHeader(400)
	w.Write(`{"foo": "invalid"}`)

	// body:
	{
		"status": "fail",
		"data": {
			"foo": "invalid"
		}
	}

Error:
	w.WriteHeader(500)
	w.Write(`we are closed`)

	// body:
	{
		"status": "error",
		"message": "we are closed"
	}

Note: Write method does not json-encode your data.

Use status functions

You can use Success, Fail and Error functions directly to write json responses with those statuses.

Success:
	jsend.Success(w, data, 200) // w is ResponseWriter

	// body:
	{
		"status": "success",
		"data": // your json encoded data
	}

Fail:
	jsend.Fail(w, data, 400)

	// body:
	{
		"status": "fail",
		"data": // your json encoded data
	}

Error:
	jsend.Fail(w, "something bad happened", 500)

	{
		"status": "error",
		"message": "something bad happened"
	}

Success, Fail and Error functions write given statusCode to response. Also
"Content-Type" header is set to "application/json" if it is not set already.


* See http://labs.omniti.com/labs/jsend for details.
*/
package jsend

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
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
	ErrWrittenAlready = errors.New("jsend: written already")
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

// Wrap wraps given http.ResponseWriter and returns a response object which
// implements http.ResponseWriter interface.
func Wrap(rw http.ResponseWriter) http.ResponseWriter {
	if rw.Header().Get("Content-Type") == "" {
		rw.Header().Set("Content-Type", "application/json")
	}

	return &response{rw: rw}
}

// A response wraps a http.ResponseWriter.
type response struct {
	rw      http.ResponseWriter
	code    int
	written bool
	sync.Mutex
}

func (r *response) Header() http.Header {
	return r.rw.Header()
}

func (r *response) WriteHeader(code int) {
	r.code = code
	r.rw.WriteHeader(code)
}

func (r *response) Write(data []byte) (int, error) {
	r.Lock()
	defer r.Unlock()

	if r.written {
		return 0, ErrWrittenAlready
	}
	r.written = true

	st := getStatus(r.code)
	jr := &jsonResponse{Status: st}
	switch st {
	case StatusError:
		jr.Message = string(data)
	case StatusFail:
		jr.Data = data
	default:
		jr.Data = data
	}

	return writeJSONResponse(r.rw, jr)
}

func getStatus(code int) string {
	switch {
	case code >= 500:
		return StatusError
	case code >= 400 && code < 500:
		return StatusFail
	}

	return StatusSuccess
}
