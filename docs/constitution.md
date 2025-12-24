# Project Development Constitution
# Version: 1.1, Ratified: 2025-12-20

This document defines the immutable core development principles for this project. All AI Agents must unconditionally follow these principles when performing technical planning and code implementation.

---

## Principle 1: Simplicity First

**Core Tenet:** Follow Go's "less is more" philosophy. Never create unnecessary abstractions. Never introduce non-essential dependencies.

- **1.1 (YAGNI):** You Ain't Gonna Need It. Implement only features explicitly required.
- **1.2 (Standard Library First):** Unless there is an exceptionally compelling reason, always prioritize the Go standard library. For example, use `net/http` for web services, not Gin or Echo.
- **1.3 (Anti-Over-Engineering):** Avoid complex design patterns. Simple functions and data structures are superior to complex interfaces and inheritance hierarchies.

---

## Principle 2: Test-First Imperative - Non-Negotiable

**Core Tenet:** All new features or bug fixes MUST begin by writing one (or more) failing tests.

- **2.1 (TDD Cycle):** Strictly adhere to the "Red-Green-Refactor" cycle (write failing test - make it pass - refactor).
- **2.2 (Table-Driven Tests):** Unit tests MUST prioritize table-driven testing style to cover multiple inputs and edge cases.
- **2.3 (No Excessive Mocks):** Prioritize integration tests using real dependencies or fake objects (such as in-memory GitHub API mock servers) rather than over-relying on mocks.

---

## Principle 3: Clarity and Explicitness

**Core Tenet:** Code's primary purpose is to be easily understood by humans; execution by machines is secondary.

- **3.1 (Error Handling):** **Non-Negotiable**: All errors MUST be explicitly handled. Never use `_` to discard errors. When propagating errors, MUST wrap them using `fmt.Errorf("...: %w", err)`.
- **3.2 (No Global Variables):** Global variables for passing state are strictly forbidden. All dependencies MUST be explicitly injected via function parameters or struct fields.
- **3.3 (Meaningful Comments):** Comments should explain "why", not "what". All public APIs MUST have clear GoDoc comments.

---

## Principle 4: Single Responsibility

**Core Tenet:** Every package, every file, every function should do one thing well.

- **4.1 (Package Cohesion):** Packages under the `internal` directory should maintain high cohesion and low coupling. For example, the `github` package is responsible only for interacting with the GitHub API and MUST NOT contain Markdown conversion logic.
- **4.2 (Interface Segregation):** Define small, purpose-specific interfaces rather than large, all-encompassing "god interfaces".

---

## Governance

This Constitution holds the highest priority and supersedes any instructions in `CLAUDE.md` or individual session directives. Any plan generated MUST first undergo a "constitutionality review" to ensure compliance with these principles.
