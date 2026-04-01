# charm-table-cli

CLI tool to render CSV, JSON, and TSV as beautiful styled terminal tables.

> **[Leia em Portugues (PT-BR)](README.pt-br.md)**

## Features

- Reads from **stdin** or a **file argument**
- Auto-detects format: **CSV**, **JSON array**, **TSV**, **JSONL**
- Styled borders: `rounded`, `thick`, `double`, `ascii`
- Configurable header color and max width
- Powered by [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## Install

```bash
go install github.com/junhinhow/charm-table-cli@latest
```

## Usage

```bash
# From file
charm-table-cli data.csv

# From stdin
cat data.json | charm-table-cli

# With options
charm-table-cli --style double --color "212" --max-width 80 data.tsv
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--style` | `rounded` | Border style: `rounded`, `thick`, `double`, `ascii` |
| `--header` | `true` | Use first row as header |
| `--max-width` | `0` | Max table width (0 = unlimited) |
| `--color` | `63` | Header color (ANSI number or hex) |

### Supported Formats

- **CSV** — comma-separated values
- **TSV** — tab-separated values
- **JSON** — array of objects (`[{"key": "value"}, ...]`)
- **JSONL** — one JSON object per line

## Examples

```bash
# CSV
echo "name,age,city
Alice,30,NYC
Bob,25,LA" | charm-table-cli

# JSON
echo '[{"name":"Alice","age":30},{"name":"Bob","age":25}]' | charm-table-cli

# ASCII style
charm-table-cli --style ascii data.csv
```

## License

[MIT](LICENSE) - junhinhow
