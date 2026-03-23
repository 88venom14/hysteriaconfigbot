package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hysconfigbot/pkg/consts"
	"strings"
	"text/template"
)

// SanitizeFileName удаляет опасные символы из имени файла
// Возвращает безопасное имя для использования в файловой системе
func SanitizeFileName(name string) string {
	// Заменяем потенциально опасные символы
	sanitized := strings.ReplaceAll(name, "/", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")
	sanitized = strings.ReplaceAll(sanitized, "..", "")
	sanitized = strings.ReplaceAll(sanitized, "\x00", "") // null byte

	// Оставляем только безопасные символы: буквы, цифры, дефис, подчёркивание
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_') // Заменяем небезопасные символы на _
		}
	}

	// Гарантируем непустое имя
	if result.Len() == 0 {
		return "config"
	}

	return result.String()
}

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
  default-nameserver: [{{range $i, $dns := .DNSServers}}{{if $i}}, {{end}}{{$dns}}{{end}}]
  proxy-server-nameserver: [{{range $i, $dns := .DNSServers}}{{if $i}}, {{end}}{{$dns}}{{end}}]
  direct-nameserver: [{{range $i, $dns := .DNSServers}}{{if $i}}, {{end}}{{$dns}}{{end}}]
  nameserver: [https://cloudflare-dns.com/dns-query]

proxies:
  - name: ⚡️ Hysteria2
    type: hysteria2
    server: {{.Server}}
    port: {{.Port}}
    password: "{{.Password}}"
    sni: {{.Server}}
    skip-cert-verify: false
    up: {{.Up}}
    down: {{.Down}}

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

type ConfigParams struct {
	Server     string
	Port       int
	Password   string
	Up         int
	Down       int
	DNSServers []string
}

// configTemplate кэширует скомпилированный шаблон
var configTemplate *template.Template

func init() {
	var err error
	configTemplate, err = template.New("config").Parse(ConfigTemplate)
	if err != nil {
		panic(fmt.Sprintf("failed to parse config template: %v", err))
	}
}

func GeneratePassword() (string, error) {
	bytes := make([]byte, consts.PasswordByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateConfig(userName, password, server string, up, down int) (string, error) {
	passwordField := fmt.Sprintf("%s:%s", userName, password)

	params := ConfigParams{
		Server:     server,
		Port:       consts.DefaultPort,
		Password:   passwordField,
		Up:         up,
		Down:       down,
		DNSServers: consts.DefaultDNSServers,
	}

	var builder strings.Builder
	if err := configTemplate.Execute(&builder, params); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

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

func IsValidServerAddress(server string) bool {
	if len(server) == 0 || len(server) > consts.MaxServerLength {
		return false
	}

	// Запретить символы template injection
	if strings.ContainsAny(server, "{}") {
		return false
	}

	// Домен не должен начинаться или заканчиваться на дефис/подчёркивание
	if len(server) > 0 {
		first := server[0]
		last := server[len(server)-1]
		if first == '-' || first == '_' || last == '-' || last == '_' {
			return false
		}
	}

	// Нет двойных точек
	if strings.Contains(server, "..") {
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
