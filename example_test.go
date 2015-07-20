package jsend_test

import (
	"encoding/json"
	"net/http"

	"github.com/gamegos/jsend"
)

func Example() {
	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"id":   1,
			"name": "foo",
		}

		jsend.Success(w, data, 200)
	}))

	/*
		HTTP/1.1 200 OK
		Content-Type: application/json

		{
		  "status": "success",
		  "data": {
		    "id": 1,
		    "name": "foo"
		  }
		}
	*/
}

func Example_wrap() {
	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(map[string]interface{}{
			"id":   "missing",
			"name": "invalid",
		})

		w = jsend.Wrap(w)
		w.WriteHeader(400)
		w.Write(data)
	}))

	/*
	  HTTP/1.1 400 Bad Request
	  Content-Type: application/json

	  {
	      "status": "fail",
	      "data": {
	          "id": "missing",
	          "name": "invalid"
	      }
	  }
	*/
}
