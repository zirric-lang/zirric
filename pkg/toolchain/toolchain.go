// Package toolchain reports the version of the running Zirric build, as ldflags stamped it into github.com/metal-stack/v.
package toolchain

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
	mv "github.com/metal-stack/v"
)

// developmentVersion stands in for metal-stack/v's default, which instructs whoever builds Zirric rather than whoever runs it.
const developmentVersion = "devel"

func init() {
	if _, err := version.ParseSemver(mv.Version); err != nil {
		mv.Version = developmentVersion
	}
}

// Version returns the version this binary was built as, or nil when the build stamped none.
// Nil means unknown, not old: a development build cannot be held to a version predicate.
func Version() version.Version {
	semver, err := version.ParseSemver(mv.Version)
	if err != nil {
		return nil
	}
	return semver
}

// Description returns the version, commit, revision, build date and Go version.
func Description() string {
	return mv.V.String()
}

// Revision returns the Git revision the build was made from, or "" when the build stamped none.
func Revision() string {
	return mv.Revision
}
