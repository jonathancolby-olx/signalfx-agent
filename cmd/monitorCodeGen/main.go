package main

import (
	"bytes"
	"errors"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path/filepath"
	"strings"

	"text/template"
)

var tmpl = `// DO NOT EDIT. This file is auto-generated.

package {{.package}}

import "github.com/signalfx/signalfx-agent/internal/monitors"

{{define "Metrics"}}nil{{end}}

{{define "Groups"}}nil{{end}}

{{define "Dimensions"}}nil{{end}}

{{define "Properties"}}nil{{end}}

const (
{{- range .monitor.Groups }}
	{{(printf "group.%s" .Name) | formatVariable}} = "{{.Name}}"
{{- end}}
)

var groupSet = map[string]bool {
{{- range .monitor.Groups}}
	{{(printf "group.%s" .Name) | formatVariable}}: true,
{{- end}}
}

const (
{{- range .monitor.Metrics}}
	{{.Name | formatVariable }} = "{{.Name}}"
{{- end}}
)

var metricSet = map[string]bool {
{{- range .monitor.Metrics}}
	{{.Name | formatVariable}}: true,
{{- end}}
}

var includedMetrics = map[string]bool {
{{- range .monitor.Metrics}}
{{- if .Included}}
	{{.Name | formatVariable}}: true,
{{- end}}
{{- end}}
}

var groupMetricsMap = map[string][]string {
{{- range $group, $metrics := .groupMetricsMap}}
	{{(printf "group.%s" $group) | formatVariable}}: []string {
		{{- range $metrics}}
		{{. | formatVariable}},
		{{- end}}
	},
{{- end}}
}

var metadata = monitors.Metadata{
	MonitorType: "{{.monitor.MonitorType}}",
	IncludedMetrics: includedMetrics,
	Metrics: metricSet,
	Groups: groupSet,
	GroupMetricsMap: groupMetricsMap,
	SendAll: {{ .monitor.SendAll }},
}
`

// TODO: check mtimes
func main() {
	if pkgs, err := monitors.CollectMetadata("internal/monitors"); err != nil {
		log.Fatal(err)
	} else {
		template := template.Must(template.New("").Funcs(template.FuncMap{
			"formatVariable": func(s string) (string, error) {
				if s == "" {
					return "", errors.New("string cannot be empty")
				}
				// Convert various characters to . for strings.Title to operate on.
				replace := strings.NewReplacer("_", ".", "-", ".", "<", ".", ">", ".")
				str := replace.Replace(s)
				str = strings.Title(str)
				str = strings.ReplaceAll(str, ".", "")
				// Lower case first letter.
				return strings.ToLower(string(str[0])) + str[1:], nil
			},
		}).Parse(tmpl))
		for _, pkg := range pkgs {
			for _, mon := range pkg.Monitors {
				if mon.MonitorType != "kubernetes-cluster" {
					continue
				}

				writer := &bytes.Buffer{}
				groupMetricsMap := map[string][]string{}

				for _, metric := range mon.Metrics {
					if metric.Group == nil {
						continue
					}
					groupMetricsMap[*metric.Group] = append(groupMetricsMap[*metric.Group], metric.Name)
				}

				if err := template.Execute(writer, map[string]interface{}{
					"monitor":         mon,
					"package":         filepath.Base(pkg.Package),
					"groupMetricsMap": groupMetricsMap,
				}); err != nil {
					log.Fatalf("failed executing template for %s: %s", mon.MonitorType, err)
				}

				if err := ioutil.WriteFile(filepath.Join(pkg.Package, "generatedMetadata.go"), writer.Bytes(),
					0644); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
