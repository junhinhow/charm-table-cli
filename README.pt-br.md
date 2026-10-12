# charm-table-cli

Ferramenta CLI para renderizar CSV, TSV, JSON e JSONL como tabelas estilizadas no terminal.

> **[Read in English](README.md)**

## Funcionalidades

- Lê de **stdin** ou de um **arquivo como argumento**
- Detecta o formato automaticamente: **CSV**, **TSV**, **JSON array**, **JSONL**
- Estilos de borda: `rounded`, `thick`, `double`, `ascii` (ASCII de verdade: `+---+`)
- Ordem de colunas determinística em JSON/JSONL — é usada a união das chaves de
  **todos** os registros, então nenhuma coluna é descartada em silêncio
- Registros CSV/TSV com quantidades diferentes de campos são normalizados em
  vez de abortar o programa
- A largura das colunas é medida em **células de terminal**, então textos CJK e
  emoji continuam alinhados
- `--max-width` trunca com segurança e nunca ultrapassa o limite pedido
- Construído com [Lip Gloss](https://github.com/charmbracelet/lipgloss)

## Instalação

```bash
go install github.com/junhinhow/charm-table-cli@latest
```

## Uso

```bash
# De um arquivo
charm-table-cli dados.csv

# Do stdin
cat dados.json | charm-table-cli

# Com opções
charm-table-cli --style double --color "212" --max-width 80 dados.tsv
```

### Flags

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `--style` | `rounded` | Estilo da borda: `rounded`, `thick`, `double`, `ascii` |
| `--header` | `true` | Usar a primeira linha como cabeçalho (vale para todos os formatos) |
| `--max-width` | `0` | Largura máxima da tabela em colunas, bordas incluídas (0 = sem limite) |
| `--color` | `63` | Cor do cabeçalho (número ANSI ou hex) |

### Formatos Suportados

- **CSV** — valores separados por vírgula
- **TSV** — valores separados por tabulação
- **JSON** — array de objetos (`[{"chave": "valor"}, ...]`)
- **JSONL** — um objeto JSON por linha

## Comportamento

### Ordem das colunas (JSON e JSONL)

A ordem de iteração de um map em Go é aleatória, então as colunas são
coletadas de forma explícita: entram primeiro as chaves do primeiro registro
que as contém (em ordem alfabética dentro de cada registro) e depois as chaves
que só aparecem nos registros seguintes. A mesma entrada sempre produz a mesma
tabela.

```bash
echo '[{"nome":"Alice","idade":30},{"nome":"Bob","cidade":"LA"}]' | charm-table-cli
# idade | nome | cidade
```

### Registros irregulares (CSV e TSV)

Linhas com uma quantidade diferente de campos não abortam o programa: a
tabela cresce até o registro mais largo e as células que faltam ficam vazias.

```bash
printf 'a,b,c\n1,2\n3,4,5,6\n' | charm-table-cli
```

### `--header=false`

A primeira linha passa a ser tratada como dado e as colunas recebem os nomes
`Col1`, `Col2`, … Isso vale para todos os formatos (CSV, TSV, JSON e JSONL).

### `--max-width`

A largura da tabela inclui as bordas e o padding das células. Quando o
conteúdo não cabe, as colunas mais largas são reduzidas e o texto é truncado
com segurança (nunca no meio de um caractere). Um limite pequeno demais até
para as bordas é reportado como erro, em vez de gerar uma tabela quebrada.

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

```text
$ echo "nome,idade,cidade
Alice,30,NYC" | charm-table-cli --style ascii
+-------+-------+--------+
| nome  | idade | cidade |
+-------+-------+--------+
| Alice | 30    | NYC    |
+-------+-------+--------+
```

## Licença

[MIT](LICENSE) - junhinhow
