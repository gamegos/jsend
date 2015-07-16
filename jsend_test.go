package jsend

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"
)

type testCaseIn struct {
	status     string
	statusCode int
	data       interface{}
	error      string
}

const jsonContentType = "application/json"

var (
	testBody0 = map[string]interface{}{"foo": "bar", "baz": "qux"}
	testBody1 = map[string]interface{}{"id": "invalid", "dob": "empty"}
	testBody2 = map[string]interface{}{"foos": map[[2]byte]string{
		[2]byte{'2', '3'}: "4",
		[2]byte{'a', 'b'}: "c",
	}}
)

type testCaseOut struct {
	statusCode  int
	contentType string
	body        map[string]interface{}
}

var testCases = []struct {
	in  *testCaseIn
	out *testCaseOut
}{
	{
		&testCaseIn{StatusSuccess, 200, testBody0, ""},
		&testCaseOut{200, jsonContentType, map[string]interface{}{
			"status": StatusSuccess,
			"data":   testBody0,
		}},
	},
	{
		&testCaseIn{StatusFail, 400, testBody1, ""},
		&testCaseOut{400, jsonContentType, map[string]interface{}{
			"status": StatusFail,
			"data":   testBody1,
		}},
	},
	{
		&testCaseIn{StatusError, 500, nil, "something wrong"},
		&testCaseOut{500, jsonContentType, map[string]interface{}{
			"status":  StatusError,
			"message": "something wrong",
		}},
	},

	{
		&testCaseIn{StatusSuccess, 200, testBody2, ""},
		&testCaseOut{200, "", nil},
	},
}

func compare(t *testing.T, label string, rw *httptest.ResponseRecorder, out *testCaseOut) {
	if rw.Header().Get("Content-Type") != out.contentType {
		t.Errorf("%s: Content-Type: have: %q, want: %q", label, rw.Header().Get("Content-Type"), out.contentType)
	}

	if rw.Code != out.statusCode {
		t.Errorf("%s: statusCode: have: %d, want: %d", label, rw.Code, out.statusCode)
	}

	var body interface{}
	json.Unmarshal(rw.Body.Bytes(), &body)

	if rw.Body.Len() != 0 && out.body != nil && !reflect.DeepEqual(body, out.body) {
		t.Errorf("%s: body: have: %#v, want: %#v", label, body, out.body)
	}
}

func TestWrite(t *testing.T) {
	for _, tt := range testCases {
		rw := httptest.NewRecorder()
		write(rw, tt.in.status, tt.in.statusCode, tt.in.data, tt.in.error)
		compare(t, "write", rw, tt.out)
	}
}

func TestSuccess(t *testing.T) {
	for _, tt := range testCases {
		if tt.in.status != StatusSuccess {
			continue
		}
		rw := httptest.NewRecorder()
		Success(rw, tt.in.data, tt.in.statusCode)
		compare(t, "Success", rw, tt.out)
	}
}

func TestFail(t *testing.T) {
	for _, tt := range testCases {
		if tt.in.status != StatusFail {
			continue
		}
		rw := httptest.NewRecorder()
		Fail(rw, tt.in.data, tt.in.statusCode)
		compare(t, "Fail", rw, tt.out)
	}
}

func TestError(t *testing.T) {
	for _, tt := range testCases {
		if tt.in.status != StatusError {
			continue
		}
		rw := httptest.NewRecorder()
		Error(rw, tt.in.error, tt.in.statusCode)
		compare(t, "Error", rw, tt.out)
	}
}

func TestWriteJSONResponse(t *testing.T) {
	res := &jsonResponse{Status: StatusSuccess, Data: []byte{'"'}}
	rw := httptest.NewRecorder()
	n, err := writeJSONResponse(rw, res)
	if n != 0 || err != ErrInvalidRawJSON {
		t.Errorf("writeJSONResponse(%q): have: (%d, %q), want: (%d, %q)", res, n, err, 0, ErrInvalidRawJSON)
	}
}

type writeIn struct {
	statusCode  int
	data        string
	contentType string
}

type writeOut struct {
	statusCode  int
	body        string
	err         error
	contentType string
}

var wrapTestCases = []struct {
	in  *writeIn
	out *writeOut
}{
	{
		&writeIn{200, `{"foo":"bar"}`, ""},
		&writeOut{200, `{"status":"success","data":{"foo":"bar"}}`, nil, jsonContentType},
	},
	{
		&writeIn{400, `{"foo":"bar"}`, ""},
		&writeOut{400, `{"status":"fail","data":{"foo":"bar"}}`, nil, jsonContentType},
	},
	{
		&writeIn{503, `something wrong`, ""},
		&writeOut{503, `{"status":"error","message":"something wrong"}`, nil, jsonContentType},
	},
	{
		&writeIn{200, `some invalid json`, ""},
		&writeOut{200, ``, ErrInvalidRawJSON, jsonContentType},
	},
	{
		&writeIn{200, `"foo"`, "application/foo+json"},
		&writeOut{200, `{"status":"success","data":"foo"}`, nil, "application/foo+json"},
	},
}

func TestWrapWrite(t *testing.T) {
	for _, tt := range wrapTestCases {
		rw := httptest.NewRecorder()
		w := Wrap(rw)

		if tt.in.contentType != "" {
			w.Header().Set("Content-Type", tt.in.contentType)
		}

		w.WriteHeader(tt.in.statusCode)
		n, err := w.Write([]byte(tt.in.data))
		label := fmt.Sprintf("wrap(w).Write(%v)", tt.in)

		if n != len(tt.out.body) {
			t.Errorf("%s: n: have: %d, want: %d", label, n, len(tt.out.body))
		}

		if err != tt.out.err {
			t.Errorf("%s: err: have: %q, want: %q", label, err, tt.out.err)
		}

		if rw.Header().Get("Content-Type") != tt.out.contentType {
			t.Errorf("%s: content-type: have: %q, want: %q", label, rw.Header().Get("Content-Type"), tt.out.contentType)
		}

		if rw.Body.String() != tt.out.body {
			t.Errorf("%s: body: have: %q, want: %q", label, rw.Body.String(), tt.out.body)
		}
	}
}

func TestMultipleWrite(t *testing.T) {
	rw := httptest.NewRecorder()
	w := Wrap(rw)

	_, err0 := w.Write([]byte(`"hello"`))
	if err0 != nil {
		t.Fatalf("MultipleWrite: first write must succeed")
	}
	n1, err1 := w.Write([]byte(`"world"`))
	if n1 != 0 || err1 != ErrWrittenAlready {
		t.Errorf("MultipleWrite: have: (%d, %q), want: (%d, %q)", n1, err1, 0, ErrWrittenAlready)
	}
}
