package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/miekg/dns"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
)

const zoneTemplate = `resource "aws_route53_zone" "{{ .Id }}" {
   name = "{{ .Domain }}"
}
`
const recordTemplate = `{{- range .Record.Comments }}
# {{ . }}{{ end }}
resource "aws_route53_record" "{{ .ResourceId }}" {
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
	Name     string
	Type     string
	Ttl      uint32
	Data     []string
	Comments []string
}
type RecordKey struct {
	Name string
	Type string
}
type RecordKeySlice []RecordKey

func (records RecordKeySlice) Len() int {
	return len(records)
}
func (records RecordKeySlice) Less(i, j int) bool {
	genKey := func(k RecordKey) string {
		return fmt.Sprintf("%s-%s", k.Name, k.Type)
	}
	return genKey(records[i]) < genKey(records[j])
}
func (records RecordKeySlice) Swap(i, j int) {
	tmp := records[i]
	records[i] = records[j]
	records[j] = tmp
}

var (
	excludedTypesRaw = flag.String("exclude", "SOA,NS", "Comma-separated list of record types to ignore")
	excludedTypes    map[string]bool
	domain           = flag.String("domain", "", "Name of domain")
	zoneFile         = flag.String("zone-file", "", "Path to zone file. Defaults to <domain>.zone in working dir")
	showVersion      = flag.Bool("version", false, "Show version")
)

func init() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("bzfttr53rdutil %s (%s/%s) (%s on %s)", Version, Branch, Revision, BuildUser, BuildDate)
		os.Exit(0)
	}

	if *domain == "" {
		log.Fatal("Domain is required")
	}
	if *zoneFile == "" {
		*zoneFile = fmt.Sprintf("%s.zone", *domain)
	}

	excludedTypes = make(map[string]bool)
	for _, t := range strings.Split(*excludedTypesRaw, ",") {
		excludedTypes[strings.ToUpper(t)] = true
	}
}

func main() {
	bytes, err := ioutil.ReadFile(*zoneFile)
	if err != nil {
		log.Panic(err)
	}
	zone := string(bytes)
	reader := strings.NewReader(zone)

	records := make(map[RecordKey]*TerraformRecord)
	for rr := range dns.ParseZone(reader, *domain, *zoneFile) {
		if rr.Error != nil {
			log.Printf("Error: %v\n", rr.Error)
		} else {
			header := rr.Header()
			recordType := dns.Type(header.Rrtype).String()
			isExcluded, ok := excludedTypes[recordType]

			if ok && isExcluded {
				continue
			}

			name := strings.ToLower(header.Name)
			data := strings.TrimPrefix(rr.String(), header.String())
			if recordType == "CNAME" {
				data = strings.ToLower(data)
			}

			key := RecordKey{
				Name: name,
				Type: recordType,
			}
			if rec, ok := records[key]; ok {
				rec.Data = append(rec.Data, data)
				if rr.Comment != "" {
					rec.Comments = append(rec.Comments, strings.TrimLeft(rr.Comment, ";"))
				}
			} else {
				comments := make([]string, 0)
				if rr.Comment != "" {
					comments = append(comments, strings.TrimLeft(rr.Comment, ";"))
				}
				records[key] = &TerraformRecord{
					Name:     key.Name,
					Type:     key.Type,
					Ttl:      header.Ttl,
					Data:     []string{data},
					Comments: comments,
				}
			}
		}
	}

	zoneName := strings.TrimRight(*domain, ".")
	ZoneTemplateData := ZoneTemplateData{
		Id:     strings.Replace(zoneName, ".", "-", -1),
		Domain: zoneName,
	}
	terraformZone := template.Must(template.New("zone").Parse(zoneTemplate))
	terraformZone.Execute(os.Stdout, ZoneTemplateData)

	resource := template.Must(template.New("resource").Funcs(template.FuncMap{"ensureQuoted": ensureQuoted}).Parse(recordTemplate))
	recordKeys := make(RecordKeySlice, 0, len(records))
	for key, _ := range records {
		recordKeys = append(recordKeys, key)
	}
	sort.Sort(sort.Reverse(recordKeys))

	for _, key := range recordKeys {
		rec := records[key]
		hyphenatedName := strings.Replace(strings.TrimRight(rec.Name, "."), ".", "-", -1)
		wildcardCleanedName := strings.Replace(hyphenatedName, "*", "wildcard", -1)
		id := fmt.Sprintf("%s-%s", wildcardCleanedName, rec.Type)
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
