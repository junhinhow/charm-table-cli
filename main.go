// Command charm-table-cli renderiza dados CSV, TSV, JSON e JSONL como tabelas
// estilizadas no terminal.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// options representa as opções da linha de comando.
type options struct {
	style    string
	header   bool
	maxWidth int
	color    string
	file     string
}

func main() {
	opts, err := parseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if err := run(opts, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags processa os argumentos da linha de comando.
func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("charm-table-cli", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	style := fs.String("style", "rounded", "Estilo da borda: rounded, thick, double, ascii")
	header := fs.Bool("header", true, "Usa a primeira linha como cabeçalho")
	maxWidth := fs.Int("max-width", 0, "Largura máxima da tabela em colunas (0 = sem limite)")
	color := fs.String("color", "63", "Cor do cabeçalho (número ANSI ou hex)")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	file := ""
	if fs.NArg() > 0 {
		file = fs.Arg(0)
	}

	return options{
		style:    *style,
		header:   *header,
		maxWidth: *maxWidth,
		color:    *color,
		file:     file,
	}, nil
}

// run executa o fluxo completo: leitura da entrada, detecção do formato,
// interpretação dos dados e renderização da tabela.
func run(opts options, stdin *os.File, stdout io.Writer) error {
	if _, ok := borderStyles[opts.style]; !ok {
		return fmt.Errorf("estilo de borda desconhecido %q: use um de %s",
			opts.style, strings.Join(borderStyleNames(), ", "))
	}

	data, err := readInput(opts.file, stdin)
	if err != nil {
		return err
	}

	if strings.TrimSpace(data) == "" {
		return errors.New("entrada vazia: nenhum dado para renderizar")
	}

	headers, rows, err := detectAndParse(data, opts.header)
	if err != nil {
		return err
	}

	output, err := renderTable(headers, rows, opts)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(stdout, output); err != nil {
		return fmt.Errorf("não foi possível escrever a tabela na saída: %w", err)
	}

	return nil
}
