package main

import (
	"fmt"
	"sort"
	"strings"
)

type imageType struct {
	Export string
	ISO    bool
}

var supportedImageTypes = map[string]imageType{
	"ami":          imageType{Export: "image"},
	"qcow2":        imageType{Export: "qcow2"},
	"raw":          imageType{Export: "image"},
	"vmdk":         imageType{Export: "vmdk"},
	"vhd":          imageType{Export: "vpc"},
	"anaconda-iso": imageType{Export: "anaconda-iso", ISO: true},
	"iso":          imageType{Export: "iso", ISO: true},
}

// allImageTypesString returns a comma-separated list of supported types
func allImageTypesString() string {
	keys := make([]string, 0, len(supportedImageTypes))
	for k := range supportedImageTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return strings.Join(keys, ", ")
}

type ImageTypes []string

func NewImageTypes(imageTypeNames []string) (ImageTypes, error) {
	if len(imageTypeNames) == 0 {
		return nil, fmt.Errorf("cannot use an empty array as a build request")
	}

	var ISOs, disks int
	for _, name := range imageTypeNames {
		imgType, ok := supportedImageTypes[name]
		if !ok {
			return nil, fmt.Errorf("unsupported image type %q, valid types are %s", name, allImageTypesString())
		}
		if imgType.ISO {
			ISOs++
		} else {
			disks++
		}
	}
	if ISOs > 0 && disks > 0 {
		return nil, fmt.Errorf("cannot mix ISO/disk images in request %v", imageTypeNames)
	}

	return ImageTypes(imageTypeNames), nil
}

func (it ImageTypes) Exports() []string {
	exports := make([]string, 0, len(it))
	// XXX: this assumes we have validated
	for _, name := range it {
		imgType := supportedImageTypes[name]
		exports = append(exports, imgType.Export)
	}

	return exports
}

func (it ImageTypes) BuildsISO() bool {
	// XXX: this assumes we have validated
	return supportedImageTypes[it[0]].ISO
}
