package dnf

import (
	"fmt"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/dnfjson"

	"github.com/osbuild/bootc-image-builder/bib/internal/source"
)

func NewContainerSolver(cacheRoot string, architecture arch.Arch, sourceInfo *source.Info, depsolverCmd []string) *dnfjson.Solver {
	solver := dnfjson.NewSolver(
		sourceInfo.OSRelease.PlatformID,
		sourceInfo.OSRelease.VersionID,
		architecture.String(),
		fmt.Sprintf("%s-%s", sourceInfo.OSRelease.ID, sourceInfo.OSRelease.VersionID),
		cacheRoot)
	solver.SetDNFJSONPath(depsolverCmd[0], depsolverCmd[1:]...)
	solver.SetRootDir("/")
	return solver
}
