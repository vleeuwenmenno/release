package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	flags := Flags{}

	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.StringVar(&flags.Tag, "tag", "", "Manual tag to create (skips version detection and bump menu)")
	flag.StringVar(&flags.Message, "message", "", "Tag/release message (default: \"Release <tag>\")")
	flag.BoolVar(&flags.Push, "push", false, "Automatically push tag to all remotes")
	flag.BoolVar(&flags.Release, "release", false, "Automatically create release on all detected forges")
	flag.BoolVar(&flags.DryRun, "dry-run", false, "Preview the execution plan without making changes")
	flag.BoolVar(&flags.Force, "force", false, "Proceed despite dirty working tree or existing tag")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "release -- interactive git tag & release manager\n\n")
		fmt.Fprintf(os.Stderr, "Usage: release [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  release                       Interactive mode (default)\n")
		fmt.Fprintf(os.Stderr, "  release -tag v2.0.0           Create a specific tag\n")
		fmt.Fprintf(os.Stderr, "  release -push -release        Auto-push and create forge releases\n")
		fmt.Fprintf(os.Stderr, "  release -dry-run              Preview without executing\n")
		fmt.Fprintf(os.Stderr, "  release -force                Proceed despite dirty working tree\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Println("release " + version)
		os.Exit(0)
	}

	m := initialModel(flags)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
