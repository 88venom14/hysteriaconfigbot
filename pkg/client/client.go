package client

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"hysconfigbot/pkg/consts"

	"golang.org/x/net/proxy"
)

func NewHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: consts.HTTPTimeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   consts.TLSHandshakeTimeout,
			ResponseHeaderTimeout: consts.ResponseTimeout,
		},
	}

	proxyURL := os.Getenv("HTTPS_PROXY")
	if proxyURL == "" {
		proxyURL = os.Getenv("HTTP_PROXY")
	}

	if proxyURL != "" {
		proxyParsedURL, err := url.Parse(proxyURL)
		if err != nil {
			log.Printf("[WARN] Invalid proxy URL: %v", err)
			return client
		}

		client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyParsedURL)
		log.Printf("[INFO] Using HTTP proxy: %s", proxyURL)
		return client
	}

	socks5Proxy := os.Getenv("SOCKS5_PROXY")
	if socks5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			log.Printf("[WARN] Failed to create SOCKS5 dialer: %v", err)
			return client
		}

		client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		log.Printf("[INFO] Using SOCKS5 proxy: %s", socks5Proxy)
		return client
	}

	return client
}
