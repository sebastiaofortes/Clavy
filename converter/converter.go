// Package converter fornece funcionalidades para converter páginas de
// arquivos PDF para HTML5 utilizando o pdftohtml do Poppler.
package converter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// Options configura o comportamento da conversão.
type Options struct {
	Page        int           // Página a converter (1-indexed). 0 = todas.
	Zoom        float64       // Fator de zoom (default: 1.5).
	ImageFmt    string        // Formato de imagem: "png" ou "jpg" (default: "png").
	Timeout     time.Duration // Timeout da conversão (default: 30s).
	EmbedImages bool          // Embute imagens como data URIs no HTML.
}

// DefaultOptions retorna opções com valores padrão sensatos.
func DefaultOptions() Options {
	return Options{
		Page:        0,
		Zoom:        1.5,
		ImageFmt:    "png",
		Timeout:     30 * time.Second,
		EmbedImages: true,
	}
}

// Result contém o resultado da conversão.
type Result struct {
	HTML    string // Conteúdo HTML gerado.
	PageNum int    // Página que foi convertida (0 = todas).
}

// Convert converte um arquivo PDF (ou uma página específica) para HTML.
// Usa o pdftohtml do Poppler via os/exec.
func Convert(pdfPath string, opts Options) (*Result, error) {
	// 1. Verificar se o pdftohtml está instalado
	binPath, err := exec.LookPath("pdftohtml")
	if err != nil {
		return nil, fmt.Errorf(
			"pdftohtml não encontrado no PATH: %w\n"+
				"Instale com: brew install poppler (macOS) ou apt install poppler-utils (Linux)", err,
		)
	}

	// 2. Verificar se o arquivo PDF existe
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("arquivo PDF não encontrado: %s", pdfPath)
	}

	// 3. Montar argumentos
	args := buildArgs(pdfPath, opts)

	// 4. Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// 5. Executar o comando
	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("conversão excedeu o timeout de %s", opts.Timeout)
		}
		return nil, fmt.Errorf("erro ao executar pdftohtml: %w\nstderr: %s", err, stderr.String())
	}

	// 6. Retornar resultado
	return &Result{
		HTML:    stdout.String(),
		PageNum: opts.Page,
	}, nil
}

// buildArgs monta os argumentos do comando pdftohtml.
func buildArgs(pdfPath string, opts Options) []string {
	args := []string{
		"-noframes", // HTML único, sem frameset
		"-stdout",   // saída para stdout (capturamos no Go)
		"-q",        // modo silencioso
	}

	// Página específica
	if opts.Page > 0 {
		pageStr := strconv.Itoa(opts.Page)
		args = append(args, "-f", pageStr, "-l", pageStr)
	}

	// Zoom
	if opts.Zoom > 0 {
		args = append(args, "-zoom", strconv.FormatFloat(opts.Zoom, 'f', 2, 64))
	}

	// Formato de imagem
	if opts.ImageFmt == "jpg" || opts.ImageFmt == "png" {
		args = append(args, "-fmt", opts.ImageFmt)
	}

	// Embutir imagens como data URIs
	if opts.EmbedImages {
		args = append(args, "-dataurls")
	}

	// Arquivo de entrada (deve ser o último argumento)
	args = append(args, pdfPath)

	return args
}

// CheckDependencies verifica se o pdftohtml (Poppler) está instalado
// e imprime informações sobre a versão encontrada.
func CheckDependencies() error {
	path, err := exec.LookPath("pdftohtml")
	if err != nil {
		return fmt.Errorf(
			"pdftohtml não encontrado.\n"+
				"  macOS:  brew install poppler\n"+
				"  Ubuntu: sudo apt install poppler-utils\n"+
				"  Fedora: sudo dnf install poppler-utils",
		)
	}

	// pdftohtml -v imprime a versão no stderr e retorna exit code 1
	cmd := exec.Command(path, "-v")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Run() //nolint:errcheck // -v sempre retorna exit code != 0

	fmt.Printf("pdftohtml encontrado: %s\n", path)
	if v := stderr.String(); v != "" {
		fmt.Printf("Versão: %s", v)
	}

	return nil
}
