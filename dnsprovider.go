package main

type dnsProvider struct {
	zoneTemplate   string
	recordTemplate string
}

var (
	route53DNS = dnsProvider{
		zoneTemplate: `resource "aws_route53_zone" "{{ .ID }}" {
  name = "{{ .Domain }}"
}
`,
		recordTemplate: `{{- range .Record.Comments }}
# {{ . }}{{ end }}
resource "aws_route53_record" "{{ .ResourceID }}" {
  zone_id = {{ zoneReference .ZoneID }}
  name    = "{{ .Record.Name }}"
  type    = "{{ .Record.Type }}"
  ttl     = "{{ .Record.TTL }}"
  records = [{{ range $idx, $elem := .Record.Data }}{{ if $idx }}, {{ end }}{{ ensureQuoted $elem }}{{ end }}]
}
`,
	}

	cloudDNS = dnsProvider{
		zoneTemplate: `resource "google_dns_managed_zone" "{{ .ID }}" {
  name = "{{ .ID }}"
  dns_name = "{{ .Domain }}"
  visibility = "public"
}`,
		recordTemplate: `{{- range .Record.Comments }}
# {{ . }}{{ end }}
resource "google_dns_record_set" "{{ .ResourceID }}" {
  managed_zone = {{ zoneReference .ZoneID }}
  name    = "{{ .Record.Name }}"
  type    = "{{ .Record.Type }}"
  ttl     = "{{ .Record.TTL }}"
  rrdatas = [{{ range $idx, $elem := .Record.Data }}{{ if $idx }}, {{ end }}{{ ensureQuoted $elem }}{{ end }}]
}
`,
	}
)
