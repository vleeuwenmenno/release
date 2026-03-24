package pkgmanager

// Manager defines how to detect and update a version in a project's package file.
// Each supported language/ecosystem implements this interface.
type Manager interface {
	// Name returns a human-readable name (e.g., "Flutter", "Node.js").
	Name() string

	// Detect checks if this package type exists at rootPath.
	// Returns nil when the project type is not found.
	Detect(rootPath string) (*ProjectInfo, error)

	// TargetVersionForTag converts a git tag to the version format used by this package type.
	TargetVersionForTag(tag string) string

	// ShouldPromptUpdate returns true if the user should be prompted about a version update.
	ShouldPromptUpdate(info *ProjectInfo, target string) bool

	// NeedsUpdate returns true if the current version differs from the target.
	NeedsUpdate(info *ProjectInfo, target string) bool

	// UpdateVersion writes the new version to the file at the given path.
	UpdateVersion(filePath, newVersion string) error
}

// ProjectInfo holds detected project information for any package manager.
type ProjectInfo struct {
	Detected   bool
	FilePath   string
	Version    string
	HasVersion bool
	Manager    Manager
}

// VersionUpdate holds information needed to update a package version file.
type VersionUpdate struct {
	Path           string
	CurrentVersion string
	NewVersion     string
	CommitMessage  string
	Manager        Manager
}

var registry []Manager

// Register adds a package manager to the detection registry.
func Register(m Manager) {
	registry = append(registry, m)
}

// DetectAll runs all registered managers and returns the first detected project, or nil.
func DetectAll(rootPath string) (*ProjectInfo, error) {
	for _, m := range registry {
		info, err := m.Detect(rootPath)
		if err != nil {
			return nil, err
		}
		if info != nil && info.Detected {
			return info, nil
		}
	}
	return nil, nil
}
