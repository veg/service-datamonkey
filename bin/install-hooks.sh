#!/bin/bash
# Install Git hooks for the project

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$BIN_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing Git hooks..."

# Install pre-commit hook
if [ -f "$BIN_DIR/pre-commit" ]; then
    cp "$BIN_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
    chmod +x "$HOOKS_DIR/pre-commit"
    echo "✓ Installed pre-commit hook (gofmt)"
else
    echo "✗ pre-commit hook not found at $BIN_DIR/pre-commit"
    exit 1
fi

echo ""
echo "Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will automatically:"
echo "  - Format Go files with gofmt before each commit"
echo "  - Run golangci-lint (or go vet if not installed)"
echo "  - Re-stage formatted files"
echo ""
echo "To install golangci-lint for comprehensive checks:"
echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
echo ""
echo "To bypass the hook (not recommended):"
echo "  git commit --no-verify"
