# charm-table-cli

Ferramenta CLI para renderizar CSV, JSON e TSV como tabelas estilizadas no terminal.

> **[Read in English](README.md)**

## Funcionalidades

- Le de **stdin** ou de um **arquivo como argumento**
- Detecta formato automaticamente: **CSV**, **JSON array**, **TSV**, **JSONL**
- Estilos de borda: `rounded`, `thick`, `double`, `ascii`
- Cor do cabecalho e largura maxima configuraveis
- Construido com [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## Instalacao

```bash
go install github.com/junhinhow/charm-table-cli@latest
```

## Uso

```bash
# De um arquivo
charm-table-cli dados.csv

# Do stdin
cat dados.json | charm-table-cli

# Com opcoes
charm-table-cli --style double --color "212" --max-width 80 dados.tsv
```

### Flags

| Flag | Padrao | Descricao |
|------|--------|-----------|
| `--style` | `rounded` | Estilo da borda: `rounded`, `thick`, `double`, `ascii` |
| `--header` | `true` | Usar primeira linha como cabecalho |
| `--max-width` | `0` | Largura maxima da tabela (0 = sem limite) |
| `--color` | `63` | Cor do cabecalho (numero ANSI ou hex) |

### Formatos Suportados

- **CSV** — valores separados por virgula
- **TSV** — valores separados por tab
- **JSON** — array de objetos (`[{"chave": "valor"}, ...]`)
- **JSONL** — um objeto JSON por linha

## Exemplos

```bash
# CSV
echo "nome,idade,cidade
Alice,30,NYC
Bob,25,LA" | charm-table-cli

# JSON
echo '[{"nome":"Alice","idade":30},{"nome":"Bob","idade":25}]' | charm-table-cli

# Estilo ASCII
charm-table-cli --style ascii dados.csv
```

## Licenca

[MIT](LICENSE) - junhinhow
