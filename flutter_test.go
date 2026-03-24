package main

import "testing"

func TestShouldPromptFlutterVersionUpdateWhenFlutterProjectDetected(t *testing.T) {
	info := FlutterProjectInfo{
		Detected:   true,
		HasVersion: true,
		Version:    "1.10.12+24032026",
	}

	if !shouldPromptFlutterVersionUpdate(info, "1.10.12+24032026") {
		t.Fatal("expected Flutter confirmation step when target version is available")
	}
}

func TestNeedsFlutterVersionUpdateOnlyWhenVersionDiffers(t *testing.T) {
	info := FlutterProjectInfo{
		Detected:   true,
		HasVersion: true,
		Version:    "1.10.12+24032026",
	}

	if needsFlutterVersionUpdate(info, "1.10.12+24032026") {
		t.Fatal("did not expect update when pubspec already matches target")
	}

	if !needsFlutterVersionUpdate(info, "1.10.13+24032026") {
		t.Fatal("expected update when pubspec differs from target")
	}
}

func TestBuildReleasePlanAddsFlutterCommitStepBeforeTag(t *testing.T) {
	plan := BuildReleasePlan(
		"v1.10.12+24032026",
		"Release v1.10.12+24032026",
		"",
		false,
		&FlutterVersionUpdate{
			Path:           "/tmp/pubspec.yaml",
			CurrentVersion: "1.10.10+19032026",
			NewVersion:     "1.10.12+24032026",
			CommitMessage:  "chore: bump version to 1.10.12+24032026",
		},
		"main",
		[]RemoteInfo{{Name: "origin"}},
		nil,
		false,
	)

	if len(plan.Steps) < 4 {
		t.Fatalf("expected at least 4 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].Type != ExecUpdateFlutterPubspec {
		t.Fatalf("expected first step to update pubspec, got %v", plan.Steps[0].Type)
	}

	if plan.Steps[1].Type != ExecCommitFlutterPubspec {
		t.Fatalf("expected second step to commit pubspec, got %v", plan.Steps[1].Type)
	}

	if plan.Steps[2].Type != ExecPushBranch {
		t.Fatalf("expected third step to push branch, got %v", plan.Steps[2].Type)
	}

	if plan.Steps[3].Type != ExecCreateTag {
		t.Fatalf("expected fourth step to create tag, got %v", plan.Steps[3].Type)
	}
}
