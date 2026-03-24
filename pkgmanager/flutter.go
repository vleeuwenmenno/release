package pkgmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FlutterManager detects and updates Flutter pubspec.yaml version fields.
type FlutterManager struct{}

func init() {
	Register(&FlutterManager{})
}

func (f *FlutterManager) Name() string { return "Flutter" }

var pubspecVersionLineRegex = regexp.MustCompile(`(?m)^(\s*version:\s*)(['"]?)([^'"\r\n#]+)(['"]?)(\s*(?:#.*)?)$`)

func (f *FlutterManager) Detect(rootPath string) (*ProjectInfo, error) {
	for _, name := range []string{"pubspec.yaml", "pubspec.yml"} {
		path := filepath.Join(rootPath, name)
		info, err := f.readPubspec(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return info, nil
	}
	return nil, nil
}

func (f *FlutterManager) readPubspec(path string) (*ProjectInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := &ProjectInfo{
		Detected: true,
		FilePath: path,
		Manager:  f,
	}

	matches := pubspecVersionLineRegex.FindSubmatch(content)
	if len(matches) >= 5 {
		info.Version = strings.TrimSpace(string(matches[3]))
		info.HasVersion = info.Version != ""
	}

	return info, nil
}

func (f *FlutterManager) TargetVersionForTag(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

func (f *FlutterManager) ShouldPromptUpdate(info *ProjectInfo, target string) bool {
	if info == nil || !info.Detected || !info.HasVersion {
		return false
	}
	return strings.TrimSpace(target) != ""
}

func (f *FlutterManager) NeedsUpdate(info *ProjectInfo, target string) bool {
	if info == nil || !info.Detected || !info.HasVersion {
		return false
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	return info.Version != target
}

func (f *FlutterManager) UpdateVersion(path, newVersion string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	replaced := false
	updated := pubspecVersionLineRegex.ReplaceAllStringFunc(string(content), func(line string) string {
		if replaced {
			return line
		}
		match := pubspecVersionLineRegex.FindStringSubmatch(line)
		if len(match) != 6 {
			return line
		}
		replaced = true
		return match[1] + match[2] + newVersion + match[4] + match[5]
	})

	if !replaced {
		return fmt.Errorf("no version field found in %s", path)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(updated), stat.Mode()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
