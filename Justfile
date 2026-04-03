set dotenv-load := false

plugin_dir := env("HOME") / ".grove/plugins"

# List available recipes
default:
    @just --list

# Build and install locally (no push/release needed)
dev:
    go build -o "{{ plugin_dir }}/gw-dash" .
    @echo "✓ Installed to {{ plugin_dir }}/gw-dash"
    @echo "  Run: gw dash"

# Run all checks
check: lint test

# Run tests
test *args:
    go test ./... {{ args }}

# Run tests verbose
test-v *args:
    go test ./... -v {{ args }}

# Run linter
lint:
    go vet ./...

# Build binary (local dir)
build:
    go build -o gw-dash .

# Clean build artifacts
clean:
    rm -f gw-dash
    go clean

# Run the dashboard directly (without installing as plugin)
run:
    go run .

# Tag a new release (usage: just release 0.1.0)
release version:
    git tag -a "v{{ version }}" -m "Release {{ version }}"
    git push origin "v{{ version }}"
