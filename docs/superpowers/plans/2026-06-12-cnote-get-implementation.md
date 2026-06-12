# CNote Get Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `cyeam cnote get <title> --format markdown|text` for terminal-friendly note reading.

**Architecture:** Reuse the existing OneDrive `ReadFile` behavior and add a small CLI-level OneDrive interface for test isolation. Convert the note HTML locally with a dependency-free formatter before writing stdout.

**Tech Stack:** Go 1.23, Cobra, standard library HTML/XML tokenization helpers.

---

## File Structure

- Modify `internal/cli/root.go`: add OneDrive dependency injection, `cnote get`, and HTML conversion helpers.
- Modify `internal/cli/root_test.go`: add fake OneDrive client tests for `cnote get` and focused converter assertions.
- Modify `README.md`: document `cnote get`.
- Modify `skills/cyeam-cli/SKILL.md`: add `cnote get` to supported command instructions.

### Task 1: Add Testable OneDrive Dependency

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

- [ ] **Step 1: Add failing command test using fake OneDrive**

Add a fake OneDrive client to `internal/cli/root_test.go` with:

```go
type fakeOneDrive struct {
	readFolder string
	readName   string
	readBody   []byte
}

func (f *fakeOneDrive) ListFolder(ctx context.Context, folderPath string) ([]onedrive.Item, error) {
	return nil, nil
}

func (f *fakeOneDrive) ReadFileByID(ctx context.Context, itemID string) ([]byte, error) {
	return nil, nil
}

func (f *fakeOneDrive) ReadFile(ctx context.Context, folderPath, filename string) ([]byte, error) {
	f.readFolder = folderPath
	f.readName = filename
	return f.readBody, nil
}

func (f *fakeOneDrive) WriteFile(ctx context.Context, folderPath, filename, contentType string, content []byte) error {
	return nil
}

func (f *fakeOneDrive) CreateShareLink(ctx context.Context, folderPath, filename string) (string, error) {
	return "", nil
}
```

Add a test that runs `cnote get 日记` and expects `Notes/日记.html` to be read.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestCnoteGetDefaultsToMarkdown -count=1`

Expected: FAIL because `Dependencies` has no fake OneDrive hook and `cnote get` is not registered.

- [ ] **Step 3: Add minimal dependency injection**

In `internal/cli/root.go`, define:

```go
type OneDriveClient interface {
	ListFolder(ctx context.Context, folderPath string) ([]onedrive.Item, error)
	ReadFileByID(ctx context.Context, itemID string) ([]byte, error)
	ReadFile(ctx context.Context, folderPath, filename string) ([]byte, error)
	WriteFile(ctx context.Context, folderPath, filename, contentType string, content []byte) error
	CreateShareLink(ctx context.Context, folderPath, filename string) (string, error)
}
```

Add `OneDrive func() OneDriveClient` to `Dependencies` and helper:

```go
func oneDriveClient(deps Dependencies) OneDriveClient {
	if deps.OneDrive != nil {
		return deps.OneDrive()
	}
	return onedrive.NewClient(auth.GetAccessToken)
}
```

Replace direct `onedrive.NewClient(auth.GetAccessToken)` calls in CLI commands with `oneDriveClient(deps)`.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/cli -count=1`

Expected: existing CLI tests pass or fail only because `cnote get` is still unimplemented.

### Task 2: Implement `cnote get` and Format Validation

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

- [ ] **Step 1: Write failing tests**

Add tests:

```go
func TestCnoteGetDefaultsToMarkdown(t *testing.T) {
	od := &fakeOneDrive{readBody: []byte(`<h1>标题</h1><p>Hello <strong>world</strong></p>`)}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Stdout: stdout,
		OneDrive: func() OneDriveClient {
			return od
		},
	})
	cmd.SetArgs([]string{"cnote", "get", "日记"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute cnote get: %v", err)
	}
	if od.readFolder != "Notes" || od.readName != "日记.html" {
		t.Fatalf("read = %s/%s", od.readFolder, od.readName)
	}
	if stdout.String() != "# 标题\n\nHello **world**\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
```

```go
func TestCnoteGetSupportsTextFormat(t *testing.T) {
	od := &fakeOneDrive{readBody: []byte(`<h2>标题</h2><p>Hello <em>world</em></p>`)}
	stdout := new(bytes.Buffer)
	cmd := NewRootCommand(Dependencies{
		Stdout: stdout,
		OneDrive: func() OneDriveClient {
			return od
		},
	})
	cmd.SetArgs([]string{"cnote", "get", "日记", "--format", "text"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute cnote get: %v", err)
	}
	if stdout.String() != "标题\n\nHello world\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
```

```go
func TestCnoteGetRejectsUnsupportedFormat(t *testing.T) {
	cmd := NewRootCommand(Dependencies{Stdout: new(bytes.Buffer)})
	cmd.SetArgs([]string{"cnote", "get", "日记", "--format", "html"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unsupported format "html"`) {
		t.Fatalf("error = %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/cli -run 'TestCnoteGet' -count=1`

Expected: FAIL because `cnote get` is not implemented.

- [ ] **Step 3: Add command**

Add `cmd.AddCommand(newCnoteGetCommand(deps))` in `newCnoteCommand`.

Implement `newCnoteGetCommand` with `--format`, validation, `oneDriveClient(deps).ReadFile`, conversion, and stdout write.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/cli -run 'TestCnoteGet' -count=1`

Expected: tests pass after converter exists.

### Task 3: Add HTML Converter Tests and Implementation

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

- [ ] **Step 1: Write converter tests**

Add tests for:

```go
func TestFormatCnoteHTMLMarkdown(t *testing.T) {
	html := `<h1>A &amp; B</h1><p>Line<br>Next</p><ul><li>One</li><li><a href="https://example.com">Link</a></li></ul>`
	got := formatCnoteHTML([]byte(html), "markdown")
	want := "# A & B\n\nLine\nNext\n\n- One\n- [Link](https://example.com)"
	if got != want {
		t.Fatalf("markdown = %q", got)
	}
}
```

```go
func TestFormatCnoteHTMLText(t *testing.T) {
	html := `<h1>A &amp; B</h1><p>Line<br>Next</p><ol><li>One</li><li><a href="https://example.com">Link</a></li></ol>`
	got := formatCnoteHTML([]byte(html), "text")
	want := "A & B\n\nLine\nNext\n\n1. One\n1. Link (https://example.com)"
	if got != want {
		t.Fatalf("text = %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/cli -run 'TestFormatCnoteHTML|TestCnoteGet' -count=1`

Expected: FAIL until `formatCnoteHTML` is implemented.

- [ ] **Step 3: Implement converter**

Use `golang.org/x/net/html` or a small tokenizer-free standard-library approach. Keep output deterministic and trim excessive blank lines.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/cli -run 'TestFormatCnoteHTML|TestCnoteGet' -count=1`

Expected: PASS.

### Task 4: Update Docs and Skill

**Files:**
- Modify: `README.md`
- Modify: `skills/cyeam-cli/SKILL.md`

- [ ] **Step 1: Update README**

Add:

```bash
# 读取笔记详情，默认输出 Markdown 风格文本
cyeam cnote get "日记"

# 读取笔记详情，输出纯文本
cyeam cnote get "日记" --format text
```

- [ ] **Step 2: Update skill**

Add:

```bash
cyeam cnote get "note-title"
cyeam cnote get "note-title" --format text
```

Update behavior notes to say `cnote get` reads `Notes/<title>.html`, requires login, and outputs Markdown by default or text with `--format text`.

- [ ] **Step 3: Run full verification**

Run: `go test ./...`

Expected: PASS.

Run: `git diff --check`

Expected: no output.
