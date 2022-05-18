// VulcanizeDB
// Copyright © 2022 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
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
