package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// borderStyles mapeia o nome do estilo para a borda correspondente.
// O estilo "ascii" usa a borda ASCII de verdade (+---+), e não a borda
// Unicode padrão do lipgloss.
var borderStyles = map[string]lipgloss.Border{
	"rounded": lipgloss.RoundedBorder(),
	"thick":   lipgloss.ThickBorder(),
	"double":  lipgloss.DoubleBorder(),
	"ascii":   lipgloss.ASCIIBorder(),
}

// borderStyleNames devolve os nomes de estilo disponíveis, em ordem alfabética.
func borderStyleNames() []string {
	names := make([]string, 0, len(borderStyles))
	for name := range borderStyles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// renderTable renderiza a tabela estilizada com lipgloss.
func renderTable(headers []string, rows [][]string, opts options) (string, error) {
	if len(headers) == 0 {
		return "", errors.New("nenhuma coluna para renderizar")
	}

	border, ok := borderStyles[opts.style]
	if !ok {
		return "", fmt.Errorf("estilo de borda desconhecido %q: use um de %s",
			opts.style, strings.Join(borderStyleNames(), ", "))
	}

	colWidths := computeColWidths(headers, rows)

	if opts.maxWidth > 0 {
		fitted, err := fitToMaxWidth(colWidths, opts.maxWidth)
		if err != nil {
			return "", err
		}
		colWidths = fitted
	}

	// No lipgloss v2, Width(n) é a largura total do bloco: o padding lateral
	// de 1 célula de cada lado já faz parte das n colunas. Por isso as células
	// usam largura + 2 (o padding), enquanto a borda também reserva w + 2.
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(opts.color)).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	var sb strings.Builder

	// Borda superior.
	writeBorderLine(&sb, border, colWidths, border.TopLeft, border.Top, border.MiddleTop, border.TopRight)

	// Linha de cabeçalho.
	sb.WriteString(border.Left)
	for i, header := range headers {
		content := truncate(sanitizeCell(header), colWidths[i])
		sb.WriteString(headerStyle.Width(colWidths[i] + 2).Render(content))
		if i < len(headers)-1 {
			sb.WriteString(border.Left)
		}
	}
	sb.WriteString(border.Right)
	sb.WriteString("\n")

	// Separador entre cabeçalho e corpo.
	writeBorderLine(&sb, border, colWidths, border.MiddleLeft, border.Top, border.Middle, border.MiddleRight)

	// Linhas de dados.
	for _, row := range rows {
		sb.WriteString(border.Left)
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			content := truncate(sanitizeCell(cell), colWidths[i])
			sb.WriteString(cellStyle.Width(colWidths[i] + 2).Render(content))
			if i < len(headers)-1 {
				sb.WriteString(border.Left)
			}
		}
		sb.WriteString(border.Right)
		sb.WriteString("\n")
	}

	// Borda inferior.
	writeBorderLine(&sb, border, colWidths, border.BottomLeft, border.Bottom, border.MiddleBottom, border.BottomRight)

	return sb.String(), nil
}

// writeBorderLine escreve uma linha horizontal da tabela (borda superior,
// separador do cabeçalho ou borda inferior).
func writeBorderLine(sb *strings.Builder, border lipgloss.Border, colWidths []int,
	left, horizontal, junction, right string) {
	sb.WriteString(left)
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat(horizontal, width+2))
		if i < len(colWidths)-1 {
			sb.WriteString(junction)
		}
	}
	sb.WriteString(right)
	sb.WriteString("\n")
}

// computeColWidths calcula a largura de exibição de cada coluna. A medição usa
// a largura de exibição (células do terminal) e nunca o número de bytes ou de
// runes: CJK e emoji ocupam duas células.
func computeColWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))

	for i, header := range headers {
		widths[i] = ansi.StringWidth(sanitizeCell(header))
	}

	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if width := ansi.StringWidth(sanitizeCell(cell)); width > widths[i] {
				widths[i] = width
			}
		}
	}

	return widths
}

// fitToMaxWidth redistribui as larguras das colunas para que a tabela inteira
// caiba em maxWidth, contando bordas e padding. As colunas são limitadas por
// um teto comum (as mais largas encolhem até o limite das mais estreitas) e o
// conteúdo é truncado com segurança.
func fitToMaxWidth(widths []int, maxWidth int) ([]int, error) {
	// Overhead: 2 células de padding por coluna + 1 caractere de borda por
	// coluna + 1 de cada lado.
	overhead := 3*len(widths) + 1
	budget := maxWidth - overhead

	if budget < len(widths) {
		return nil, fmt.Errorf(
			"largura máxima %d insuficiente para %d colunas: são necessárias ao menos %d colunas",
			maxWidth, len(widths), overhead+len(widths))
	}

	total := 0
	for _, width := range widths {
		total += width
	}
	if total <= budget {
		return widths, nil
	}

	// Procura o maior limite T tal que a soma das colunas limitadas por T
	// ainda cabe no orçamento.
	low, high := 1, largestWidth(widths)
	for low < high {
		mid := (low + high + 1) / 2
		if sumCapped(widths, mid) <= budget {
			low = mid
		} else {
			high = mid - 1
		}
	}

	fitted := make([]int, len(widths))
	used := 0
	for i, width := range widths {
		if width > low {
			fitted[i] = low
		} else {
			fitted[i] = width
		}
		used += fitted[i]
	}

	// Distribui a sobra entre as colunas que foram truncadas.
	for i := range fitted {
		if used >= budget {
			break
		}
		if widths[i] > low {
			fitted[i]++
			used++
		}
	}

	return fitted, nil
}

// sumCapped soma as larguras limitadas por um teto.
func sumCapped(widths []int, cap int) int {
	total := 0
	for _, width := range widths {
		if width > cap {
			total += cap
			continue
		}
		total += width
	}
	return total
}

// largestWidth devolve a maior largura da slice.
func largestWidth(widths []int) int {
	largest := 0
	for _, width := range widths {
		if width > largest {
			largest = width
		}
	}
	return largest
}

// truncate corta o texto para caber na largura máxima sem quebrar glifos de
// CJK, emoji ou caracteres acentuados.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= len(ellipsis) {
		// Sem espaço para o sufixo: corta o texto puro.
		return ansi.Truncate(s, maxWidth, "")
	}
	return ansi.Truncate(s, maxWidth, ellipsis)
}

// ellipsis é o sufixo usado quando o conteúdo é truncado.
const ellipsis = "..."

// sanitizeCell troca quebras de linha e tabulações por espaços: uma célula
// com várias linhas ou com um tab quebraria o alinhamento da tabela.
func sanitizeCell(s string) string {
	if !strings.ContainsAny(s, "\r\n\t") {
		return s
	}
	replacer := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ")
	return replacer.Replace(s)
}
