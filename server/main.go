package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	tracecontext "github.com/lightstep/tracecontext.go"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	client := http.DefaultClient

	handler := func(w http.ResponseWriter, req *http.Request) {
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		var body []testRequest
		if err = json.Unmarshal(b, &body); err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		var tc tracecontext.TraceContext
		if tc, err = tracecontext.FromHeader(req.Header); err != nil {
			for i := 0; i < len(tc.TraceParent.TraceID); i++ {
				tc.TraceParent.TraceID[i] = byte(rand.Intn(255))
			}
			for i := 0; i < len(tc.TraceParent.SpanID); i++ {
				tc.TraceParent.SpanID[i] = byte(rand.Intn(255))
			}
		}

		for _, item := range body {
			u, err := url.Parse(item.URL)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}

			b, err := json.Marshal(item.Arguments)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}

			r := &http.Request{
				URL:    u,
				Method: "POST",
				Body:   ioutil.NopCloser(bytes.NewBuffer(b)),
				Header: make(http.Header),
			}

			for i := 0; i < len(tc.TraceParent.SpanID); i++ {
				tc.TraceParent.SpanID[i] = byte(rand.Intn(255))
			}

			tc.SetHeaders(r.Header)

			if _, err = client.Do(r); err != nil {
				w.Write([]byte(err.Error()))
				return
			}
		}
	}

	var port = os.Getenv("PORT")
	if port == "" {
		port = "4567"
	}

	http.HandleFunc("/test", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

type testRequest struct {
	URL       string        `json:"url"`
	Arguments []interface{} `json:"arguments"`
}
