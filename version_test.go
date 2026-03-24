package main

import (
	"testing"
	"time"
)

func TestParseVersionSupportsDateBuildMetadata(t *testing.T) {
	version, ok := parseVersion("v1.10.11+24032026")
	if !ok {
		t.Fatal("expected version to parse")
	}

	if version.Pattern != PatternSemver {
		t.Fatalf("expected semver pattern, got %v", version.Pattern)
	}

	if version.BuildMeta != "24032026" {
		t.Fatalf("expected build metadata 24032026, got %q", version.BuildMeta)
	}

	if formatVersion(version) != "v1.10.11+24032026" {
		t.Fatalf("unexpected formatted version %q", formatVersion(version))
	}
}

func TestCompareVersionsSortsDateBuildMetadataChronologically(t *testing.T) {
	a, _ := parseVersion("v1.0.11+31032025")
	b, _ := parseVersion("v1.0.11+01042025")

	if compareVersions(a, b) >= 0 {
		t.Fatalf("expected %q to sort before %q", a.Raw, b.Raw)
	}
}

func TestBumpSemverRefreshesDateBuildMetadata(t *testing.T) {
	originalNow := timeNow
	timeNow = func() time.Time {
		return time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC)
	}
	defer func() {
		timeNow = originalNow
	}()

	version, _ := parseVersion("v1.10.11+19032026")
	bumped := bumpVersion(version, BumpPatch, "", "")

	if bumped.Raw != "v1.10.12+24032026" {
		t.Fatalf("expected bumped version v1.10.12+24032026, got %q", bumped.Raw)
	}
}

func TestFlutterTargetVersionKeepsBuildMetadata(t *testing.T) {
	got := flutterTargetVersionForTag("v1.10.12+24032026")
	if got != "1.10.12+24032026" {
		t.Fatalf("expected flutter target to preserve build metadata, got %q", got)
	}
}
