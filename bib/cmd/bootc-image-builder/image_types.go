package main

import (
	"fmt"
	"slices"
)

type ImageType int

const (
	ImageTypeDisk ImageType = iota
	ImageTypeISO
)

var supportedImageTypes = map[string]bool{
	"ami":          true,
	"qcow2":        true,
	"raw":          true,
	"vmdk":         true,
	"anaconda-iso": true,
	"iso":          true,
}

type ImageTypes []string

func (it ImageTypes) Validate() error {
	// Disk image types all share the same manifest but with
	// different export pipelines. ISO images can't be built
	// alongside other image types.
	if len(it) > 1 && (slices.Contains(it, "iso") || slices.Contains(it, "anaconda-iso")) {
		return fmt.Errorf("cannot build iso with different target types")
	}

	for _, typ := range it {
		if !supportedImageTypes[typ] {
			return fmt.Errorf("Manifest(): unsupported image type %q", typ)
		}
	}

	return nil
}

func (it ImageTypes) Type() ImageType {
	if it[0] == "iso" || it[0] == "anaconda-iso" {
		return ImageTypeISO
	}

	return ImageTypeDisk
}
