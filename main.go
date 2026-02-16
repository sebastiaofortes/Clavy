package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sebastiaofortes/pdf-to-html/converter"
)

const samplesDir = "samples"

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

	http.HandleFunc("/", handleList)
	http.HandleFunc("/convert", handleConvert)
	http.HandleFunc("/health", handleHealth)

	addr := ":" + port
	fmt.Printf("\nServidor iniciado em http://localhost%s\n", addr)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /                                                         → lista PDFs disponíveis")
	fmt.Println("  GET /convert?pdf=<caminho>&page=<N>&zoom=<N>&fmt=<png|jpg>    → converte PDF para HTML")
	fmt.Println("  GET /health                                                   → status do servidor")
	fmt.Println()
	fmt.Printf("Abra no navegador: http://localhost%s\n", addr)
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, nil))
}

// listTemplate é o template HTML para a página de listagem de PDFs.
var listTemplate = template.Must(template.New("list").Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PDF to HTML — Arquivos disponíveis</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
            padding: 2rem;
        }
        h1 {
            font-size: 1.5rem;
            margin-bottom: 0.5rem;
        }
        .subtitle {
            color: #666;
            margin-bottom: 2rem;
            font-size: 0.9rem;
        }
        .pdf-list {
            list-style: none;
            max-width: 600px;
        }
        .pdf-item {
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 8px;
            margin-bottom: 0.75rem;
            transition: box-shadow 0.2s;
        }
        .pdf-item:hover {
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .pdf-item a {
            display: flex;
            align-items: center;
            padding: 1rem 1.25rem;
            text-decoration: none;
            color: #333;
        }
        .pdf-icon {
            font-size: 1.5rem;
            margin-right: 1rem;
            flex-shrink: 0;
        }
        .pdf-name {
            font-weight: 500;
        }
        .pdf-arrow {
            margin-left: auto;
            color: #999;
            font-size: 1.2rem;
        }
        .empty {
            color: #999;
            font-style: italic;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <h1>PDF to HTML</h1>
    <p class="subtitle">Clique em um PDF para visualizar a primeira página convertida em HTML.</p>
    {{if .Files}}
    <ul class="pdf-list">
        {{range .Files}}
        <li class="pdf-item">
            <a href="/convert?pdf={{.Path}}&page=1">
                <span class="pdf-icon">&#128196;</span>
                <span class="pdf-name">{{.Name}}</span>
                <span class="pdf-arrow">&#8594;</span>
            </a>
        </li>
        {{end}}
    </ul>
    {{else}}
    <p class="empty">Nenhum arquivo PDF encontrado na pasta "{{.Dir}}".</p>
    <p class="empty">Coloque seus PDFs na pasta e recarregue a página.</p>
    {{end}}
</body>
</html>`))

// pdfFile representa um arquivo PDF para o template.
type pdfFile struct {
	Name string
	Path string
}

// handleList lista os PDFs disponíveis na pasta samples.
func handleList(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var files []pdfFile

	entries, err := os.ReadDir(samplesDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
				files = append(files, pdfFile{
					Name: entry.Name(),
					Path: filepath.Join(samplesDir, entry.Name()),
				})
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	listTemplate.Execute(w, struct {
		Files []pdfFile
		Dir   string
	}{
		Files: files,
		Dir:   samplesDir,
	})
}

// viewerTemplate é o template HTML que envolve o conteúdo convertido
// com uma barra de navegação para navegar entre páginas.
var viewerTemplate = template.Must(template.New("viewer").Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.FileName}} — Página {{.Page}} de {{.TotalPages}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
        }
        .navbar {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            z-index: 1000;
            display: flex;
            align-items: center;
            justify-content: space-between;
            background: #1a1a2e;
            color: #fff;
            padding: 0.6rem 1.5rem;
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .navbar a {
            color: #8ecae6;
            text-decoration: none;
            font-size: 0.85rem;
        }
        .navbar a:hover { text-decoration: underline; }
        .nav-center {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }
        .nav-btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            background: #16213e;
            color: #fff;
            border: 1px solid #0f3460;
            border-radius: 6px;
            padding: 0.4rem 1rem;
            font-size: 0.85rem;
            cursor: pointer;
            text-decoration: none;
            transition: background 0.2s;
        }
        .nav-btn:hover { background: #0f3460; color: #fff; text-decoration: none; }
        .nav-btn.disabled {
            opacity: 0.35;
            pointer-events: none;
            cursor: default;
        }
        .page-info {
            font-size: 0.85rem;
            color: #ccc;
            min-width: 120px;
            text-align: center;
        }
        .file-name {
            font-size: 0.85rem;
            color: #aaa;
            max-width: 200px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .content {
            margin-top: 52px;
        }
    </style>
</head>
<body>
    <nav class="navbar">
        <a href="/">&#8592; Voltar à lista</a>
        <div class="nav-center">
            <a class="nav-btn {{if le .Page 1}}disabled{{end}}"
               href="/convert?pdf={{.PdfPath}}&page={{.PrevPage}}&zoom={{.Zoom}}&fmt={{.Fmt}}">
                &#9664; Anterior
            </a>
            <span class="page-info">Página {{.Page}} de {{.TotalPages}}</span>
            <a class="nav-btn {{if ge .Page .TotalPages}}disabled{{end}}"
               href="/convert?pdf={{.PdfPath}}&page={{.NextPage}}&zoom={{.Zoom}}&fmt={{.Fmt}}">
                Próxima &#9654;
            </a>
        </div>
        <span class="file-name">{{.FileName}}</span>
    </nav>
    <div class="content">
        {{.HTMLContent}}
    </div>
</body>
</html>`))

// viewerData contém os dados para o template de visualização.
type viewerData struct {
	HTMLContent template.HTML
	PdfPath     string
	FileName    string
	Page        int
	TotalPages  int
	PrevPage    int
	NextPage    int
	Zoom        string
	Fmt         string
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

	zoomStr := query.Get("zoom")
	if zoomStr != "" {
		z, err := strconv.ParseFloat(zoomStr, 64)
		if err != nil || z <= 0 {
			http.Error(w, "Parâmetro 'zoom' deve ser um número positivo", http.StatusBadRequest)
			return
		}
		opts.Zoom = z
	}

	fmtStr := query.Get("fmt")
	if fmtStr != "" {
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

	// Obter total de páginas para a navegação
	totalPages, err := converter.PageCount(pdfPath)
	if err != nil {
		log.Printf("Aviso: não foi possível contar páginas: %v", err)
		totalPages = 0
	}

	// Se page=0 (todas) ou não há info de páginas, retornar HTML puro
	if opts.Page == 0 || totalPages == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(result.HTML))
		log.Printf("Conversão concluída: %d bytes", len(result.HTML))
		return
	}

	// Renderizar com barra de navegação
	prevPage := opts.Page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := opts.Page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	// Preservar zoom e fmt nos links
	displayZoom := strconv.FormatFloat(opts.Zoom, 'f', 2, 64)
	displayFmt := opts.ImageFmt

	data := viewerData{
		HTMLContent: template.HTML(result.HTML),
		PdfPath:     pdfPath,
		FileName:    filepath.Base(pdfPath),
		Page:        opts.Page,
		TotalPages:  totalPages,
		PrevPage:    prevPage,
		NextPage:    nextPage,
		Zoom:        displayZoom,
		Fmt:         displayFmt,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	viewerTemplate.Execute(w, data)

	log.Printf("Conversão concluída: página %d/%d", opts.Page, totalPages)
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
