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
	"vhd":          imageType{Export: "vhd"},
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

type BuildRequest struct {
	Exports []string
	ISO     bool

	// XXX does not quite fit
	Manifest []byte
}

func NewBuildRequest(imageTypeNames []string) (*BuildRequest, error) {
	if len(imageTypeNames) == 0 {
		return nil, fmt.Errorf("cannot convert empty array of image types")
	}

	var ISOs, Disks int
	exports := make([]string, 0, len(imageTypeNames))
	for _, name := range imageTypeNames {
		imgType, ok := supportedImageTypes[name]
		if !ok {
			return nil, fmt.Errorf("unsupported image type %q, valid types are %s", name, allImageTypesString())
		}
		if imgType.ISO {
			ISOs++
		} else {
			Disks++
		}
		exports = append(exports, imgType.Export)
	}
	if ISOs > 0 && Disks > 0 {
		return nil, fmt.Errorf("cannot mix ISO/disk images in request %v", imageTypeNames)
	}

	return &BuildRequest{
		Exports: exports,
		ISO:     (ISOs > 0),
	}, nil
}
