package main

import (
	"fmt"
	"strings"

	"github.com/vleeuwenmenno/release/pkgmanager"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// step represents the current screen in the TUI flow
type step int

const (
	stepInit                  step = iota // Loading repo info, fetching tags
	stepDirtyWarning                      // Warn about dirty working tree
	stepDetachedWarning                   // Warn about detached HEAD
	stepFirstRelease                      // No tags found, pick a template
	stepFirstReleaseEdit                  // Edit starting version for first release
	stepVersionLine                       // Pick version line (if multiple)
	stepBumpType                          // Pick bump type
	stepReleaseType                       // Stable or pre-release?
	stepPreReleaseLabel                   // Pick pre-release label
	stepPreReleaseLabelCustom             // Custom pre-release label input
	stepDescriptorInput                   // Enter descriptor for build-number pattern
	stepCustomTag                         // Enter custom tag manually
	stepPreReleaseConfirm                 // Always confirm whether this tag is a pre-release
	stepTagReview                         // Review proposed tag, option to edit
	stepPackageVersionConfirm             // Optionally update a detected package manager version before release
	stepRemotes                           // Pick which remote(s) to push to
	stepForgeRelease                      // Pick which forge(s) to create release on
	stepReleaseNotesMode                  // Choose release notes mode
	stepReleaseNotesInput                 // Write release notes
	stepSummary                           // Final execution plan summary
	stepExecuting                         // Running commands
	stepDone                              // Finished
	stepUndone                            // Tag deleted after undo
	stepError                             // Fatal error
)

// Flags holds parsed CLI flags
type Flags struct {
	Tag     string
	Message string
	Push    bool
	Release bool
	DryRun  bool
	Force   bool
}

// Custom tea.Msg types for async operations
type gitInfoMsg struct {
	repo     RepoInfo
	tags     []string
	remotes  []RemoteInfo
	pkgInfo  *pkgmanager.ProjectInfo
	fetchErr error
	err      error
}

type execStepDoneMsg struct {
	index int
	err   error
}

// model is the main bubbletea model
type model struct {
	// Current step in the flow
	step step

	// CLI flags
	flags Flags

	// Window dimensions
	width  int
	height int

	// Git repo info
	repo    RepoInfo
	remotes []RemoteInfo
	pkgInfo *pkgmanager.ProjectInfo

	// Version analysis
	allTags  []string
	versions []Version
	pattern  VersionPattern
	lines    []VersionLine

	// User selections
	selectedLineIdx  int
	selectedBump     BumpType
	latestVersion    Version
	hasLatest        bool
	newTag           string
	tagMessage       string
	releaseNotes     string
	pkgVersionTarget string

	// Remote selections (indices into remotes slice)
	pushSelected  map[int]bool
	forgeSelected map[int]bool

	// Execution plan
	plan      ReleasePlan
	execIndex int

	// Menu cursor and choices
	cursor  int
	choices []Choice

	// Pending version from bump selection (before stable/pre-release choice)
	pendingBump Version

	// Tracks whether the user explicitly marked this tag as a pre-release.
	preReleaseExplicit bool
	updatePkgVersion   bool

	// True when the pending bump is from build-number pattern.
	preReleaseTargetIsBuild bool

	// Multi-select cursor (for remotes)
	multiCursor int

	// Bubbles components
	spinner   spinner.Model
	textInput textinput.Model
	textArea  textarea.Model

	// Release notes mode choices
	releaseNotesChoices []Choice

	// Commits since last tag (for auto release notes)
	commitsSinceTag []string

	// Error state
	err error
}

// initialModel creates the initial model state.
func initialModel(flags Flags) model {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorPrimary)),
	)

	ti := textinput.New()
	ti.CharLimit = 100
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPrimary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)

	ta := textarea.New()
	ta.SetWidth(60)
	ta.SetHeight(8)
	ta.CharLimit = 5000
	ta.Placeholder = "Enter release notes..."

	return model{
		step:          stepInit,
		flags:         flags,
		spinner:       s,
		textInput:     ti,
		textArea:      ta,
		pushSelected:  make(map[int]bool),
		forgeSelected: make(map[int]bool),
	}
}

// Init implements tea.Model. Starts the spinner and kicks off git info gathering.
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		gatherGitInfo,
	)
}

// gatherGitInfo is a tea.Cmd that collects repo info, fetches tags, and detects remotes.
func gatherGitInfo() tea.Msg {
	// Check if we're in a git repo
	if !isGitRepo() {
		return gitInfoMsg{err: fmt.Errorf("not a git repository")}
	}

	// Get repo info
	repo, err := getRepoInfo()
	if err != nil {
		return gitInfoMsg{err: err}
	}

	// Fetch tags from remotes
	fetchErr := fetchTags()

	// Get all tags
	tags, err := getAllTags()
	if err != nil {
		return gitInfoMsg{repo: repo, err: err}
	}

	// Detect remotes
	remotes, _ := detectRemotes()

	pkgInfo, err := pkgmanager.DetectAll(repo.RootPath)
	if err != nil {
		return gitInfoMsg{repo: repo, tags: tags, remotes: remotes, fetchErr: fetchErr, err: err}
	}

	return gitInfoMsg{
		repo:     repo,
		tags:     tags,
		remotes:  remotes,
		pkgInfo:  pkgInfo,
		fetchErr: fetchErr,
	}
}

// executeStepCmd returns a tea.Cmd that executes a single plan step.
func executeStepCmd(plan *ReleasePlan, index int) tea.Cmd {
	return func() tea.Msg {
		err := ExecuteStep(plan, index)
		return execStepDoneMsg{index: index, err: err}
	}
}

// Update implements tea.Model. Handles all messages and state transitions.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Step-specific key handling
		return m.handleKeyMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case gitInfoMsg:
		return m.handleGitInfo(msg)

	case execStepDoneMsg:
		return m.handleExecStepDone(msg)
	}

	// Pass through to active sub-components
	return m.updateComponents(msg)
}

// updateComponents passes messages to the currently active bubbles component.
func (m model) updateComponents(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.step {
	case stepCustomTag, stepFirstReleaseEdit, stepPreReleaseLabelCustom, stepDescriptorInput:
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case stepReleaseNotesInput:
		m.textArea, cmd = m.textArea.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleGitInfo processes the gathered git information and transitions to the next step.
func (m model) handleGitInfo(msg gitInfoMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.step = stepError
		return m, nil
	}

	m.repo = msg.repo
	m.allTags = msg.tags
	m.remotes = msg.remotes
	m.pkgInfo = msg.pkgInfo

	// Parse and filter version tags
	m.versions = filterVersionTags(m.allTags)
	m.pattern = detectPattern(m.versions)
	m.lines = detectVersionLines(m.versions, m.pattern)

	// If a manual tag was provided via flag, skip to tag review
	if m.flags.Tag != "" {
		m.newTag = m.flags.Tag
		m.tagMessage = m.buildTagMessage()
		return m.transitionToTagReview()
	}

	// Check for dirty working tree
	if m.repo.IsDirty && !m.flags.Force {
		m.step = stepDirtyWarning
		return m, nil
	}

	return m.afterDirtyCheck()
}

// afterDirtyCheck continues the flow after handling dirty state.
func (m model) afterDirtyCheck() (model, tea.Cmd) {
	// Check for detached HEAD
	if m.repo.IsDetached {
		m.step = stepDetachedWarning
		return m, nil
	}

	return m.afterWarnings()
}

// afterWarnings continues after all warnings have been shown.
func (m model) afterWarnings() (model, tea.Cmd) {
	// No version tags found -> first release flow
	if len(m.versions) == 0 {
		m.choices = buildFirstReleaseChoices()
		m.cursor = 0
		m.step = stepFirstRelease
		return m, nil
	}

	// Multiple version lines -> ask which to bump
	if len(m.lines) > 1 {
		m.cursor = len(m.lines) - 1 // default to latest line
		m.step = stepVersionLine
		return m, nil
	}

	// Single version line -> go to bump type
	if len(m.lines) == 1 {
		m.selectedLineIdx = 0
		m.latestVersion = m.lines[0].Latest
		m.hasLatest = true
	} else if latest, ok := latestVersionForPattern(m.versions, m.pattern); ok {
		m.latestVersion = latest
		m.hasLatest = true
	}

	return m.transitionToBumpType()
}

// transitionToBumpType sets up the bump type selection step.
func (m model) transitionToBumpType() (model, tea.Cmd) {
	if !m.hasLatest {
		// Shouldn't happen, but fall back to first release
		m.choices = buildFirstReleaseChoices()
		m.cursor = 0
		m.step = stepFirstRelease
		return m, nil
	}

	// Reset release-type state for a fresh bump selection.
	m.pendingBump = Version{}
	m.preReleaseExplicit = false
	m.preReleaseTargetIsBuild = false

	m.choices = buildBumpChoices(m.latestVersion, m.pattern)
	m.cursor = 0
	m.step = stepBumpType
	return m, nil
}

// transitionToTagReview always asks whether the tag is a pre-release,
// then routes to the actual tag review screen.
func (m model) transitionToTagReview() (model, tea.Cmd) {
	m.choices = []Choice{
		{
			Label: "No",
		},
		{
			Label: "Yes",
		},
	}

	if m.preReleaseExplicit || inferPreReleaseFromTag(m.newTag) {
		m.cursor = 1
	} else {
		m.cursor = 0
	}

	m.step = stepPreReleaseConfirm
	return m, nil
}

// enterTagReview sets up the actual tag review step.
func (m model) enterTagReview() (model, tea.Cmd) {
	if m.tagMessage == "" {
		m.tagMessage = m.buildTagMessage()
	}

	// Get commits since last tag for release notes
	prevTag := ""
	if m.hasLatest {
		prevTag = m.latestVersion.Raw
	}
	m.commitsSinceTag, _ = getCommitsSinceTag(prevTag)

	m.step = stepTagReview
	return m, nil
}

// transitionAfterTagReview optionally prompts to update a detected package manager version
// before continuing with the normal release flow.
func (m model) transitionAfterTagReview() (model, tea.Cmd) {
	m.updatePkgVersion = false

	if m.pkgInfo != nil && m.pkgInfo.Manager != nil {
		m.pkgVersionTarget = m.pkgInfo.Manager.TargetVersionForTag(m.newTag)
		if m.pkgInfo.Manager.ShouldPromptUpdate(m.pkgInfo, m.pkgVersionTarget) {
			if m.pkgInfo.Manager.NeedsUpdate(m.pkgInfo, m.pkgVersionTarget) {
				m.choices = []Choice{
					{Label: "No"},
					{Label: "Yes"},
				}
				m.cursor = 1
			} else {
				m.choices = []Choice{
					{Label: "Continue"},
				}
				m.cursor = 0
			}
			m.step = stepPackageVersionConfirm
			return m, nil
		}
	}

	return m.transitionToRemotes()
}

// transitionToRemotes sets up the remote selection step.
func (m model) transitionToRemotes() (model, tea.Cmd) {
	if len(m.remotes) == 0 {
		// No remotes, skip to summary
		return m.transitionToSummary()
	}

	if m.flags.Push {
		// Auto-select all remotes
		for i := range m.remotes {
			m.pushSelected[i] = true
		}
		return m.transitionToForgeRelease()
	}

	m.multiCursor = 0
	m.step = stepRemotes
	return m, nil
}

// transitionToForgeRelease sets up the forge release selection step.
func (m model) transitionToForgeRelease() (model, tea.Cmd) {
	capable := releaseCapableRemotes(m.remotes)
	if len(capable) == 0 {
		// No forge CLIs available, skip to release notes or summary
		return m.transitionToReleaseNotes()
	}

	if m.flags.Release {
		// Auto-select all capable forges
		for i, r := range m.remotes {
			if r.HasCLI && r.Forge != ForgeUnknown {
				m.forgeSelected[i] = true
			}
		}
		return m.transitionToReleaseNotes()
	}

	m.multiCursor = 0
	m.step = stepForgeRelease
	return m, nil
}

// transitionToReleaseNotes sets up the release notes step.
func (m model) transitionToReleaseNotes() (model, tea.Cmd) {
	// If no forge releases selected, skip release notes entirely
	hasForgeRelease := false
	for _, selected := range m.forgeSelected {
		if selected {
			hasForgeRelease = true
			break
		}
	}

	if !hasForgeRelease {
		return m.transitionToSummary()
	}

	m.releaseNotesChoices = []Choice{
		{Label: "Auto-generate from commits", Description: "List commits since last tag"},
		{Label: "Write notes", Description: "Enter release notes manually"},
		{Label: "Empty", Description: "No release notes"},
	}
	m.choices = m.releaseNotesChoices
	m.cursor = 0
	m.step = stepReleaseNotesMode
	return m, nil
}

// transitionToSummary builds the execution plan and shows the summary.
func (m model) transitionToSummary() (model, tea.Cmd) {
	// Collect selected push remotes
	var pushRemotes []RemoteInfo
	for i, r := range m.remotes {
		if m.pushSelected[i] {
			pushRemotes = append(pushRemotes, r)
		}
	}

	// Collect selected forge remotes
	var forgeRemotes []RemoteInfo
	for i, r := range m.remotes {
		if m.forgeSelected[i] {
			forgeRemotes = append(forgeRemotes, r)
		}
	}

	m.plan = BuildReleasePlan(
		m.newTag,
		m.tagMessage,
		m.releaseNotes,
		m.preReleaseExplicit,
		m.pkgVersionUpdate(),
		m.repo.Branch,
		pushRemotes,
		forgeRemotes,
		m.flags.DryRun,
	)

	m.cursor = 0 // reuse cursor for confirm (0=No, move to 1=Yes)
	m.step = stepSummary
	return m, nil
}

// transitionToExecuting starts executing the release plan.
func (m model) transitionToExecuting() (model, tea.Cmd) {
	m.step = stepExecuting
	m.execIndex = 0

	if len(m.plan.Steps) == 0 {
		m.step = stepDone
		return m, nil
	}

	return m, executeStepCmd(&m.plan, 0)
}

// handleExecStepDone handles completion of a single execution step.
func (m model) handleExecStepDone(msg execStepDoneMsg) (model, tea.Cmd) {
	// Move to next step
	m.execIndex = msg.index + 1

	if msg.err != nil {
		// Step failed, but continue with remaining steps
		// (don't abort the whole plan)
	}

	// Check if there are more steps
	if m.execIndex < len(m.plan.Steps) {
		return m, executeStepCmd(&m.plan, m.execIndex)
	}

	// All steps done
	m.step = stepDone
	return m, nil
}

// handleDone handles key input on the done screen.
// If there are failures, offer retry/undo/quit. Otherwise any key quits.
func (m model) handleDone(msg tea.KeyMsg) (model, tea.Cmd) {
	if !HasFailures(m.plan) {
		// No failures -- any key exits
		if msg.String() == "q" || msg.String() == "enter" || msg.String() == "esc" {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "r":
		// Retry failed steps: reset them to pending and re-execute
		for i := range m.plan.Steps {
			if m.plan.Steps[i].Status == ExecFailed {
				m.plan.Steps[i].Status = ExecPending
				m.plan.Steps[i].Error = nil
				m.plan.Steps[i].Duration = 0
			}
		}
		// Find first pending step
		m.step = stepExecuting
		for i, s := range m.plan.Steps {
			if s.Status == ExecPending {
				m.execIndex = i
				return m, executeStepCmd(&m.plan, i)
			}
		}
		// No pending steps found (shouldn't happen)
		m.step = stepDone
		return m, nil

	case "u":
		// Undo: delete the local tag and quit
		if m.plan.Tag != "" && !m.plan.DryRun {
			_ = deleteTag(m.plan.Tag)
		}
		m.step = stepUndone
		return m, nil

	case "q", "esc", "enter":
		return m, tea.Quit
	}

	return m, nil
}

// handleKeyMsg routes key events to the appropriate step handler.
func (m model) handleKeyMsg(msg tea.KeyMsg) (model, tea.Cmd) {
	switch m.step {
	case stepInit, stepExecuting:
		// No key handling during loading/executing
		return m, nil

	case stepError, stepUndone:
		if msg.String() == "q" || msg.String() == "enter" || msg.String() == "esc" {
			return m, tea.Quit
		}
		return m, nil

	case stepDone:
		return m.handleDone(msg)

	case stepDirtyWarning:
		return m.handleDirtyWarning(msg)

	case stepDetachedWarning:
		return m.handleDetachedWarning(msg)

	case stepFirstRelease:
		return m.handleMenuSelect(msg, m.onFirstReleaseSelect)

	case stepFirstReleaseEdit:
		return m.handleTextInput(msg, m.onFirstReleaseEditDone)

	case stepVersionLine:
		return m.handleVersionLineSelect(msg)

	case stepBumpType:
		return m.handleMenuSelect(msg, m.onBumpTypeSelect)

	case stepReleaseType:
		return m.handleMenuSelect(msg, m.onReleaseTypeSelect)

	case stepPreReleaseLabel:
		return m.handleMenuSelect(msg, m.onPreReleaseLabelSelect)

	case stepPreReleaseLabelCustom:
		return m.handleTextInput(msg, m.onCustomPreReleaseDone)

	case stepDescriptorInput:
		return m.handleTextInput(msg, m.onDescriptorDone)

	case stepCustomTag:
		return m.handleTextInput(msg, m.onCustomTagDone)

	case stepPreReleaseConfirm:
		return m.handleMenuSelect(msg, m.onPreReleaseConfirmSelect)

	case stepTagReview:
		return m.handleTagReview(msg)

	case stepPackageVersionConfirm:
		return m.handleMenuSelect(msg, m.onPackageVersionConfirmSelect)

	case stepRemotes:
		return m.handleMultiSelect(msg, len(m.remotes), m.pushSelected, m.onRemotesDone)

	case stepForgeRelease:
		return m.handleForgeReleaseSelect(msg)

	case stepReleaseNotesMode:
		return m.handleMenuSelect(msg, m.onReleaseNotesModeSelect)

	case stepReleaseNotesInput:
		return m.handleReleaseNotesInput(msg)

	case stepSummary:
		return m.handleSummary(msg)
	}

	return m, nil
}

// onPreReleaseConfirmSelect handles explicit pre-release confirmation for the chosen tag.
func (m model) onPreReleaseConfirmSelect(idx int) (model, tea.Cmd) {
	m.preReleaseExplicit = idx == 1
	return m.enterTagReview()
}

// handleDirtyWarning handles the dirty working tree warning screen.
func (m model) handleDirtyWarning(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.afterDirtyCheck()
	case "n", "N", "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// handleDetachedWarning handles the detached HEAD warning screen.
func (m model) handleDetachedWarning(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		return m.afterWarnings()
	case "n", "N", "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// handleMenuSelect handles up/down/enter for a simple menu.
func (m model) handleMenuSelect(msg tea.KeyMsg, onSelect func(int) (model, tea.Cmd)) (model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter":
		return onSelect(m.cursor)
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// handleVersionLineSelect handles version line selection.
func (m model) handleVersionLineSelect(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.lines)-1 {
			m.cursor++
		}
	case "enter":
		m.selectedLineIdx = m.cursor
		m.latestVersion = m.lines[m.cursor].Latest
		m.hasLatest = true
		return m.transitionToBumpType()
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// handleTextInput handles text input fields (custom tag, descriptor, etc.)
func (m model) handleTextInput(msg tea.KeyMsg, onDone func(string) (model, tea.Cmd)) (model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value == "" {
			return m, nil
		}
		return onDone(value)
	case "esc":
		return m, tea.Quit
	}

	// Let textinput handle the key
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// handleTagReview handles the tag review screen.
func (m model) handleTagReview(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y", "Y":
		return m.transitionAfterTagReview()
	case "e":
		// Edit tag
		m.textInput.SetValue(m.newTag)
		m.textInput.Focus()
		m.textInput.CursorEnd()
		m.step = stepCustomTag
		return m, textinput.Blink
	case "q", "esc", "n", "N":
		return m, tea.Quit
	}
	return m, nil
}

// onPackageVersionConfirmSelect handles the optional package version update choice.
func (m model) onPackageVersionConfirmSelect(idx int) (model, tea.Cmd) {
	if m.pkgInfo != nil {
		m.updatePkgVersion = m.pkgInfo.Manager.NeedsUpdate(m.pkgInfo, m.pkgVersionTarget) && idx == 1
	}
	return m.transitionToRemotes()
}

// handleMultiSelect handles multi-select (checkboxes) for remotes.
func (m model) handleMultiSelect(msg tea.KeyMsg, itemCount int, selected map[int]bool, onDone func() (model, tea.Cmd)) (model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.multiCursor > 0 {
			m.multiCursor--
		}
	case "down", "j":
		if m.multiCursor < itemCount-1 {
			m.multiCursor++
		}
	case " ", "x":
		// Toggle selection
		selected[m.multiCursor] = !selected[m.multiCursor]
	case "a":
		// Select all
		for i := 0; i < itemCount; i++ {
			selected[i] = true
		}
	case "enter":
		return onDone()
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// handleForgeReleaseSelect handles the forge release multi-select.
// Only shows remotes that have a CLI available.
func (m model) handleForgeReleaseSelect(msg tea.KeyMsg) (model, tea.Cmd) {
	capable := releaseCapableRemotes(m.remotes)
	capableCount := len(capable)

	switch msg.String() {
	case "up", "k":
		if m.multiCursor > 0 {
			m.multiCursor--
		}
	case "down", "j":
		if m.multiCursor < capableCount-1 {
			m.multiCursor++
		}
	case " ", "x":
		// Map capable index back to remotes index
		idx := m.capableRemoteIndex(m.multiCursor)
		if idx >= 0 {
			m.forgeSelected[idx] = !m.forgeSelected[idx]
		}
	case "a":
		for i, r := range m.remotes {
			if r.HasCLI && r.Forge != ForgeUnknown {
				m.forgeSelected[i] = true
			}
		}
	case "s":
		// Skip release creation
		m.forgeSelected = make(map[int]bool)
		return m.transitionToReleaseNotes()
	case "enter":
		return m.transitionToReleaseNotes()
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// capableRemoteIndex maps a capable-list index to the full remotes slice index.
func (m model) capableRemoteIndex(capableIdx int) int {
	count := 0
	for i, r := range m.remotes {
		if r.HasCLI && r.Forge != ForgeUnknown {
			if count == capableIdx {
				return i
			}
			count++
		}
	}
	return -1
}

// handleReleaseNotesInput handles the textarea for release notes.
func (m model) handleReleaseNotesInput(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+d":
		// Submit notes
		m.releaseNotes = m.textArea.Value()
		return m.transitionToSummary()
	case "esc":
		m.releaseNotes = m.textArea.Value()
		return m.transitionToSummary()
	}

	var cmd tea.Cmd
	m.textArea, cmd = m.textArea.Update(msg)
	return m, cmd
}

// handleSummary handles the final summary/confirmation screen.
func (m model) handleSummary(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		return m.transitionToExecuting()
	case "n", "N", "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// --- Selection callbacks ---

// onFirstReleaseSelect handles picking a first release template.
func (m model) onFirstReleaseSelect(idx int) (model, tea.Cmd) {
	choice := m.choices[idx]

	if choice.Preview == "" || idx == len(m.choices)-1 {
		// Custom entry
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., v0.1.0"
		m.textInput.Focus()
		m.step = stepFirstReleaseEdit
		return m, textinput.Blink
	}

	// Use the template preview as the starting tag
	m.newTag = choice.Preview
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

// onFirstReleaseEditDone handles the custom first release tag input.
func (m model) onFirstReleaseEditDone(value string) (model, tea.Cmd) {
	m.newTag = value
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

// onBumpTypeSelect handles picking a bump type.
func (m model) onBumpTypeSelect(idx int) (model, tea.Cmd) {
	choice := m.choices[idx]
	m.selectedBump = choice.BumpType

	switch choice.BumpType {
	case BumpCustom:
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., " + m.latestVersion.Prefix + "X.Y.Z"
		m.textInput.Focus()
		m.step = stepCustomTag
		return m, textinput.Blink

	case BumpPreRelease:
		if !m.latestVersion.HasPreRelease() {
			// Need to pick a pre-release label
			m.choices = buildPreReleaseLabelChoices(m.latestVersion)
			m.cursor = 0
			m.step = stepPreReleaseLabel
			return m, nil
		}
		// Bumping existing pre-release counter
		bumped := bumpVersion(m.latestVersion, BumpPreRelease, "", "")
		m.newTag = formatVersion(bumped)
		m.tagMessage = m.buildTagMessage()
		return m.transitionToTagReview()

	case BumpDescriptor:
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., podman"
		m.textInput.Focus()
		m.step = stepDescriptorInput
		return m, textinput.Blink

	default:
		bumped := bumpVersion(m.latestVersion, choice.BumpType, "", "")

		// Ask stable vs pre-release for normal bump actions:
		// - semver: patch/minor/major
		// - build-number: build/minor/major
		shouldAskReleaseType := false
		if m.pattern == PatternSemver {
			shouldAskReleaseType = choice.BumpType == BumpPatch || choice.BumpType == BumpMinor || choice.BumpType == BumpMajor
		} else if m.pattern == PatternBuildNumber {
			shouldAskReleaseType = choice.BumpType == BumpBuild || choice.BumpType == BumpMinor || choice.BumpType == BumpMajor
		}

		if shouldAskReleaseType {
			m.pendingBump = bumped
			m.preReleaseTargetIsBuild = m.pattern == PatternBuildNumber
			m.choices = buildReleaseTypeChoices(bumped)
			m.choices = append(m.choices, Choice{
				Label:       "Edit full tag",
				Description: "Manually edit the complete tag value",
				Preview:     "(enter custom tag)",
				BumpType:    BumpCustom,
			})
			m.cursor = 0
			m.step = stepReleaseType
			return m, nil
		}

		// Other bump actions go straight to tag review.
		m.newTag = formatVersion(bumped)
		m.tagMessage = m.buildTagMessage()
		return m.transitionToTagReview()
	}
}

// onReleaseTypeSelect handles stable vs pre-release choice after bump selection.
func (m model) onReleaseTypeSelect(idx int) (model, tea.Cmd) {
	switch idx {
	case 0:
		// Stable release
		m.preReleaseExplicit = false
		m.newTag = formatVersion(m.pendingBump)
		m.tagMessage = m.buildTagMessage()
		return m.transitionToTagReview()

	case 1:
		// Pre-release
		m.preReleaseExplicit = true

		// For build-number, route to full-tag edit so the user can control exact formatting safely.
		if m.preReleaseTargetIsBuild {
			suggested := formatVersion(m.pendingBump) + "-beta.1"
			m.textInput.SetValue(suggested)
			m.textInput.Placeholder = "e.g., " + suggested
			m.textInput.Focus()
			m.textInput.CursorEnd()
			m.step = stepCustomTag
			return m, textinput.Blink
		}

		// Semver flow: pick a pre-release label.
		m.choices = buildPreReleaseLabelChoicesForVersion(m.pendingBump)
		m.cursor = 0
		m.step = stepPreReleaseLabel
		return m, nil

	case 2:
		// Edit full tag
		m.textInput.SetValue(formatVersion(m.pendingBump))
		m.textInput.Placeholder = "Enter full tag"
		m.textInput.Focus()
		m.textInput.CursorEnd()
		m.step = stepCustomTag
		return m, textinput.Blink
	}
	return m, nil
}

// onPreReleaseLabelSelect handles picking a pre-release label.
func (m model) onPreReleaseLabelSelect(idx int) (model, tea.Cmd) {
	choice := m.choices[idx]

	if choice.BumpType == BumpCustom {
		// Custom label
		m.textInput.SetValue("")
		m.textInput.Placeholder = "e.g., beta"
		m.textInput.Focus()
		m.step = stepPreReleaseLabelCustom
		return m, textinput.Blink
	}

	// Use the chosen label (alpha.1, beta.1, rc.1)
	label := choice.Label + ".1"

	// Use pendingBump if we came from the release type step, otherwise bump from latest
	base := m.pendingBump
	if base.Pattern == PatternUnknown {
		// Came from the old pre-release flow (top-level BumpPreRelease option)
		base = m.latestVersion
		base.Patch = base.Patch + 1
	}
	base.PreRelease = label
	base.Raw = formatVersion(base)
	m.preReleaseExplicit = true
	m.newTag = base.Raw
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

// onCustomPreReleaseDone handles custom pre-release label input.
func (m model) onCustomPreReleaseDone(value string) (model, tea.Cmd) {
	label := value + ".1"

	// Use pendingBump if available, otherwise bump from latest
	base := m.pendingBump
	if base.Pattern == PatternUnknown {
		base = m.latestVersion
		base.Patch = base.Patch + 1
	}
	base.PreRelease = label
	base.Raw = formatVersion(base)
	m.preReleaseExplicit = true
	m.newTag = base.Raw
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

// onDescriptorDone handles descriptor input.
func (m model) onDescriptorDone(value string) (model, tea.Cmd) {
	bumped := bumpVersion(m.latestVersion, BumpDescriptor, "", value)
	m.newTag = formatVersion(bumped)
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

// onCustomTagDone handles custom tag input.
func (m model) onCustomTagDone(value string) (model, tea.Cmd) {
	m.newTag = value
	m.tagMessage = m.buildTagMessage()
	return m.transitionToTagReview()
}

func (m model) pkgVersionUpdate() *pkgmanager.VersionUpdate {
	if !m.updatePkgVersion {
		return nil
	}
	if m.pkgInfo == nil || !m.pkgInfo.Detected || !m.pkgInfo.HasVersion || m.pkgVersionTarget == "" {
		return nil
	}
	return &pkgmanager.VersionUpdate{
		Path:           m.pkgInfo.FilePath,
		CurrentVersion: m.pkgInfo.Version,
		NewVersion:     m.pkgVersionTarget,
		CommitMessage:  m.buildPkgVersionCommitMessage(),
		Manager:        m.pkgInfo.Manager,
	}
}

func (m model) buildPkgVersionCommitMessage() string {
	if m.pkgVersionTarget == "" {
		return "chore: bump version"
	}
	return "chore: bump version to " + m.pkgVersionTarget
}

// onRemotesDone handles completion of remote selection.
func (m model) onRemotesDone() (model, tea.Cmd) {
	return m.transitionToForgeRelease()
}

// onReleaseNotesModeSelect handles release notes mode selection.
func (m model) onReleaseNotesModeSelect(idx int) (model, tea.Cmd) {
	switch idx {
	case 0:
		// Auto-generate from commits
		prevTag := ""
		if m.hasLatest {
			prevTag = m.latestVersion.Raw
		}
		m.releaseNotes = generateReleaseNotes(prevTag)
		return m.transitionToSummary()

	case 1:
		// Write notes manually
		m.textArea.SetValue("")
		m.textArea.Focus()
		m.step = stepReleaseNotesInput
		return m, textarea.Blink

	case 2:
		// Empty notes
		m.releaseNotes = ""
		return m.transitionToSummary()
	}

	return m, nil
}

// buildTagMessage creates the default tag message.
func (m model) buildTagMessage() string {
	if m.flags.Message != "" {
		return m.flags.Message
	}
	tag := m.newTag
	if tag == "" {
		return "Release"
	}
	return "Release " + tag
}

// inferPreReleaseFromTag tries to infer if a tag looks like a pre-release.
func inferPreReleaseFromTag(tag string) bool {
	v, ok := parseVersion(tag)
	if !ok {
		return false
	}

	return v.HasPreRelease()
}
