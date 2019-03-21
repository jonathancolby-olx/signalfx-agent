package monitors

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Monitor metadata file.
const MonitorMetadataFile = "metadata.yaml"

type MetricMetadata struct {
	Name        string  `json:"name"`
	Alias       string  `json:"alias,omitempty"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Group       *string `json:"group"`
	Included    bool    `json:"included" default:"false"`
}

type PropMetadata struct {
	Name        string `json:"name"`
	Dimension   string `json:"dimension"`
	Description string `json:"description"`
}

type GroupMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MonitorMetadata struct {
	MonitorType string           `json:"monitorType" yaml:"monitorType"`
	SendAll     bool             `json:"sendAll" yaml:"sendAll"`
	Dimensions  []DimMetadata    `json:"dimensions" yaml:"dimensions"`
	Doc         string           `json:"doc" yaml:"doc"`
	Groups      []GroupMetadata  `json:"groups" yaml:"groups"`
	Metrics     []MetricMetadata `json:"metrics" yaml:"metrics"`
	Properties  []PropMetadata   `json:"properties" yaml:"properties"`
}

type PackageMetadata struct {
	Monitors []MonitorMetadata
	Package  string `yaml:"-"`
	Path     string `json:"-", yaml:"-"`
}

type DimMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func CollectMetadata(root string) ([]PackageMetadata, error) {
	var packages []PackageMetadata

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != MonitorMetadataFile {
			return nil
		}

		var monitorDocs []MonitorMetadata

		if bytes, err := ioutil.ReadFile(path); err != nil {
			return errors.Errorf("Unable to read metadata file %s: %s", path, err)
		} else if err := yaml.UnmarshalStrict(bytes, &monitorDocs); err != nil {
			return errors.Wrapf(err, "unable to unmarshal file %s", path)
		}

		packages = append(packages, PackageMetadata{
			Monitors: monitorDocs,
			Package:  filepath.Dir(path),
			Path:     path,
		})

		return nil
	}); err != nil {
		return nil, err
	}

	return packages, nil
}
