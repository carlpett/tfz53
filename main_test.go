package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRecordResource(t *testing.T) {
	record := dnsRecord{
		Name:     "foo.bar",
		Data:     []string{"127.0.0.1"},
		Type:     "A",
		TTL:      3600,
		Comments: []string{"This is a test"},
	}
	expected := `# This is a test
resource "aws_route53_record" "foo-bar-A" {
   zone_id = "${aws_route53_zone.test-zone.zone_id}"
   name = "foo.bar"
   type = "A"
   ttl = "3600"
   records = ["127.0.0.1"]
}`

	var buf bytes.Buffer
	err := generateRecordResource(record, "test-zone", &buf)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(buf.String()) != expected {
		t.Errorf("Unexpected result from resource generation. Expected:\n%s\nGot:\n%s", expected, buf.String())
	}
}

func TestResourceNameSanitation(t *testing.T) {
	cases := []struct {
		name           string
		expectedOutput string
	}{
		{"foo.bar.com", "foo-bar-com"},
		{"*.bar.com", "wildcard-bar-com"},
		{"åäö.bar.com", "xn---bar-com-zzaj2q"},
		{"#issue-2.github.com", "_issue-2-github-com"},
		{"//issue-2.github.com", "__issue-2-github-com"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id := sanitizeRecordName(c.name)
			if id != c.expectedOutput {
				t.Errorf("Expected %q, got %q", c.expectedOutput, id)
			}
		})
	}
}

func TestAcceptance(t *testing.T) {
	fileNames, err := filepath.Glob("testdata/*.zone")
	if err != nil {
		panic(err)
	}

	for _, n := range fileNames {
		t.Run(n, func(t *testing.T) {
			file, err := os.Open(n)
			if err != nil {
				panic(err)
			}
			expected, err := ioutil.ReadFile(strings.Replace(n, ".zone", ".expected", 1))
			if err != nil {
				panic(err)
			}

			var buf bytes.Buffer
			domain := strings.Replace(filepath.Base(n), ".zone", "", 1)
			excludedTypes := excludedTypesFromString("SOA,NS")
			generateTerraformForZone(domain, excludedTypes, file, &buf)

			if strings.TrimSpace(buf.String()) != strings.TrimSpace(string(expected)) {
				t.Errorf("Unexpected result from full Terraform output. Expected:\n%s\nGot:\n%s", expected, buf.String())
			}
		})
	}
}
