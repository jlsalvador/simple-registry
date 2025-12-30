package filesystem

import (
	"iter"
	"os"
	"path/filepath"

	pkgDigest "github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func (s *FilesystemDataStorage) ReferrersGet(
	repo,
	manifestDigest string,
) (digests iter.Seq[string], err error) {
	algo, hash, err := pkgDigest.Parse(manifestDigest)
	if err != nil {
		return nil, err
	}

	refDir := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"referrers", algo, hash,
	)

	entries, err := os.ReadDir(refDir)
	if err != nil {
		return nil, err
	}

	return func(yield func(string) bool) {
		for _, e := range entries {
			name := e.Name()

			if !registry.RegExprDigest.MatchString(name) {
				continue
			}

			if !yield(name) {
				return
			}
		}
	}, nil
}
