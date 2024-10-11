package distrodef

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"

	"github.com/hashicorp/go-version"

	"github.com/osbuild/bootc-image-builder/bib/data"
)

var dataDistroDefs = data.DistroDefs

// ImageDef is a structure containing extra information needed to build an image that cannot be extracted
// from the container image itself. Currently, this is only the list of packages needed for the installer
// ISO.
type ImageDef struct {
	Packages []string `yaml:"packages"`
}

func glob(fsys embed.FS, patternPath string) ([]string, error) {
	var matches []string

	defsDir := filepath.Dir(patternPath)
	dents, err := fsys.ReadDir(defsDir)
	if err != nil {
		return nil, err
	}
	pattern := filepath.Base(patternPath)
	for _, dent := range dents {
		match, err := filepath.Match(pattern, dent.Name())
		if err != nil {
			return nil, err
		}
		if match {
			matches = append(matches, filepath.Join(defsDir, dent.Name()))
		}
	}
	return matches, nil
}

func findDistroDef(fsys embed.FS, distro, wantedVerStr string) (string, error) {
	var bestFuzzyMatch string

	bestFuzzyVer := &version.Version{}
	wantedVer, err := version.NewVersion(wantedVerStr)
	if err != nil {
		return "", fmt.Errorf("cannot parse wanted version string: %w", err)
	}

	// exact match
	matches, err := glob(fsys, fmt.Sprintf("defs/%s-%s.yaml", distro, wantedVerStr))
	if err != nil {
		return "", err
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// fuzzy match
	matches, err = glob(fsys, fmt.Sprintf("defs/%s-[0-9]*.yaml", distro))
	if err != nil {
		return "", err
	}
	for _, m := range matches {
		baseNoExt := strings.TrimSuffix(filepath.Base(m), ".yaml")
		haveVerStr := strings.SplitN(baseNoExt, "-", 2)[1]
		// this should never error (because of the glob above) but be defensive
		haveVer, err := version.NewVersion(haveVerStr)
		if err != nil {
			return "", fmt.Errorf("cannot parse distro version from %q: %w", m, err)
		}
		if wantedVer.Compare(haveVer) > 0 && haveVer.Compare(bestFuzzyVer) > 0 {
			bestFuzzyVer = haveVer
			bestFuzzyMatch = m
		}
	}

	if bestFuzzyMatch == "" {
		return "", fmt.Errorf("could not find def file for distro %s-%s", distro, wantedVerStr)
	}

	return bestFuzzyMatch, nil
}

func loadFile(fsys embed.FS, distro, ver string) ([]byte, error) {
	defPath, err := findDistroDef(fsys, distro, ver)
	if err != nil {
		return nil, err
	}

	content, err := fsys.ReadFile(defPath)
	if err != nil {
		return nil, fmt.Errorf("could not read def file %s for distro %s-%s: %v", defPath, distro, ver, err)
	}
	return content, nil
}

// Loads a definition file for a given distro and image type
func LoadImageDef(distro, ver, it string) (*ImageDef, error) {
	data, err := loadFile(dataDistroDefs, distro, ver)
	if err != nil {
		return nil, err
	}

	var defs map[string]ImageDef
	if err := yaml.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("could not unmarshal def file for distro %s: %v", distro, err)
	}

	d, ok := defs[it]
	if !ok {
		return nil, fmt.Errorf("could not find def for distro %s and image type %s, available types: %s", distro, it, strings.Join(maps.Keys(defs), ", "))
	}

	return &d, nil
}
