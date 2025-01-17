//  Copyright 2017 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package precheck

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/osconfig/osinfo"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
)

const (
	docsURL = "https://cloud.google.com/sdk/gcloud/reference/compute/images/import"
)

// OSVersionCheck is a precheck.Check that verifies the disk's operating system is importable.
type OSVersionCheck struct {
	OSInfo *osinfo.OSInfo
}

// GetName returns the name of the precheck step; this is shown to the user.
func (c *OSVersionCheck) GetName() string {
	return "OS Version Check"
}

// Run executes the precheck step.
func (c *OSVersionCheck) Run() (*Report, error) {
	r := &Report{name: c.GetName()}
	// Find osID from OS config's detection results.
	major, minor := splitOSVersion(c.OSInfo.Version)
	osID := c.createOSID(major, minor, r)
	if osID == "" {
		r.Info("Unable to determine whether your system is supported for import. " +
			"For supported versions, see " + docsURL)
		r.result = Skipped
		return r, nil
	}
	// Check whether the osID is supported for import.
	// Some systems are only available as BYOL, so check for both osID variants.
	var supported bool
	for _, suffix := range []string{"", "-byol"} {
		if daisyutils.ValidateOS(osID+suffix) == nil {
			supported = true
			break
		}
	}
	if supported {
		if c.OSInfo.ShortName == osinfo.Windows {
			// Emit the NT version for Windows, since the same NT version is
			// either Desktop or Server, and we don't want to emit a misleading message.
			r.Info(fmt.Sprintf("Detected Windows version number: NT %s", c.OSInfo.Version))
		} else {
			r.Info(fmt.Sprintf("Detected system: %s", osID))
		}
	} else {
		r.Fatal(osID + " is not supported for import. For supported versions, see " + docsURL)
	}
	return r, nil
}

// createOSID creates the osID, as used in the `--os` flag of the CLI tools. An empty string is
// return when unable to determine the osID.
func (c *OSVersionCheck) createOSID(originalMajor string, originalMinor string, r *Report) string {
	major, minor := originalMajor, originalMinor

	switch c.OSInfo.ShortName {
	case "":
		r.Info("Unable to determine OS.")
		return ""
	case osinfo.Linux:
		// OS config returns "linux" as the distro when it can't find a more specific match.
		r.Info("Detected generic Linux system.")
		return ""
	case osinfo.Windows:
		r.Info("Detected Windows system.")
		// OS config uses NT version numbers, while cli_tools/common/distro uses marketing verions.
		windowsMajor, windowsMinor, err :=
			distro.WindowsServerVersionforNTVersion(originalMajor, originalMinor)
		if err == nil {
			major, minor = windowsMajor, windowsMinor
		}
	}

	release, err := distro.FromComponents(c.OSInfo.ShortName, major, minor, c.OSInfo.Architecture)
	if err != nil {
		r.Info(err.Error())
		return ""
	}
	osID := release.AsGcloudArg()
	if osID != "" {
		return osID
	}
	// If the distro package can't determine the osID, attempt to create one using
	// the format "os-version".
	if c.OSInfo.ShortName != osinfo.Linux && c.OSInfo.ShortName != "" && c.OSInfo.Version != "" {
		return fmt.Sprintf("%s-%s", c.OSInfo.ShortName, c.OSInfo.Version)
	}
	return ""
}

func splitOSVersion(version string) (major, minor string) {
	if version == "" {
		return "", ""
	}
	if !strings.Contains(version, ".") {
		return version, ""
	}
	parts := strings.Split(version, ".")
	return parts[0], parts[1]
}
