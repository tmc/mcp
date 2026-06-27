### Go-Team Panel Verdict: Ready for Execution Plan

The panel has reviewed the triaged, filesystem-verified blockers **(B11–B14)** alongside the proposed post-1.0 polish items. Because the filesystem shows that the codebase compiles cleanly and passes its tests at `HEAD` `a2499b9cb` [1], we have discarded all speculative compile-time hallucinations [1] and focused purely on concrete structural, concurrency, and API-boundary corrections. 

Below is the design for the **Execution Workflow**, addressing each of your questions with a clear panel verdict.

---

### 1. Ordering: The Bisect-Safe Landing Sequence

To guarantee that the tree remains fully compilable (`go build ./...`) and passes all tests (`go test ./...`) at every single commit, the sequence must land leaf-level, non-breaking bug fixes first, followed by internal behavioral enhancements, and finally high-churn package-level type changes.

```
[B13: Darwin Read Fix] ──> [B14: Timer Leak Fixes] ──> [B12: Server Pagination] ──> [B11: Type Consolidation]
 (Platform Isolated)        (Leaf Multi-Package)          (Internal Server)           (Cross-Package Breaking)
```

#### Parallel vs. Serialized Tracks:
*   **Parallel Tracks (Independent):** 
    *   **B13 (Darwin Read Contract)** and **B14 (Timer Leaks)** are entirely independent of each other and of the rest of the core system. They can be developed concurrently in separate branches and merged in either order.
    *   **B12 (List Pagination)** is independent of the transport (B13) and cache/limiter lifecycles (B14). It can be developed in parallel, but should be merged after B13 and B14.
*   **Serialized Track (Strict Dependency):**
    *   **B11 (Type Consolidation)** must be serialized **last**. Because B11 alters package imports and types across the root module, experimental subpackages, and examples, attempting to land it first will create a massive, unstable branch where any compiler break in other files halts the entire pipeline. Landing B11 last ensures we perform high-churn refactoring on a codebase that is already verified correct on Darwin [2, 3] and leak-free [4, 5].

#### Bisect-Safe Implementation Steps:
1.  **Commit 1 (Leaf - B13):** Repair `AppleOptimizedTransport.Read` to respect `len(p)` [2, 3]. Delete the redundant `init()` panic block and the dummy `sysctlByName` stub in `platform_darwin.go` [5-7].
2.  **Commit 2 (Leaf - B14):** Add a `Close()` method to `InMemoryCache`, `TokenBucketRateLimiter`, and `SlidingWindowRateLimiter` to cleanly halt their background cleanup timers [4, 5].
3.  **Commit 3 (Internal - B12):** Update the four list handlers in `server.go` to evaluate client-supplied cursors, slice the registry data, and set `NextCursor` [5, 8]. Since `Cursor` already exists on the public request structs [9-12], this does not break API compatibility.
4.  **Commit 4 (High-Churn - B11):** Purge `protocol/types.go` [2, 13, 14] and standardize the system on the canonical root-level `types.go` and `modelcontextprotocol/` relationship [2, 13, 14].

---

### 2. B11 Scope for v1: The Surgical "Option A" Path

**Verdict: Option (a) is the only correct v1 move.**

```
                                  ┌── [mcp (root)] (Canonical Wire Types: types.go)
[DELETE protocol/types.go] ───────┼── [modelcontextprotocol/] (Legacy/Draft Reference)
                                  └── [Align LATEST_PROTOCOL_VERSION to "2025-11-25"]
```

#### Why Full Consolidation (Option B) is a Trappy Non-Goal for v1:
Attempting a full single-package type consolidation right now (e.g., deleting `modelcontextprotocol/` entirely or rewriting all core components and example servers to share a single struct definitions namespace) is extremely risky [2, 13]. 
*   `modelcontextprotocol/` is deeply integrated: it is used by 17 example servers, fuzzer tests, has its own custom JSON unmarshaling, a separate `draft/` subpackage, and implements strict "2025-03-26" specifications [2, 3, 13, 15].
*   The root `types.go` package represents a more advanced "2025-11-25" specification layer [3].
*   Combining them now would trigger a massive breaking churn across all examples and third-party consumers just before tagging, defeating the purpose of a stable v1.0.0.

#### The Surgical Path:
1.  **Delete `protocol/types.go` immediately [2, 13, 14]:** It is deadweight, only imported by 2 experimental files under `exp/` [1].
2.  **Align the version constants:** Correct the drift so that the client and server negotiation layer resolves to the canonical `LATEST_PROTOCOL_VERSION = "2025-11-25"` [3].
3.  **Document the boundary:** Clearly state in `docs/` and package doc comments that the root package `types.go` defines the canonical wire models consumed by the core SDK [2, 13], while `modelcontextprotocol/` remains as a strict reference package for legacy and draft-level compatibility [3].

---

### 3. Pure-Additive Warm-Up Tasks: B13 First

**Verdict: B13 is a pure bug fix; B14 is pure-additive. B13 must go first.**

*   **B13 (Darwin Read Contract Fix)** is a pure, self-contained **correctness fix** [2]. It is entirely confined to `platform_darwin.go` [3, 6], which compiles only on macOS via build tags [5-7]. It carries zero runtime risk for Linux, Windows, or CI runners. It is the perfect, zero-overhead task to verify local testing, CI build gates, and git-notes metadata tracking before any core code is touched.
*   **B14 (Timer/Routine Leaks)** is **purely additive** but touches multiple packages (`middleware_advanced.go` and `ratelimit.go`) [4, 5]. Because it introduces new public `Close()` methods and lifecycle requirements on cache and rate-limiting instances, it should be merged second.

---

### 4. Polish Items: What to Fold into v1 vs. What Must Wait

#### FOLD into v1 Pass (Safe, Non-Breaking, or High-Impact):
*   **String Context Keys (`progress.go`):** **Fold.** Changing `"mcp_progress"` and `"mcp_cancel_manager"` to an unexported type (e.g., `type contextKey string`) [4] is completely non-breaking [4]. Downstream consumers do not interact with the string values directly; they use `ProgressFromContext(ctx)` and `CancelManagerFromContext(ctx)` [16, 17]. This eliminates a silent namespace collision risk without breaking any public signatures.
*   **Placeholder Middleware Factories (`middleware_registry.go`):** **Fold.** Wiring up the registry factories so they initialize the real `InMemoryCache` or `CompressionMiddleware` instead of returning a `NoOpMiddleware` [18] is a critical correctness fix [18]. It prevents configuration files (JSON/YAML) from silently discarding middleware steps [18].
*   **Encoder-Pool Theater (`performance.go`):** **Fold.** The dummy `jsonEncoderPool` fetches are discarded and re-allocated anyway [19]. Replacing this with direct, clean allocation (or utilizing `.Reset(w)` on retrieval since Go 1.12 now supports it) [5, 19] is an internal-only performance cleanup that does not affect public APIs.
*   **Skipped/Deadlocking Headline Example (`example_test.go`):** **Fold.** A skipped, deadlocking example under `Example` erodes trust [2]. This must be un-commented, debugged, and verified passing to protect the SDK's entry-level documentation [2].

#### DEFER to v2 (API-Breaking or Major Churn):
*   **NotificationHandler Context Parameter (`types.go`):** **Must Defer.** Changing the public signature `func(method string, params json.RawMessage) error` to accept a `context.Context` [2] is a major breaking change that will break every custom client-side notification handler registered in the ecosystem [2, 7]. This must wait.
*   **Constructor Asymmetry (`client.go` vs `server.go`):** **Defer.** Changing `NewServer` to return `(*Server, error)` to match `NewClient` is a direct breaking API change [6]. It can be safely documented for v1.

---

### 5. Single-Most-Important-Next-Step

**Verdict: Correct the Darwin `io.Reader` buffer reslicing and compile-time guards in `platform_darwin.go` [2, 3].**

Executing this targeted correctness fix immediately achieves several critical goals:
1.  Resolves an active safety and memory corruption vulnerability on macOS [2].
2.  Purges redundant, unreachable startup panics that bypass Go's native compilation hygiene [7].
3.  Acts as the low-risk "warm-up" that establishes release-pipeline confidence without introducing cross-package API churn.

📊 Want me to generate the exact git patch to resolve the Darwin `io.Reader` reslicing and clear the redundant startup panics in `platform_darwin.go`?
