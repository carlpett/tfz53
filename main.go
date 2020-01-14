package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/miekg/dns"
	"golang.org/x/net/idna"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Branch    string
	BuildUser string
	BuildDate string
)

const (
	zoneTemplateStr = `resource "aws_route53_zone" "{{ .ID }}" {
  name = "{{ .Domain }}"
}
`
	recordTemplateStr = `{{- range .Record.Comments }}
# {{ . }}{{ end }}
resource "aws_route53_record" "{{ .ResourceID }}" {
  count   = "{{ env_check }}"
  zone_id = {{ zoneReference .ZoneID }}
  name    = "{{ .Record.Name }}"
  type    = "{{ .Record.Type }}"
  ttl     = "{{ .Record.TTL }}"
  records = [{{ range $idx, $elem := .Record.Data }}{{ if $idx }}, {{ end }}{{ ensureQuoted $elem }}{{ end }}]
}
`
)

type syntaxMode uint8

func (m syntaxMode) String() string {
	switch m {
	case Modern:
		return "modern"
	case Legacy:
		return "legacy"
	default:
		panic("Unknown syntax")
	}
}

const (
	Modern syntaxMode = iota
	Legacy
)

type configGenerator struct {
	zoneTemplate   *template.Template
	recordTemplate *template.Template

	syntax syntaxMode
}

func env_check() string {
	env_chk := fmt.Sprintf("${var.env == \"%s\" ? 1 : 0}", *env)
	return env_chk
}

func newConfigGenerator(syntax syntaxMode) *configGenerator {
	g := &configGenerator{syntax: syntax}
	g.zoneTemplate = template.Must(template.New("zone").Parse(zoneTemplateStr))
	g.recordTemplate = template.Must(template.New("record").Funcs(template.FuncMap{
		"ensureQuoted":  ensureQuoted,
		"env_check":     env_check,
		"zoneReference": g.zoneReference,
	}).Parse(recordTemplateStr))
	return g
}

type zoneTemplateData struct {
	ID     string
	Domain string
}
type recordTemplateData struct {
	ResourceID string
	Record     dnsRecord
	ZoneID     string
}
type dnsRecord struct {
	Name     string
	Type     string
	TTL      uint32
	Data     []string
	Comments []string
}
type recordKey struct {
	Name string
	Type string
}
type recordKeySlice []recordKey

func (records recordKeySlice) Len() int {
	return len(records)
}
func (records recordKeySlice) Less(i, j int) bool {
	genKey := func(k recordKey) string {
		return fmt.Sprintf("%s-%s", k.Name, k.Type)
	}
	return genKey(records[i]) < genKey(records[j])
}
func (records recordKeySlice) Swap(i, j int) {
	tmp := records[i]
	records[i] = records[j]
	records[j] = tmp
}

var (
	excludedTypesRaw = flag.String("exclude", "SOA,NS", "Comma-separated list of record types to ignore")
	domain           = flag.String("domain", "", "Name of domain")
	env              = flag.String("env", "", "Environment to use")
	zoneFile         = flag.String("zone-file", "", "Path to zone file. Defaults to <domain>.zone in working dir")
	showVersion      = flag.Bool("version", false, "Show version")
	legacySyntax     = flag.Bool("legacy-syntax", false, "Generate legacy terraform syntax (versions older than 0.12)")
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("tfz53 %s (%s/%s) (%s on %s)", Version, Branch, Revision, BuildUser, BuildDate)
		os.Exit(0)
	}

	if *domain == "" {
		log.Fatal("Domain is required")
	}
	if *env == "" {
		log.Fatal("Env is required")
	}
	if *zoneFile == "" {
		*zoneFile = fmt.Sprintf("%s.zone", *domain)
	}

	excludedTypes := excludedTypesFromString(*excludedTypesRaw)

	fileReader, err := os.Open(*zoneFile)
	if err != nil {
		log.Fatal(err)
	}

	var syntax syntaxMode
	if !*legacySyntax {
		syntax = Modern
	} else {
		syntax = Legacy
	}

	g := newConfigGenerator(syntax)
	g.generateTerraformForZone(*domain, excludedTypes, fileReader, os.Stdout)
}

func (g *configGenerator) generateTerraformForZone(domain string, excludedTypes map[uint16]bool, zoneReader io.Reader, output io.Writer) {
	records := readZoneRecords(zoneReader, excludedTypes)

	zoneID := g.generateZoneResource(domain, output)

	recordKeys := make(recordKeySlice, 0, len(records))
	for key := range records {
		recordKeys = append(recordKeys, key)
	}
	sort.Sort(sort.Reverse(recordKeys))

	for _, key := range recordKeys {
		rec := records[key]
		err := g.generateRecordResource(rec, zoneID, output)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
	}
}

func readZoneRecords(zoneReader io.Reader, excludedTypes map[uint16]bool) map[recordKey]dnsRecord {
	records := make(map[recordKey]dnsRecord)
	for rr := range dns.ParseZone(zoneReader, *domain, *zoneFile) {
		if rr.Error != nil {
			log.Printf("Error: %v\n", rr.Error)
			continue
		}

		recordType := rr.Header().Rrtype
		isExcluded, ok := excludedTypes[recordType]
		if ok && isExcluded {
			continue
		}

		record := generateRecord(rr)

		key := recordKey{record.Name, record.Type}
		if _, ok := records[key]; ok {
			record = mergeRecords(records[key], record)
		}

		records[key] = record
	}
	return records
}

func (g *configGenerator) generateZoneResource(domain string, w io.Writer) string {
	zoneName := strings.TrimRight(domain, ".")
	data := zoneTemplateData{
		ID:     strings.Replace(zoneName, ".", "-", -1),
		Domain: zoneName,
	}
	if strings.Contains(data.ID, "arpa") {
		data.ID = "r-" + data.ID
	}
	//err := g.zoneTemplate.Execute(w, data)
	return data.ID
}

func (g *configGenerator) generateRecordResource(record dnsRecord, zoneID string, w io.Writer) error {
	sanitizedName := sanitizeRecordName(record.Name)
	id := fmt.Sprintf("%s-%s", sanitizedName, record.Type)

	data := recordTemplateData{
		ResourceID: id,
		Record:     record,
		ZoneID:     zoneID,
	}

	return g.recordTemplate.Execute(w, data)
}

func mergeRecords(a, b dnsRecord) dnsRecord {
	a.Data = append(a.Data, b.Data...)
	a.Comments = append(a.Comments, b.Comments...)

	return a
}

func generateRecord(rr *dns.Token) dnsRecord {
	header := rr.Header()
	name := strings.ToLower(header.Name)

	key := recordKey{
		Name: name,
		Type: dns.TypeToString[header.Rrtype],
	}

	data := strings.TrimPrefix(rr.String(), header.String())
	if key.Type == "CNAME" {
		data = strings.ToLower(data)
	}

	if key.Type == "TXT" {
		// TXT records can be up to 255 characters long in BIND format. Cloud
		// DNS Terraform providers lets them be longer by joining them with
		// a \"\" sequence. So we split by " " (which is inserted by miekg/dns
		// unless already in the source file), trim away any spaces, then join
		// by the escape sequence. So the following:
		// foo IN TXT "long-[250 chars]-string"
		// ... will be hava a data section like this before being adjusted:
		// "long-[250 chars]" "-string"
		// Below, we merge this into
		// "long-[250 chars]\"\"-string"
		// Which is then properly passed from Terraform to Route 53
		parts := strings.Split(data, `" "`)
		for pidx := range parts {
			parts[pidx] = strings.TrimSpace(parts[pidx])
		}
		data = strings.Join(parts, `\"\"`)
	}

	comments := make([]string, 0)
	if rr.Comment != "" {
		comments = append(comments, strings.TrimLeft(rr.Comment, ";"))
	}
	return dnsRecord{
		Name:     key.Name,
		Type:     key.Type,
		TTL:      header.Ttl,
		Data:     []string{data},
		Comments: comments,
	}
}

// sanitizeRecordName creates a normalized record name that Terraform accepts.
// Terraform only allows letters, numbers, dashes and underscores, while DNS
// records allow far more.
// 1. All dots are replaced with -
// 2. * is replaced by the string "wildcard"
// 3. IDN records are cleaned using punycode conversion
// 4. Any remaining non-allowed characters are replaced underscore
// 5. If the start of the record name is not a valid Terraform identifier,
//    then prepend an underscore.
func sanitizeRecordName(name string) string {
	withoutDots := strings.Replace(strings.TrimRight(name, "."), ".", "-", -1)
	withoutAsterisk := strings.Replace(withoutDots, "*", "wildcard", -1)

	punycoded, err := idna.Punycode.ToASCII(withoutAsterisk)
	if err != nil {
		log.Fatalf("Cannot create resource name from record %s: %v", name, err)
	}

	id := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			(r == '-' || r == '_') {
			return r
		}
		return '_'
	}, punycoded)

	if (id[0] >= 'a' && id[0] <= 'z') ||
		(id[0] >= 'A' && id[0] <= 'Z') ||
		(id[0] == '_') {
		return id
	}

	return fmt.Sprintf("_%s", id)
}

func excludedTypesFromString(s string) map[uint16]bool {
	excludedTypes := make(map[uint16]bool)
	for _, t := range strings.Split(s, ",") {
		t = strings.ToUpper(t) // ensure upper case
		rrType := dns.StringToType[t]
		excludedTypes[rrType] = true
	}
	return excludedTypes
}

func ensureQuoted(s string) string {
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s
	}
	return fmt.Sprintf("%q", s)
}

func (g *configGenerator) zoneReference(zone string) string {
	switch g.syntax {
	case Modern:
		return fmt.Sprintf("\"${aws_route53_zone.%s.*.zone_id[count.index]}\"", zone)
	case Legacy:
		return fmt.Sprintf(`"\"${aws_route53_zone.%s.*.zone_id[count.index]}\"`, zone)
	default:
		panic(fmt.Sprintf("Unknown mode %v", g.syntax))
	}
}
