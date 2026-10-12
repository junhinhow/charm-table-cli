package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// byteOrderMark é o BOM UTF-8, que às vezes aparece no início de arquivos
// exportados por planilhas e ferramentas de BI.
const byteOrderMark = "\ufeff"

// readInput lê os dados do arquivo informado ou, na ausência dele, do stdin.
func readInput(file string, stdin *os.File) (string, error) {
	var reader io.Reader

	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return "", fmt.Errorf("não foi possível abrir o arquivo %q: %w", file, err)
		}
		defer f.Close()
		reader = f
	} else {
		if stdin == nil {
			return "", errors.New("nenhuma entrada fornecida — use pipe ou passe um arquivo como argumento")
		}
		terminal, err := isCharDevice(stdin)
		if err != nil {
			return "", err
		}
		if terminal {
			return "", errors.New("nenhuma entrada fornecida — use pipe ou passe um arquivo como argumento")
		}
		reader = stdin
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("não foi possível ler a entrada: %w", err)
	}

	return strings.TrimPrefix(string(b), byteOrderMark), nil
}

// isCharDevice informa se o arquivo é um terminal (dispositivo de caracteres).
// O erro do Stat é sempre verificado: descartá-lo provoca um nil dereference
// quando o stdin está fechado.
func isCharDevice(f *os.File) (bool, error) {
	stat, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("não foi possível inspecionar o stdin: %w", err)
	}
	return stat.Mode()&os.ModeCharDevice != 0, nil
}

// detectAndParse detecta o formato dos dados e faz o parse do conteúdo.
func detectAndParse(data string, useHeader bool) ([]string, [][]string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(data, byteOrderMark))

	switch {
	case trimmed == "":
		return nil, nil, errors.New("entrada vazia: nenhum dado para renderizar")
	case strings.HasPrefix(trimmed, "["):
		return parseJSONArray(trimmed, useHeader)
	case strings.HasPrefix(trimmed, "{"):
		return parseJSONL(trimmed, useHeader)
	}

	return parseDelimited(trimmed, detectDelimiter(trimmed), useHeader)
}

// detectDelimiter escolhe o delimitador mais provável entre tabulação e
// vírgula. Separadores que estejam dentro de campos citados não contam, o que
// evita tratar um CSV com tabulação em um campo citado como se fosse TSV.
func detectDelimiter(data string) rune {
	linha := firstNonEmptyLine(data)

	if countUnquoted(linha, '\t') > 0 {
		return '\t'
	}
	if countUnquoted(linha, ',') > 0 {
		return ','
	}

	// Sem separadores visíveis na primeira linha: compara a quantidade de
	// campos obtida com cada delimitador candidato.
	if countFirstRecordFields(data, '\t') > countFirstRecordFields(data, ',') {
		return '\t'
	}

	return ','
}

// firstNonEmptyLine devolve a primeira linha não vazia do conteúdo.
func firstNonEmptyLine(data string) string {
	for _, line := range strings.Split(data, "\n") {
		if linha := strings.TrimSpace(line); linha != "" {
			return linha
		}
	}
	return ""
}

// countUnquoted conta as ocorrências do separador que estejam fora de campos
// citados entre aspas.
func countUnquoted(line string, sep byte) int {
	total := 0
	inQuotes := false

	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			// Aspas duplicadas dentro de um campo citado são escape, não
			// delimitador de campo.
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				i++
				continue
			}
			inQuotes = !inQuotes
		case sep:
			if !inQuotes {
				total++
			}
		}
	}

	return total
}

// countFirstRecordFields conta os campos do primeiro registro lido com o
// delimitador informado.
func countFirstRecordFields(data string, delimiter rune) int {
	reader := csv.NewReader(strings.NewReader(data))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	record, err := reader.Read()
	if err != nil {
		return 0
	}

	return len(record)
}
