# echo-plugin-for-coredns
A CoreDNS plugin that encoded received DNS queries and sends it back within a TXT record in the response. The sender's IP address and the transport protocol (TCP/UDP) over that the request reached the name server are also featured in the response. 

# Funcionality explained
For general details, how to develop custom CoreDNS plugins, pleasre refer to [this article](https://coredns.io/2016/12/19/writing-plugins-for-coredns/).


## Retrieving the Transport Protocol
To save the correct protocol in the context variable however, some small changes in the original CoreDNS code are necessary (in _core/dnsserver/server.go_). As a reference, the changed file is added to the repository and the changed are highlighted. 
To realize the retieval of the transport protocol, the module _util.go_ needs to be added to the go source at first (usually located in _/usr/local/go/src_). server.go uses it to add the respective protocol to the context variable in its **Serve** and **ServePacket** function.

### Serve
~~~
s.server[tcp] = &dns.Server{Listener: l, Net: "tcp", MsgAcceptFunc: MyMsgAcceptFunc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		...
		ctx = context.WithValue(ctx, util.CtxKey{}, "TCP")
		s.ServeDNS(ctx, w, r)
	})}

~~~

### ServePacket
~~~
s.server[udp] = &dns.Server{PacketConn: p, Net: "udp", MsgAcceptFunc: **MyMsgAcceptFunc**, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		...
		ctx = context.WithValue(ctx, util.CtxKey{}, "UDP")
		s.ServeDNS(ctx, w, r)
	})}
~~~
Note that we have notice during tests that some requests using specific EDNS(0) Options were blocked by CoreDNS. Therefore, an own _MsgAcceptFunc_, **MyMsgAcceptFunc** was introduced to leave all request through to the plugin. In the **ServeDNS** (see echo.go) function, _echo_ firstly retrieves the transport protocol the request was sent over using the **context-variable (ctx)** passed as parameter: 

~~~
protocol, _ := util.GetProtocolFromContext(ctx)
~~~


## Retreiving Request Data 
The remaining data from the incoming request is as well taken from **ctx**: 
~~~
data := getData(ctx, state)

msg := new(dns.Msg)
msg.SetReply(r)
msg.Authoritative = true
msg.Rcode = dns.RcodeSuccess

rr, err := assembleRR(data, protocol)
~~~

The body of the response message is filled by Resource Record returned from **assembleRR**.

**assbemleRR** takes the sender's IP (**data.Remote**) and the DNS message (**data.Message**), encodes all information and creates a TXT record containing the encoding
~~~
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
~~~

An example for a DNS message (sent over UDP, from localhost) and its encoding: 
**Original**
~~~~
;; opcode: QUERY, status: NOERROR, id: 48286 
;; flags: rd ad; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 1

;;OPT PSEUDOSECTION:
; EDNS: version 0, flags:; udp: 4096
; COOKIE: 29e4df0bf7cfe3b2
;;QUESTION SECTION:
membrain-it.technology.		IN		A
~~~~
**Encoding**
~~~~
FROM_127.0.0.1 
Protocol_UDP 
opcode:&QUERY,&status:&NOERROR,&id:&48286$
flags:&rd&ad%&QUERY:&1,&ANSWER:&0,&AUTHORITY:&0,&ADDITIONAL:&1$$
OPT&PSEUDOSECTION:$%&EDNS:&version&0%&flags:&%&udp:&4096$
%&COOKIE:&29e4df0bf7cfe3b
QUESTION&SECTION:$%membrain-it.technology.?IN?&A$
~~~~


the used transport protocol, we need to make some small adjustments in the original CoreDNS code in . When requests over one protocol are received, this information is stored using context-variables. As a reference, the file has been added to the repository of echo and the respective changes have been marked. It is also necessary to add the the module util.go to the go-source /usr/local/go/src. As we have furthermore observed during tests that DNS messages containing specific EDNS options (like TCP Keepalive) were blocked by the CoreDNS server, we additionally added a custom MsgAcceptFunc to leave all requests through to the plugins. It can also be found the repository and needs to be added to the CoreDNS folder coredns/core/dnsserver/. Some additional changes in server.go are again highlighted.
To be able to run the plugins, their folders need to be pasted into coredns/plugin and their names registered in coredns/plugin.cfg (also listed in the repository). Finally, the usage of either of the plugins needs to be configured in the Corefile. An example can be found in each repository. After building, the respective name servers running the plugins can be deployed using systemd following the instructions given in [111].
