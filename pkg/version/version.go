package version

import "fmt"

// A reusable structure to allow developers to set their application versions.
type Version struct {
	Major int    // Major version component of the current release
	Minor int    // Minor version component of the current release
	Patch int    // Patch version component of the current release
	Meta  string // Version metadata to append to the version string
}

// Provides a string with the version
func (version *Version) GetVersion() string {
	return fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
}

// Provides a string with the version and Meta.
func (version *Version) GetVersionWithMeta() string {
	v := version.GetVersion()
	if version.Meta != "" {
		v += "-" + version.Meta
	}
	return v
}
