# Contributing to DioramaOps

Thank you for your interest in contributing to DioramaOps! We welcome bug fixes, documentation improvements, and thoughtful feature contributions.

---

## Development Setup

### Prerequisites
- **Go** >= 1.24
- **Bun** >= 1.0 (or Node.js >= 20)
- **Git**

### Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/ramhandean/diorama-ops.git
   cd dioramaops
   ```

2. Install frontend dependencies:
   ```bash
   cd web && bun install && cd ..
   ```

3. Build and run tests:
   ```bash
   go test -v ./...
   cd web && bun run build && cd ..
   ```

4. Start local development server:
   ```bash
   # Terminal 1: Frontend dev server with HMR
   cd web && bun run dev

   # Terminal 2: Backend server
   DEMO_MODE=true ADMIN_SECRET="devsecret" go run ./cmd/dioramaops serve
   ```

---

## Code Guidelines

- **Go Idioms:** Follow standard Go idioms (`gofmt`, error wrapping, explicit context cancellation). No unchecked panics.
- **Frontend Craft & Aesthetics:** We follow strict 60-30-10 neutral color rules, border-first component architecture, and WCAG AA contrast standards. Avoid generic AI gradients or unneeded decorative clutter.
- **Privacy First:** Never log raw IP addresses, cookies, or user identifiers.
- **Pull Requests:** Ensure all unit tests pass (`go test ./...`) and the frontend bundle compiles (`cd web && bun run build`) before opening a PR.
