package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// maxJSONLLineSize é o tamanho máximo aceito para uma linha JSONL (16 MiB).
// O limite padrão do bufio.Scanner (64 KiB) é pequeno demais para payloads
// grandes e a falha só aparecia depois, como "nenhum registro encontrado".
const maxJSONLLineSize = 16 * 1024 * 1024

// parseJSONArray processa um array JSON e extrai cabeçalhos e linhas.
func parseJSONArray(data string, useHeader bool) ([]string, [][]string, error) {
	var records []map[string]any
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return nil, nil, fmt.Errorf("JSON inválido: esperava um array de objetos: %w", err)
	}

	if len(records) == 0 {
		return nil, nil, errors.New("array JSON vazio: nenhum registro para renderizar")
	}

	headers, rows := buildTable(records, useHeader)

	return headers, rows, nil
}

// parseJSONL processa linhas JSON (um objeto por linha).
func parseJSONL(data string, useHeader bool) ([]string, [][]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxJSONLLineSize)

	var records []map[string]any
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, nil, fmt.Errorf("JSONL inválido na linha %d: %w", lineNumber, err)
		}

		records = append(records, record)
	}

	// O erro do scanner precisa ser verificado: sem isso, uma linha maior que
	// o limite do buffer produziria a mensagem enganosa "nenhum registro
	// JSONL encontrado".
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("erro ao ler JSONL: %w", err)
	}

	if len(records) == 0 {
		return nil, nil, errors.New("nenhum registro JSONL encontrado")
	}

	headers, rows := buildTable(records, useHeader)

	return headers, rows, nil
}

// parseDelimited processa dados CSV ou TSV.
func parseDelimited(data string, delimiter rune, useHeader bool) ([]string, [][]string, error) {
	reader := csv.NewReader(strings.NewReader(data))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	// Registros com quantidades diferentes de campos são aceitos e
	// normalizados; sem isso o csv.Reader aborta a leitura com
	// "wrong number of fields".
	reader.FieldsPerRecord = -1

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler dados %s: %w", delimiterName(delimiter), err)
	}

	if len(allRows) == 0 {
		return nil, nil, fmt.Errorf("nenhum registro encontrado nos dados %s", delimiterName(delimiter))
	}

	allRows = normalizeRows(allRows)

	var headers []string
	var rows [][]string

	if useHeader {
		// A primeira linha é sempre o cabeçalho, mesmo quando o arquivo tem
		// apenas uma linha (caso em que não existe nenhuma linha de dados).
		headers = allRows[0]
		rows = allRows[1:]
	} else {
		headers = genericHeaders(len(allRows[0]))
		rows = allRows
	}

	return headers, rows, nil
}

// normalizeRows completa com células vazias os registros que têm menos campos
// que o registro mais largo da tabela.
func normalizeRows(rows [][]string) [][]string {
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	for i, row := range rows {
		if len(row) == maxCols {
			continue
		}
		normalized := make([]string, maxCols)
		copy(normalized, row)
		rows[i] = normalized
	}

	return rows
}

// buildTable monta cabeçalhos e linhas a partir de uma lista de registros JSON.
func buildTable(records []map[string]any, useHeader bool) ([]string, [][]string) {
	keys := collectKeys(records)

	headers := keys
	if !useHeader {
		headers = genericHeaders(len(keys))
	}

	rows := make([][]string, 0, len(records))
	for _, record := range records {
		row := make([]string, len(keys))
		for i, key := range keys {
			row[i] = formatValue(record[key])
		}
		rows = append(rows, row)
	}

	return headers, rows
}

// collectKeys une as chaves de todos os registros preservando a ordem de
// aparição: as chaves do primeiro registro que as contém entram primeiro e,
// dentro de cada registro, a ordem é alfabética — assim a saída é
// determinística entre execuções (a ordem de iteração de um map em Go é
// aleatória).
func collectKeys(records []map[string]any) []string {
	seen := make(map[string]struct{})
	keys := make([]string, 0)

	for _, record := range records {
		current := make([]string, 0, len(record))
		for key := range record {
			current = append(current, key)
		}
		sort.Strings(current)

		for _, key := range current {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}

	return keys
}

// genericHeaders gera nomes de coluna genéricos (Col1, Col2, ...).
func genericHeaders(count int) []string {
	headers := make([]string, count)
	for i := range headers {
		headers[i] = fmt.Sprintf("Col%d", i+1)
	}
	return headers
}

// formatValue converte um valor do JSON para string.
func formatValue(value any) string {
	switch val := value.(type) {
	case nil:
		return ""
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		// FormatFloat evita a conversão para int64, que estoura em números
		// grandes, e mantém inteiros sem casas decimais.
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		encoded, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(encoded)
	}
}

// delimiterName devolve um nome legível para o delimitador.
func delimiterName(delimiter rune) string {
	if delimiter == '\t' {
		return "TSV"
	}
	return "CSV"
}
