package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FlutterProjectInfo struct {
	Detected    bool
	PubspecPath string
	Version     string
	HasVersion  bool
}

var pubspecVersionLineRegex = regexp.MustCompile(`(?m)^(\s*version:\s*)(['"]?)([^'"\r\n#]+)(['"]?)(\s*(?:#.*)?)$`)

func detectFlutterProject(rootPath string) (FlutterProjectInfo, error) {
	for _, name := range []string{"pubspec.yaml", "pubspec.yml"} {
		path := filepath.Join(rootPath, name)
		info, err := readFlutterPubspec(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return FlutterProjectInfo{}, err
		}
		return info, nil
	}

	return FlutterProjectInfo{}, nil
}

func readFlutterPubspec(path string) (FlutterProjectInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return FlutterProjectInfo{}, err
	}

	info := FlutterProjectInfo{
		Detected:    true,
		PubspecPath: path,
	}

	matches := pubspecVersionLineRegex.FindSubmatch(content)
	if len(matches) >= 5 {
		info.Version = strings.TrimSpace(string(matches[3]))
		info.HasVersion = info.Version != ""
	}

	return info, nil
}

func flutterTargetVersionForTag(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

func shouldPromptFlutterVersionUpdate(info FlutterProjectInfo, target string) bool {
	if !info.Detected || !info.HasVersion {
		return false
	}

	target = strings.TrimSpace(target)
	return target != ""
}

func needsFlutterVersionUpdate(info FlutterProjectInfo, target string) bool {
	if !info.Detected || !info.HasVersion {
		return false
	}

	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	return info.Version != target
}

func updateFlutterPubspecVersion(path, newVersion string) error {
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
