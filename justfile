set shell := ["bash", "-cu"]

binary    := "ihatt"
bin_dir   := "bin"
pkg       := "github.com/rlespinasse/ihatt/cmd"
version   := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags   := "-s -w -X " + pkg + ".version=" + version

# List available recipes.
default:
    @just --list

# --- Build & install -------------------------------------------------------

# Build a release-style binary into ./bin/ihatt with embedded version.
build:
    @mkdir -p {{bin_dir}}
    go build -trimpath -ldflags "{{ldflags}}" -o {{bin_dir}}/{{binary}} .
    @echo "built {{bin_dir}}/{{binary}} ({{version}})"

# Install ihatt into $GOBIN / $GOPATH/bin.
install:
    go install -trimpath -ldflags "{{ldflags}}" .

# Build a snapshot release with goreleaser (no publish).
release-snapshot:
    goreleaser build --snapshot --clean --single-target

# Remove build artifacts.
clean:
    rm -rf {{bin_dir}} dist

# --- Run -------------------------------------------------------------------

# Run any ihatt subcommand: `just run scan --root ~/code`.
run *ARGS:
    go run . {{ARGS}}

# Launch the interactive TUI.
tui:
    go run . tui

# Show today's activity across tracked repos.
today:
    go run . today

# Show yesterday's activity.
yesterday:
    go run . yesterday

# Show this week's activity.
week:
    go run . week

# Initialise config + database.
init:
    go run . init

# Scan one or more roots for git repos and index commits.
scan *ROOTS:
    go run . scan {{ if ROOTS == "" { "" } else { "--root " + replace(ROOTS, " ", " --root ") } }}

# Pull GitHub issues / PRs for tracked repos (uses gh auth).
gh-sync:
    go run . github sync

# Build the cross-reference graph between tracked repos.
xref:
    go run . xref

# --- Quality ---------------------------------------------------------------

# Format Go sources in place.
fmt:
    gofmt -w .

# Fail if any file is not gofmt-clean.
fmt-check:
    @diff=$(gofmt -l .); if [ -n "$diff" ]; then echo "unformatted files:"; echo "$diff"; exit 1; fi

# Static analysis.
vet:
    go vet ./...

# Run tests with race detector.
test:
    go test -race ./...

# Tidy go.mod / go.sum.
tidy:
    go mod tidy

# CI gate: format, vet, test.
check: fmt-check vet test

# --- Diagnostics -----------------------------------------------------------

# Print the resolved config + data paths (no side effects).
paths:
    @echo "config: ${XDG_CONFIG_HOME:-$HOME/.config}/ihatt/config.yaml"
    @echo "data:   ${XDG_DATA_HOME:-$HOME/.local/share}/ihatt/ihatt.db"

# Delete the local ihatt database (asks for confirmation).
db-reset:
    @db="${XDG_DATA_HOME:-$HOME/.local/share}/ihatt/ihatt.db"; \
     read -r -p "delete $db ? [y/N] " ans; \
     [ "$ans" = "y" ] || { echo "aborted"; exit 1; }; \
     rm -f "$db" && echo "removed $db"

# Generate a shell completion script: `just completion zsh > _ihatt`.
completion shell="bash":
    go run . completion {{shell}}
