# AGENTS.md - Mini ETCD Developer Guide

This document provides essential information for AI agents and developers working on the Mini ETCD codebase. It covers build instructions, code style guidelines, and project structure to ensure consistency and quality.

## 1. Project Overview

Mini ETCD is a lightweight implementation of a distributed key-value store using the Raft consensus algorithm. It supports basic key-value operations and node clustering.
The project is written in Go (version 1.14+).

## 2. Build, Run, and Lint Commands

### Building the Project
To build the executable, run the following command from the project root:

```bash
go build -o mini-etcd cmd/server/main.go
```

### Running the Application
To run the application directly:

```bash
go run cmd/server/main.go
```

**Note:** The application expects a `conf/server.ini` file to exist relative to the execution directory. Ensure you are in the project root when running.

### Testing
Currently, there are no existing test files (`*_test.go`).
To run tests (when added), use the standard Go test command:

```bash
# Run all tests
go test ./...

# Run a specific test function (verbose)
go test -v -run TestName ./package/path
```

### Linting and Verification
Use `go vet` to catch common errors:

```bash
go vet ./...
```

It is also recommended to ensure code is formatted:

```bash
gofmt -w .
```

## 3. Code Style and Conventions

Adhere strictly to the following guidelines when modifying or adding code.

### General formatting
- **Tooling:** Always run `gofmt` on modified files.
- **Indentation:** Use tabs for indentation (standard Go style).
- **Line Length:** Keep lines reasonably short (under 120 characters preferred), but do not break long lines artificially if it hurts readability.

### Naming Conventions
- **Packages:** Use short, lowercase, single-word names (e.g., `conf`, `store`, `model`).
- **Exported Identifiers:** Use **PascalCase** for functions, structs, and variables that need to be exported (e.g., `GetCurrentConfig`, `RequestBody`).
- **Internal Identifiers:** Use **camelCase** for unexported (private) identifiers (e.g., `commandFile`, `openerr`).
- **Variables:** Use concise but descriptive names. Common abbreviations (`err`, `cfg`, `wg`) are acceptable if standard.
- **Constants:** Use **PascalCase** or **camelCase**.

### Imports
Group imports into standard library and third-party/project imports.

```go
import (
    "fmt"
    "sync"
    
    "gopkg.in/ini.v1"
    "z.cn/RaftImpl/internal/config"
)
```

### Error Handling
- **Explicit Checks:** Always check errors immediately after the function call.
- **Propagation:** Return errors to the caller unless it is the main function or a top-level goroutine.
- **Logging:**
    - Current convention uses `fmt.Println` for logging errors and status updates.
    - Format: `fmt.Println("description ,err", err)` (Note: existing code often uses a space before the comma).
    - Maintain consistency with this style unless a structured logger is introduced.

### Comments and Language
- **Language:** The codebase contains comments in **Chinese** (e.g., `//是否存入日志`).
    - When modifying existing code with Chinese comments, preserve them.
    - You may use English for new comments, but be aware of the bilingual context.
- **Documentation:** Add comments for complex logic, especially in the Raft consensus implementation and storage mechanisms.

## 4. Project Structure and Architecture

### Directory Layout
- **`cmd/server/main.go`**: Entry point. Handles wiring of Config, Store, Transport, Raft, and Server.
- **`conf/`**: Configuration files (e.g., `server.ini`).
- **`internal/`**: Core application code (private to this module).
    - **`config/`**: Configuration parsing logic.
    - **`model/`**: Data structures (Nodes, Messages, Requests).
    - **`raft/`**: Core Consensus Algorithm (Vote, Heartbeat, State transitions).
    - **`store/`**: Data persistence layer (WAL + In-memory KV).
    - **`transport/`**: Network communication (RPC connection pool).
    - **`server/`**: HTTP API and RPC Server registration.
    - **`util/`**: Utility functions (encryption).

### Key Components

#### Raft Implementation
The project implements core Raft concepts:
- **Roles:** Leader, Follower, Candidate.
- **Log Replication:** Leader receives requests and replicates commands to followers.
- **Consensus:** Operations are committed after consensus.

#### Storage Engine
- **In-Memory:** Data is stored in a `map[string]string`.
- **Persistence:** Commands are serialized to JSON, encrypted, and appended to a file.
- **Recovery:** On startup, the log file is read, decrypted, and replayed to restore state.

## 5. Development Workflow for Agents

1.  **Analysis:** Before making changes, read `main.go` and relevant package files to understand the flow.
2.  **Configuration:** Verify `conf/server.ini` settings if debugging startup issues.
3.  **Implementation:**
    - Use standard Go idioms.
    - Respect the existing error handling patterns.
    - Ensure `sync.RWMutex` is used correctly in `store` to prevent data races.
4.  **Verification:**
    - Since there are no automated tests, verify changes by understanding the logic flow.
    - Suggest adding unit tests for new logic, especially for `store` and `util` packages.

## 6. Specific Rules (if applicable)

- **Dependencies:** The project uses `gopkg.in/ini.v1`. Do not introduce new heavy dependencies without good reason.
- **Concurrency:** Be careful with goroutines. `main.go` uses `sync.WaitGroup` to keep the main process alive.

---
*Generated by OpenCode Agent*
