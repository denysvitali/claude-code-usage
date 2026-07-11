package config

import (
	"os"
	"path/filepath"
	"sort"
)

var legacyFilenames = []string{
	"claude.json",
	"kimi.json",
	"minimax.json",
	"zai.json",
}

// DetectLegacyFiles returns the paths of any legacy per-provider JSON files or
// combined credentials files found in configDir.
func DetectLegacyFiles(configDir string) ([]string, error) {
	var found []string

	for _, name := range legacyFilenames {
		p := filepath.Join(configDir, name)
		fi, err := os.Stat(p)
		if err == nil && !fi.IsDir() {
			found = append(found, p)
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	matches, err := filepath.Glob(filepath.Join(configDir, "*credentials*.json"))
	if err != nil {
		return nil, err
	}

	for _, m := range matches {
		fi, err := os.Stat(m)
		if err == nil && !fi.IsDir() {
			found = append(found, m)
		}
	}

	sort.Strings(found)

	// Deduplicate in case a glob match overlaps with one of the explicit names.
	unique := make([]string, 0, len(found))
	var prev string
	for _, f := range found {
		if f != prev {
			unique = append(unique, f)
			prev = f
		}
	}

	return unique, nil
}
