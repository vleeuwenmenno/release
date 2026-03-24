package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// RepoInfo holds information about the current git repository
type RepoInfo struct {
	Name        string
	RootPath    string
	Branch      string
	HeadCommit  string
	HeadMessage string
	IsDirty     bool
	DirtyCount  int
	IsDetached  bool
}

// gitExec runs a git command and returns trimmed stdout.
func gitExec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// isGitRepo checks if the current directory is inside a git repository.
func isGitRepo() bool {
	_, err := gitExec("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// getRepoInfo collects information about the current git repository.
func getRepoInfo() (RepoInfo, error) {
	info := RepoInfo{}

	// Repository name (from top-level directory)
	topLevel, err := gitExec("rev-parse", "--show-toplevel")
	if err != nil {
		return info, fmt.Errorf("not a git repository: %w", err)
	}
	parts := strings.Split(topLevel, "/")
	if len(parts) > 0 {
		info.Name = parts[len(parts)-1]
	}
	info.RootPath = topLevel

	// Current branch or detached state
	branch, err := gitExec("symbolic-ref", "--short", "HEAD")
	if err != nil {
		// Likely detached HEAD
		info.IsDetached = true
		hash, hashErr := gitExec("rev-parse", "--short", "HEAD")
		if hashErr != nil {
			return info, fmt.Errorf("cannot determine HEAD: %w", hashErr)
		}
		info.Branch = "detached at " + hash
	} else {
		info.Branch = branch
	}

	// HEAD commit short hash
	commit, err := gitExec("rev-parse", "--short", "HEAD")
	if err != nil {
		return info, fmt.Errorf("cannot get HEAD commit: %w", err)
	}
	info.HeadCommit = commit

	// HEAD commit message (first line)
	message, err := gitExec("log", "-1", "--format=%s")
	if err != nil {
		info.HeadMessage = "(unknown)"
	} else {
		info.HeadMessage = message
	}

	// Dirty state
	status, err := gitExec("status", "--porcelain")
	if err == nil && status != "" {
		info.IsDirty = true
		lines := strings.Split(status, "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		info.DirtyCount = count
	}

	return info, nil
}

// fetchTags fetches tags from all remotes.
func fetchTags() error {
	_, err := gitExec("fetch", "--tags", "--all", "--quiet")
	return err
}

// getAllTags returns all tag names in the repository.
func getAllTags() ([]string, error) {
	out, err := gitExec("tag", "--list")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	var tags []string
	for _, line := range lines {
		tag := strings.TrimSpace(line)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// createAnnotatedTag creates a signed annotated git tag at HEAD.
func createAnnotatedTag(tag, message string) error {
	_, err := gitExec("tag", "-s", tag, "-m", message)
	if err != nil {
		return fmt.Errorf("failed to create tag %s: %w", tag, err)
	}
	return nil
}

// commitTrackedFile creates a commit for the current working tree contents of a tracked file.
func commitTrackedFile(path, message string) error {
	_, err := gitExec("commit", "--only", "-m", message, "--", path)
	if err != nil {
		return fmt.Errorf("failed to commit %s: %w", path, err)
	}
	return nil
}

// deleteTag deletes a local git tag.
func deleteTag(tag string) error {
	_, err := gitExec("tag", "-d", tag)
	return err
}

// pushTag pushes a specific tag to a remote.
func pushTag(tag, remote string) error {
	_, err := gitExec("push", remote, tag)
	if err != nil {
		return fmt.Errorf("failed to push tag %s to %s: %w", tag, remote, err)
	}
	return nil
}

// pushBranch pushes a local branch to a remote.
func pushBranch(branch, remote string) error {
	_, err := gitExec("push", remote, branch)
	if err != nil {
		return fmt.Errorf("failed to push branch %s to %s: %w", branch, remote, err)
	}
	return nil
}

// getCommitsSinceTag returns a list of one-line commit summaries since the given tag.
// If tag is empty, returns all commits.
func getCommitsSinceTag(tag string) ([]string, error) {
	var out string
	var err error

	if tag == "" {
		out, err = gitExec("log", "--oneline", "--no-decorate")
	} else {
		out, err = gitExec("log", "--oneline", "--no-decorate", tag+"..HEAD")
	}
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	var commits []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits, nil
}

// getTagCount returns the total number of tags in the repository.
func getTagCount() (int, error) {
	tags, err := getAllTags()
	if err != nil {
		return 0, err
	}
	return len(tags), nil
}

// tagExistsInRepo checks if a tag already exists in the repository.
func tagExistsInRepo(tag string) bool {
	_, err := gitExec("rev-parse", "--verify", "refs/tags/"+tag)
	return err == nil
}
