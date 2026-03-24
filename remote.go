package main

import (
	"os/exec"
	"strings"
)

// ForgeType represents the type of git forge hosting the remote
type ForgeType int

const (
	ForgeUnknown ForgeType = iota
	ForgeGitHub
	ForgeGitLab
	ForgeGitea
)

func (f ForgeType) String() string {
	switch f {
	case ForgeGitHub:
		return "GitHub"
	case ForgeGitLab:
		return "GitLab"
	case ForgeGitea:
		return "Gitea"
	default:
		return "Unknown"
	}
}

// CLITool returns the CLI tool name for this forge type
func (f ForgeType) CLITool() string {
	switch f {
	case ForgeGitHub:
		return "gh"
	case ForgeGitLab:
		return "glab"
	case ForgeGitea:
		return "tea"
	default:
		return ""
	}
}

// RemoteInfo holds information about a git remote
type RemoteInfo struct {
	Name     string
	URL      string
	Forge    ForgeType
	HasCLI   bool
	Selected bool // whether user selected this remote for push/release
}

// Label returns a display label like "origin (github.com/user/repo)"
func (r RemoteInfo) Label() string {
	shortURL := r.ShortURL()
	if shortURL != "" {
		return r.Name + " (" + shortURL + ")"
	}
	return r.Name
}

// ShortURL strips protocol and .git suffix for display
func (r RemoteInfo) ShortURL() string {
	url := r.URL

	// Strip protocol
	for _, prefix := range []string{"https://", "http://", "ssh://", "git@", "git://"} {
		url = strings.TrimPrefix(url, prefix)
	}

	// Handle git@host:path format -> host/path
	url = strings.Replace(url, ":", "/", 1)

	// Strip .git suffix
	url = strings.TrimSuffix(url, ".git")

	return url
}

// detectRemotes discovers git remotes and classifies them by forge type.
func detectRemotes() ([]RemoteInfo, error) {
	out, err := gitExec("remote", "-v")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	// Parse "origin\thttps://github.com/user/repo.git (fetch)" lines.
	// We only care about fetch URLs (push URLs are usually the same).
	seen := make(map[string]bool)
	var remotes []RemoteInfo

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Only process fetch lines to avoid duplicates
		if !strings.HasSuffix(line, "(fetch)") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		url := parts[1]

		if seen[name] {
			continue
		}
		seen[name] = true

		forge := classifyForge(url)
		remote := RemoteInfo{
			Name:   name,
			URL:    url,
			Forge:  forge,
			HasCLI: hasCLI(forge.CLITool()),
		}
		remotes = append(remotes, remote)
	}

	return remotes, nil
}

// classifyForge determines the forge type from a remote URL.
func classifyForge(url string) ForgeType {
	lower := strings.ToLower(url)

	if strings.Contains(lower, "github.com") {
		return ForgeGitHub
	}
	if strings.Contains(lower, "gitlab.com") || strings.Contains(lower, "gitlab.") {
		return ForgeGitLab
	}

	// For non-GitHub/GitLab remotes, assume Gitea if tea is available.
	// This covers self-hosted Gitea/Forgejo instances.
	return ForgeGitea
}

// hasCLI checks if a CLI tool is available in PATH.
func hasCLI(name string) bool {
	if name == "" {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// shouldMarkPreRelease returns true only when the user explicitly selected
// pre-release during the interactive flow.
func shouldMarkPreRelease(preReleaseExplicit bool) bool {
	return preReleaseExplicit
}

// createForgeRelease creates a release on the given forge using the appropriate CLI tool.
func createForgeRelease(remote RemoteInfo, tag, title, notes string, preReleaseExplicit bool) error {
	switch remote.Forge {
	case ForgeGitHub:
		return createGitHubRelease(tag, title, notes, preReleaseExplicit)
	case ForgeGitLab:
		return createGitLabRelease(tag, title, notes, preReleaseExplicit)
	case ForgeGitea:
		return createGiteaRelease(tag, title, notes, preReleaseExplicit)
	default:
		return nil
	}
}

// createGitHubRelease creates a release via the gh CLI.
func createGitHubRelease(tag, title, notes string, preReleaseExplicit bool) error {
	args := []string{"release", "create", tag, "--title", title}
	if shouldMarkPreRelease(preReleaseExplicit) {
		args = append(args, "--prerelease")
	}
	if notes != "" {
		args = append(args, "--notes", notes)
	} else {
		args = append(args, "--generate-notes")
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return formatExecError("gh", out, err)
	}
	return nil
}

// createGitLabRelease creates a release via the glab CLI.
func createGitLabRelease(tag, title, notes string, preReleaseExplicit bool) error {
	args := []string{"release", "create", tag, "--name", title}
	if shouldMarkPreRelease(preReleaseExplicit) {
		args = append(args, "--pre-release")
	}
	if notes != "" {
		args = append(args, "--notes", notes)
	}

	cmd := exec.Command("glab", args...)
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return formatExecError("glab", out, err)
	}
	return nil
}

// createGiteaRelease creates a release via the tea CLI.
func createGiteaRelease(tag, title, notes string, preReleaseExplicit bool) error {
	args := []string{"release", "create", "--tag", tag, "--title", title}
	if shouldMarkPreRelease(preReleaseExplicit) {
		args = append(args, "--prerelease")
	}
	if notes != "" {
		args = append(args, "--note", notes)
	}

	cmd := exec.Command("tea", args...)
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return formatExecError("tea", out, err)
	}
	return nil
}

// formatExecError creates a readable error from a command's combined output.
func formatExecError(tool string, output []byte, err error) error {
	msg := strings.TrimSpace(string(output))
	if msg != "" {
		return &execError{tool: tool, message: msg}
	}
	return &execError{tool: tool, message: err.Error()}
}

type execError struct {
	tool    string
	message string
}

func (e *execError) Error() string {
	return e.tool + ": " + e.message
}

// releaseCapableRemotes returns remotes that have a forge CLI available.
func releaseCapableRemotes(remotes []RemoteInfo) []RemoteInfo {
	var capable []RemoteInfo
	for _, r := range remotes {
		if r.HasCLI && r.Forge != ForgeUnknown {
			capable = append(capable, r)
		}
	}
	return capable
}

// generateReleaseNotes creates auto-generated release notes from commits since the previous tag.
func generateReleaseNotes(prevTag string) string {
	commits, err := getCommitsSinceTag(prevTag)
	if err != nil || len(commits) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## What's Changed\n\n")
	for _, commit := range commits {
		b.WriteString("* " + commit + "\n")
	}
	return b.String()
}