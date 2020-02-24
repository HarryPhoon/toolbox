package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/containers/toolbox/pkg/podman"
	"github.com/shirou/gopsutil/host"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	idTruncLength = 12
)

// GetCgroupsVersion returns the cgroups version of the host
//
// The information is taken from the output of `podman info` command
func GetCgroupsVersion() (string, error) {
	podmanInfo, err := podman.PodmanInfo()
	if err != nil {
		return "", err
	}
	cgroupVersion := fmt.Sprint(podmanInfo["host"].(map[string]interface{})["CgroupVersion"])

	return cgroupVersion, nil
}

// GetHostPlatform returns the name of host system.
//
// Examples:
// - host is Fedora, returned string is 'fedora'
// - host is Ubuntu, returned string is 'ubuntu'
func GetHostPlatform() string {
	hostInfo, err := host.Info()
	if err != nil {
		logrus.Error(err)
	}
	return hostInfo.Platform
}

// GetHostVersionID returns the version of host system.
//
// Examples:
// - host is Fedora 31, returned string is '31'
// - host is Ubuntu 19.04, returned string is '19.04'
func GetHostVersionID() string {
	hostInfo, err := host.Info()
	if err != nil {
		logrus.Error(err)
	}
	return hostInfo.PlatformVersion
}

// GetGroupForSudo returns the name of the sudoers group.
//
// Some distros call it 'sudo' (eg. Ubuntu) and some call it 'wheel' (eg. Fedora).
func GetGroupForSudo() string {
	group := ""
	if _, err := user.LookupGroup("sudo"); err == nil {
		group = "sudo"
	} else if _, err := user.LookupGroup("wheel"); err == nil {
		group = "wheel"
	}
	return group
}

// GetMountPoint returns the mount point of a target.
func GetMountPoint(target string) (string, error) {
	cmd := exec.Command("df", "--output=target", target)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	options := strings.SplitAfter(string(output), "\n")

	return strings.Trim(string(options[1]), "\n"), nil
}

// GetMountOptions returns the mount options of a target.
func GetMountOptions(target string) (string, error) {
	cmd := exec.Command("findmnt", "--noheadings", "--output", "OPTIONS", target)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	options := strings.SplitAfter(string(output), "\n")

	return strings.Trim(options[0], "\n"), nil
}

// GetUID returns the user ID
func GetUID() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", errors.New("Failed getting user information")
	}
	return currentUser.Uid, nil
}

// ShortID shortens provided id to first 12 characters.
func ShortID(id string) string {
	if len(id) > idTruncLength {
		return id[:idTruncLength]
	}
	return id
}

// ReferenceCanBeID checks if the provided text matches a format for an ID.
func ReferenceCanBeID(text string) bool {
	matched, err := regexp.MatchString(`^[a-f0-9]\{6,64\}$`, text)
	if err != nil {
		logrus.Error(err)
	}
	return matched
}

// ReferenceHasDomain checks if the provided text has a domain definition in it.
func ReferenceHasDomain(text string) bool {
	i := strings.IndexRune(text, '/')
	if i == -1 {
		return false
	}

	// A domain should contain a top level domain name. An exception is 'localhost'
	if strings.ContainsAny(text[:i], ".:") && text[:i] != "localhost" {
		return false
	}

	return true
}

// PathExists wraps around os.Stat providing a nice interface for checking an existence of a path.
func PathExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

// UpdateContainerAndImageNames takes care of standardizing names of containers and images.
//
// If no image name is specified then the base image will reflect the platform of the host (even the version).
// If no container name is specified then the name of the image will be used.
//
// If the host system is unknown then the base image will be 'fedora-toolbox' with a default version
func UpdateContainerAndImageNames(containerName string, imageName string, release string) (string, string) {
	hostPlatform := GetHostPlatform()

	if release == "" {
		if hostPlatform == "fedora" {
			release = GetHostVersionID()
			if release == "rawhide" {
				release = "32"
			}
		} else {
			release = viper.GetString("RELEASE_DEFAULT")
		}
	}

	if imageName == "" {
		if hostPlatform == "fedora" {
			imageName = fmt.Sprintf("f%s/fedora-toolbox:%s", release, release)
		} else {
			imageName = fmt.Sprintf("f%s/fedora-toolbox:%s", release, release)
		}
	} else {
		// If the image name for fedora toolbox does not have the parent defined (eg. f31/ for
		// fedora-toolbox:31) then add it here.
		reg, err := regexp.Compile("^fedora-toolbox:[1-9]{1}[0-9]*")
		if err == nil {
			if reg.MatchString(imageName) {
				imageNameParts := strings.SplitN(imageName, ":", 2)
				imageName = fmt.Sprintf("f%s/%s:%s", imageNameParts[1], imageNameParts[0], imageNameParts[1])
			}
		} else {
			logrus.Debug(err)
		}
	}

	// If no container name is specified then use the image name and it's version
	if containerName == "" {
		containerName = strings.ReplaceAll(imageName, ":", "-")
		if strings.Contains(containerName, "/") {
			nameSplit := strings.Split(containerName, "/")
			containerName = nameSplit[len(nameSplit)-1]
		}
	}

	return containerName, imageName
}

// IsContainerNameValid checks if the name of a container matches the right pattern
func IsContainerNameValid(containerName string) bool {
	reg, err := regexp.Compile("^[a-zA-Z0-9][a-zA-Z0-9_.-]*")
	if err != nil {
		return false
	}

	if !reg.MatchString(containerName) {
		return false
	}

	return true
}

// NumberPrompt creates an interactive prompt that expects and returns an integer
func NumberPrompt(defaultValue int, min int, max int, prompt string) int {
	var tmp string
	var response int = 0

	for true {
		fmt.Printf("%s (%d to abort) [%d-%d]: ", prompt, defaultValue, min, max)
		fmt.Scanf("%s", &tmp)

		tmpresponse, err := strconv.ParseInt(tmp, 10, 0)
		if err != nil {
			continue
		}

		response = int(tmpresponse)

		if response >= min && response <= max {
			break
		}
	}

	return response
}

func JoinJSON(joinkey string, maps ...[]map[string]interface{}) []map[string]interface{} {
	var json []map[string]interface{}
	found := make(map[string]bool)

	// Iterate over every json provided and check if it is already in the final json
	// If it contains some invalid entry (equals nil), then skip that entry

	for _, m := range maps {
		for _, entry := range m {
			if entry["names"] == nil && entry["Names"] == nil {
				continue
			}
			key := entry[joinkey].(string)
			if _, ok := found[key]; !ok {
				found[key] = true
				json = append(json, entry)
			}
		}
	}
	return json
}

func SortJSON(json []map[string]interface{}, key string, hasInterface bool) []map[string]interface{} {
	sort.Slice(json, func(i, j int) bool {
		if hasInterface {
			return json[i][key].([]interface{})[0].(string) < json[j][key].([]interface{})[0].(string)
		}
		return json[i][key].(string) < json[j][key].(string)
	})

	return json
}

func SystemctlOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}
