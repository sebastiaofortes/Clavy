# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o clavy .

# Run server
./clavy -serve 8080

# CLI: convert a single page
./clavy -pdf samples/doc.pdf -page 1 -zoom 1.5 -fmt png -output output/page1.html

# CLI: convert all pages to stdout
./clavy -pdf samples/doc.pdf

# Check Poppler dependencies
./clavy -check

# Run tests (if added)
go test ./...
go test ./converter/...   # single package
```

**System dependency:** Poppler must be installed (`brew install poppler` on macOS, `apt install poppler-utils` on Linux). The project uses `pdftohtml` and `pdfinfo` from Poppler — not the `pdftohtml` Xpdf package (incompatible flags).

## Architecture

The app runs in two modes selected by flags: `-serve <port>` for HTTP server, or `-pdf <file>` for CLI. Both modes share the same `converter` and `store` packages.

### Packages

- **`main`** ([main.go](main.go)) — HTTP handlers, embedded HTML templates, CLI runner, and entry point. All templates (upload form, PDF list, viewer) are defined as Go string constants in this file and rendered via `html/template`.
- **`converter`** ([converter/converter.go](converter/converter.go)) — Wraps Poppler's `pdftohtml` CLI via `os/exec`. Converts a PDF page to HTML, then rewrites `<img src="file.png">` tags to base64 data URIs so the output is a single self-contained HTML file.
- **`store`** ([store/store.go](store/store.go)) — Thread-safe JSON file store (`data/books.json`) for PDF metadata (filename, language, upload time). Protected by `sync.RWMutex`.

### HTTP Routes

| Method | Route | Handler |
|--------|-------|---------|
| GET | `/` | List PDFs with bookmarks |
| GET/POST | `/upload` | Upload PDF with language selection |
| GET | `/convert` | Convert page and serve viewer |
| DELETE | `/delete` | Remove PDF and metadata |
| GET | `/health` | JSON health check |

### Conversion Flow

1. `handleConvert()` parses query params (`pdf`, `page`, `zoom`, `fmt`, `lang`)
2. Calls `converter.Convert(pdfPath, opts)` → runs `pdftohtml` in a temp dir → reads output HTML → `embedImages()` inlines all images as base64
3. Calls `converter.PageCount()` → runs `pdfinfo` to get total pages
4. Renders `viewerTemplate` with HTML content and navigation controls

### Data & State

- **PDF metadata**: `data/books.json` (gitignored) — JSON map keyed by file path
- **Bookmarks & UI prefs** (theme, font size): browser `localStorage` — no server state
- **Uploads**: stored in `samples/` (gitignored); 100 MB upload limit

### Key Design Points

- Zero external Go dependencies — stdlib only
- Single binary deploys both CLI and server
- Converted HTML is fully self-contained (base64 images)
- Conversion runs with a 30-second context timeout
- Path validation guards against directory traversal on uploaded PDFs
