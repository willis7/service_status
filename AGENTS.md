# AGENTS.md - Coding Agent Guidelines

This document provides guidance for AI coding agents working in this repository.

## Project Overview

Go service status monitoring application that generates a static status page. Uses the standard library `net/http` for the web server and `html/template` for templating.

**Go Version:** 1.21+ (uses Go modules for dependency management)

## Build/Lint/Test Commands

```bash
make build          # Build with race detector
make run            # Run directly: go run main.go config.json
make test           # Run all tests with race detector
make coverage       # Run all tests with coverage
make vet            # Run Go vet (static analysis)
make lint           # Run golint
make docker-build   # Build Docker image
make docker-run     # Run Docker container

# Run a single test function
go test -race -v -run TestPingSuccess ./status

# Run tests matching a pattern
go test -race -v -run TestGrep ./status

# Run all tests in a specific package
go test -race -v ./status
```

## Code Style Guidelines

### Imports

Organize imports in two groups separated by a blank line:
1. Standard library (alphabetically sorted)
2. External packages

```go
import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/willis7/service_status/status"
)
```

### Formatting

- **Indentation:** Tabs (4-space width)
- **Line endings:** LF (Unix-style)
- **Final newline:** Insert, **Trailing whitespace:** Trim

See `.editorconfig` for full configuration.

### Naming Conventions

| Element | Style | Example |
|---------|-------|---------|
| Exported functions/types | PascalCase | `LoadConfiguration`, `Service` |
| Unexported functions | camelCase | `validStatus` |
| Local variables | camelCase, short | `configFile`, `resp`, `err` |
| Test variables | Short descriptive | `tc` (test case), `tt` (test table) |
| Sentinel errors | `Err` prefix | `ErrServiceUnavailable` |
| Interfaces | Verb-like or `er` suffix | `Pinger`, `PingerFactory` |
| Files | lowercase, underscore | `check_test.go` |

### Types and Structs

JSON struct tags with lowercase keys; pointer receivers; struct embedding:

```go
type Service struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Port  string `json:"port,omitempty"`
}

type Ping struct {
	Service  // embedded
}

func (p *Ping) Status() error { ... }
```

### Error Handling

Define sentinel errors at package level:

```go
var (
	ErrServiceUnavailable = errors.New("commands: service unavailable")
	ErrRegexNotFound      = errors.New("commands: regex not found")
)
```

Use `log.Fatalf` for fatal errors in main; standard `if err != nil` pattern elsewhere.

### Documentation

Godoc-style comments starting with the function/type name:

```go
// LoadConfiguration takes a configuration file and returns a Config struct
func LoadConfiguration(file string) (Config, error) { ... }
```

### Testing Patterns

**Table-driven tests:**

```go
func TestValidStatus(t *testing.T) {
	tt := []struct {
		name   string
		code   int
		output bool
	}{
		{name: "status ok", code: http.StatusOK, output: true},
		{name: "bad gateway", code: http.StatusBadGateway, output: false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if validStatus(tc.code) != tc.output {
				t.Fail()
			}
		})
	}
}
```

**HTTP test server pattern:**

```go
ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "<html><body>Hello World!</body></html>")
}))
defer ts.Close()
```

## Design Patterns Used

- **Factory Pattern:** `PingFactory`, `GrepFactory` create `Pinger` implementations
- **Interface-based design:** `Pinger` interface for polymorphic status checking
- **Closure pattern:** HTTP handlers returned as closures (see `status.Index`)
- **Struct embedding:** Composition over inheritance

## Project Structure

```
.
├── main.go              # Application entry point, config loading
├── config.json          # Service configuration
├── Makefile             # Build automation
├── status/
│   ├── check.go         # Core Pinger interface, Ping/Grep types
│   ├── check_test.go    # Unit tests
│   └── page.go          # HTTP handler, Page struct
└── templates/
    └── status.gohtml    # HTML template (Bootstrap 3.3.7)
```

## Configuration

Services are defined in `config.json`:

```json
{
  "services": [
    {"type": "ping", "url": "http://example.com"},
    {"type": "grep", "url": "http://example.com", "regex": "pattern"}
  ]
}
```

## Notes for Agents

- Always run tests with `-race` flag for race condition detection
- Use `httptest.NewServer` for HTTP mocking in tests
- Check errors before deferring close operations (current code has a bug in `LoadConfiguration`)
- Run `go mod tidy` after adding or removing dependencies

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
