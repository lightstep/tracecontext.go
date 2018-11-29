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
	// Version represents the maximum header version supported.
	// The library attempts optimistic forwards compatibility with higher versions.
	Version = traceparent.Version
)

var (
	// ErrInvalidHeadersMultipleTraceParent occurs when there are multiple `traceparent` headers present.
	ErrInvalidHeadersMultipleTraceParent = errors.New("tracecontext: Multiple traceparent headers")
)

var (
	traceParentHeader = textproto.CanonicalMIMEHeaderKey("traceparent")
	traceStateHeader  = textproto.CanonicalMIMEHeaderKey("tracestate")
)

// TraceContext represents a paired TraceParent and TraceState that are intended to be propagated together.
type TraceContext struct {
	TraceParent traceparent.TraceParent
	TraceState  tracestate.TraceState
}

// FromHeaders attempts to parse a TraceContext from a set of headers.
// The returned `TraceContext` value should be considered valid so long as no error error is returned.
//
// It is considered an error for the `traceparent` header to be invalid, but not for the `tracestate` header(s) to be invalid.
// If the `traceparent` header is valid and `tracestate` is not, a `TraceContext` with an empty `TraceState` will still be returned.
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

// SetHeaders sets the `traceparent` and `tracestate` headers based on the `TraceContext`'s fields.
func (tc TraceContext) SetHeaders(headers http.Header) {
	headers.Set(traceParentHeader, tc.TraceParent.String())
	headers.Set(traceStateHeader, tc.TraceState.String())
}
