package pkgmanager

import "testing"

func TestFlutterShouldPromptUpdateWhenDetected(t *testing.T) {
	f := &FlutterManager{}
	info := &ProjectInfo{
		Detected:   true,
		HasVersion: true,
		Version:    "1.10.12+24032026",
	}

	if !f.ShouldPromptUpdate(info, "1.10.12+24032026") {
		t.Fatal("expected prompt when target version is available")
	}
}

func TestFlutterNeedsUpdateOnlyWhenVersionDiffers(t *testing.T) {
	f := &FlutterManager{}
	info := &ProjectInfo{
		Detected:   true,
		HasVersion: true,
		Version:    "1.10.12+24032026",
	}

	if f.NeedsUpdate(info, "1.10.12+24032026") {
		t.Fatal("did not expect update when pubspec already matches target")
	}

	if !f.NeedsUpdate(info, "1.10.13+24032026") {
		t.Fatal("expected update when pubspec differs from target")
	}
}

func TestFlutterTargetVersionForTagStripsVPrefix(t *testing.T) {
	f := &FlutterManager{}
	got := f.TargetVersionForTag("v1.10.12+24032026")
	if got != "1.10.12+24032026" {
		t.Fatalf("expected flutter target to preserve build metadata, got %q", got)
	}
}
