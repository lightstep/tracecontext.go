package traceparent

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
)

const (
	Version    = 0
	MaxVersion = 254
)

var (
	ErrInvalidFormat  = errors.New("tracecontext: Invalid traceparent format")
	ErrInvalidVersion = errors.New("tracecontext: Invalid traceparent version")
	ErrInvalidTraceID = errors.New("tracecontext: Invalid traceparent trace ID")
	ErrInvalidSpanID  = errors.New("tracecontext: Invalid traceparent span ID")
)

const (
	numVersionBytes = 1
	numTraceIDBytes = 16
	numSpanIDBytes  = 8
	numFlagBytes    = 1
)

var (
	re = regexp.MustCompile(`^([a-f0-9]{2})-([a-f0-9]{32})-([a-f0-9]{16})-([a-f0-9]{2})(-.*)?$`)

	invalidTraceIDAllZeroes = make([]byte, numTraceIDBytes, numTraceIDBytes)
	invalidSpanIDAllZeroes  = make([]byte, numSpanIDBytes, numSpanIDBytes)
)

type Flags struct {
	Recorded bool
}

func (f Flags) String() string {
	var flags [1]byte
	if f.Recorded {
		flags[0] = 1
	}
	return fmt.Sprintf("%02x", flags)
}

type TraceParent struct {
	Version uint8
	TraceID [16]byte
	SpanID  [8]byte
	Flags   Flags
}

func (tp TraceParent) String() string {
	return fmt.Sprintf("%02x-%032x-%016x-%s", tp.Version, tp.TraceID, tp.SpanID, tp.Flags)
}

func Parse(b []byte) (TraceParent, error) {
	return parse(b)
}

func ParseString(s string) (TraceParent, error) {
	return parse([]byte(s))
}

func parse(b []byte) (tp TraceParent, err error) {
	matches := re.FindSubmatch(b)
	if len(matches) < 6 {
		err = ErrInvalidFormat
		return
	}

	var version uint8
	if version, err = parseVersion(matches[1]); err != nil {
		return
	}
	if version == Version && len(matches[5]) > 0 {
		err = ErrInvalidFormat
		return
	}

	var traceID [16]byte
	if traceID, err = parseTraceID(matches[2]); err != nil {
		return
	}

	var spanID [8]byte
	if spanID, err = parseSpanID(matches[3]); err != nil {
		return
	}

	var flags Flags
	if flags, err = parseFlags(matches[4]); err != nil {
		return
	}

	tp.Version = Version
	tp.TraceID = traceID
	tp.SpanID = spanID
	tp.Flags = flags

	return tp, nil
}

func parseVersion(b []byte) (uint8, error) {
	version, ok := parseEncodedSegment(b, numVersionBytes)
	if !ok {
		return 0, ErrInvalidFormat
	}
	if version[0] > MaxVersion {
		return 0, ErrInvalidVersion
	}
	return version[0], nil
}

func parseTraceID(b []byte) (traceID [16]byte, err error) {
	id, ok := parseEncodedSegment(b, numTraceIDBytes)
	if !ok {
		return traceID, ErrInvalidFormat
	}
	if bytes.Equal(id, invalidTraceIDAllZeroes) {
		return traceID, ErrInvalidTraceID
	}

	copy(traceID[:], id)

	return traceID, nil
}

func parseSpanID(b []byte) (spanID [8]byte, err error) {
	id, ok := parseEncodedSegment(b, numSpanIDBytes)
	if !ok {
		return spanID, ErrInvalidFormat
	}
	if bytes.Equal(id, invalidSpanIDAllZeroes) {
		return spanID, ErrInvalidSpanID
	}

	copy(spanID[:], id)

	return spanID, nil
}

func parseFlags(b []byte) (Flags, error) {
	flags, ok := parseEncodedSegment(b, numFlagBytes)
	if !ok {
		return Flags{}, ErrInvalidFormat
	}

	return Flags{
		Recorded: (flags[0] & 1) == 1,
	}, nil
}

func parseEncodedSegment(src []byte, expectedLen int) ([]byte, bool) {
	dst := make([]byte, hex.DecodedLen(len(src)))
	if n, err := hex.Decode(dst, src); n != expectedLen || err != nil {
		return dst, false
	}
	return dst, true
}
