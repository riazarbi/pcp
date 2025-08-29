# GitHub Workflows

This project uses GitHub Actions for automated CI/CD. The workflow system includes two main workflows:

## Workflows

### 1. CI (`ci.yml`)
**Triggers:** Push to `main`, Pull Requests to `main`

**What it does:**
- Runs on every push and PR
- Tests code formatting with `gofmt`
- Runs `go vet` for static analysis
- Executes full test suite with coverage reporting
- Enforces 80% minimum test coverage
- Tests cross-platform builds (Linux, macOS, Windows)
- Performs basic functionality test on each platform

### 2. Auto Release (`auto-release.yml`)
**Triggers:** Push to `main`, Manual dispatch

**What it does:**
- Automatically determines version bump based on commit messages:
  - **Major**: `BREAKING CHANGE`, `feat!:`, `fix!:`, etc.
  - **Minor**: `feat:`, `feature:` commits
  - **Patch**: `fix:`, `chore:`, `docs:`, etc.
- Creates semantic version tags (v1.0.0 format)
- Runs full test suite before release
- Builds cross-platform binaries:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64, arm64)
- Generates SHA256 checksums and install scripts
- Creates GitHub release with:
  - Auto-generated release notes from commits
  - Binary attachments for all platforms
  - One-liner installation instructions
- Can be manually triggered with custom version bump
- Skips release if no version-worthy changes detected

## Release Process

### Automatic Release
1. Push commits to `main` branch
2. Auto Release workflow detects changes, creates tag, and builds release
3. Binaries are built and GitHub release is created automatically

### Manual Release
1. Go to Actions → Auto Release → Run workflow
2. Select version bump type (patch/minor/major)
3. Release is created immediately in the same workflow

## Commit Message Conventions

For automatic version detection, use conventional commits:

```
feat: add new delimiter style
fix: resolve circular reference detection
BREAKING CHANGE: change CLI flag format
chore: update dependencies
docs: improve README examples
```

## Binary Distribution

Released binaries follow this naming pattern:
- `pcp-linux-amd64`
- `pcp-linux-arm64`
- `pcp-darwin-amd64` (macOS Intel)
- `pcp-darwin-arm64` (macOS Apple Silicon)
- `pcp-windows-amd64.exe`
- `pcp-windows-arm64.exe`

All binaries are stripped (`-ldflags="-s -w"`) for smaller size and include SHA256 checksums.

## Coverage Requirements

- Minimum 80% test coverage enforced in CI
- Coverage reports generated for all workflow runs
- Tests use real file system operations (no mocking)
- Cross-platform compatibility verified

## Security

- Uses `GITHUB_TOKEN` with minimal permissions
- No external secrets required
- All builds are reproducible and verifiable via checksums