package tracestate_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	. "github.com/lightstep/tracecontext.go/tracestate"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	allowedVendorChars        = "abcdefghijklmnopqrstuvwxyz0123456789_-*/"
	allowedTenantChars        = allowedVendorChars
	allowedValueNonBlankChars = "!\"#$%&'()*+-./0123456789:;<>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	allowedValueChars         = allowedValueNonBlankChars + " "
)

func TestTracestate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracestate Suite")
}

var _ = Describe(".Parse", func() {
	testParsing(func(headers string) (TraceState, error) {
		return Parse([]byte(headers))
	})
})

func testParsing(parse func(string) (TraceState, error)) {
	It("allows up to 32 valid, non-empty list members", func() {
		quick.Check(func(testMembers []TestMember) bool {
			var expectedMembers []Member
			vendors := make(map[string]interface{})

			var memberStrings []string

			for _, tm := range testMembers {
				if len(expectedMembers) > 32 {
					break
				}

				tm.Clean()
				if tm.Vendor == "" || tm.Value == "" {
					continue
				}

				vendorKey := fmt.Sprintf("%s@%s", tm.Vendor, tm.Tenant)
				if _, ok := vendors[vendorKey]; ok {
					continue
				}
				vendors[vendorKey] = nil

				expectedMembers = append(expectedMembers, Member{
					Vendor: tm.Vendor,
					Tenant: tm.Tenant,
					Value:  tm.Value,
				})
				memberStrings = append(memberStrings, tm.String())

				if len(expectedMembers) == 32 {
					break
				}
			}

			ts, err := parse(strings.Join(memberStrings, ","))
			Expect(err).NotTo(HaveOccurred())
			Expect(ts).To(ConsistOf(expectedMembers))

			return true
		}, nil)
	})
}

type TestMember struct {
	Vendor string
	Tenant string
	Value  string
}

func (m TestMember) String() string {
	if m.Tenant == "" {
		return fmt.Sprintf("%s=%s", m.Vendor, m.Value)
	}
	return fmt.Sprintf("%s@%s=%s", m.Vendor, m.Tenant, m.Value)
}

func (m *TestMember) Clean() {
	if m.Tenant == "" {
		if len(m.Vendor) > 256 {
			m.Vendor = m.Vendor[:256]
		}
	} else {
		if len(m.Vendor) > 241 {
			m.Vendor = m.Vendor[:241]
		}
		if len(m.Tenant) > 14 {
			m.Tenant = m.Tenant[:14]
		}
	}

	if len(m.Value) > 256 {
		m.Value = m.Value[:256]
	}
}

func (m TestMember) Generate(rg *rand.Rand, size int) reflect.Value {
	vendor := make([]byte, size)
	var tenant []byte
	hasTenant := rg.Int()%1 == 0
	if hasTenant {
		tenant = make([]byte, size)
	}
	value := make([]byte, size)

	for i := 0; i < size; i++ {
		vendor[i] = randChar(rg, allowedVendorChars)
		if hasTenant {
			tenant[i] = randChar(rg, allowedTenantChars)
		}
		if i == size-1 {
			value[i] = randChar(rg, allowedValueNonBlankChars)
		} else {
			value[i] = randChar(rg, allowedValueChars)
		}
	}

	v := reflect.New(reflect.TypeOf(m)).Elem()
	v.Field(0).SetString(string(vendor))
	v.Field(1).SetString(string(tenant))
	v.Field(2).SetString(string(value))

	return v
}

func randChar(rg *rand.Rand, allowedChars string) byte {
	return allowedChars[rg.Intn(len(allowedChars))]
}
