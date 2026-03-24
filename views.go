package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model. Renders the current step.
func (m model) View() string {
	switch m.step {
	case stepInit:
		return m.viewInit()
	case stepDirtyWarning:
		return m.viewDirtyWarning()
	case stepDetachedWarning:
		return m.viewDetachedWarning()
	case stepFirstRelease:
		return m.viewFirstRelease()
	case stepFirstReleaseEdit:
		return m.viewFirstReleaseEdit()
	case stepVersionLine:
		return m.viewVersionLine()
	case stepBumpType:
		return m.viewBumpType()
	case stepReleaseType:
		return m.viewReleaseType()
	case stepPreReleaseLabel:
		return m.viewPreReleaseLabel()
	case stepPreReleaseLabelCustom:
		return m.viewPreReleaseLabelCustom()
	case stepDescriptorInput:
		return m.viewDescriptorInput()
	case stepCustomTag:
		return m.viewCustomTag()
	case stepPreReleaseConfirm:
		return m.viewPreReleaseConfirm()
	case stepTagReview:
		return m.viewTagReview()
	case stepPackageVersionConfirm:
		return m.viewPackageVersionConfirm()
	case stepRemotes:
		return m.viewRemotes()
	case stepForgeRelease:
		return m.viewForgeRelease()
	case stepReleaseNotesMode:
		return m.viewReleaseNotesMode()
	case stepReleaseNotesInput:
		return m.viewReleaseNotesInput()
	case stepSummary:
		return m.viewSummary()
	case stepExecuting:
		return m.viewExecuting()
	case stepDone:
		return m.viewDone()
	case stepUndone:
		return m.viewUndone()
	case stepError:
		return m.viewError()
	default:
		return "Unknown state"
	}
}

// --- Header ---

func (m model) viewHeader() string {
	return titleStyle.Render("release") + " " + dimStyle.Render("-- git tag & release manager") + "\n"
}

// viewRepoInfo renders a compact repo info block shown on most screens.
func (m model) viewRepoInfo() string {
	var b strings.Builder

	b.WriteString(renderInfoLine("Repository", m.repo.Name) + "\n")
	b.WriteString(renderInfoLine("Branch", m.repo.Branch) + "\n")

	commitLine := m.repo.HeadCommit
	if m.repo.HeadMessage != "" {
		commitLine += " " + dimStyle.Render(m.repo.HeadMessage)
	}
	b.WriteString(renderInfoLine("HEAD", commitLine) + "\n")

	if m.hasLatest {
		b.WriteString(renderInfoLine("Latest tag", infoStyle.Render(m.latestVersion.Raw)) + "\n")
	}

	if len(m.versions) > 0 {
		patternLabel := m.pattern.Description()
		prefix := detectPrefix(m.versions)
		if prefix == "v" {
			patternLabel += " " + dimStyle.Render("(v-prefixed)")
		}
		b.WriteString(renderInfoLine("Pattern", patternLabel) + "\n")
		b.WriteString(renderInfoLine("Version tags", fmt.Sprintf("%d", len(m.versions))) + "\n")
	}

	return b.String()
}

// --- Step views ---

func (m model) viewInit() string {
	return m.viewHeader() + "\n" +
		m.spinner.View() + " Gathering repository information...\n"
}

func (m model) viewDirtyWarning() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	warning := fmt.Sprintf(
		"Working tree has %d uncommitted change(s).\nCreating a tag will point to the current HEAD commit,\nbut your working directory has modifications.",
		m.repo.DirtyCount,
	)
	b.WriteString(warningBoxStyle.Render(renderWarning("Dirty working tree") + "\n\n" + warning))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Continue anyway? ") + boldStyle.Render("[y/N]"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewDetachedWarning() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	warning := "HEAD is detached. The tag will be created at " + m.repo.HeadCommit + ".\nThis is usually fine for tagging, but make sure you're on the right commit."
	b.WriteString(warningBoxStyle.Render(renderWarning("Detached HEAD") + "\n\n" + warning))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Continue? ") + boldStyle.Render("[y/N]"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewFirstRelease() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render("No version tags found -- First Release"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Choose a versioning scheme:"))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		if choice.Description != "" {
			line += " " + dimStyle.Render(choice.Description)
		}
		if choice.Preview != "" && i == m.cursor {
			line += " " + previewStyle.Render(glyphArrow+" "+choice.Preview)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewFirstReleaseEdit() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Enter starting version"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter to confirm, esc to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewVersionLine() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render("Multiple version lines detected"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Which version line to release from?"))
	b.WriteString("\n\n")

	for i, line := range m.lines {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		label := line.Label()
		latest := line.Latest.Raw
		count := len(line.Versions)

		lineStr := cursorStyle.Render(cursor) + " " +
			style.Render(label) +
			"  " + dimStyle.Render(fmt.Sprintf("latest: %s", latest)) +
			"  " + mutedStyle.Render(fmt.Sprintf("(%d tags)", count))

		b.WriteString(lineStr + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewBumpType() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render("Select version bump"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Current: ") + infoStyle.Render(m.latestVersion.Raw))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		if choice.Preview != "" {
			line += " " + previewStyle.Render(glyphArrow+" "+choice.Preview)
		}
		if choice.Description != "" && i == m.cursor {
			line += " " + dimStyle.Render("("+choice.Description+")")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewReleaseType() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render("Stable or pre-release?"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Current: ") + infoStyle.Render(m.latestVersion.Raw))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		if choice.Preview != "" {
			line += " " + previewStyle.Render(glyphArrow+" "+choice.Preview)
		}
		if choice.Description != "" && i == m.cursor {
			line += " " + dimStyle.Render("("+choice.Description+")")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewPreReleaseLabel() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Select pre-release label"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Current: ") + infoStyle.Render(m.latestVersion.Raw))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		if choice.Description != "" {
			line += " " + dimStyle.Render(choice.Description)
		}
		if choice.Preview != "" && i == m.cursor {
			line += " " + previewStyle.Render(glyphArrow+" "+choice.Preview)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewPreReleaseLabelCustom() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Enter custom pre-release label"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("This will be used as: <version>-<label>.1"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter to confirm, esc to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewDescriptorInput() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Enter descriptor"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("This will be appended as: <version>-<build>-<descriptor>"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter to confirm, esc to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewCustomTag() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Enter custom tag"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	if m.hasLatest {
		b.WriteString(dimStyle.Render("Previous: ") + mutedStyle.Render(m.latestVersion.Raw))
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("enter to confirm, esc to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewPreReleaseConfirm() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render("Is this tag a pre-release?"))
	b.WriteString("\n")
	b.WriteString(renderInfoLine("Tag", infoStyle.Render(m.newTag)))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewTagReview() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	// Show the proposed tag prominently
	newTagDisplay := successStyle.Render(m.newTag)

	tagBox := subtitleStyle.Render("Tag Review") + "\n\n"
	tagBox += renderInfoLine("New tag", newTagDisplay) + "\n"
	tagBox += renderInfoLine("Tag message", dimStyle.Render(m.tagMessage)) + "\n"
	if m.pkgInfo != nil && m.pkgInfo.Detected && m.pkgInfo.HasVersion {
		label := m.pkgInfo.Manager.Name() + " version"
		tagBox += renderInfoLine(label, dimStyle.Render(m.pkgInfo.Version)) + "\n"
		target := m.pkgInfo.Manager.TargetVersionForTag(m.newTag)
		if target != "" {
			if target == m.pkgInfo.Version {
				tagBox += renderInfoLine("Package target", successStyle.Render(target)+dimStyle.Render(" (already matches)")) + "\n"
			} else {
				tagBox += renderInfoLine("Package target", infoStyle.Render(target)) + "\n"
			}
		}
	}

	// Check if tag already exists
	if tagExistsInRepo(m.newTag) {
		tagBox += "\n" + renderWarning("Tag already exists! Use --force to overwrite.") + "\n"
	}

	// Show commits since last tag
	if len(m.commitsSinceTag) > 0 {
		tagBox += "\n" + dimStyle.Render(fmt.Sprintf("Commits since last tag (%d):", len(m.commitsSinceTag))) + "\n"
		maxShow := 10
		if len(m.commitsSinceTag) < maxShow {
			maxShow = len(m.commitsSinceTag)
		}
		for i := 0; i < maxShow; i++ {
			tagBox += "  " + mutedStyle.Render(m.commitsSinceTag[i]) + "\n"
		}
		if len(m.commitsSinceTag) > maxShow {
			tagBox += "  " + dimStyle.Render(fmt.Sprintf("... and %d more", len(m.commitsSinceTag)-maxShow)) + "\n"
		}
	}

	b.WriteString(boxStyle.Render(tagBox))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("[enter/y] Accept  [e] Edit tag  [q/esc] Cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewPackageVersionConfirm() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewRepoInfo())
	b.WriteString("\n")

	managerName := ""
	if m.pkgInfo != nil && m.pkgInfo.Manager != nil {
		managerName = m.pkgInfo.Manager.Name()
	}

	needsUpdate := m.pkgInfo != nil && m.pkgInfo.Manager.NeedsUpdate(m.pkgInfo, m.pkgVersionTarget)
	if needsUpdate {
		b.WriteString(subtitleStyle.Render("Update " + managerName + " package version?"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("A " + managerName + " project was detected and the version does not match the release tag."))
	} else {
		b.WriteString(subtitleStyle.Render(managerName + " version check"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("A " + managerName + " project was detected and the version already matches the release tag."))
	}
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(
		renderInfoLine("File", m.pkgInfo.FilePath) + "\n" +
			renderInfoLine("Current version", dimStyle.Render(m.pkgInfo.Version)) + "\n" +
			renderInfoLine("Target version", successStyle.Render(m.pkgVersionTarget)),
	))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		b.WriteString(line + "\n")
	}

	if needsUpdate {
		b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	} else {
		b.WriteString("\n" + dimStyle.Render("press enter to continue, q to quit"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m model) viewRemotes() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Push tag to remote(s)"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Select remotes to push the tag to:"))
	b.WriteString("\n\n")

	for i, remote := range m.remotes {
		cursor := " "
		if i == m.multiCursor {
			cursor = cursorStyle.Render(glyphSelected)
		}

		checkbox := uncheckedStyle.Render(glyphUnchecked)
		nameStyle := unselectedItemStyle
		if m.pushSelected[i] {
			checkbox = checkedStyle.Render(glyphChecked)
			nameStyle = selectedItemStyle
		}

		forgeLabel := ""
		if remote.Forge != ForgeUnknown {
			forgeLabel = " " + dimStyle.Render("["+remote.Forge.String()+"]")
		}

		line := cursor + " " + checkbox + " " + nameStyle.Render(remote.Name) +
			" " + mutedStyle.Render(remote.ShortURL()) + forgeLabel

		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("space/x toggle, a select all, enter to confirm, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewForgeRelease() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Create release on platform(s)"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Select platforms to create a release on:"))
	b.WriteString("\n\n")

	capableIdx := 0
	for i, remote := range m.remotes {
		if !remote.HasCLI || remote.Forge == ForgeUnknown {
			continue
		}

		cursor := " "
		if capableIdx == m.multiCursor {
			cursor = cursorStyle.Render(glyphSelected)
		}

		checkbox := uncheckedStyle.Render(glyphUnchecked)
		nameStyle := unselectedItemStyle
		if m.forgeSelected[i] {
			checkbox = checkedStyle.Render(glyphChecked)
			nameStyle = selectedItemStyle
		}

		cliInfo := dimStyle.Render("via " + remote.Forge.CLITool())

		line := cursor + " " + checkbox + " " +
			nameStyle.Render(remote.Forge.String()+" release") +
			" " + mutedStyle.Render("on "+remote.Name) +
			" " + cliInfo

		b.WriteString(line + "\n")
		capableIdx++
	}

	if capableIdx == 0 {
		b.WriteString(dimStyle.Render("  No release CLIs available (gh, glab, tea)") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("space/x toggle, a select all, s skip, enter to confirm, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewReleaseNotesMode() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Release notes"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("How would you like to generate release notes?"))
	b.WriteString("\n\n")

	for i, choice := range m.releaseNotesChoices {
		cursor := glyphUnselected
		style := unselectedItemStyle
		if i == m.cursor {
			cursor = glyphSelected
			style = selectedItemStyle
		}

		line := cursorStyle.Render(cursor) + " " + style.Render(choice.Label)
		if choice.Description != "" && i == m.cursor {
			line += " " + dimStyle.Render("("+choice.Description+")")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("j/k or arrows to move, enter to select, q to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewReleaseNotesInput() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Write release notes"))
	b.WriteString("\n\n")
	b.WriteString(m.textArea.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("ctrl+d or esc to finish"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewSummary() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Build summary content
	var summary strings.Builder

	summary.WriteString(subtitleStyle.Render("Execution Plan") + "\n\n")
	summary.WriteString(renderInfoLine("Repository", m.repo.Name) + "\n")
	summary.WriteString(renderInfoLine("Branch", m.repo.Branch) + "\n")
	summary.WriteString(renderInfoLine("Commit", m.repo.HeadCommit+" "+dimStyle.Render(m.repo.HeadMessage)) + "\n")

	if m.hasLatest {
		summary.WriteString(renderInfoLine("Current tag", m.latestVersion.Raw) + "\n")
	}

	summary.WriteString(renderInfoLine("New tag", successStyle.Render(m.newTag)) + "\n")
	summary.WriteString(renderInfoLine("Tag message", dimStyle.Render(m.tagMessage)) + "\n")
	if update := m.pkgVersionUpdate(); update != nil {
		label := update.Manager.Name() + " version"
		summary.WriteString(renderInfoLine(label, dimStyle.Render(update.CurrentVersion+" -> ")+successStyle.Render(update.NewVersion)) + "\n")
	}

	if m.flags.DryRun {
		summary.WriteString("\n" + warningStyle.Render("DRY RUN -- no changes will be made") + "\n")
	}

	summary.WriteString("\n" + boldStyle.Render("Steps:") + "\n")
	summary.WriteString(PlanSummary(m.plan))

	if m.releaseNotes != "" {
		summary.WriteString("\n" + boldStyle.Render("Release notes:") + "\n")
		// Show a preview (first few lines)
		lines := strings.Split(m.releaseNotes, "\n")
		maxLines := 8
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			summary.WriteString("  " + dimStyle.Render(lines[i]) + "\n")
		}
		if len(lines) > maxLines {
			summary.WriteString("  " + dimStyle.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxLines)) + "\n")
		}
	}

	b.WriteString(summaryBoxStyle.Render(summary.String()))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("Execute this plan? ") + boldStyle.Render("[y/enter] Yes  [n/q] Cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewExecuting() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	if m.flags.DryRun {
		b.WriteString(warningStyle.Render("DRY RUN") + "\n\n")
	}

	b.WriteString(subtitleStyle.Render("Executing...") + "\n\n")

	for i, step := range m.plan.Steps {
		var glyph string
		var style lipgloss.Style

		switch step.Status {
		case ExecSuccess:
			glyph = glyphSuccess
			style = successStyle
		case ExecFailed:
			glyph = glyphFailed
			style = errorStyle
		case ExecRunning:
			glyph = m.spinner.View()
			style = infoStyle
		default:
			glyph = glyphPending
			style = lipgloss.NewStyle().Foreground(colorMuted)
		}

		_ = i
		line := renderExecStep(glyph, step.Label, style)
		b.WriteString(line + "\n")

		if step.Error != nil {
			b.WriteString("     " + errorStyle.Render(step.Error.Error()) + "\n")
		}
	}

	b.WriteString("\n")

	return b.String()
}

func (m model) viewDone() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	if m.flags.DryRun {
		b.WriteString(warningStyle.Render("DRY RUN -- no actual changes were made") + "\n\n")
	}

	if HasFailures(m.plan) {
		b.WriteString(errorBoxStyle.Render(
			renderError("Some steps failed") + "\n\n" + ExecutionSummary(m.plan),
		))
		b.WriteString("\n\n")
		b.WriteString(boldStyle.Render("[r]") + dimStyle.Render(" Retry failed steps") + "  ")
		undoLabel := " Undo (delete local tag and exit)"
		if m.plan.PackageUpdate != nil {
			undoLabel = " Undo (delete local tag only; keep commit and any pushed branch changes)"
		}
		b.WriteString(boldStyle.Render("[u]") + dimStyle.Render(undoLabel) + "  ")
		b.WriteString(boldStyle.Render("[q]") + dimStyle.Render(" Quit (keep tag)"))
		b.WriteString("\n")
	} else {
		doneContent := renderSuccess("Release "+m.newTag+" completed successfully!") + "\n\n" +
			ExecutionSummary(m.plan)
		b.WriteString(successBoxStyle.Render(doneContent))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Press q or enter to exit"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m model) viewError() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	errMsg := "An error occurred"
	if m.err != nil {
		errMsg = m.err.Error()
	}

	b.WriteString(errorBoxStyle.Render(renderError(errMsg)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press q or enter to exit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewUndone() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	content := renderSuccess("Tag "+m.plan.Tag+" deleted from local repository.") + "\n\n" +
		dimStyle.Render("You can re-run release to try again.")
	b.WriteString(successBoxStyle.Render(content))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press q or enter to exit"))
	b.WriteString("\n")

	return b.String()
}
