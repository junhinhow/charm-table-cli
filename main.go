package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

// Estilos de borda disponiveis para a tabela
var borderStyles = map[string]lipgloss.Border{
	"rounded": lipgloss.RoundedBorder(),
	"thick":   lipgloss.ThickBorder(),
	"double":  lipgloss.DoubleBorder(),
	"ascii":   lipgloss.NormalBorder(),
}

// Opcoes de linha de comando
type options struct {
	style    string
	header   bool
	maxWidth int
	color    string
	file     string
}

func main() {
	opts := parseFlags()

	data, err := readInput(opts.file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler entrada: %v\n", err)
		os.Exit(1)
	}

	if len(data) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: entrada vazia")
		os.Exit(1)
	}

	headers, rows, err := detectAndParse(data, opts.header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao processar dados: %v\n", err)
		os.Exit(1)
	}

	output := renderTable(headers, rows, opts)
	fmt.Println(output)
}

// parseFlags processa os argumentos da linha de comando
func parseFlags() options {
	style := flag.String("style", "rounded", "Estilo da borda: rounded, thick, double, ascii")
	header := flag.Bool("header", true, "Primeira linha como cabecalho")
	maxWidth := flag.Int("max-width", 0, "Largura maxima da tabela (0 = sem limite)")
	color := flag.String("color", "63", "Cor do cabecalho (numero ANSI ou hex)")
	flag.Parse()

	file := ""
	if flag.NArg() > 0 {
		file = flag.Arg(0)
	}

	return options{
		style:    *style,
		header:   *header,
		maxWidth: *maxWidth,
		color:    *color,
		file:     file,
	}
}

// readInput le dados do stdin ou de um arquivo
func readInput(file string) (string, error) {
	var reader io.Reader

	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return "", fmt.Errorf("nao foi possivel abrir arquivo %s: %w", file, err)
		}
		defer f.Close()
		reader = f
	} else {
		// Verifica se ha dados no stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return "", fmt.Errorf("nenhuma entrada fornecida — use pipe ou passe um arquivo como argumento")
		}
		reader = os.Stdin
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// detectAndParse detecta o formato e faz o parse dos dados
func detectAndParse(data string, useHeader bool) ([]string, [][]string, error) {
	trimmed := strings.TrimSpace(data)

	// Tenta JSON array
	if strings.HasPrefix(trimmed, "[") {
		return parseJSONArray(trimmed)
	}

	// Tenta JSONL (cada linha e um objeto JSON)
	if strings.HasPrefix(trimmed, "{") {
		return parseJSONL(trimmed)
	}

	// Tenta TSV (detecta tabs)
	if strings.Contains(strings.SplitN(trimmed, "\n", 2)[0], "\t") {
		return parseDelimited(trimmed, '\t', useHeader)
	}

	// Fallback: CSV
	return parseDelimited(trimmed, ',', useHeader)
}

// parseJSONArray processa um array JSON e extrai cabecalhos e linhas
func parseJSONArray(data string) ([]string, [][]string, error) {
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return nil, nil, fmt.Errorf("JSON invalido: %w", err)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("array JSON vazio")
	}

	// Extrai chaves do primeiro registro como cabecalhos
	headers := extractKeys(records[0])

	var rows [][]string
	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = formatValue(rec[h])
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

// parseJSONL processa linhas JSON (um objeto por linha)
func parseJSONL(data string) ([]string, [][]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	var records []map[string]interface{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, nil, fmt.Errorf("JSONL invalido na linha: %w", err)
		}
		records = append(records, rec)
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("nenhum registro JSONL encontrado")
	}

	headers := extractKeys(records[0])

	var rows [][]string
	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = formatValue(rec[h])
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

// parseDelimited processa dados CSV ou TSV
func parseDelimited(data string, delimiter rune, useHeader bool) ([]string, [][]string, error) {
	reader := csv.NewReader(strings.NewReader(data))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao processar dados delimitados: %w", err)
	}

	if len(allRows) == 0 {
		return nil, nil, fmt.Errorf("dados vazios")
	}

	var headers []string
	var rows [][]string

	if useHeader && len(allRows) > 1 {
		headers = allRows[0]
		rows = allRows[1:]
	} else {
		// Gera cabecalhos numericos
		numCols := len(allRows[0])
		headers = make([]string, numCols)
		for i := range headers {
			headers[i] = fmt.Sprintf("Col%d", i+1)
		}
		rows = allRows
	}

	return headers, rows, nil
}

// extractKeys extrai as chaves de um mapa mantendo ordem de insercao
func extractKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// formatValue converte um valor interface{} para string
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// renderTable renderiza a tabela estilizada com lipgloss
func renderTable(headers []string, rows [][]string, opts options) string {
	border, ok := borderStyles[opts.style]
	if !ok {
		border = lipgloss.RoundedBorder()
	}

	// Calcula a largura de cada coluna
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				w := utf8.RuneCountInString(cell)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}

	// Aplica max-width se definido
	if opts.maxWidth > 0 {
		totalWidth := 0
		for _, w := range colWidths {
			totalWidth += w + 3 // padding + separador
		}
		if totalWidth > opts.maxWidth {
			excess := totalWidth - opts.maxWidth
			perCol := excess/len(colWidths) + 1
			for i := range colWidths {
				colWidths[i] -= perCol
				if colWidths[i] < 3 {
					colWidths[i] = 3
				}
			}
		}
	}

	// Estilo do cabecalho
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(opts.color)).
		Padding(0, 1)

	// Estilo das celulas
	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	// Monta a tabela como string
	var sb strings.Builder

	// Borda superior
	sb.WriteString(border.TopLeft)
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat(border.Top, w+2))
		if i < len(colWidths)-1 {
			sb.WriteString(border.MiddleTop)
		}
	}
	sb.WriteString(border.TopRight)
	sb.WriteString("\n")

	// Linha de cabecalho
	sb.WriteString(border.Left)
	for i, h := range headers {
		content := truncate(h, colWidths[i])
		styled := headerStyle.Width(colWidths[i]).Render(content)
		sb.WriteString(styled)
		if i < len(headers)-1 {
			sb.WriteString(border.Left)
		}
	}
	sb.WriteString(border.Right)
	sb.WriteString("\n")

	// Separador cabecalho/corpo
	sb.WriteString(border.MiddleLeft)
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat(border.Bottom, w+2))
		if i < len(colWidths)-1 {
			sb.WriteString(border.Middle)
		}
	}
	sb.WriteString(border.MiddleRight)
	sb.WriteString("\n")

	// Linhas de dados
	for _, row := range rows {
		sb.WriteString(border.Left)
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			content := truncate(cell, colWidths[i])
			styled := cellStyle.Width(colWidths[i]).Render(content)
			sb.WriteString(styled)
			if i < len(headers)-1 {
				sb.WriteString(border.Left)
			}
		}
		sb.WriteString(border.Right)
		sb.WriteString("\n")
	}

	// Borda inferior
	sb.WriteString(border.BottomLeft)
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat(border.Bottom, w+2))
		if i < len(colWidths)-1 {
			sb.WriteString(border.MiddleBottom)
		}
	}
	sb.WriteString(border.BottomRight)

	return sb.String()
}

// truncate corta o texto se exceder a largura maxima
func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}
