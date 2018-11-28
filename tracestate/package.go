package tracestate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxMembers = 32
)

var (
	ErrInvalidListMember      = errors.New("tracecontext: Invalid tracestate list member")
	ErrDuplicateListMemberKey = errors.New("tracecontext: Duplicate list member key in tracestate")
	ErrTooManyListMembers     = errors.New("tracecontext: Too many list members in tracestate")
)

const (
	delimiter = ","
)

var (
	re = regexp.MustCompile(`^\s*(?:([a-z0-9_\-*/]{1,241})@([a-z0-9_\-*/]{1,14})|([a-z0-9_\-*/]{1,256}))=([\x20-\x2b\x2d-\x3c\x3e-\x7e]*[\x21-\x2b\x2d-\x3c\x3e-\x7e])\s*$`)
)

type Member struct {
	Vendor string
	Tenant string
	Value  string
}

func (m Member) String() string {
	if m.Tenant == "" {
		return fmt.Sprintf("%s=%s", m.Vendor, m.Value)
	}
	return fmt.Sprintf("%s@%s=%s", m.Vendor, m.Tenant, m.Value)
}

type TraceState []Member

func (ts TraceState) String() string {
	var members []string
	for _, member := range ts {
		members = append(members, member.String())
	}
	return strings.Join(members, ",")
}

func Parse(traceState []byte) (TraceState, error) {
	return parse(string(traceState))
}

func ParseString(traceState string) (TraceState, error) {
	return parse(traceState)
}

func parse(traceState string) (ts TraceState, err error) {
	found := make(map[string]interface{})

	members := strings.Split(traceState, delimiter)

	for _, member := range members {
		if len(member) == 0 {
			continue
		}

		var m Member
		m, err = parseMember(member)
		if err != nil {
			return
		}

		key := fmt.Sprintf("%s%s", m.Vendor, m.Tenant)
		if _, ok := found[key]; ok {
			err = ErrDuplicateListMemberKey
			return
		}
		found[key] = nil

		ts = append(ts, m)

		if len(ts) > MaxMembers {
			err = ErrTooManyListMembers
			return
		}
	}

	return
}

func parseMember(s string) (Member, error) {
	matches := re.FindStringSubmatch(s)
	if len(matches) != 5 {
		return Member{}, ErrInvalidListMember
	}

	vendor := matches[1]
	if vendor == "" {
		vendor = matches[3]
	}

	return Member{
		Vendor: vendor,
		Tenant: matches[2],
		Value:  matches[4],
	}, nil
}
