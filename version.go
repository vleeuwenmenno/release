package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VersionPattern represents the detected versioning scheme
type VersionPattern int

const (
	PatternUnknown     VersionPattern = iota
	PatternSemver                     // X.Y.Z or vX.Y.Z (optionally with pre-release/build metadata)
	PatternBuildNumber                // X.Y-N or X.Y-N-descriptor
)

func (p VersionPattern) String() string {
	switch p {
	case PatternSemver:
		return "semver"
	case PatternBuildNumber:
		return "build-number"
	default:
		return "unknown"
	}
}

func (p VersionPattern) Description() string {
	switch p {
	case PatternSemver:
		return "Semantic Versioning (X.Y.Z, optional +build metadata)"
	case PatternBuildNumber:
		return "Build Number (X.Y-N)"
	default:
		return "Unknown"
	}
}

// Version represents a parsed version tag
type Version struct {
	Major      int
	Minor      int
	Patch      int    // semver only
	Build      int    // build-number pattern only
	Prefix     string // "v" or ""
	Descriptor string // e.g. "podman" in 8.3-64-podman
	PreRelease string // e.g. "beta.1" in v1.2.3-beta.1
	BuildMeta  string // e.g. "24032026" in v1.2.3+24032026
	Raw        string // original tag string
	Pattern    VersionPattern
}

// String returns the formatted version string
func (v Version) String() string {
	return formatVersion(v)
}

// HasPreRelease returns true if version has a pre-release suffix
func (v Version) HasPreRelease() bool {
	return v.PreRelease != ""
}

// HasBuildMetadata returns true if version has semver build metadata.
func (v Version) HasBuildMetadata() bool {
	return v.BuildMeta != ""
}

// HasDateBuildMetadata returns true when build metadata matches DDMMYYYY.
func (v Version) HasDateBuildMetadata() bool {
	return isDateBuildMetadata(v.BuildMeta)
}

// HasDescriptor returns true if version has a descriptor suffix
func (v Version) HasDescriptor() bool {
	return v.Descriptor != ""
}

// VersionLine represents a group of versions sharing the same major (semver)
// or major.minor (build-number) prefix
type VersionLine struct {
	Major    int
	Minor    int // only meaningful for build-number pattern
	Pattern  VersionPattern
	Latest   Version
	Versions []Version
}

// Label returns a human-readable label for this version line
func (vl VersionLine) Label() string {
	switch vl.Pattern {
	case PatternBuildNumber:
		return fmt.Sprintf("%d.%d", vl.Major, vl.Minor)
	default:
		prefix := vl.Latest.Prefix
		return fmt.Sprintf("%s%d.x", prefix, vl.Major)
	}
}

// BumpType represents the kind of version bump
type BumpType int

const (
	BumpPatch           BumpType = iota // semver: X.Y.Z -> X.Y.(Z+1)
	BumpMinor                           // semver: X.Y.Z -> X.(Y+1).0; build: X.Y-N -> X.(Y+1)-1
	BumpMajor                           // semver: X.Y.Z -> (X+1).0.0; build: X.Y-N -> (X+1).0-1
	BumpBuild                           // build-number: X.Y-N -> X.Y-(N+1)
	BumpPreRelease                      // start or bump pre-release counter
	BumpRelease                         // strip pre-release: X.Y.Z-beta.1 -> X.Y.Z
	BumpDescriptor                      // change descriptor
	BumpStripDescriptor                 // remove descriptor
	BumpCustom                          // manual entry
)

func (b BumpType) String() string {
	switch b {
	case BumpPatch:
		return "Patch"
	case BumpMinor:
		return "Minor"
	case BumpMajor:
		return "Major"
	case BumpBuild:
		return "Build"
	case BumpPreRelease:
		return "Pre-release"
	case BumpRelease:
		return "Release"
	case BumpDescriptor:
		return "Change descriptor"
	case BumpStripDescriptor:
		return "Strip descriptor"
	case BumpCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// Choice represents a menu item in the TUI
type Choice struct {
	Label       string
	Description string
	Preview     string
	BumpType    BumpType
}

// Regex patterns for version detection
var (
	timeNow = time.Now

	// Semver: v1.2.3 or 1.2.3, optionally with pre-release like -beta.1 and build metadata like +24032026
	semverRegex = regexp.MustCompile(`^(v?)(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z][0-9A-Za-z.-]*))?(?:\+([0-9A-Za-z][0-9A-Za-z.-]*))?$`)

	// Build-number: 8.3-64 or 8.3-64-podman
	buildNumberRegex = regexp.MustCompile(`^(v?)(\d+)\.(\d+)-(\d+)(?:-([a-zA-Z][a-zA-Z0-9]*))?$`)

	// Pre-release label with counter: beta.1, alpha.3, rc.12
	preReleaseCounterRegex = regexp.MustCompile(`^([a-zA-Z]+)\.(\d+)$`)

	// Date build metadata: DDMMYYYY
	dateBuildMetadataRegex = regexp.MustCompile(`^\d{8}$`)
)

// parseVersion tries to parse a tag string into a Version.
// Returns the parsed version and true if successful, or zero Version and false.
func parseVersion(tag string) (Version, bool) {
	// Try semver first (more specific: 3 numeric components)
	if m := semverRegex.FindStringSubmatch(tag); m != nil {
		major, _ := strconv.Atoi(m[2])
		minor, _ := strconv.Atoi(m[3])
		patch, _ := strconv.Atoi(m[4])
		return Version{
			Major:      major,
			Minor:      minor,
			Patch:      patch,
			Prefix:     m[1],
			PreRelease: m[5],
			BuildMeta:  m[6],
			Raw:        tag,
			Pattern:    PatternSemver,
		}, true
	}

	// Try build-number pattern
	if m := buildNumberRegex.FindStringSubmatch(tag); m != nil {
		major, _ := strconv.Atoi(m[2])
		minor, _ := strconv.Atoi(m[3])
		build, _ := strconv.Atoi(m[4])
		return Version{
			Major:      major,
			Minor:      minor,
			Build:      build,
			Prefix:     m[1],
			Descriptor: m[5],
			Raw:        tag,
			Pattern:    PatternBuildNumber,
		}, true
	}

	return Version{}, false
}

// formatVersion produces the string representation of a Version
func formatVersion(v Version) string {
	switch v.Pattern {
	case PatternSemver:
		s := fmt.Sprintf("%s%d.%d.%d", v.Prefix, v.Major, v.Minor, v.Patch)
		if v.PreRelease != "" {
			s += "-" + v.PreRelease
		}
		if v.BuildMeta != "" {
			s += "+" + v.BuildMeta
		}
		return s
	case PatternBuildNumber:
		s := fmt.Sprintf("%s%d.%d-%d", v.Prefix, v.Major, v.Minor, v.Build)
		if v.Descriptor != "" {
			s += "-" + v.Descriptor
		}
		return s
	default:
		return v.Raw
	}
}

// compareVersions compares two versions.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Pre-release versions are considered less than the same version without pre-release.
func compareVersions(a, b Version) int {
	// Compare major
	if a.Major != b.Major {
		return intCmp(a.Major, b.Major)
	}

	// Compare minor
	if a.Minor != b.Minor {
		return intCmp(a.Minor, b.Minor)
	}

	// Pattern-specific comparison
	if a.Pattern == PatternSemver && b.Pattern == PatternSemver {
		if a.Patch != b.Patch {
			return intCmp(a.Patch, b.Patch)
		}
		// Pre-release comparison: no pre-release > has pre-release (semver spec)
		if cmp := comparePreRelease(a.PreRelease, b.PreRelease); cmp != 0 {
			return cmp
		}
		return compareBuildMetadata(a.BuildMeta, b.BuildMeta)
	}

	if a.Pattern == PatternBuildNumber && b.Pattern == PatternBuildNumber {
		if a.Build != b.Build {
			return intCmp(a.Build, b.Build)
		}
		// Descriptor doesn't affect ordering meaningfully
		return strings.Compare(a.Descriptor, b.Descriptor)
	}

	// Mixed patterns: fall back to string comparison
	return strings.Compare(a.Raw, b.Raw)
}

// comparePreRelease compares two pre-release strings per semver rules.
// Empty pre-release (stable) > any pre-release.
func comparePreRelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1 // stable > pre-release
	}
	if b == "" {
		return -1
	}

	// Split by dots and compare each component
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}

	for i := 0; i < minLen; i++ {
		aNum, aIsNum := strconv.Atoi(aParts[i])
		bNum, bIsNum := strconv.Atoi(bParts[i])

		if aIsNum == nil && bIsNum == nil {
			// Both numeric
			if aNum != bNum {
				return intCmp(aNum, bNum)
			}
		} else if aIsNum == nil {
			// Numeric < string (semver spec)
			return -1
		} else if bIsNum == nil {
			return 1
		} else {
			// Both strings
			cmp := strings.Compare(aParts[i], bParts[i])
			if cmp != 0 {
				return cmp
			}
		}
	}

	// Longer pre-release > shorter when all preceding components are equal
	return intCmp(len(aParts), len(bParts))
}

func compareBuildMetadata(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return -1
	}
	if b == "" {
		return 1
	}

	if ad, ok := parseDateBuildMetadata(a); ok {
		if bd, ok := parseDateBuildMetadata(b); ok {
			if ad.Before(bd) {
				return -1
			}
			if ad.After(bd) {
				return 1
			}
			return 0
		}
	}

	return strings.Compare(a, b)
}

func intCmp(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// sortVersions sorts a slice of versions in ascending order
func sortVersions(versions []Version) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) < 0
	})
}

// filterVersionTags takes raw tag strings and returns parsed Version structs,
// filtering out non-version tags.
func filterVersionTags(tags []string) []Version {
	var versions []Version
	for _, tag := range tags {
		if v, ok := parseVersion(tag); ok {
			versions = append(versions, v)
		}
	}
	sortVersions(versions)
	return versions
}

// detectPattern analyzes a list of versions and returns the dominant pattern.
// Returns PatternUnknown if no versions are provided.
func detectPattern(versions []Version) VersionPattern {
	if len(versions) == 0 {
		return PatternUnknown
	}

	semverCount := 0
	buildCount := 0

	for _, v := range versions {
		switch v.Pattern {
		case PatternSemver:
			semverCount++
		case PatternBuildNumber:
			buildCount++
		}
	}

	if semverCount >= buildCount {
		return PatternSemver
	}
	return PatternBuildNumber
}

// detectVersionLines groups versions into lines (e.g., v3.x, v4.x for semver
// or 8.3, 8.4 for build-number). Returns lines sorted by latest version ascending.
func detectVersionLines(versions []Version, pattern VersionPattern) []VersionLine {
	if len(versions) == 0 {
		return nil
	}

	lineMap := make(map[string]*VersionLine)

	for _, v := range versions {
		if v.Pattern != pattern {
			continue
		}

		var key string
		switch pattern {
		case PatternSemver:
			key = fmt.Sprintf("%d", v.Major)
		case PatternBuildNumber:
			key = fmt.Sprintf("%d.%d", v.Major, v.Minor)
		default:
			continue
		}

		if line, ok := lineMap[key]; ok {
			line.Versions = append(line.Versions, v)
			if compareVersions(v, line.Latest) > 0 {
				line.Latest = v
			}
		} else {
			lineMap[key] = &VersionLine{
				Major:    v.Major,
				Minor:    v.Minor,
				Pattern:  pattern,
				Latest:   v,
				Versions: []Version{v},
			}
		}
	}

	// Collect and sort lines by latest version
	var lines []VersionLine
	for _, line := range lineMap {
		lines = append(lines, *line)
	}
	sort.Slice(lines, func(i, j int) bool {
		return compareVersions(lines[i].Latest, lines[j].Latest) < 0
	})

	return lines
}

// bumpVersion creates a new version based on the bump type.
// For BumpPreRelease, preReleaseLabel should be provided (e.g., "beta.1").
// For BumpDescriptor, newDescriptor should be provided.
func bumpVersion(v Version, bump BumpType, preReleaseLabel, newDescriptor string) Version {
	next := Version{
		Prefix:  v.Prefix,
		Pattern: v.Pattern,
	}

	switch v.Pattern {
	case PatternSemver:
		next = bumpSemver(v, bump, preReleaseLabel)
	case PatternBuildNumber:
		next = bumpBuildNumber(v, bump, newDescriptor)
	}

	next.Raw = formatVersion(next)
	return next
}

func bumpSemver(v Version, bump BumpType, preReleaseLabel string) Version {
	next := Version{
		Major:     v.Major,
		Minor:     v.Minor,
		Patch:     v.Patch,
		Prefix:    v.Prefix,
		BuildMeta: nextSemverBuildMetadata(v),
		Pattern:   PatternSemver,
	}

	switch bump {
	case BumpPatch:
		next.Patch = v.Patch + 1
		next.PreRelease = ""
	case BumpMinor:
		next.Minor = v.Minor + 1
		next.Patch = 0
		next.PreRelease = ""
	case BumpMajor:
		next.Major = v.Major + 1
		next.Minor = 0
		next.Patch = 0
		next.PreRelease = ""
	case BumpRelease:
		// Strip pre-release
		next.PreRelease = ""
	case BumpPreRelease:
		if preReleaseLabel != "" {
			// Use the provided label directly
			next.PreRelease = preReleaseLabel
		} else if v.HasPreRelease() {
			// Bump existing pre-release counter
			next.PreRelease = bumpPreReleaseCounter(v.PreRelease)
			next.BuildMeta = v.BuildMeta
		} else {
			// Start new pre-release on next patch
			next.Patch = v.Patch + 1
			next.PreRelease = "beta.1"
		}
	default:
		// For custom/unknown, just return a copy
		next.PreRelease = v.PreRelease
		next.BuildMeta = v.BuildMeta
	}

	return next
}

func nextSemverBuildMetadata(v Version) string {
	if v.HasDateBuildMetadata() {
		return currentDateBuildMetadata()
	}
	return v.BuildMeta
}

func currentDateBuildMetadata() string {
	return timeNow().Format("02012006")
}

func isDateBuildMetadata(value string) bool {
	return dateBuildMetadataRegex.MatchString(value)
}

func parseDateBuildMetadata(value string) (time.Time, bool) {
	if !isDateBuildMetadata(value) {
		return time.Time{}, false
	}

	parsed, err := time.Parse("02012006", value)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func bumpBuildNumber(v Version, bump BumpType, newDescriptor string) Version {
	next := Version{
		Major:      v.Major,
		Minor:      v.Minor,
		Build:      v.Build,
		Prefix:     v.Prefix,
		Descriptor: v.Descriptor,
		Pattern:    PatternBuildNumber,
	}

	switch bump {
	case BumpBuild:
		next.Build = v.Build + 1
		// Keep descriptor from current version
	case BumpMinor:
		next.Minor = v.Minor + 1
		next.Build = 1
		next.Descriptor = ""
	case BumpMajor:
		next.Major = v.Major + 1
		next.Minor = 0
		next.Build = 1
		next.Descriptor = ""
	case BumpDescriptor:
		next.Build = v.Build + 1
		next.Descriptor = newDescriptor
	case BumpStripDescriptor:
		next.Build = v.Build + 1
		next.Descriptor = ""
	default:
		// Copy as-is for custom/unknown
	}

	return next
}

// bumpPreReleaseCounter increments the numeric counter in a pre-release label.
// e.g., "beta.1" -> "beta.2", "rc.3" -> "rc.4"
// If no counter is found, appends ".1": "beta" -> "beta.1"
func bumpPreReleaseCounter(pre string) string {
	if m := preReleaseCounterRegex.FindStringSubmatch(pre); m != nil {
		label := m[1]
		counter, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s.%d", label, counter+1)
	}
	// No counter found, start at 1
	return pre + ".1"
}

// parsePreReleaseLabel extracts the label name and counter from a pre-release string.
// e.g., "beta.1" -> ("beta", 1, true), "beta" -> ("beta", 0, false)
func parsePreReleaseLabel(pre string) (label string, counter int, hasCounter bool) {
	if m := preReleaseCounterRegex.FindStringSubmatch(pre); m != nil {
		c, _ := strconv.Atoi(m[2])
		return m[1], c, true
	}
	return pre, 0, false
}

// buildBumpChoices generates the bump menu choices for the given version and pattern.
func buildBumpChoices(latest Version, pattern VersionPattern) []Choice {
	var choices []Choice

	switch pattern {
	case PatternSemver:
		choices = buildSemverChoices(latest)
	case PatternBuildNumber:
		choices = buildBuildNumberChoices(latest)
	}

	// Always add custom as the last option
	choices = append(choices, Choice{
		Label:       "Custom",
		Description: "Enter a tag manually",
		BumpType:    BumpCustom,
	})

	return choices
}

func buildSemverChoices(v Version) []Choice {
	var choices []Choice

	if v.HasPreRelease() {
		// If current version has pre-release, offer release (strip) first
		released := bumpVersion(v, BumpRelease, "", "")
		choices = append(choices, Choice{
			Label:       "Release (stable)",
			Description: "Strip pre-release suffix",
			Preview:     formatVersion(released),
			BumpType:    BumpRelease,
		})

		// Bump pre-release counter
		bumped := bumpVersion(v, BumpPreRelease, "", "")
		choices = append(choices, Choice{
			Label:       "Bump pre-release",
			Description: "Increment pre-release counter",
			Preview:     formatVersion(bumped),
			BumpType:    BumpPreRelease,
		})
	}

	// Standard bumps
	patch := bumpVersion(v, BumpPatch, "", "")
	minor := bumpVersion(v, BumpMinor, "", "")
	major := bumpVersion(v, BumpMajor, "", "")

	choices = append(choices,
		Choice{
			Label:       "Patch",
			Description: semverBumpDescription(v, patch),
			Preview:     formatVersion(patch),
			BumpType:    BumpPatch,
		},
		Choice{
			Label:       "Minor",
			Description: semverBumpDescription(v, minor),
			Preview:     formatVersion(minor),
			BumpType:    BumpMinor,
		},
		Choice{
			Label:       "Major",
			Description: semverBumpDescription(v, major),
			Preview:     formatVersion(major),
			BumpType:    BumpMajor,
		},
	)

	return choices
}

func semverBumpDescription(current, next Version) string {
	description := fmt.Sprintf("%d.%d.%d -> %d.%d.%d", current.Major, current.Minor, current.Patch, next.Major, next.Minor, next.Patch)
	if current.HasDateBuildMetadata() && next.BuildMeta != "" {
		description += fmt.Sprintf(" +%s", next.BuildMeta)
	}
	return description
}

// buildReleaseTypeChoices generates the stable vs pre-release choice after selecting a bump type.
func buildReleaseTypeChoices(bumped Version) []Choice {
	return []Choice{
		{
			Label:       "Stable release",
			Description: "No pre-release suffix",
			Preview:     formatVersion(bumped),
			BumpType:    BumpRelease,
		},
		{
			Label:       "Pre-release",
			Description: "Add a pre-release label",
			Preview:     formatVersion(bumped) + "-???.1",
			BumpType:    BumpPreRelease,
		},
	}
}

// buildPreReleaseLabelChoicesForVersion generates pre-release label choices for a specific version.
// Unlike buildPreReleaseLabelChoices, this does not bump the patch -- it uses the version as-is.
func buildPreReleaseLabelChoicesForVersion(v Version) []Choice {
	tag := formatVersion(v)
	return []Choice{
		{
			Label:       "alpha",
			Description: "Early development, unstable",
			Preview:     tag + "-alpha.1",
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "beta",
			Description: "Feature complete, may have bugs",
			Preview:     tag + "-beta.1",
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "rc",
			Description: "Release candidate, final testing",
			Preview:     tag + "-rc.1",
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "Custom",
			Description: "Enter a custom pre-release label",
			BumpType:    BumpCustom,
		},
	}
}

func buildBuildNumberChoices(v Version) []Choice {
	var choices []Choice

	// Build bump (most common)
	build := bumpVersion(v, BumpBuild, "", "")
	choices = append(choices, Choice{
		Label:       "Build bump",
		Description: fmt.Sprintf("%d.%d-%d -> %d.%d-%d", v.Major, v.Minor, v.Build, build.Major, build.Minor, build.Build),
		Preview:     formatVersion(build),
		BumpType:    BumpBuild,
	})

	// Minor bump
	minor := bumpVersion(v, BumpMinor, "", "")
	choices = append(choices, Choice{
		Label:       "Minor bump",
		Description: fmt.Sprintf("%d.%d -> %d.%d", v.Major, v.Minor, minor.Major, minor.Minor),
		Preview:     formatVersion(minor),
		BumpType:    BumpMinor,
	})

	// Major bump
	major := bumpVersion(v, BumpMajor, "", "")
	choices = append(choices, Choice{
		Label:       "Major bump",
		Description: fmt.Sprintf("%d.x -> %d.0", v.Major, major.Major),
		Preview:     formatVersion(major),
		BumpType:    BumpMajor,
	})

	if v.HasDescriptor() {
		// Change descriptor
		choices = append(choices, Choice{
			Label:       "Change descriptor",
			Description: fmt.Sprintf("Change \"%s\" to something else", v.Descriptor),
			Preview:     "(enter descriptor next)",
			BumpType:    BumpDescriptor,
		})

		// Strip descriptor
		stripped := bumpVersion(v, BumpStripDescriptor, "", "")
		choices = append(choices, Choice{
			Label:       "Strip descriptor",
			Description: fmt.Sprintf("Remove \"%s\" suffix", v.Descriptor),
			Preview:     formatVersion(stripped),
			BumpType:    BumpStripDescriptor,
		})
	} else {
		// Add descriptor
		choices = append(choices, Choice{
			Label:       "Add descriptor",
			Description: "Add a descriptor suffix",
			Preview:     "(enter descriptor next)",
			BumpType:    BumpDescriptor,
		})
	}

	return choices
}

// buildPreReleaseLabelChoices generates choices for selecting a pre-release label.
func buildPreReleaseLabelChoices(v Version) []Choice {
	nextPatch := v.Patch + 1
	prefix := v.Prefix

	return []Choice{
		{
			Label:       "alpha",
			Description: "Early development, unstable",
			Preview:     fmt.Sprintf("%s%d.%d.%d-alpha.1", prefix, v.Major, v.Minor, nextPatch),
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "beta",
			Description: "Feature complete, may have bugs",
			Preview:     fmt.Sprintf("%s%d.%d.%d-beta.1", prefix, v.Major, v.Minor, nextPatch),
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "rc",
			Description: "Release candidate, final testing",
			Preview:     fmt.Sprintf("%s%d.%d.%d-rc.1", prefix, v.Major, v.Minor, nextPatch),
			BumpType:    BumpPreRelease,
		},
		{
			Label:       "Custom",
			Description: "Enter a custom pre-release label",
			BumpType:    BumpCustom,
		},
	}
}

// buildFirstReleaseChoices generates template choices for the first release
func buildFirstReleaseChoices() []Choice {
	return []Choice{
		{
			Label:       "vX.Y.Z (semver with v prefix)",
			Description: "Most common, e.g., v0.1.0, v1.0.0",
			Preview:     "v0.1.0",
			BumpType:    BumpCustom,
		},
		{
			Label:       "X.Y.Z (semver without v prefix)",
			Description: "Plain semver, e.g., 0.1.0, 1.0.0",
			Preview:     "0.1.0",
			BumpType:    BumpCustom,
		},
		{
			Label:       "vX.Y.Z-beta.1 (semver pre-release)",
			Description: "Start with a pre-release, e.g., v0.1.0-beta.1",
			Preview:     "v0.1.0-beta.1",
			BumpType:    BumpCustom,
		},
		{
			Label:       "vX.Y.Z+DDMMYYYY (semver with date build metadata)",
			Description: "Semver with a date suffix, e.g., v1.10.11+24032026",
			Preview:     "v0.1.0+" + currentDateBuildMetadata(),
			BumpType:    BumpCustom,
		},
		{
			Label:       "X.Y-N (build number)",
			Description: "Build number scheme, e.g., 1.0-1",
			Preview:     "1.0-1",
			BumpType:    BumpCustom,
		},
		{
			Label:       "Custom",
			Description: "Enter any tag manually",
			BumpType:    BumpCustom,
		},
	}
}

// tagExists checks if a tag string already exists in the list of versions
func tagExists(tag string, versions []Version) bool {
	for _, v := range versions {
		if v.Raw == tag {
			return true
		}
	}
	return false
}

// latestVersion returns the highest version from a sorted slice.
// Returns zero Version if empty.
func latestVersion(versions []Version) (Version, bool) {
	if len(versions) == 0 {
		return Version{}, false
	}
	return versions[len(versions)-1], true
}

// latestVersionForPattern returns the highest version matching a specific pattern.
func latestVersionForPattern(versions []Version, pattern VersionPattern) (Version, bool) {
	var filtered []Version
	for _, v := range versions {
		if v.Pattern == pattern {
			filtered = append(filtered, v)
		}
	}
	return latestVersion(filtered)
}

// detectPrefix determines the dominant prefix (e.g., "v" or "") from a list of versions.
func detectPrefix(versions []Version) string {
	vCount := 0
	noCount := 0
	for _, v := range versions {
		if v.Prefix == "v" {
			vCount++
		} else {
			noCount++
		}
	}
	if vCount >= noCount {
		return "v"
	}
	return ""
}
