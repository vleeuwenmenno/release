package main

import (
	"testing"

	"github.com/vleeuwenmenno/release/pkgmanager"
)

func TestBuildReleasePlanAddsPackageCommitStepBeforeTag(t *testing.T) {
	plan := BuildReleasePlan(
		"v1.10.12+24032026",
		"Release v1.10.12+24032026",
		"",
		false,
		&pkgmanager.VersionUpdate{
			Path:           "/tmp/pubspec.yaml",
			CurrentVersion: "1.10.10+19032026",
			NewVersion:     "1.10.12+24032026",
			CommitMessage:  "chore: bump version to 1.10.12+24032026",
			Manager:        &stubManager{},
		},
		"main",
		[]RemoteInfo{{Name: "origin"}},
		nil,
		false,
	)

	if len(plan.Steps) < 4 {
		t.Fatalf("expected at least 4 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].Type != ExecUpdatePackageVersion {
		t.Fatalf("expected first step to update package version, got %v", plan.Steps[0].Type)
	}

	if plan.Steps[1].Type != ExecCommitPackageVersion {
		t.Fatalf("expected second step to commit package version, got %v", plan.Steps[1].Type)
	}

	if plan.Steps[2].Type != ExecPushBranch {
		t.Fatalf("expected third step to push branch, got %v", plan.Steps[2].Type)
	}

	if plan.Steps[3].Type != ExecCreateTag {
		t.Fatalf("expected fourth step to create tag, got %v", plan.Steps[3].Type)
	}
}

// stubManager is a minimal pkgmanager.Manager implementation for testing.
type stubManager struct{}

func (s *stubManager) Name() string                                            { return "Stub" }
func (s *stubManager) Detect(rootPath string) (*pkgmanager.ProjectInfo, error) { return nil, nil }
func (s *stubManager) TargetVersionForTag(tag string) string                   { return tag }
func (s *stubManager) ShouldPromptUpdate(info *pkgmanager.ProjectInfo, target string) bool {
	return false
}
func (s *stubManager) NeedsUpdate(info *pkgmanager.ProjectInfo, target string) bool { return false }
func (s *stubManager) UpdateVersion(filePath, newVersion string) error              { return nil }
