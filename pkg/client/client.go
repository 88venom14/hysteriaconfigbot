package client

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"hysconfigbot/pkg/consts"

	"golang.org/x/net/proxy"
)

// isValidProxyAddress проверяет, что прокси не указывает на внутренний адрес
// Разрешаем localhost (127.0.0.1) для локальных VPN-клиентов (NekoBox, Clash и т.д.)
func isValidProxyAddress(addr string) bool {
	// Разрешаем localhost для локальных прокси-клиентов
	if strings.HasPrefix(addr, "127.0.0.1:") ||
		strings.HasPrefix(addr, "localhost:") ||
		addr == "127.0.0.1" || addr == "localhost" {
		return true
	}

	// Блокируем private IP
	forbidden := []string{
		"10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
		"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
		"169.254.", "::1", "fe80:", "0.0.0.0",
	}
	addrLower := strings.ToLower(addr)
	for _, prefix := range forbidden {
		if strings.HasPrefix(addrLower, prefix) {
			return false
		}
	}
	return true
}

// sanitizeProxyURL скрывает учётные данные прокси для логирования
func sanitizeProxyURL(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "[INVALID_URL]"
	}
	if parsed.User != nil {
		if _, ok := parsed.User.Password(); ok {
			parsed.User = url.UserPassword(parsed.User.Username(), "***")
		}
	}
	return parsed.String()
}

func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		TLSHandshakeTimeout:   consts.TLSHandshakeTimeout,
		ResponseHeaderTimeout: consts.ResponseTimeout,
	}

	client := &http.Client{
		Timeout:   consts.HTTPTimeout,
		Transport: transport,
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
		transport.Proxy = http.ProxyURL(proxyParsedURL)
		log.Printf("[INFO] Using HTTP proxy: %s", sanitizeProxyURL(proxyURL))
		return client
	}

	socks5Proxy := os.Getenv("SOCKS5_PROXY")
	if socks5Proxy != "" {
		if !isValidProxyAddress(socks5Proxy) {
			log.Printf("[WARN] SOCKS5 proxy points to internal address: %s", socks5Proxy)
			return client
		}

		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			log.Printf("[WARN] Failed to create SOCKS5 dialer: %v", err)
			return client
		}

		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		log.Printf("[INFO] Using SOCKS5 proxy: %s", sanitizeProxyURL("socks5://"+socks5Proxy))
		return client
	}

	return client
}
