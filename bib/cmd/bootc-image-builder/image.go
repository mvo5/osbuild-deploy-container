package main

import (
	cryptorand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
)

// TODO: Auto-detect this from container image metadata
const DEFAULT_SIZE = uint64(10 * GibiByte)

type ManifestConfig struct {
	// OCI image path (without the transport, that is always docker://)
	Imgref string

	// Image type to build (currently: qcow2, ami)
	//
	// TODO: Make this an enum.
	ImgType string

	// Build config
	Config *BuildConfig

	// CPU architecture of the image
	Architecture arch.Arch

	// TLSVerify specifies whether HTTPS and a valid TLS certificate are required
	TLSVerify bool
}

func Manifest(c *ManifestConfig) (*manifest.Manifest, error) {
	rng := createRand()

	// TODO: make this a interface again
	var img *image.BootcDiskImage
	var err error

	switch c.ImgType {
	case "ami", "qcow2":
		img, err = pipelinesForDiskImage(c, rng)
	default:
		fail(fmt.Sprintf("Manifest(): unsupported image type %q", c.ImgType))
	}

	if err != nil {
		fail(err.Error())
	}

	mf := manifest.New()
	mf.Distro = manifest.DISTRO_FEDORA
	runner := &runner.Fedora{Version: 39}
	containerSources := []container.SourceSpec{
		{
			Source:    c.Imgref,
			Name:      c.Imgref,
			TLSVerify: &c.TLSVerify,
		},
	}
	_, err = img.InstantiateManifestFromContainers(&mf, containerSources, runner, rng)

	return &mf, err
}

func pipelinesForDiskImage(c *ManifestConfig, rng *rand.Rand) (*image.BootcDiskImage, error) {
	if c.Imgref == "" {
		fail("pipeline: no base image defined")
	}
	containerSource := container.SourceSpec{
		Source:    c.Imgref,
		Name:      c.Imgref,
		TLSVerify: &c.TLSVerify,
	}

	var customizations *blueprint.Customizations
	if c.Config != nil && c.Config.Blueprint != nil {
		customizations = c.Config.Blueprint.Customizations
	}

	img := image.NewBootcDiskImage(containerSource)
	img.Users = users.UsersFromBP(customizations.GetUsers())
	img.Groups = users.GroupsFromBP(customizations.GetGroups())

	img.KernelOptionsAppend = []string{
		"rw",
		// TODO: Drop this as we expect kargs to come from the container image,
		// xref https://github.com/CentOS/centos-bootc-layered/blob/main/cloud/usr/lib/bootc/install/05-cloud-kargs.toml
		"console=tty0",
		"console=ttyS0",
	}

	img.SysrootReadOnly = true

	var imageFormat platform.ImageFormat
	var filename string
	switch c.ImgType {
	case "qcow2":
		imageFormat = platform.FORMAT_QCOW2
		filename = "disk.qcow2"
	case "ami":
		imageFormat = platform.FORMAT_RAW
		filename = "disk.raw"
	}

	switch c.Architecture {
	case arch.ARCH_X86_64:
		img.Platform = &platform.X86{
			BasePlatform: platform.BasePlatform{
				ImageFormat: imageFormat,
			},
			BIOS:       true,
			UEFIVendor: "fedora",
		}
	case arch.ARCH_AARCH64:
		img.Platform = &platform.Aarch64{
			UEFIVendor: "fedora",
			BasePlatform: platform.BasePlatform{
				ImageFormat: imageFormat,
				QCOW2Compat: "1.1",
			},
		}
	}

	if kopts := customizations.GetKernel(); kopts != nil && kopts.Append != "" {
		img.KernelOptionsAppend = append(img.KernelOptionsAppend, kopts.Append)
	}

	basept, ok := partitionTables[c.Architecture.String()]
	if !ok {
		fail(fmt.Sprintf("pipelines: no partition tables defined for %s", c.Architecture))
	}
	pt, err := disk.NewPartitionTable(&basept, nil, DEFAULT_SIZE, disk.RawPartitioningMode, nil, rng)
	check(err)
	img.PartitionTable = pt

	img.Filename = filename

	return img, nil
}

func createRand() *rand.Rand {
	seed, err := cryptorand.Int(cryptorand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		panic("Cannot generate an RNG seed.")
	}

	// math/rand is good enough in this case
	/* #nosec G404 */
	return rand.New(rand.NewSource(seed.Int64()))
}
