package newdns

import (
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertResolverAnswer(t *testing.T, ret *dns.Msg) {
	require.NotNil(t, ret)
	assert.True(t, ret.Response)
	assert.True(t, ret.RecursionDesired)
	assert.True(t, ret.RecursionAvailable)
	require.Len(t, ret.Question, 1)
	assert.Equal(t, "example.newdns.256dpi.com.", ret.Question[0].Name)
	assert.Equal(t, uint16(dns.TypeA), ret.Question[0].Qtype)
	assert.Equal(t, uint16(dns.ClassINET), ret.Question[0].Qclass)
	require.GreaterOrEqual(t, len(ret.Answer), 2)
	hasCNAME := false
	hasA := false
	for _, rr := range ret.Answer {
		switch v := rr.(type) {
		case *dns.CNAME:
			if v.Hdr.Name == "example.newdns.256dpi.com." && v.Target == "example.com." {
				hasCNAME = true
			}
		case *dns.A:
			if v.Hdr.Name == "example.com." {
				hasA = true
			}
		}
	}
	assert.True(t, hasCNAME)
	assert.True(t, hasA)
}

func TestResolver(t *testing.T) {
	ret, err := Query("tcp", "1.1.1.1:53", "example.newdns.256dpi.com.", "A", func(msg *dns.Msg) {
		msg.RecursionDesired = true
	})
	if isIOError(err) {
		t.Skipf("skip resolver test due network issue: %v", err)
	}
	require.NoError(t, err)
	assertResolverAnswer(t, ret)

	addr := "0.0.0.0:53002"
	mux := dns.NewServeMux()
	mux.Handle("newdns.256dpi.com", Proxy(awsNS[0]+":53", nil))
	mux.Handle("example.com", Proxy("a.iana-servers.net:53", nil))
	handler := Resolver(mux)

	serve(handler, addr, func() {
		ret, err := Query("udp", addr, "example.newdns.256dpi.com.", "A", func(msg *dns.Msg) {
			msg.RecursionDesired = true
		})
		if isIOError(err) {
			t.Skipf("skip resolver proxy test due network issue: %v", err)
		}
		require.NoError(t, err)
		assertResolverAnswer(t, ret)
	})
}
