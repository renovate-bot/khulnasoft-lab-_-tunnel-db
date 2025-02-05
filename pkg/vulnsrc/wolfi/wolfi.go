package wolfi

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/khulnasoft-lab/tunnel-db/pkg/db"
	"github.com/khulnasoft-lab/tunnel-db/pkg/types"
	"github.com/khulnasoft-lab/tunnel-db/pkg/utils"
	"github.com/khulnasoft-lab/tunnel-db/pkg/vulnsrc/vulnerability"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/xerrors"
)

const (
	wolfiDir   = "wolfi"
	distroName = "wolfi"
)

var (
	source = types.DataSource{
		ID:   vulnerability.Wolfi,
		Name: "Wolfi Secdb",
		URL:  "https://packages.wolfi.dev/os/security.json",
	}
)

type VulnSrc struct {
	dbc db.Operation
}

func NewVulnSrc() VulnSrc {
	return VulnSrc{
		dbc: db.Config{},
	}
}

func (vs VulnSrc) Name() types.SourceID {
	return source.ID
}

func (vs VulnSrc) Update(dir string) error {
	rootDir := filepath.Join(dir, "vuln-list", wolfiDir)
	var advisories []advisory
	err := utils.FileWalk(rootDir, func(r io.Reader, path string) error {
		var advisory advisory
		if err := json.NewDecoder(r).Decode(&advisory); err != nil {
			return xerrors.Errorf("failed to decode Wolfi advisory: %w", err)
		}
		advisories = append(advisories, advisory)
		return nil
	})
	if err != nil {
		return xerrors.Errorf("error in Wolfi walk: %w", err)
	}

	if err = vs.save(advisories); err != nil {
		return xerrors.Errorf("error in Wolfi save: %w", err)
	}

	return nil
}

func (vs VulnSrc) save(advisories []advisory) error {
	err := vs.dbc.BatchUpdate(func(tx *bolt.Tx) error {
		for _, adv := range advisories {
			bucket := distroName
			if err := vs.dbc.PutDataSource(tx, bucket, source); err != nil {
				return xerrors.Errorf("failed to put data source: %w", err)
			}
			if err := vs.saveSecFixes(tx, distroName, adv.PkgName, adv.Secfixes); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return xerrors.Errorf("error in db batch update: %w", err)
	}
	return nil
}

func (vs VulnSrc) saveSecFixes(tx *bolt.Tx, platform, pkgName string, secfixes map[string][]string) error {
	for fixedVersion, vulnIDs := range secfixes {
		advisory := types.Advisory{
			FixedVersion: fixedVersion,
		}
		for _, vulnID := range vulnIDs {
			// See https://gitlab.alpinelinux.org/alpine/infra/docker/secdb/-/issues/3
			// e.g. CVE-2017-2616 (+ regression fix)
			ids := strings.Fields(vulnID)
			for _, cveID := range ids {
				cveID = strings.ReplaceAll(cveID, "CVE_", "CVE-")
				if !strings.HasPrefix(cveID, "CVE-") {
					continue
				}
				if err := vs.dbc.PutAdvisoryDetail(tx, cveID, pkgName, []string{platform}, advisory); err != nil {
					return xerrors.Errorf("failed to save Wolfi advisory: %w", err)
				}

				// for optimization
				if err := vs.dbc.PutVulnerabilityID(tx, cveID); err != nil {
					return xerrors.Errorf("failed to save the vulnerability ID: %w", err)
				}
			}
		}
	}
	return nil
}

func (vs VulnSrc) Get(_, pkgName string) ([]types.Advisory, error) {
	bucket := distroName
	advisories, err := vs.dbc.GetAdvisories(bucket, pkgName)
	if err != nil {
		return nil, xerrors.Errorf("failed to get Wolfi advisories: %w", err)
	}
	return advisories, nil
}
