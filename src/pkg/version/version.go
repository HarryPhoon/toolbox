package version

import "fmt"

// Version is the version of Toolbox
type Version struct {
	Major int
	Minor int
	Patch int
}

// CurrentVersion holds the information about current build version
var CurrentVersion = Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// GetVersion returns string with the version of Toolbox
func GetVersion() string {
	return fmt.Sprintf("%d.%d.%d", CurrentVersion.Major, CurrentVersion.Minor, CurrentVersion.Patch)
}
