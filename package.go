package tracecontext

import (
	"errors"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/lightstep/tracecontext.go/traceparent"
	"github.com/lightstep/tracecontext.go/tracestate"
)

const (
	Version = traceparent.Version
)

var (
	ErrInvalidHeadersMultipleTraceParent = errors.New("tracecontext: Multiple traceparent headers")
)

var (
	traceParentHeader = textproto.CanonicalMIMEHeaderKey("traceparent")
	traceStateHeader  = textproto.CanonicalMIMEHeaderKey("tracestate")
)

type TraceContext struct {
	TraceParent traceparent.TraceParent
	TraceState  tracestate.TraceState
}

func FromHeaders(headers http.Header) (TraceContext, error) {
	var tc TraceContext

	h := textproto.MIMEHeader(headers)
	if len(h[traceParentHeader]) > 1 {
		return tc, ErrInvalidHeadersMultipleTraceParent
	}

	var err error
	if tc.TraceParent, err = traceparent.ParseString(h.Get(traceParentHeader)); err != nil {
		return tc, err
	}

	var traceStates []string
	for _, traceState := range h[traceStateHeader] {
		traceStates = append(traceStates, traceState)
	}

	traceState, err := tracestate.ParseString(strings.Join(traceStates, ","))
	if err == nil {
		tc.TraceState = traceState
	}

	return tc, nil
}

func (tc TraceContext) SetHeaders(headers http.Header) {
	headers.Set(traceParentHeader, tc.TraceParent.String())
	headers.Set(traceStateHeader, tc.TraceState.String())
}
