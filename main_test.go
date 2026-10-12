package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// linhaSemANSI remove as sequências de escape da linha.
func linhaSemANSI(s string) string {
	return ansi.Strip(s)
}

// largurasDasLinhas devolve a largura de exibição de cada linha da tabela.
func largurasDasLinhas(tabela string) []int {
	linhas := strings.Split(strings.TrimRight(tabela, "\n"), "\n")
	larguras := make([]int, 0, len(linhas))
	for _, linha := range linhas {
		larguras = append(larguras, ansi.StringWidth(linhaSemANSI(linha)))
	}
	return larguras
}

// TestRenderTableNaoQuebraConteudoLargo cobre o achado 1: no lipgloss v2,
// Width(n) é a largura total do bloco (conteúdo + padding). Passar a largura
// do conteúdo fazia a célula mais larga quebrar em duas linhas.
func TestRenderTableNaoQuebraConteudoLargo(t *testing.T) {
	tabela, err := renderTable(
		[]string{"name", "age", "city"},
		[][]string{{"Alice", "30", "NYC"}},
		options{style: "rounded", color: "63"},
	)
	if err != nil {
		t.Fatalf("renderTable retornou erro inesperado: %v", err)
	}

	linhas := strings.Split(strings.TrimRight(tabela, "\n"), "\n")
	if len(linhas) != 5 {
		t.Fatalf("esperava 5 linhas (borda, cabeçalho, separador, dados, borda), obtizou %d:\n%s",
			len(linhas), tabela)
	}

	// Todas as linhas devem ter exatamente a mesma largura de exibição.
	larguras := largurasDasLinhas(tabela)
	for i, largura := range larguras {
		if largura != larguras[0] {
			t.Errorf("linha %d tem largura %d, esperava %d (todas as linhas devem ter a mesma largura):\n%s",
				i, largura, larguras[0], tabela)
		}
	}

	// O conteúdo não pode ter sido quebrado em duas linhas.
	if !strings.Contains(linhaSemANSI(tabela), "Alice") {
		t.Errorf("esperava encontrar o conteúdo \"Alice\" intacto na tabela:\n%s", tabela)
	}
	if strings.Count(linhaSemANSI(tabela), "Alice") != 1 {
		t.Errorf("esperava exatamente uma ocorrência de \"Alice\":\n%s", tabela)
	}
}

// TestRenderTableLarguraDeExibicaoCJK cobre os achados 10 e 14: a largura da
// coluna deve ser medida em células de terminal, nunca em bytes ou runes.
func TestRenderTableLarguraDeExibicaoCJK(t *testing.T) {
	tabela, err := renderTable(
		[]string{"idioma", "valor"},
		[][]string{{"日本語", "42"}, {"🎉", "ok"}},
		options{style: "rounded", color: "63"},
	)
	if err != nil {
		t.Fatalf("renderTable retornou erro inesperado: %v", err)
	}

	larguras := largurasDasLinhas(tabela)
	for i, largura := range larguras {
		if largura != larguras[0] {
			t.Errorf("linha %d tem largura %d, esperava %d:\n%s", i, largura, larguras[0], tabela)
		}
	}

	// "日本語" ocupa 6 células; a coluna precisa ter pelo menos 6 + 2 (padding).
	if larguras[0] < 6+2+1+2+1 {
		t.Errorf("largura inesperada para conteúdo CJK: %d\n%s", larguras[0], tabela)
	}

	// O texto não pode ter sido cortado no meio de um glifo.
	if !strings.Contains(linhaSemANSI(tabela), "日本語") {
		t.Errorf("esperava encontrar \"日本語\" intacto na tabela:\n%s", tabela)
	}
	if !strings.Contains(linhaSemANSI(tabela), "🎉") {
		t.Errorf("esperava encontrar o emoji intacto na tabela:\n%s", tabela)
	}
}

// TestParseJSONArrayOrdemDeterministica cobre o achado 2: a ordem das colunas
// não pode depender da ordem aleatória de iteração de um map.
func TestParseJSONArrayOrdemDeterminitica(t *testing.T) {
	dados := `[{"zebra":1,"alfa":2,"meio":3},{"alfa":4,"novo":5}]`

	var anterior []string
	for i := 0; i < 50; i++ {
		headers, rows, err := parseJSONArray(dados, true)
		if err != nil {
			t.Fatalf("parseJSONArray retornou erro inesperado: %v", err)
		}

		if len(headers) != 4 {
			t.Fatalf("esperava 4 colunas (união das chaves de todos os registros), obtizou %d: %v",
				len(headers), headers)
		}

		// Ordem de aparição: as chaves do primeiro registro (em ordem
		// alfabética) e, depois, as chaves que só existem nos demais.
		esperado := []string{"alfa", "meio", "zebra", "novo"}
		if strings.Join(headers, ",") != strings.Join(esperado, ",") {
			t.Fatalf("ordem das colunas inesperada: %v (esperava %v)", headers, esperado)
		}

		if anterior != nil && strings.Join(headers, ",") != strings.Join(anterior, ",") {
			t.Fatalf("a ordem das colunas variou entre execuções: %v e %v", anterior, headers)
		}
		anterior = headers

		// A coluna que só existe no segundo registro não pode ser descartada.
		if rows[1][3] != "5" {
			t.Errorf("esperava o valor \"5\" na coluna \"novo\", obtive %q", rows[1][3])
		}
	}
}

// TestParseJSONLUneTodasAsChaves cobre o achado 3 para JSONL.
func TestParseJSONLUneTodasAsChaves(t *testing.T) {
	dados := "{\"a\":1}\n{\"b\":2}\n{\"a\":3,\"c\":4}"

	headers, rows, err := parseJSONL(dados, true)
	if err != nil {
		t.Fatalf("parseJSONL retornou erro inesperado: %v", err)
	}

	if strings.Join(headers, ",") != "a,b,c" {
		t.Fatalf("esperava as colunas a,b,c (união de todos os registros), obtive %v", headers)
	}
	if len(rows) != 3 {
		t.Fatalf("esperava 3 linhas, obtive %d", len(rows))
	}
	if rows[1][1] != "2" {
		t.Errorf("esperava o valor \"2\" na coluna b, obtive %q", rows[1][1])
	}
}

// TestParseJSONLOrdemDeterministica cobre o achado 2 para JSONL.
func TestParseJSONLOrdemDeterministica(t *testing.T) {
	dados := "{\"zebra\":1,\"alfa\":2}\n{\"alfa\":3}"

	var anterior string
	for i := 0; i < 50; i++ {
		headers, _, err := parseJSONL(dados, true)
		if err != nil {
			t.Fatalf("parseJSONL retornou erro inesperado: %v", err)
		}
		atual := strings.Join(headers, ",")
		if atual != "alfa,zebra" {
			t.Fatalf("ordem das colunas inesperada: %v", headers)
		}
		if anterior != "" && atual != anterior {
			t.Fatalf("a ordem das colunas variou entre execuções: %s e %s", anterior, atual)
		}
		anterior = atual
	}
}

// TestParseJSONLVerificaScannerErr cobre o achado 8: linhas maiores que o
// limite padrão do bufio.Scanner (64 KiB) devem ser lidas, e o erro do
// scanner não pode ser engolido.
func TestParseJSONLVerificaScannerErr(t *testing.T) {
	valor := strings.Repeat("x", 200*1024)
	dados := `{"grande":"` + valor + `","id":1}`

	headers, rows, err := parseJSONL(dados, true)
	if err != nil {
		t.Fatalf("parseJSONL retornou erro inesperado para linha grande: %v", err)
	}
	if len(headers) != 2 {
		t.Fatalf("esperava 2 colunas, obtive %v", headers)
	}
	if len(rows) != 1 {
		t.Fatalf("esperava 1 linha, obtive %d", len(rows))
	}
	if rows[0][0] != valor {
		t.Errorf("o valor grande não foi preservado")
	}
}

// TestParseJSONLErroDeLinhaInvalida verifica que o erro informa a linha.
func TestParseJSONLErroDeLinhaInvalida(t *testing.T) {
	_, _, err := parseJSONL("{\"a\":1}\nnao e json", true)
	if err == nil {
		t.Fatal("esperava um erro para a linha inválida")
	}
	if !strings.Contains(err.Error(), "linha 2") {
		t.Errorf("esperava que o erro informasse a linha 2, obtive: %v", err)
	}
}

// TestParseDelimitedCabecalhoDeUmaLinha cobre o achado 5: um CSV com apenas
// uma linha deve preservar o cabeçalho real.
func TestParseDelimitedCabecalhoDeUmaLinha(t *testing.T) {
	headers, rows, err := parseDelimited("nome,idade,cidade", ',', true)
	if err != nil {
		t.Fatalf("parseDelimited retornou erro inesperado: %v", err)
	}

	if strings.Join(headers, ",") != "nome,idade,cidade" {
		t.Errorf("esperava o cabeçalho real, obtive %v", headers)
	}
	if len(rows) != 0 {
		t.Errorf("esperava nenhuma linha de dados, obtive %d", len(rows))
	}
}

// TestParseDelimitedColunasIrregulares cobre o achado 6: registros com
// quantidades diferentes de campos não podem abortar o programa.
func TestParseDelimitedColunasIrregulares(t *testing.T) {
	headers, rows, err := parseDelimited("a,b,c\n1,2\n3,4,5,6", ',', true)
	if err != nil {
		t.Fatalf("parseDelimited retornou erro inesperado: %v", err)
	}

	if len(headers) != 4 {
		t.Fatalf("esperava 4 colunas (a mais larga), obtive %v", headers)
	}
	for i, row := range rows {
		if len(row) != len(headers) {
			t.Errorf("linha %d tem %d campos, esperava %d (linhas devem ser normalizadas)", i, len(row), len(headers))
		}
	}
	if rows[0][2] != "" || rows[0][3] != "" {
		t.Errorf("esperava células vazias nas colunas ausentes da primeira linha, obtive %v", rows[0])
	}
	if rows[1][3] != "6" {
		t.Errorf("esperava o valor \"6\" na última coluna, obtive %q", rows[1][3])
	}
}

// TestParseDelimitedSemHeader gera nomes genéricos quando --header=false.
func TestParseDelimitedSemHeader(t *testing.T) {
	headers, rows, err := parseDelimited("a,b\n1,2", ',', false)
	if err != nil {
		t.Fatalf("parseDelimited retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "Col1,Col2" {
		t.Errorf("esperava Col1,Col2, obtive %v", headers)
	}
	if len(rows) != 2 {
		t.Errorf("esperava 2 linhas de dados, obtive %d", len(rows))
	}
}

// TestParseJSONArraySemHeader cobre o achado 9: --header=false precisa valer
// também para JSON e JSONL.
func TestParseJSONArraySemHeader(t *testing.T) {
	headers, rows, err := parseJSONArray(`[{"nome":"Alice","idade":30}]`, false)
	if err != nil {
		t.Fatalf("parseJSONArray retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "Col1,Col2" {
		t.Errorf("esperava Col1,Col2, obtive %v", headers)
	}
	if len(rows) != 1 {
		t.Fatalf("esperava 1 linha de dados, obtive %d", len(rows))
	}
}

func TestParseJSONLSemHeader(t *testing.T) {
	headers, rows, err := parseJSONL("{\"nome\":\"Alice\"}\n{\"nome\":\"Bob\"}", false)
	if err != nil {
		t.Fatalf("parseJSONL retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "Col1" {
		t.Errorf("esperava Col1, obtive %v", headers)
	}
	if len(rows) != 2 {
		t.Errorf("esperava 2 linhas de dados, obtive %d", len(rows))
	}
}

// TestReadInputStdinFechadoNaoPanica cobre o achado 4: o erro de os.Stdin.Stat
// precisa ser verificado, senão um stdin fechado causa nil dereference.
func TestReadInputStdinFechadoNaoPanica(t *testing.T) {
	arquivo, err := os.CreateTemp(t.TempDir(), "stdin-*.txt")
	if err != nil {
		t.Fatalf("não foi possível criar o arquivo temporário: %v", err)
	}
	if _, err := arquivo.WriteString("a,b\n1,2\n"); err != nil {
		t.Fatalf("não foi possível escrever no arquivo temporário: %v", err)
	}
	if err := arquivo.Close(); err != nil {
		t.Fatalf("não foi possível fechar o arquivo temporário: %v", err)
	}

	// O arquivo já está fechado: Stat() falha e o erro precisa ser tratado.
	_, err = readInput("", arquivo)
	if err == nil {
		t.Fatal("esperava um erro ao inspecionar um stdin fechado")
	}
	if !strings.Contains(err.Error(), "stdin") {
		t.Errorf("esperava um erro com contexto sobre o stdin, obtive: %v", err)
	}
}

// TestReadInputArquivoInexistente verifica o erro com contexto do arquivo.
func TestReadInputArquivoInexistente(t *testing.T) {
	_, err := readInput(filepath.Join(t.TempDir(), "nao-existe.csv"), nil)
	if err == nil {
		t.Fatal("esperava um erro para arquivo inexistente")
	}
	if !strings.Contains(err.Error(), "nao-existe.csv") {
		t.Errorf("esperava que o erro citasse o arquivo, obtive: %v", err)
	}
}

// TestStyleAsciiUsaBordaASCII cobre o achado 7: o estilo ascii precisa usar a
// borda ASCII de verdade (+---+), e não a borda Unicode.
func TestStyleAsciiUsaBordaASCII(t *testing.T) {
	tabela, err := renderTable(
		[]string{"a", "b"},
		[][]string{{"1", "2"}},
		options{style: "ascii", color: "63"},
	)
	if err != nil {
		t.Fatalf("renderTable retornou erro inesperado: %v", err)
	}

	limpo := linhaSemANSI(tabela)
	if !strings.Contains(limpo, "+---+") {
		t.Errorf("esperava a borda ASCII (+---+), obtive:\n%s", limpo)
	}
	for _, caractere := range []string{"─", "│", "╭", "╮", "╰", "╯", "├", "┤", "┬", "┴", "┼"} {
		if strings.Contains(limpo, caractere) {
			t.Errorf("a tabela em estilo ascii não deve conter o caractere Unicode %q:\n%s", caractere, limpo)
		}
	}
}

// TestRenderTableEstiloDesconhecido verifica o erro com contexto.
func TestRenderTableEstiloDesconhecido(t *testing.T) {
	_, err := renderTable([]string{"a"}, nil, options{style: "inexistente", color: "63"})
	if err == nil {
		t.Fatal("esperava um erro para estilo desconhecido")
	}
	if !strings.Contains(err.Error(), "inexistente") {
		t.Errorf("esperava que o erro citasse o estilo inválido, obtive: %v", err)
	}
}

// TestMaxWidthRespeitaLimite cobre o achado 11: a tabela truncada não pode
// ultrapassar a largura máxima pedida.
func TestMaxWidthRespeitaLimite(t *testing.T) {
	headers := []string{"nome", "descricao", "cidade"}
	rows := [][]string{{"Alice", "Uma descrição bem longa mesmo", "São Paulo"}}

	for _, limite := range []int{20, 30, 40, 60} {
		tabela, err := renderTable(headers, rows, options{style: "rounded", color: "63", maxWidth: limite})
		if err != nil {
			t.Fatalf("renderTable retornou erro inesperado para max-width %d: %v", limite, err)
		}

		for i, largura := range largurasDasLinhas(tabela) {
			if largura > limite {
				t.Errorf("com max-width %d a linha %d tem largura %d, ou seja, ultrapassou o limite:\n%s",
					limite, i, largura, tabela)
			}
		}
	}
}

// TestMaxWidthImpossivelRetornaErro verifica que um limite inviável gera um
// erro claro em vez de uma tabela quebrada.
func TestMaxWidthImpossivelRetornaErro(t *testing.T) {
	_, err := renderTable(
		[]string{"a", "b", "c"},
		[][]string{{"1", "2", "3"}},
		options{style: "rounded", color: "63", maxWidth: 8},
	)
	if err == nil {
		t.Fatal("esperava um erro para max-width insuficiente")
	}
	if !strings.Contains(err.Error(), "8") {
		t.Errorf("esperava que o erro citasse o limite, obtive: %v", err)
	}
}

// TestTruncateSeguroComCJKEmoji cobre o achado 14: o corte não pode quebrar
// glifos multibyte.
func TestTruncateSeguroComCJKEmoji(t *testing.T) {
	casos := []struct {
		texto   string
		largura int
	}{
		{"日本語テキスト", 5},
		{"🎉🎉🎉🎉", 6},
		{"café", 3},
		{"acentuação", 8},
	}

	for _, caso := range casos {
		resultado := truncate(caso.texto, caso.largura)
		if largura := ansi.StringWidth(resultado); largura > caso.largura {
			t.Errorf("truncate(%q, %d) devolveu largura %d", caso.texto, caso.largura, largura)
		}
		if !utf8Valido(resultado) {
			t.Errorf("truncate(%q, %d) devolveu texto inválido: %q", caso.texto, caso.largura, resultado)
		}
	}
}

// utf8Valido informa se a string é um UTF-8 válido.
func utf8Valido(s string) bool {
	return strings.ToValidUTF8(s, "�") == s
}

// TestDetectaTSVComTabEmCampoCitado cobre o achado 15: um CSV com tabulação
// dentro de um campo citado não pode ser interpretado como TSV.
func TestDetectaTSVComTabEmCampoCitado(t *testing.T) {
	// Tabulação dentro de campo citado na primeira linha.
	dados := "\"nome\tcom aspas\",idade\nAlice,30"

	if delim := detectDelimiter(dados); delim != ',' {
		t.Errorf("esperava detecção como CSV, obtive %q", delim)
	}

	headers, rows, err := detectAndParse(dados, true)
	if err != nil {
		t.Fatalf("detectAndParse retornou erro inesperado: %v", err)
	}
	if len(headers) != 2 {
		t.Errorf("esperava 2 colunas, obtive %v", headers)
	}
	if headers[0] != "nome\tcom aspas" {
		t.Errorf("esperava preservar a tabulação dentro do campo citado, obtive %q", headers[0])
	}
	if rows[0][0] != "Alice" {
		t.Errorf("esperava o valor \"Alice\", obtive %q", rows[0][0])
	}
}

// TestDetectaCSVCOMTabEmCampoCitadoNaSegundaLinha garante que uma tabulação
// em campo citado de qualquer linha não vira TSV.
func TestDetectaCSVCOMTabEmCampoCitadoNaSegundaLinha(t *testing.T) {
	dados := "nome,nota\nAlice,\"a\tb\""

	if delim := detectDelimiter(dados); delim != ',' {
		t.Errorf("esperava detecção como CSV, obtive %q", delim)
	}

	headers, rows, err := detectAndParse(dados, true)
	if err != nil {
		t.Fatalf("detectAndParse retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "nome,nota" {
		t.Errorf("esperava as colunas nome,nota, obtive %v", headers)
	}
	if rows[0][1] != "a\tb" {
		t.Errorf("esperava preservar a tabulação dentro do campo citado, obtive %q", rows[0][1])
	}
}

// TestDetectaTSV verifica a detecção de tabulação.
func TestDetectaTSV(t *testing.T) {
	dados := "nome\tidade\nAlice\t30"

	if delim := detectDelimiter(dados); delim != '\t' {
		t.Errorf("esperava detecção como TSV, obtive %q", delim)
	}

	headers, _, err := detectAndParse(dados, true)
	if err != nil {
		t.Fatalf("detectAndParse retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "nome,idade" {
		t.Errorf("esperava as colunas nome,idade, obtive %v", headers)
	}
}

// TestTrimLeadingSpacePreservado cobre o achado 16: espaços iniciais legítimos
// não podem ser removidos.
func TestTrimLeadingSpacePreservado(t *testing.T) {
	_, rows, err := parseDelimited("nome, valor ,outro\nAlice, Bob ,Carol", ',', true)
	if err != nil {
		t.Fatalf("parseDelimited retornou erro inesperado: %v", err)
	}
	if rows[0][1] != " Bob " {
		t.Errorf("esperava preservar os espaços do valor, obtive %q", rows[0][1])
	}
}

// TestRunComArquivo exercita o fluxo completo da CLI.
func TestRunComArquivo(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "dados.csv")
	conteudo := "nome,idade\nAlice,30\nBob,25\n"
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("não foi possível escrever o arquivo: %v", err)
	}

	var saida strings.Builder
	opts := options{style: "rounded", header: true, color: "63", file: caminho}
	if err := run(opts, nil, &saida); err != nil {
		t.Fatalf("run retornou erro inesperado: %v", err)
	}

	resultado := linhaSemANSI(saida.String())
	if !strings.Contains(resultado, "Alice") || !strings.Contains(resultado, "Bob") {
		t.Errorf("a saída não contém os dados:\n%s", resultado)
	}
}

// TestRunEntradaVazia verifica o erro com contexto para entrada vazia.
func TestRunEntradaVazia(t *testing.T) {
	var saida strings.Builder
	opts := options{style: "rounded", header: true, color: "63", file: ""}

	// Um stdin fechado produz erro em vez de panic.
	arquivo, err := os.CreateTemp(t.TempDir(), "stdin-*.txt")
	if err != nil {
		t.Fatalf("não foi possível criar o arquivo temporário: %v", err)
	}
	if err := arquivo.Close(); err != nil {
		t.Fatalf("não foi possível fechar o arquivo temporário: %v", err)
	}

	if err := run(opts, arquivo, &saida); err == nil {
		t.Fatal("esperava um erro com stdin fechado")
	}
}

// TestErroFormatoDesconhecidoJSON verifica que o erro diz qual formato falhou.
func TestErroFormatoDesconhecidoJSON(t *testing.T) {
	_, _, err := detectAndParse("[1,2,3]", true)
	if err == nil {
		t.Fatal("esperava um erro para JSON inválido")
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("esperava que o erro mencionasse o formato JSON, obtive: %v", err)
	}
}

// TestColunasUnidasEmTodosOsRegistros cobre o achado 3 de ponta a ponta.
func TestColunasUnidasEmTodosOsRegistros(t *testing.T) {
	dados := `[{"a":1,"b":2},{"a":3,"c":9}]`

	headers, rows, err := detectAndParse(dados, true)
	if err != nil {
		t.Fatalf("detectAndParse retornou erro inesperado: %v", err)
	}
	if strings.Join(headers, ",") != "a,b,c" {
		t.Fatalf("esperava as colunas a,b,c, obtive %v", headers)
	}
	if rows[0][2] != "" {
		t.Errorf("esperava célula vazia para a coluna ausente no primeiro registro, obtive %q", rows[0][2])
	}
	if rows[1][2] != "9" {
		t.Errorf("esperava o valor \"9\" na coluna c, obtive %q", rows[1][2])
	}
}

// posicoesDe devolve as colunas (em células de terminal) onde o caractere
// alvo aparece na linha.
func posicoesDe(linha string, alvo rune) []int {
	var posicoes []int
	coluna := 0

	for _, r := range linhaSemANSI(linha) {
		if r == alvo {
			posicoes = append(posicoes, coluna)
		}
		coluna += ansi.StringWidth(string(r))
	}

	return posicoes
}

// TestJuncoesConectamComBordasHorizontais cobre o achado 13: as junções das
// bordas horizontais (┬, ┼, ┴) precisam ficar exatamente na mesma coluna dos
// separadores verticais das células (│). O desalinhamento relatado na
// auditoria era consequência do achado 1 (largura errada no lipgloss v2).
func TestJuncoesConectamComBordasHorizontais(t *testing.T) {
	tabela, err := renderTable(
		[]string{"name", "age", "city"},
		[][]string{{"Alice", "30", "NYC"}, {"Bob", "25", "LA"}},
		options{style: "rounded", color: "63"},
	)
	if err != nil {
		t.Fatalf("renderTable retornou erro inesperado: %v", err)
	}

	linhas := strings.Split(strings.TrimRight(tabela, "\n"), "\n")
	if len(linhas) != 6 {
		t.Fatalf("esperava 6 linhas, obtive %d:\n%s", len(linhas), tabela)
	}

	verticais := posicoesDe(linhas[1], '│')
	if len(verticais) != 4 {
		t.Fatalf("esperava 4 separadores verticais, obtive %v", verticais)
	}

	// As junções internas (┬, ┼, ┴) devem ficar exatamente na mesma coluna
	// dos separadores verticais internos das células.
	internos := verticais[1 : len(verticais)-1]

	// Borda superior (┬), separador do cabeçalho (┼) e borda inferior (┴).
	casos := []struct {
		linha int
		alvo  rune
	}{
		{0, '┬'},
		{2, '┼'},
		{5, '┴'},
	}

	for _, caso := range casos {
		posicoes := posicoesDe(linhas[caso.linha], caso.alvo)
		if len(posicoes) != len(internos) {
			t.Errorf("linha %d: esperava %d junções, obtive %v", caso.linha, len(internos), posicoes)
			continue
		}
		for i := range posicoes {
			if posicoes[i] != internos[i] {
				t.Errorf("linha %d: junção %d na coluna %d, esperava %d (as junções devem conectar com as bordas horizontais):\n%s",
					caso.linha, i, posicoes[i], internos[i], tabela)
			}
		}
	}
}
