package echo

import (
	"fmt"
	"strings"
	"context"


	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/request"
	"util"
	"github.com/miekg/dns"
)

// Handler is a plugin handler that takes a query and templates a response.
type Handler struct {
	Zones []string
	Next      plugin.Handler
}


type queryData struct {
	Name     string
	Remote	 string
	Message  *dns.Msg
	md       map[string]metadata.Func
}

func (data *queryData) Meta(metaName string) string {
	if data.md == nil {
		return ""
	}

	if f, ok := data.md[metaName]; ok {
		return f()
	}

	return ""
}

// ServeDNS implements the plugin.Handler interface.
func (h Handler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	protocol, _ := util.GetProtocolFromContext(ctx)
        fmt.Println("DNS Request over", protocol)

	//if zone == "" {
	//	fmt.Println("Aborting")
//		return plugin.NextOrFailure(h.Name(), h.Next, ctx, w, r)
//	}


	data := getData(ctx, state)
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.Rcode = dns.RcodeSuccess

	rr, err := assembleRR(data, protocol)
	if err != nil {
	   return dns.RcodeServerFailure, err
	}

	msg.Answer = append(msg.Answer, rr)

	w.WriteMsg(msg)
	return dns.RcodeSuccess, nil

}

// Name implements the plugin.Handler interface.
func (h Handler) Name() string { return "echo" }

func assembleRR(data *queryData, protocol string) (dns.RR, error) {
	from_str := fmt.Sprintf("FROM_%s Protocol_%s", data.Remote, protocol)

	buffer_str := strings.Replace(data.Message.String(), "\n", "$", -1)
	buffer_str = strings.Replace(buffer_str, " ", "&", -1)
	buffer_str = strings.Replace(buffer_str, ";;", " ", -1)
	buffer_str = strings.Replace(buffer_str, ";", "%", -1)
	buffer_str = strings.Replace(buffer_str, "\t", "?", -1)
	buffer_str = strings.Replace(buffer_str, " &", " ", -1)
	buffer_str = strings.Replace(buffer_str, "& ", " ", -1)
	str := fmt.Sprintf("%s IN TXT %s %s", data.Name, from_str, buffer_str)

	rr, err := dns.NewRR(str)
	if err != nil {
	   return rr, err
	}
	return rr, nil
}

func getData(ctx context.Context, state request.Request) (*queryData) {
	data := &queryData{md: metadata.ValueFuncs(ctx), Remote: state.IP()}
	data.Name = state.Name()
        data.Message = state.Req
	return data
}
