package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/miekg/dns"
)

const zoneTemplate = `resource "aws_route53_zone" "{{ .Id }}" {
   name = "{{ .Domain }}"
}
`
const recordTemplate = `resource "aws_route53_record" "{{ .ResourceId }}" {
   zone_id = "${aws_route53_zone.{{ .Zone.Id }}.zone_id}"
   name = "{{ .Record.Name }}"
   type = "{{ .Record.Type }}"
   ttl = "{{ .Record.Ttl }}"
   records = [{{ range $idx, $elem := .Record.Data }}{{ if $idx }},{{ end }}{{ ensureQuoted $elem }}{{ end }}]
}
`

type ZoneTemplateData struct {
	Id     string
	Domain string
}
type RecordTemplateData struct {
	ResourceId string
	Record     *TerraformRecord
	Zone       ZoneTemplateData
}
type TerraformRecord struct {
	Name string
	Type string
	Ttl  uint32
	Data []string
}
type RecordKey struct {
	Name string
	Type string
}

func main() {
	domain := os.Args[1]
	filename := fmt.Sprintf("%s.zone", domain)
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panic(err)
	}
	zone := string(bytes)
	reader := strings.NewReader(zone)

	records := make(map[RecordKey]*TerraformRecord)
	for rr := range dns.ParseZone(reader, "", "") {
		if rr.Error != nil {
			log.Printf("Error: %v\n", rr.Error)
		} else {
			header := rr.Header()
			data := strings.TrimPrefix(rr.String(), header.String())
			key := RecordKey{
				Name: header.Name,
				Type: dns.Type(header.Rrtype).String(),
			}
			if rec, ok := records[key]; ok {
				rec.Data = append(rec.Data, data)
			} else {
				records[key] = &TerraformRecord{
					Name: key.Name,
					Type: key.Type,
					Ttl:  header.Ttl,
					Data: []string{data},
				}
			}
		}
	}

	zoneName := strings.TrimRight(domain, ".")
	ZoneTemplateData := ZoneTemplateData{
		Id:     strings.Replace(zoneName, ".", "-", -1),
		Domain: zoneName,
	}
	terraformZone := template.Must(template.New("zone").Parse(zoneTemplate))
	terraformZone.Execute(os.Stdout, ZoneTemplateData)

	resource := template.Must(template.New("resource").Funcs(template.FuncMap{"ensureQuoted": ensureQuoted}).Parse(recordTemplate))
	for _, rec := range records {
		hyphenatedName := strings.Replace(strings.TrimRight(rec.Name, "."), ".", "-", -1)
		id := fmt.Sprintf("%s-%s", hyphenatedName, rec.Type)
		info := RecordTemplateData{
			ResourceId: id,
			Record:     rec,
			Zone:       ZoneTemplateData,
		}
		resource.Execute(os.Stdout, info)
	}
}

func ensureQuoted(s string) string {
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s
	}
	return fmt.Sprintf("%q", s)
}
