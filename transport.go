package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// NewTransport returns pre-configured *http.transport
func NewTransport() *http.Transport {
	netDialer := NetDialerOpt()

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           netDialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       10,
		DisableCompression:    false,
		// TLSClientConfig:TLSClientConfigOpt() ,
	}
}

func NetDialerOpt() *net.Dialer {
	resolver := NetResolverOpt()

	return &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver:  resolver,
	}
}

func NetResolverOpt() *net.Resolver {
	return &net.Resolver{
		PreferGo: true, // Using Go's built-in resolver
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 2 * time.Second, // DNS timeout
			}
			return d.DialContext(ctx, network, address)
		},
	}
}

func TLSClientConfigOpt() *tls.Config {
	certFile := "cert.pem"
	KeyFile := "key.pem"
	caFile := "ca.pem" // TODO: Make all of these variable configurable
	clientCert, err := tls.LoadX509KeyPair(certFile, KeyFile)
	if err != nil {
		slog.Error("error occured while loading certificate",
			slog.String("error", err.Error()),
			slog.String("cert_file", certFile),
			slog.String("key_file", KeyFile))
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			clientCert,
		},
	}

	if len(caFile) != 0 {
		cert, err := os.ReadFile(caFile)
		if err != nil {
			slog.Error("unable to load root CA file,", slog.String("ca file", caFile), slog.String("error", err.Error()))
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig
}
