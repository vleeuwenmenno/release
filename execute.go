package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"release/pkgmanager"
)

// ExecStepType identifies what kind of execution step this is
type ExecStepType int

const (
	ExecUpdatePackageVersion ExecStepType = iota
	ExecCommitPackageVersion
	ExecPushBranch
	ExecCreateTag
	ExecPushTag
	ExecCreateRelease
)

func (e ExecStepType) String() string {
	switch e {
	case ExecUpdatePackageVersion:
		return "Update package version"
	case ExecCommitPackageVersion:
		return "Commit package version"
	case ExecPushBranch:
		return "Push branch"
	case ExecCreateTag:
		return "Create tag"
	case ExecPushTag:
		return "Push tag"
	case ExecCreateRelease:
		return "Create release"
	default:
		return "Unknown"
	}
}

// ExecStepStatus tracks the state of an execution step
type ExecStepStatus int

const (
	ExecPending ExecStepStatus = iota
	ExecRunning
	ExecSuccess
	ExecFailed
	ExecSkipped
)

// ExecStep represents a single step in the release execution plan
type ExecStep struct {
	Type        ExecStepType
	Label       string
	Description string
	Status      ExecStepStatus
	Error       error
	Duration    time.Duration
}

// ReleasePlan holds the complete execution plan built from user choices
type ReleasePlan struct {
	Tag                string
	TagMessage         string
	ReleaseNotes       string
	PreReleaseExplicit bool
	PackageUpdate      *pkgmanager.VersionUpdate
	Branch             string
	PushRemotes        []RemoteInfo
	ForgeRemotes       []RemoteInfo
	DryRun             bool
	Steps              []ExecStep
}

// BuildReleasePlan constructs an execution plan from the collected user choices.
func BuildReleasePlan(tag, tagMessage, releaseNotes string, preReleaseExplicit bool, packageUpdate *pkgmanager.VersionUpdate, branch string, pushRemotes, forgeRemotes []RemoteInfo, dryRun bool) ReleasePlan {
	plan := ReleasePlan{
		Tag:                tag,
		TagMessage:         tagMessage,
		ReleaseNotes:       releaseNotes,
		PreReleaseExplicit: preReleaseExplicit,
		PackageUpdate:      packageUpdate,
		Branch:             branch,
		PushRemotes:        pushRemotes,
		ForgeRemotes:       forgeRemotes,
		DryRun:             dryRun,
	}

	if packageUpdate != nil {
		plan.Steps = append(plan.Steps, ExecStep{
			Type:        ExecUpdatePackageVersion,
			Label:       fmt.Sprintf("Update %s version", packageUpdate.Manager.Name()),
			Description: fmt.Sprintf("Set %s version from %s to %s", packageUpdate.Path, packageUpdate.CurrentVersion, packageUpdate.NewVersion),
			Status:      ExecPending,
		})
		plan.Steps = append(plan.Steps, ExecStep{
			Type:        ExecCommitPackageVersion,
			Label:       fmt.Sprintf("Commit %s version bump", packageUpdate.Manager.Name()),
			Description: fmt.Sprintf("git commit --only -m \"%s\" -- %s", escapeSummary(packageUpdate.CommitMessage), packageUpdate.Path),
			Status:      ExecPending,
		})

		if branch != "" {
			for _, remote := range pushRemotes {
				plan.Steps = append(plan.Steps, ExecStep{
					Type:        ExecPushBranch,
					Label:       fmt.Sprintf("Push branch %s to %s", branch, remote.Label()),
					Description: fmt.Sprintf("git push %s %s", remote.Name, branch),
					Status:      ExecPending,
				})
			}
		}
	}

	// Step 1: Create the signed annotated tag
	plan.Steps = append(plan.Steps, ExecStep{
		Type:        ExecCreateTag,
		Label:       fmt.Sprintf("Create tag %s", tag),
		Description: fmt.Sprintf("git tag -s %s -m \"%s\"", tag, escapeSummary(tagMessage)),
		Status:      ExecPending,
	})

	// Step 2: Push tag to selected remotes
	for _, remote := range pushRemotes {
		plan.Steps = append(plan.Steps, ExecStep{
			Type:        ExecPushTag,
			Label:       fmt.Sprintf("Push tag to %s", remote.Label()),
			Description: fmt.Sprintf("git push %s %s", remote.Name, tag),
			Status:      ExecPending,
		})
	}

	// Step 3: Create releases on selected forges
	for _, remote := range forgeRemotes {
		plan.Steps = append(plan.Steps, ExecStep{
			Type:        ExecCreateRelease,
			Label:       fmt.Sprintf("Create %s release on %s", remote.Forge, remote.Name),
			Description: fmt.Sprintf("%s release create %s", remote.Forge.CLITool(), tag),
			Status:      ExecPending,
		})
	}

	return plan
}

// ExecuteStep runs a single step in the release plan.
// Returns an error if the step fails.
func ExecuteStep(plan *ReleasePlan, index int) error {
	if index < 0 || index >= len(plan.Steps) {
		return fmt.Errorf("step index %d out of range", index)
	}

	step := &plan.Steps[index]
	step.Status = ExecRunning
	start := time.Now()

	if plan.DryRun {
		// In dry-run mode, simulate success with a short delay
		time.Sleep(200 * time.Millisecond)
		step.Status = ExecSuccess
		step.Duration = time.Since(start)
		return nil
	}

	var err error

	switch step.Type {
	case ExecUpdatePackageVersion:
		if plan.PackageUpdate == nil {
			err = fmt.Errorf("no package version update configured")
		} else {
			err = plan.PackageUpdate.Manager.UpdateVersion(plan.PackageUpdate.Path, plan.PackageUpdate.NewVersion)
		}

	case ExecCommitPackageVersion:
		if plan.PackageUpdate == nil {
			err = fmt.Errorf("no package version update configured")
		} else {
			err = commitTrackedFile(plan.PackageUpdate.Path, plan.PackageUpdate.CommitMessage)
		}

	case ExecPushBranch:
		remote := findPushRemoteForStepType(plan, index, ExecPushBranch)
		if remote == nil {
			err = fmt.Errorf("no remote found for branch push step")
		} else if plan.Branch == "" {
			err = fmt.Errorf("no branch configured for branch push step")
		} else {
			err = pushBranch(plan.Branch, remote.Name)
		}

	case ExecCreateTag:
		err = createAnnotatedTag(plan.Tag, plan.TagMessage)

	case ExecPushTag:
		// Find which remote this step corresponds to
		remote := findPushRemoteForStepType(plan, index, ExecPushTag)
		if remote == nil {
			err = fmt.Errorf("no remote found for push step")
		} else {
			err = pushTag(plan.Tag, remote.Name)
		}

	case ExecCreateRelease:
		// Find which forge remote this step corresponds to
		remote := findForgeRemoteForStep(plan, index)
		if remote == nil {
			err = fmt.Errorf("no remote found for release step")
		} else {
			title := plan.Tag
			err = createForgeRelease(*remote, plan.Tag, title, plan.ReleaseNotes, plan.PreReleaseExplicit)
		}
	}

	step.Duration = time.Since(start)

	if err != nil {
		step.Status = ExecFailed
		step.Error = err
		return err
	}

	step.Status = ExecSuccess
	return nil
}

// findPushRemoteForStepType finds the RemoteInfo for a push step by counting
// matching steps up to the given index.
func findPushRemoteForStepType(plan *ReleasePlan, stepIndex int, stepType ExecStepType) *RemoteInfo {
	pushCount := 0
	for i := 0; i <= stepIndex; i++ {
		if plan.Steps[i].Type == stepType {
			if i == stepIndex {
				if pushCount < len(plan.PushRemotes) {
					return &plan.PushRemotes[pushCount]
				}
				return nil
			}
			pushCount++
		}
	}
	return nil
}

// findForgeRemoteForStep finds the RemoteInfo for a forge release step.
func findForgeRemoteForStep(plan *ReleasePlan, stepIndex int) *RemoteInfo {
	forgeCount := 0
	for i := 0; i <= stepIndex; i++ {
		if plan.Steps[i].Type == ExecCreateRelease {
			if i == stepIndex {
				if forgeCount < len(plan.ForgeRemotes) {
					return &plan.ForgeRemotes[forgeCount]
				}
				return nil
			}
			forgeCount++
		}
	}
	return nil
}

// PlanSummary returns a human-readable summary of the execution plan.
func PlanSummary(plan ReleasePlan) string {
	var b strings.Builder

	for i, step := range plan.Steps {
		prefix := fmt.Sprintf("  %d. ", i+1)
		b.WriteString(prefix + step.Label + "\n")
		b.WriteString(strings.Repeat(" ", len(prefix)) + dimStyle.Render(step.Description) + "\n")
	}

	return b.String()
}

// ExecutionSummary returns a summary of the executed plan with status indicators.
func ExecutionSummary(plan ReleasePlan) string {
	var b strings.Builder

	for _, step := range plan.Steps {
		var indicator string
		var stl lipgloss.Style

		switch step.Status {
		case ExecSuccess:
			indicator = glyphSuccess
			stl = successStyle
		case ExecFailed:
			indicator = glyphFailed
			stl = errorStyle
		case ExecSkipped:
			indicator = glyphPending
			stl = dimStyle
		case ExecRunning:
			indicator = glyphPending
			stl = infoStyle
		default:
			indicator = glyphPending
			stl = dimStyle
		}

		line := stl.Render(indicator) + " " + step.Label
		if step.Duration > 0 {
			line += " " + dimStyle.Render(fmt.Sprintf("(%s)", step.Duration.Round(time.Millisecond)))
		}
		if step.Error != nil {
			line += "\n     " + errorStyle.Render(step.Error.Error())
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// HasFailures returns true if any step in the plan failed.
func HasFailures(plan ReleasePlan) bool {
	for _, step := range plan.Steps {
		if step.Status == ExecFailed {
			return true
		}
	}
	return false
}

// AllDone returns true if all steps are in a terminal state.
func AllDone(plan ReleasePlan) bool {
	for _, step := range plan.Steps {
		if step.Status == ExecPending || step.Status == ExecRunning {
			return false
		}
	}
	return true
}

// escapeSummary truncates and cleans a string for display in command previews.
func escapeSummary(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	if len(s) > 60 {
		s = s[:57] + "..."
	}
	return s
}
