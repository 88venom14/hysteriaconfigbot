package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hysconfigbot/pkg/consts"
	"io"
	"strings"
	"text/template"
)

// ConfigTemplate — шаблон YAML-конфигурации Hysteria2
const ConfigTemplate = `mixed-port: 7890
allow-lan: true
tcp-concurrent: true
enable-process: true
find-process-mode: always
mode: rule
log-level: debug
ipv6: false
keep-alive-interval: 30
unified-delay: false

profile:
  store-selected: true
  store-fake-ip: true

sniffer:
  enable: true
  force-dns-mapping: true
  parse-pure-ip: true
  sniff:
    HTTP:
      ports: [80, 8080-8880]
      override-destination: true
    TLS:
      ports: [443, 8443]

tun:
  enable: true
  stack: mixed
  auto-route: true
  auto-detect-interface: true
  dns-hijack: ["any:53"]
  strict-route: true
  mtu: 1500

dns:
  enable: true
  prefer-h3: true
  use-hosts: true
  use-system-hosts: true
  listen: 127.0.0.1:6868
  ipv6: false
  enhanced-mode: redir-host
  default-nameserver: [tls://77.88.8.8, 195.208.4.1]
  proxy-server-nameserver: [tls://77.88.8.8, 195.208.4.1]
  direct-nameserver: [tls://77.88.8.8, 195.208.4.1]
  nameserver: [https://cloudflare-dns.com/dns-query]

proxies:
  - name: ⚡️ Hysteria2
    type: hysteria2
    server: {{.Server}}
    port: {{.Port}}
    password: "{{.Password}}"
    sni: {{.Server}}
    skip-cert-verify: false
    up: 0
    down: 0

proxy-groups:
  - name: 🌍 VPN
    icon: https://cdn.jsdelivr.net/gh/Koolson/Qure@master/IconSet/Color/Hijacking.png
    type: select
    proxies: [⚡️ Hysteria2]

rule-providers:
  ru-inline-banned:
    type: http
    url: https://github.com/legiz-ru/mihomo-rule-sets/raw/main/other/inline/ru-inline-banned.yaml
    interval: 86400
    behavior: classical
    format: yaml
    path: ./rule-sets/ru-inline-banned.yaml
  ru-inline:
    type: http
    url: https://github.com/legiz-ru/mihomo-rule-sets/raw/main/other/inline/ru-inline.yaml
    interval: 86400
    behavior: classical
    format: yaml
    path: ./rule-sets/ru-inline.yaml
  ru-banned:
    type: http
    url: https://github.com/legiz-ru/mihomo-rule-sets/raw/main/other/ru-banned.yaml
    interval: 86400
    behavior: classical
    format: yaml
    path: ./rule-sets/ru-banned.yaml

rules:
  - RULE-SET,ru-inline-banned,🌍 VPN
  - PROCESS-NAME,Discord.exe,🌍 VPN
  - PROCESS-NAME,discord,🌍 VPN
  - MATCH,🌍 VPN
`

// ConfigParams параметры для генерации конфига
type ConfigParams struct {
	Server   string
	Port     int
	Password string
}

// GeneratePassword генерирует случайный пароль (32 символа hex)
func GeneratePassword() (string, error) {
	bytes := make([]byte, consts.PasswordByteLength)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateConfig генерирует YAML-конфиг с подстановкой имени, пароля и сервера
func GenerateConfig(userName, password, server string) (string, error) {
	passwordField := fmt.Sprintf("%s:%s", userName, password)

	params := ConfigParams{
		Server:   server,
		Port:     consts.DefaultPort,
		Password: passwordField,
	}

	tmpl, err := template.New("config").Parse(ConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, params); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

// IsValidLatinName проверяет, что имя содержит только латинские буквы и цифры
// Максимальная длина имени — 32 символа
func IsValidLatinName(name string) bool {
	if len(name) == 0 || len(name) > consts.MaxNameLength {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// IsValidServerAddress проверяет формат адреса сервера
// Допускаются доменные имена, IP-адреса и простые имена
func IsValidServerAddress(server string) bool {
	if len(server) == 0 || len(server) > consts.MaxServerLength {
		return false
	}

	// Разрешаем: буквы, цифры, точки, дефисы, подчёркивания
	for _, r := range server {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_') {
			return false
		}
	}

	return true
}
