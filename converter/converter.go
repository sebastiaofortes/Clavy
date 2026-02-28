// Package converter fornece funcionalidades para converter páginas de
// arquivos PDF para HTML5 utilizando o pdftohtml do Poppler.
package converter

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

	// 3. Criar diretório temporário para a saída
	tmpDir, err := os.MkdirTemp("", "clavy-*")
	if err != nil {
		return nil, fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outFile := filepath.Join(tmpDir, "output.html")

	// 4. Montar argumentos
	args := buildArgs(pdfPath, outFile, opts)

	// 5. Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// 6. Executar o comando
	cmd := exec.CommandContext(ctx, binPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("conversão excedeu o timeout de %s", opts.Timeout)
		}
		return nil, fmt.Errorf("erro ao executar pdftohtml: %w\nstderr: %s", err, stderr.String())
	}

	// 7. Ler o HTML gerado
	htmlBytes, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler HTML gerado: %w", err)
	}

	html := string(htmlBytes)

	// 8. Embutir imagens como data URIs se solicitado
	if opts.EmbedImages {
		html = embedImages(html, tmpDir, opts.ImageFmt)
	}

	return &Result{
		HTML:    html,
		PageNum: opts.Page,
	}, nil
}

// buildArgs monta os argumentos do comando pdftohtml.
func buildArgs(pdfPath, outFile string, opts Options) []string {
	args := []string{
		"-noframes", // HTML único, sem frameset
		"-nodrm",    // ignorar restrições de DRM/cópia
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

	// Arquivo de entrada e saída
	args = append(args, pdfPath, outFile)

	return args
}

// imgSrcRegex encontra referências a imagens no HTML (src="arquivo.png" ou src="arquivo.jpg").
var imgSrcRegex = regexp.MustCompile(`src="([^"]+\.(png|jpg))"`)

// embedImages substitui referências a imagens no HTML por data URIs base64.
func embedImages(html, imgDir, imgFmt string) string {
	mimeType := "image/png"
	if imgFmt == "jpg" {
		mimeType = "image/jpeg"
	}

	return imgSrcRegex.ReplaceAllStringFunc(html, func(match string) string {
		// Extrair o nome do arquivo da referência
		parts := imgSrcRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		imgName := parts[1]
		imgPath := filepath.Join(imgDir, imgName)

		// Se o arquivo não existir no tmpDir, tentar só o basename
		if _, err := os.Stat(imgPath); os.IsNotExist(err) {
			imgPath = filepath.Join(imgDir, filepath.Base(imgName))
		}

		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			return match // manter referência original se falhar
		}

		b64 := base64.StdEncoding.EncodeToString(imgData)
		return fmt.Sprintf(`src="data:%s;base64,%s"`, mimeType, b64)
	})
}

// mimeForExt retorna o MIME type a partir da extensão do ficheiro.
func mimeForExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

// PageCount retorna o número total de páginas de um arquivo PDF
// usando o pdfinfo do Poppler.
func PageCount(pdfPath string) (int, error) {
	binPath, err := exec.LookPath("pdfinfo")
	if err != nil {
		return 0, fmt.Errorf("pdfinfo não encontrado no PATH: %w", err)
	}

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("arquivo PDF não encontrado: %s", pdfPath)
	}

	cmd := exec.Command(binPath, pdfPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("erro ao executar pdfinfo: %w", err)
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.TrimSpace(strings.TrimPrefix(line, "Pages:"))
			count, err := strconv.Atoi(parts)
			if err != nil {
				return 0, fmt.Errorf("erro ao parsear número de páginas: %w", err)
			}
			return count, nil
		}
	}

	return 0, fmt.Errorf("não foi possível determinar o número de páginas")
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
