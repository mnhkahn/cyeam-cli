# CLI Table Renderer Design

## Goal

Make CLI table output match the screenshot style: Unicode box borders, purple headers, and aligned monospace columns. Use one shared renderer for every table-like command so future table output stays consistent.

## Scope

Update the current list-style table outputs:

- `cyeam roadbook list`
- `cyeam cnote list`

Do not change JSON API command output, non-table text output, update/version output, or command semantics.

## Format

Tables use Unicode box-drawing characters:

```text
┌──────────┬────────────┬──────┐
│ 标题     │ 修改时间   │ 链接 │
├──────────┼────────────┼──────┤
│ 成都三日游 │ 2026-06-10 │ 链接 │
└──────────┴────────────┴──────┘
```

Headers are wrapped in purple ANSI color when color is enabled. Body cells keep the terminal default color. Border characters use the terminal default color.

## Shared Renderer

Add a shared table renderer inside the CLI package, for example:

```go
type table struct {
	Headers []tableCell
	Rows    [][]tableCell
}

func renderTable(out io.Writer, t table) error
```

`tableCell` keeps both rendered text and visible text. Rendered text may contain ANSI color or terminal hyperlink escape sequences; visible text is used for width calculations.

The renderer handles:

- Unicode box borders.
- One-space padding on both sides of every cell.
- Chinese and other wide runes as width 2.
- ANSI color escape sequences excluded from width.
- Terminal hyperlink escape sequences excluded from width by relying on each cell's visible text.
- Empty rows still render header and borders.

Column-specific truncation should remain at call sites where the domain knows the right limit, such as roadbook titles capped at display width 32.

## Command Output

`roadbook list` continues to show:

```text
标题 | 修改时间 | 链接
```

The link column keeps the terminal hyperlink. It displays only `链接`, not the raw URL.

`cnote list` changes from fixed-width `fmt.Fprintf` output to the shared renderer:

```text
文件名 | 修改时间
```

## Errors And Empty States

Keep existing empty-state messages:

- `No roadbooks found.`
- `No notes found.`

Auth and OneDrive errors are unchanged.

## Testing

Add or update unit tests for:

- `roadbook list` table output uses Unicode borders.
- Header cells are colored purple in rendered output.
- Hyperlink escape sequences do not expose the raw URL and do not break visible alignment.
- `cnote list` renders through the shared table format.
- Chinese wide characters align correctly.

## Documentation

README command examples do not need new syntax. If output examples are added later, they should use the shared Unicode table style.
