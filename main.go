package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sebastiaofortes/pdf-to-html/converter"
)

func main() {
	// Definir flags da CLI
	pdfPath := flag.String("pdf", "", "Caminho para o arquivo PDF (obrigatório no modo CLI)")
	page := flag.Int("page", 0, "Página a converter (0 = todas)")
	zoom := flag.Float64("zoom", 1.5, "Fator de zoom")
	output := flag.String("output", "", "Arquivo de saída HTML (vazio = stdout)")
	imgFmt := flag.String("fmt", "png", "Formato de imagem: png ou jpg")
	checkDeps := flag.Bool("check", false, "Verificar dependências e sair")
	serve := flag.String("serve", "", "Iniciar servidor HTTP na porta indicada (ex: 8080)")

	flag.Parse()

	// Verificar dependências
	if *checkDeps {
		if err := converter.CheckDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Modo servidor HTTP
	if *serve != "" {
		startServer(*serve)
		return
	}

	// Modo CLI
	runCLI(*pdfPath, *page, *zoom, *imgFmt, *output)
}

// startServer inicia o servidor HTTP na porta especificada.
func startServer(port string) {
	if err := converter.CheckDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/convert", handleConvert)
	http.HandleFunc("/health", handleHealth)

	addr := ":" + port
	fmt.Printf("\nServidor iniciado em http://localhost%s\n", addr)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /convert?pdf=<caminho>&page=<N>&zoom=<N>&fmt=<png|jpg>")
	fmt.Println("  GET /health")
	fmt.Println()
	fmt.Println("Exemplo:")
	fmt.Printf("  curl \"http://localhost%s/convert?pdf=samples/teste.pdf&page=1\" -o resultado.html\n", addr)
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleConvert processa requisições de conversão PDF → HTML.
//
// Query params:
//   - pdf  (obrigatório) caminho para o arquivo PDF
//   - page (opcional)    página a converter (0 = todas, default: 0)
//   - zoom (opcional)    fator de zoom (default: 1.5)
//   - fmt  (opcional)    formato de imagem: png ou jpg (default: png)
func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido. Use GET.", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	// Parâmetro obrigatório: pdf
	pdfPath := query.Get("pdf")
	if pdfPath == "" {
		http.Error(w, "Parâmetro 'pdf' é obrigatório. Ex: /convert?pdf=samples/teste.pdf&page=1", http.StatusBadRequest)
		return
	}

	// Parâmetros opcionais
	opts := converter.DefaultOptions()

	if pageStr := query.Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 0 {
			http.Error(w, "Parâmetro 'page' deve ser um número inteiro >= 0", http.StatusBadRequest)
			return
		}
		opts.Page = p
	}

	if zoomStr := query.Get("zoom"); zoomStr != "" {
		z, err := strconv.ParseFloat(zoomStr, 64)
		if err != nil || z <= 0 {
			http.Error(w, "Parâmetro 'zoom' deve ser um número positivo", http.StatusBadRequest)
			return
		}
		opts.Zoom = z
	}

	if fmtStr := query.Get("fmt"); fmtStr != "" {
		if fmtStr != "png" && fmtStr != "jpg" {
			http.Error(w, "Parâmetro 'fmt' deve ser 'png' ou 'jpg'", http.StatusBadRequest)
			return
		}
		opts.ImageFmt = fmtStr
	}

	// Executar conversão
	log.Printf("Convertendo: pdf=%s page=%d zoom=%.2f fmt=%s", pdfPath, opts.Page, opts.Zoom, opts.ImageFmt)

	result, err := converter.Convert(pdfPath, opts)
	if err != nil {
		log.Printf("Erro na conversão: %v", err)
		http.Error(w, fmt.Sprintf("Erro na conversão: %v", err), http.StatusInternalServerError)
		return
	}

	// Retornar HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(result.HTML)))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result.HTML))

	log.Printf("Conversão concluída: %d bytes", len(result.HTML))
}

// handleHealth retorna o status do servidor.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// runCLI executa o modo linha de comando.
func runCLI(pdfPath string, page int, zoom float64, imgFmt, output string) {
	if pdfPath == "" {
		fmt.Fprintln(os.Stderr, "Erro: caminho do PDF é obrigatório")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Uso:")
		fmt.Fprintln(os.Stderr, "  CLI:      pdf-to-html -pdf <arquivo.pdf> [-page N] [-zoom 1.5] [-output saida.html]")
		fmt.Fprintln(os.Stderr, "  Servidor: pdf-to-html -serve 8080")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		os.Exit(1)
	}

	opts := converter.DefaultOptions()
	opts.Page = page
	opts.Zoom = zoom
	opts.ImageFmt = imgFmt

	result, err := converter.Convert(pdfPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		dir := filepath.Dir(output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar diretório: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(output, []byte(result.HTML), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao salvar arquivo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("HTML salvo em: %s (%d bytes)\n", output, len(result.HTML))
	} else {
		fmt.Print(result.HTML)
	}
}
