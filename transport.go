package main

import (
	"net"
	"net/http"
	"strconv"
	"time"
)

type CustomTransport struct {
	*http.Transport
	shardId int
}

func NewCustomTransport(shardId int) *CustomTransport {
	return &CustomTransport{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       10,
			DisableCompression:    false,
		},
		shardId: shardId,
	}
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Proxy-Shard-Id", strconv.Itoa(t.shardId))
	return t.Transport.RoundTrip(req)
}
