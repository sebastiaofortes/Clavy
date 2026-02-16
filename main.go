package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sebastiaofortes/pdf-to-html/converter"
)

func main() {
	// Definir flags da CLI
	pdfPath := flag.String("pdf", "", "Caminho para o arquivo PDF (obrigatório)")
	page := flag.Int("page", 0, "Página a converter (0 = todas)")
	zoom := flag.Float64("zoom", 1.5, "Fator de zoom")
	output := flag.String("output", "", "Arquivo de saída HTML (vazio = stdout)")
	imgFmt := flag.String("fmt", "png", "Formato de imagem: png ou jpg")
	checkDeps := flag.Bool("check", false, "Verificar dependências e sair")

	flag.Parse()

	// Verificar dependências
	if *checkDeps {
		if err := converter.CheckDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validar entrada
	if *pdfPath == "" {
		fmt.Fprintln(os.Stderr, "Erro: caminho do PDF é obrigatório")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Uso: pdf-to-html -pdf <arquivo.pdf> [-page N] [-zoom 1.5] [-output saida.html]")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Configurar opções
	opts := converter.DefaultOptions()
	opts.Page = *page
	opts.Zoom = *zoom
	opts.ImageFmt = *imgFmt

	// Executar conversão
	result, err := converter.Convert(*pdfPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	// Saída
	if *output != "" {
		// Criar diretório de saída se necessário
		dir := filepath.Dir(*output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar diretório: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*output, []byte(result.HTML), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao salvar arquivo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("HTML salvo em: %s (%d bytes)\n", *output, len(result.HTML))
	} else {
		fmt.Print(result.HTML)
	}
}
