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

## Requirements

- Git
- Optional forge CLIs for creating releases: [`gh`](https://cli.github.com/), [`glab`](https://gitlab.com/gitlab-org/cli), or [`tea`](https://gitea.com/gitea/tea)

## License

See [LICENSE](LICENSE) for details.
