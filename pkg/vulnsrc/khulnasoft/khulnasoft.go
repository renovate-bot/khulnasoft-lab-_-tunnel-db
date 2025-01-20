package aqua

import (
	"path/filepath"

	"github.com/khulnasoft-lab/tunnel-db/pkg/types"
	"github.com/khulnasoft-lab/tunnel-db/pkg/vulnsrc/osv"
	"github.com/khulnasoft-lab/tunnel-db/pkg/vulnsrc/vulnerability"
)

const (
	sourceID = vulnerability.Khulnasoft
)

var vulnsDir = filepath.Join("vuln-list-khulnasoft", "vulns")

type VulnSrc struct{}

func NewVulnSrc() VulnSrc {
	return VulnSrc{}
}

func (VulnSrc) Name() types.SourceID {
	return sourceID
}

func (VulnSrc) Update(root string) error {
	dataSources := map[types.Ecosystem]types.DataSource{}
	for _, ecosystem := range vulnerability.Ecosystems {
		dataSources[ecosystem] = types.DataSource{
			ID:   sourceID,
			Name: "The Khulnasoft Security Vulnerability Database",
			URL:  "https://github.com/khulnasoft-lab/vuln-list-khulnasoft",
		}
	}

	return osv.New(vulnsDir, sourceID, dataSources, nil).Update(root)
}
