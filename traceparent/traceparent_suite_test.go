package traceparent_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"testing/quick"

	. "github.com/lightstep/tracecontext.go/traceparent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	version0                = [1]byte{0}
	invalidVersionAboveMax  = [1]byte{255}
	invalidTraceIDAllZeroes [16]byte
	invalidSpanIDAllZeroes  [8]byte
)

func TestTraceparent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Traceparent Suite")
}

var _ = Describe("#String", func() {
	It("returns a correctly formatted string", func() {
		quick.Check(func(version byte, traceID [16]byte, spanID [8]byte, recorded bool) bool {
			var flags [1]byte
			if recorded {
				flags[0] = 1
			}

			tp := TraceParent{
				Version: version,
				TraceID: traceID,
				SpanID:  spanID,
				Flags: Flags{
					Recorded: recorded,
				},
			}
			expected := string(encodeTraceParent([]byte{version}, traceID[:], spanID[:], flags[:]))

			Expect(tp.String()).To(Equal(expected))

			return true
		}, nil)
	})
})

var _ = Describe(".Parse", func() {
	testParsing(func(tp string) (TraceParent, error) {
		return Parse([]byte(tp))
	})
})

var _ = Describe(".ParseString", func() {
	testParsing(func(tp string) (TraceParent, error) {
		return ParseString(tp)
	})
})

func testParsing(parse func(string) (TraceParent, error)) {
	It("parses a valid v0 traceparent", func() {
		quick.Check(func(traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid
			if traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}

			tp, err := parse(encodeTraceParent([]byte{0}, traceID[:], spanID[:], flags[:]))
			Expect(err).NotTo(HaveOccurred())

			Expect(tp.Version).To(Equal(uint8(0)))
			Expect(tp.TraceID).To(Equal(traceID))
			Expect(tp.SpanID).To(Equal(spanID))

			recorded := (flags[0] & 1) == 1
			Expect(tp.Flags.Recorded).To(Equal(recorded))

			return true
		}, nil)
	})

	It("does not allow unexpected segments in a v0 traceparent", func() {
		quick.Check(func(traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid
			if traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}

			tpWithValidDelimiter := fmt.Sprintf("%s-extra", encodeTraceParent(version0[:], traceID[:], spanID[:], flags[:]))
			_, err := parse(tpWithValidDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			tpWithInvalidDelimiter := fmt.Sprintf("%s.extra", encodeTraceParent(version0[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithInvalidDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			tpWithNoDelimiter := fmt.Sprintf("%sextra", encodeTraceParent(version0[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithNoDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("ignores extra appended segments in a >v0 traceparent if correctly delimited", func() {
		quick.Check(func(version [1]byte, traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid
			if version == invalidVersionAboveMax || traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}
			// Extra segments aren't allowed in v0
			if version == version0 {
				return true
			}

			tpWithValidDelimiter := fmt.Sprintf("%s-extra", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err := parse(tpWithValidDelimiter)
			Expect(err).NotTo(HaveOccurred())

			tpWithValidDelimiterAndMultipleSegments := fmt.Sprintf("%s-extra-segments", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithValidDelimiterAndMultipleSegments)
			Expect(err).NotTo(HaveOccurred())

			tpWithInvalidDelimiter := fmt.Sprintf("%s.extra", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithInvalidDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			tpWithNoDelimiter := fmt.Sprintf("%sextra", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithNoDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("does not allow any characters before the version", func() {
		quick.Check(func(version [1]byte, traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if version == invalidVersionAboveMax || traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}

			tpWithValidDelimiter := fmt.Sprintf("extra-%s", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err := parse(tpWithValidDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			tpWithInvalidDelimiter := fmt.Sprintf("extra.%s", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithInvalidDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			tpWithNoDelimiter := fmt.Sprintf("extra%s", encodeTraceParent(version[:], traceID[:], spanID[:], flags[:]))
			_, err = parse(tpWithNoDelimiter)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("errors if there is a non-hex encoded segment or an invalid delimiter", func() {
		quick.Check(func(invalidChar byte, version [1]byte, traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if version == invalidVersionAboveMax || traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}
			// Discard if invalidChar is a lowercase hex byte or a valid delimiter
			if (invalidChar >= 48 && invalidChar <= 57) || (invalidChar >= 97 && invalidChar <= 122) || invalidChar == '-' {
				return true
			}

			tp := encodeTraceParent(version[:], traceID[:], spanID[:], flags[:])
			for i := 0; i < len(tp); i++ {
				invalidTP := tp[:i] + string(invalidChar) + tp[i+1:]
				_, err := parse(invalidTP)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))
			}

			return true
		}, nil)
	})

	It("errors if a segment is incorrectly delimited hex-encoded", func() {
		makeInvalidString := func(n int) string {
			var s strings.Builder
			for i := 0; i < n; i++ {
				s.WriteByte('z')
			}
			return s.String()
		}

		quick.Check(func(version [1]byte, traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if version == invalidVersionAboveMax || traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}

			nonHexString := makeInvalidString(len(version) * 2)
			tp := fmt.Sprintf("%s-%s-%s-%s", nonHexString, hex.EncodeToString(traceID[:]), hex.EncodeToString(spanID[:]), hex.EncodeToString(flags[:]))
			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			nonHexString = makeInvalidString(len(traceID) * 2)
			tp = fmt.Sprintf("%s-%s-%s-%s", hex.EncodeToString(version[:]), nonHexString, hex.EncodeToString(spanID[:]), hex.EncodeToString(flags[:]))
			_, err = parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			nonHexString = makeInvalidString(len(spanID) * 2)
			tp = fmt.Sprintf("%s-%s-%s-%s", hex.EncodeToString(version[:]), hex.EncodeToString(traceID[:]), nonHexString, hex.EncodeToString(flags[:]))
			_, err = parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			nonHexString = makeInvalidString(len(flags) * 2)
			tp = fmt.Sprintf("%s-%s-%s-%s", hex.EncodeToString(version[:]), hex.EncodeToString(traceID[:]), hex.EncodeToString(traceID[:]), nonHexString)
			_, err = parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("errors if the version is 255", func() {
		quick.Check(func(traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}

			tp := encodeTraceParent([]byte{255}, traceID[:], spanID[:], flags[:])

			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent version"))

			return true
		}, nil)
	})

	It("errors if the version is not 1 byte", func() {
		quick.Check(func(version []byte, traceID [16]byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}
			// Discard if the version length is valid
			if len(version) == 1 {
				return true
			}

			tp := encodeTraceParent(version[:], traceID[:], spanID[:], flags[:])

			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("errors if the trace ID is not 16 bytes", func() {
		quick.Check(func(version [1]byte, traceID []byte, spanID [8]byte, flags [1]byte) bool {
			// Discard example if it's invalid for other reasons
			if version == invalidVersionAboveMax || spanID == invalidSpanIDAllZeroes {
				return true
			}
			// Discard if the trace ID length is valid
			if len(traceID) == 16 {
				return true
			}

			tp := encodeTraceParent(version[:], traceID, spanID[:], flags[:])

			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("errors if the span ID is not 8 bytes", func() {
		quick.Check(func(version [1]byte, traceID [16]byte, spanID []byte, flags []byte) bool {
			// Discard example if it's invalid for other reasons
			if version == invalidVersionAboveMax || traceID == invalidTraceIDAllZeroes {
				return true
			}
			// Discard if the span ID length is valid
			if len(spanID) == 8 {
				return true
			}

			tp := encodeTraceParent(version[:], traceID[:], spanID, flags[:])

			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})

	It("errors if the flags are not 8 bits", func() {
		quick.Check(func(version [1]byte, traceID [16]byte, spanID [8]byte, flags []byte) bool {
			// Discard example if it's invalid for other reasons
			if traceID == invalidTraceIDAllZeroes || spanID == invalidSpanIDAllZeroes {
				return true
			}
			// Discard if the span ID length is valid
			if len(flags) == 1 {
				return true
			}

			tp := encodeTraceParent(version[:], traceID[:], spanID[:], flags)

			_, err := parse(tp)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("tracecontext: Invalid traceparent format"))

			return true
		}, nil)
	})
}

func encodeTraceParent(version []byte, traceID []byte, spanID []byte, flags []byte) string {
	return fmt.Sprintf("%s-%s-%s-%s", hex.EncodeToString(version), hex.EncodeToString(traceID), hex.EncodeToString(spanID), hex.EncodeToString(flags))
}
