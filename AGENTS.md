# AGENTS.md

## Project Overview

- **Purpose:** Web based dice rolling app for table-top role playing games
- **Stack:** Go, SQLite, HTMX
- **Entry point:** ./

## Setup

_to be determined_

## Common Commands

```bash
# Build
go build main.go

# Run (dev)

go run main.go

# Test

go test

# Lint / format

go fmt
```

## Project Structure

- `./` — source code here
- `bin/` — compiled executable here
- `tests/` — for tests
- `docs/` — for documentation

## Conventions

- **Language / version:** go
- **Formatting:** gofmt
- **Naming:** standard go naming convention
- **Imports:** as needed
- **Error handling:**

## Testing

- Add or update tests for any behavior change.
- Run the full test suite before declaring work done.

## Do

- Match existing code style and patterns.
- Keep changes focused and minimal.
- Update docs/comments when behavior changes.

## Don't

- Don't commit secrets, credentials, or `.env` files.
- Don't introduce new dependencies without a clear need.
- Don't reformat unrelated files.

## Git / PR

- Branch naming: main
- Commit message style: Conventional Commits
- Run lint + tests before committing.

## Notes & Gotchas

