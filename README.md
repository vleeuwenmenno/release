# release

An interactive git tag & release manager for the terminal. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Interactive TUI** — step-by-step flow for creating and pushing version tags
- **Semantic Versioning** — supports `vX.Y.Z` with pre-release and build metadata
- **Build Number** — supports `X.Y-N` patterns
- **Multi-remote** — push tags to multiple remotes at once
- **Forge releases** — create releases on GitHub (`gh`), GitLab (`glab`), and Gitea (`tea`)
- **Flutter support** — automatically detects `pubspec.yaml` and offers to update the version field
- **Dry-run mode** — preview the execution plan without making changes

## Installation

### Go install

```sh
go install github.com/vleeuwenmenno/release@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your `PATH`.

### Build from source

```sh
git clone https://github.com/vleeuwenmenno/release.git
cd release
go build -o release .
```

## Usage

```
release [flags]
```

### Flags

| Flag             | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| `-tag <tag>`     | Manual tag to create (skips version detection and bump menu) |
| `-message <msg>` | Tag/release message (default: `Release <tag>`)               |
| `-push`          | Automatically push tag to all remotes                        |
| `-release`       | Automatically create release on all detected forges          |
| `-dry-run`       | Preview the execution plan without making changes            |
| `-force`         | Proceed despite dirty working tree or existing tag           |

### Examples

```sh
release                        # Interactive mode (default)
release -tag v2.0.0            # Create a specific tag
release -push -release         # Auto-push and create forge releases
release -dry-run               # Preview without executing
release -force                 # Proceed despite dirty working tree
```

#### Overview of finished release process
<img width="799" height="450" alt="image" src="https://github.com/user-attachments/assets/cd4cc741-bb1c-4cc6-8950-5c3a9bf3406f" />

#### Inteligently suggest versioning
<img width="871" height="349" alt="image" src="https://github.com/user-attachments/assets/0d989acf-49e3-43a0-90a6-1c60bd47c79b" />

#### Review before doing
<img width="861" height="493" alt="image" src="https://github.com/user-attachments/assets/64cd7f46-bdf4-4cc6-a798-11cd3b2d00e3" />


## Requirements

- Git
- Optional forge CLIs for creating releases: [`gh`](https://cli.github.com/), [`glab`](https://gitlab.com/gitlab-org/cli), or [`tea`](https://gitea.com/gitea/tea)

## License

See [LICENSE](LICENSE) for details.
