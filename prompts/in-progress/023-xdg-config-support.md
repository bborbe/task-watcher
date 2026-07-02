---
status: committing
summary: Added XDG config directory support to task-watcher with graceful fallback to legacy ~/.task-watcher/
execution_id: task-watcher-exec-023-xdg-config-support
dark-factory-version: v0.191.0
created: "2026-07-02T00:00:00Z"
queued: "2026-07-02T16:19:32Z"
started: "2026-07-02T16:45:11Z"
completed: "2026-07-02T16:47:29Z"
---

<summary>
- Config directory follows XDG spec: `$XDG_CONFIG_HOME/task-watcher/` (or `~/.config/task-watcher/` by default) is checked first
- Legacy `~/.task-watcher/` is the fallback when XDG dir does not exist
- When neither dir exists, the XDG path is used as default (dir is created on first config write, which is a separate concern)
- `findConfigDir()` encapsulates the discovery logic: check XDG, fall back to legacy, default to XDG
- `resolveFilePath()` calls `findConfigDir()` when no explicit `--config` path is given
- CLI help text shows the XDG path as default with legacy fallback noted
- Five test scenarios cover all path resolution combinations
</summary>

<objective>
Add XDG config directory support so task-watcher follows the XDG Base Directory specification, with graceful fallback to the legacy `~/.task-watcher/` directory for existing installations.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` for Ginkgo/Gomega conventions.
Read `/home/node/.claude/plugins/marketplaces/coding/docs/go-error-wrapping-guide.md` for `bborbe/errors` patterns.
Read `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md` for changelog entry format.

Key files to read before making changes:
- `pkg/config/config.go` — contains `resolveFilePath()` and `loader.Load()`, the only functions that need changes
- `pkg/config/config_test.go` — existing tests; add new test block, do not delete existing tests
- `pkg/cli/cli.go` — CLI help text for the `--config` flag
</context>

<requirements>
### 1. Read `pkg/config/config.go` before making changes

Verify the current state of `resolveFilePath()` and the `loader` struct. The file uses the `os` package already, so `os.UserConfigDir()` will be available without new imports.

### 2. Add `findConfigDir()` to `pkg/config/config.go`

Add a new unexported function that discovers the config directory using XDG-first logic. Place this function above `resolveFilePath()`.

```go
// findConfigDir returns the config directory path.
// Prefers XDG ($XDG_CONFIG_HOME/task-watcher or ~/.config/task-watcher).
// Falls back to legacy ~/.task-watcher/ if it exists.
// When neither exists, returns the XDG path as default.
func findConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	xdgDir := filepath.Join(userConfigDir, "task-watcher")

	// If XDG dir exists, use it.
	if info, err := os.Stat(xdgDir); err == nil && info.IsDir() {
		return xdgDir, nil
	}

	// Fall back to legacy dir if it exists.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	legacyDir := filepath.Join(homeDir, ".task-watcher")
	if info, err := os.Stat(legacyDir); err == nil && info.IsDir() {
		return legacyDir, nil
	}

	// Neither exists — default to XDG path.
	return xdgDir, nil
}
```

### 3. Update `resolveFilePath()` to call `findConfigDir()` when `filePath` is empty

Current code in `resolveFilePath()` (lines 93-102):

```go
func resolveFilePath(filePath string) (string, error) {
	if filePath != "" {
		return filePath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".task-watcher", "config.yaml"), nil
}
```

Replace the `homeDir` + `filepath.Join` block inside the `filePath == ""` branch with a call to `findConfigDir()`:

```go
func resolveFilePath(filePath string) (string, error) {
	if filePath != "" {
		return filePath, nil
	}
	configDir, err := findConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}
```

The filename `config.yaml` is unchanged — only the directory resolution changes.

### 4. Update CLI help text in `pkg/cli/cli.go` and `README.md`

Current CLI line (line 110):

```go
rootCmd.Flags().
    StringVar(&configPath, "config", "", "path to config YAML file (default: ~/.task-watcher/config.yaml)")
```

Change to show the XDG path with legacy fallback:

```go
rootCmd.Flags().
    StringVar(&configPath, "config", "", "path to config YAML file (default: $XDG_CONFIG_HOME/task-watcher/config.yaml, fallback: ~/.task-watcher/config.yaml)")
```

Also update `README.md` line 11 from:

```
# default config: ~/.task-watcher/config.yaml
```

to:

```
# default config: $XDG_CONFIG_HOME/task-watcher/config.yaml (fallback: ~/.task-watcher/config.yaml)
```

### 5. Add tests to `pkg/config/config_test.go`

Read the existing test file first. Add a new `Context("findConfigDir", ...)` block below the existing `Context("file resolution", ...)` block (after line 484). Do NOT modify any existing tests.

The `findConfigDir` function is unexported, so tests live in `package config_test`. Test it indirectly through `resolveFilePath` by passing an empty `filePath` to `config.NewLoader("")` and verifying the resolved path appears in the error message returned by `Load`. Use `GinkgoT().Setenv("XDG_CONFIG_HOME", ...)` to control the XDG behavior and `os.MkdirAll` to create controlled directory trees.

Before writing each test, the test must declare `ctx := context.Background()` — follow the pattern used in the existing `BeforeEach` block (`ctx = context.Background()`). Each test uses its own `homeDir := GinkgoT().TempDir()` for isolation.

Write five scenarios:

**Scenario 1: XDG-only** — XDG dir exists, legacy dir does not exist. Resolved path begins with the XDG dir.

```go
It("prefers XDG config dir when it exists and legacy does not", func() {
    ctx := context.Background()
    homeDir := GinkgoT().TempDir()
    xdgDir := filepath.Join(homeDir, ".config", "task-watcher")
    Expect(os.MkdirAll(xdgDir, 0755)).To(Succeed())
    GinkgoT().Setenv("HOME", homeDir)
    GinkgoT().Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

    _, err := config.NewLoader("").Load(ctx)
    Expect(err).To(HaveOccurred())
    Expect(err.Error()).To(ContainSubstring(filepath.Join(xdgDir, "config.yaml")))
})
```

**Scenario 2: Legacy-only** — XDG dir does not exist, legacy dir exists. Resolved path begins with legacy dir.

```go
It("falls back to legacy dir when XDG dir does not exist but legacy does", func() {
    ctx := context.Background()
    homeDir := GinkgoT().TempDir()
    legacyDir := filepath.Join(homeDir, ".task-watcher")
    Expect(os.MkdirAll(legacyDir, 0755)).To(Succeed())
    GinkgoT().Setenv("HOME", homeDir)
    GinkgoT().Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".nonexistent-config"))

    _, err := config.NewLoader("").Load(ctx)
    Expect(err).To(HaveOccurred())
    Expect(err.Error()).To(ContainSubstring(filepath.Join(legacyDir, "config.yaml")))
})
```

**Scenario 3: Neither exists** — both XDG and legacy dirs do not exist. Resolved path defaults to the XDG path.

```go
It("defaults to XDG path when neither XDG nor legacy dir exists", func() {
    ctx := context.Background()
    homeDir := GinkgoT().TempDir()
    GinkgoT().Setenv("HOME", homeDir)
    GinkgoT().Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

    _, err := config.NewLoader("").Load(ctx)
    Expect(err).To(HaveOccurred())
    expectedPath := filepath.Join(homeDir, ".config", "task-watcher", "config.yaml")
    Expect(err.Error()).To(ContainSubstring(expectedPath))
})
```

**Scenario 4: Both exist, prefers XDG** — both dirs exist. Resolved path is from XDG dir.

```go
It("prefers XDG config dir when both XDG and legacy dirs exist", func() {
    ctx := context.Background()
    homeDir := GinkgoT().TempDir()
    xdgDir := filepath.Join(homeDir, ".config", "task-watcher")
    legacyDir := filepath.Join(homeDir, ".task-watcher")
    Expect(os.MkdirAll(xdgDir, 0755)).To(Succeed())
    Expect(os.MkdirAll(legacyDir, 0755)).To(Succeed())
    GinkgoT().Setenv("HOME", homeDir)
    GinkgoT().Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

    _, err := config.NewLoader("").Load(ctx)
    Expect(err).To(HaveOccurred())
    Expect(err.Error()).To(ContainSubstring(filepath.Join(xdgDir, "config.yaml")))
})
```

**Scenario 5: XDG path is a file not a dir** — XDG path exists but is a regular file, not a directory. The `os.Stat` + `info.IsDir()` check rejects it, so it falls through to the legacy check. Legacy dir exists so it should be picked.

```go
It("ignores XDG path when it exists as a file not a directory", func() {
    ctx := context.Background()
    homeDir := GinkgoT().TempDir()
    xdgConfigDir := filepath.Join(homeDir, ".config")
    Expect(os.MkdirAll(xdgConfigDir, 0755)).To(Succeed())
    // Create "task-watcher" as a regular file inside .config, not a directory.
    xdgFilePath := filepath.Join(xdgConfigDir, "task-watcher")
    Expect(os.WriteFile(xdgFilePath, []byte("not a dir"), 0644)).To(Succeed())
    // Legacy dir exists so it should be picked.
    legacyDir := filepath.Join(homeDir, ".task-watcher")
    Expect(os.MkdirAll(legacyDir, 0755)).To(Succeed())
    GinkgoT().Setenv("HOME", homeDir)
    GinkgoT().Setenv("XDG_CONFIG_HOME", xdgConfigDir)

    _, err := config.NewLoader("").Load(ctx)
    Expect(err).To(HaveOccurred())
    Expect(err.Error()).To(ContainSubstring(filepath.Join(legacyDir, "config.yaml")))
})
```

### 6. Run `make test`

All existing and new tests must pass.

### 7. Run `make precommit`

Must pass.

### 8. Update `CHANGELOG.md`

Add under `## v0.18.2` (the current top section):

```
- feat: Add XDG config directory support ($XDG_CONFIG_HOME/task-watcher/) with legacy ~/.task-watcher/ fallback
```
</requirements>

<constraints>
- Default path logic lives in `pkg/config/config.go`, not in CLI code
- `--config` flag overrides all discovery when provided (unchanged behavior)
- `os.UserConfigDir()` is the canonical XDG implementation — do not hardcode `~/.config`
- Existing config loading, validation, and parsing logic is unchanged — only the directory discovery changes
- `Loader` interface and `NewLoader` signature are unchanged
- Ginkgo/Gomega tests only — no stdlib table tests
- Error wrapping uses `github.com/bborbe/errors` — never `fmt.Errorf`
- Do NOT modify `pkg/config/suite_test.go`
- Do NOT delete or modify any existing test — only add new tests
- Do NOT commit — dark-factory handles git
</constraints>

<verification>
```bash
make test
make precommit
```
Must exit with code 0.
</verification>
