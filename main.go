package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sebastiaofortes/pdf-to-html/converter"
	"github.com/sebastiaofortes/pdf-to-html/store"
)

const (
	samplesDir = "samples"
	dbFile     = "data/books.json"
)

// db é o banco de dados de metadados dos PDFs.
var db *store.Store

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

	// Inicializar o banco de dados
	if err := os.MkdirAll(filepath.Dir(dbFile), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar diretório de dados: %v\n", err)
		os.Exit(1)
	}

	var err error
	db, err = store.New(dbFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inicializar banco de dados: %v\n", err)
		os.Exit(1)
	}

	// Garantir que a pasta de samples existe
	os.MkdirAll(samplesDir, 0755)

	http.HandleFunc("/", handleList)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/delete", handleDelete)
	http.HandleFunc("/convert", handleConvert)
	http.HandleFunc("/health", handleHealth)

	addr := ":" + port
	fmt.Printf("\nServidor iniciado em http://localhost%s\n", addr)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /                                                         → lista PDFs disponíveis")
	fmt.Println("  GET  /upload                                                   → página de upload")
	fmt.Println("  POST /upload                                                   → enviar PDF + idioma")
	fmt.Println("  GET  /convert?pdf=<caminho>&page=<N>&zoom=<N>&fmt=<png|jpg>    → converte PDF para HTML")
	fmt.Println("  GET  /health                                                   → status do servidor")
	fmt.Println()
	fmt.Printf("Abra no navegador: http://localhost%s\n", addr)
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, nil))
}

// uploadTemplate é o template HTML para a página de upload de PDFs.
var uploadTemplate = template.Must(template.New("upload").Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PDF to HTML — Upload</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
            padding: 2rem;
        }
        h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
        .subtitle { color: #666; margin-bottom: 2rem; font-size: 0.9rem; }
        .back-link { color: #1a73e8; text-decoration: none; font-size: 0.9rem; }
        .back-link:hover { text-decoration: underline; }
        .upload-form {
            max-width: 500px;
            background: #fff;
            border: 1px solid #ddd;
            border-radius: 12px;
            padding: 2rem;
            margin-top: 1.5rem;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        .form-group label {
            display: block;
            font-weight: 500;
            margin-bottom: 0.5rem;
            font-size: 0.9rem;
        }
        .form-group input[type="file"] {
            width: 100%;
            padding: 0.5rem;
            border: 2px dashed #ccc;
            border-radius: 8px;
            background: #fafafa;
            cursor: pointer;
        }
        .form-group input[type="file"]:hover {
            border-color: #1a73e8;
        }
        .lang-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 0.5rem;
        }
        .lang-option {
            display: flex;
            align-items: center;
            gap: 0.4rem;
            padding: 0.4rem 0.6rem;
            border: 1px solid #ddd;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.85rem;
            transition: all 0.15s;
        }
        .lang-option:hover { background: #f0f7ff; border-color: #1a73e8; }
        .lang-option input[type="radio"] { accent-color: #1a73e8; }
        .lang-option input[type="radio"]:checked + span { font-weight: 600; }
        .submit-btn {
            display: inline-block;
            background: #1a73e8;
            color: #fff;
            border: none;
            border-radius: 8px;
            padding: 0.7rem 2rem;
            font-size: 0.95rem;
            cursor: pointer;
            transition: background 0.2s;
        }
        .submit-btn:hover { background: #1557b0; }
        .msg {
            margin-top: 1rem;
            padding: 0.75rem 1rem;
            border-radius: 8px;
            font-size: 0.9rem;
        }
        .msg.success { background: #e6f4ea; color: #1e7e34; border: 1px solid #b7dfbf; }
        .msg.error { background: #fce8e6; color: #c5221f; border: 1px solid #f5c6cb; }
    </style>
</head>
<body>
    <a href="/" class="back-link">&#8592; Voltar à lista</a>
    <h1 style="margin-top: 1rem;">Upload de PDF</h1>
    <p class="subtitle">Envie um arquivo PDF e selecione o idioma original do documento.</p>

    {{if .Message}}
    <div class="msg {{if .IsError}}error{{else}}success{{end}}">{{.Message}}</div>
    {{end}}

    <div class="upload-form">
        <form method="POST" action="/upload" enctype="multipart/form-data">
            <div class="form-group">
                <label>Arquivo PDF</label>
                <input type="file" name="pdf" accept=".pdf" required>
            </div>
            <div class="form-group">
                <label>Idioma original do documento</label>
                <div class="lang-grid">
                    <label class="lang-option">
                        <input type="radio" name="language" value="en" checked>
                        <span>English</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="pt">
                        <span>Português</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="es">
                        <span>Español</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="fr">
                        <span>Français</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="de">
                        <span>Deutsch</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="it">
                        <span>Italiano</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="ja">
                        <span>日本語</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="zh">
                        <span>中文</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="ru">
                        <span>Русский</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="ko">
                        <span>한국어</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="ar">
                        <span>العربية</span>
                    </label>
                    <label class="lang-option">
                        <input type="radio" name="language" value="nl">
                        <span>Nederlands</span>
                    </label>
                </div>
            </div>
            <button type="submit" class="submit-btn">Enviar PDF</button>
        </form>
    </div>
</body>
</html>`))

// uploadData contém os dados para o template de upload.
type uploadData struct {
	Message string
	IsError bool
}

// handleUpload processa GET (exibir formulário) e POST (receber PDF).
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		uploadTemplate.Execute(w, uploadData{})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido.", http.StatusMethodNotAllowed)
		return
	}

	// Limitar tamanho do upload a 100MB
	r.ParseMultipartForm(100 << 20)

	// Obter arquivo
	file, header, err := r.FormFile("pdf")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		uploadTemplate.Execute(w, uploadData{Message: "Erro ao ler arquivo: " + err.Error(), IsError: true})
		return
	}
	defer file.Close()

	// Validar extensão
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		uploadTemplate.Execute(w, uploadData{Message: "Apenas arquivos PDF são aceitos.", IsError: true})
		return
	}

	// Obter idioma
	language := r.FormValue("language")
	if language == "" {
		language = "en"
	}

	// Salvar arquivo na pasta samples
	dstPath := filepath.Join(samplesDir, header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		uploadTemplate.Execute(w, uploadData{Message: "Erro ao salvar arquivo: " + err.Error(), IsError: true})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		uploadTemplate.Execute(w, uploadData{Message: "Erro ao copiar arquivo: " + err.Error(), IsError: true})
		return
	}

	// Salvar metadados no banco de dados
	meta := store.PDFMeta{
		Filename:   header.Filename,
		Language:   language,
		UploadedAt: time.Now(),
	}
	if err := db.Set(dstPath, meta); err != nil {
		log.Printf("Erro ao salvar metadados: %v", err)
	}

	log.Printf("Upload: %s (idioma: %s)", header.Filename, language)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	uploadTemplate.Execute(w, uploadData{
		Message: fmt.Sprintf("PDF \"%s\" enviado com sucesso! Idioma: %s", header.Filename, language),
	})
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
        .pdf-item-row {
            display: flex;
            align-items: center;
            padding: 0.5rem 1.25rem 0.5rem 0;
        }
        .pdf-item a {
            display: flex;
            align-items: center;
            flex: 1;
            padding: 0.5rem 0 0.5rem 1.25rem;
            text-decoration: none;
            color: #333;
        }
        .delete-btn {
            background: none;
            border: none;
            color: #ccc;
            font-size: 1.1rem;
            cursor: pointer;
            padding: 0.4rem 0.5rem;
            border-radius: 6px;
            transition: all 0.15s;
            flex-shrink: 0;
        }
        .delete-btn:hover {
            background: #fce8e6;
            color: #c5221f;
        }
        .pdf-icon {
            font-size: 1.5rem;
            margin-right: 1rem;
            flex-shrink: 0;
        }
        .pdf-name {
            font-weight: 500;
        }
        .lang-tag {
            background: #f0e6ff;
            color: #7c3aed;
            font-size: 0.7rem;
            padding: 0.15rem 0.5rem;
            border-radius: 10px;
            font-weight: 600;
            text-transform: uppercase;
            margin-left: 0.5rem;
        }
        .pdf-badge {
            margin-left: auto;
            background: #e0f0ff;
            color: #1a73e8;
            font-size: 0.75rem;
            padding: 0.2rem 0.5rem;
            border-radius: 10px;
            font-weight: 500;
        }
        .pdf-badge:empty { display: none; }
        .pdf-arrow {
            margin-left: 0.75rem;
            color: #999;
            font-size: 1.2rem;
            flex-shrink: 0;
        }
        .empty {
            color: #999;
            font-style: italic;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div style="display: flex; align-items: center; justify-content: space-between; max-width: 600px;">
        <h1>PDF to HTML</h1>
        <a href="/upload" style="background: #1a73e8; color: #fff; padding: 0.5rem 1.2rem; border-radius: 8px; text-decoration: none; font-size: 0.85rem;">+ Upload PDF</a>
    </div>
    <p class="subtitle">Clique em um PDF para continuar de onde parou.</p>
    {{if .Files}}
    <ul class="pdf-list">
        {{range .Files}}
        <li class="pdf-item">
            <div class="pdf-item-row">
                <a href="#" onclick="openPdf('{{.Path}}', '{{.Lang}}'); return false;">
                    <span class="pdf-icon">&#128196;</span>
                    <span class="pdf-name">{{.Name}}</span>
                    {{if .Lang}}<span class="lang-tag">{{.Lang}}</span>{{end}}
                    <span class="pdf-badge" id="badge-{{.Path}}"></span>
                    <span class="pdf-arrow">&#8594;</span>
                </a>
                <button class="delete-btn" title="Excluir" onclick="deletePdf('{{.Path}}', '{{.Name}}')">&#128465;</button>
            </div>
        </li>
        {{end}}
    </ul>
    {{else}}
    <p class="empty">Nenhum arquivo PDF encontrado na pasta "{{.Dir}}".</p>
    <p class="empty">Coloque seus PDFs na pasta e recarregue a página.</p>
    {{end}}
    <script>
        function getBookmark(pdfPath) {
            try {
                var data = JSON.parse(localStorage.getItem('pdf-bookmarks') || '{}');
                return data[pdfPath] || null;
            } catch(e) { return null; }
        }

        function deletePdf(pdfPath, name) {
            if (!confirm('Excluir "' + name + '"?\nO arquivo e seus metadados serão removidos permanentemente.')) {
                return;
            }
            fetch('/delete?pdf=' + encodeURIComponent(pdfPath), { method: 'DELETE' })
                .then(function(resp) {
                    if (resp.ok) {
                        // Remover bookmark do localStorage
                        try {
                            var data = JSON.parse(localStorage.getItem('pdf-bookmarks') || '{}');
                            delete data[pdfPath];
                            localStorage.setItem('pdf-bookmarks', JSON.stringify(data));
                        } catch(e) {}
                        window.location.reload();
                    } else {
                        resp.text().then(function(msg) { alert('Erro: ' + msg); });
                    }
                })
                .catch(function(err) { alert('Erro de rede: ' + err); });
        }

        function openPdf(pdfPath, lang) {
            var bookmark = getBookmark(pdfPath);
            var page = bookmark ? bookmark.page : 1;
            var url = '/convert?pdf=' + encodeURIComponent(pdfPath) + '&page=' + page;
            if (lang) url += '&lang=' + encodeURIComponent(lang);
            window.location.href = url;
        }

        // Mostrar badges com a última página lida
        document.addEventListener('DOMContentLoaded', function() {
            var badges = document.querySelectorAll('.pdf-badge');
            badges.forEach(function(badge) {
                var pdfPath = badge.id.replace('badge-', '');
                var bookmark = getBookmark(pdfPath);
                if (bookmark) {
                    badge.textContent = 'pág. ' + bookmark.page;
                }
            });
        });
    </script>
</body>
</html>`))

// pdfFile representa um arquivo PDF para o template.
type pdfFile struct {
	Name string
	Path string
	Lang string
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
				pdfPath := filepath.Join(samplesDir, entry.Name())
				lang := ""
				if db != nil {
					lang = db.GetLanguage(pdfPath)
				}
				files = append(files, pdfFile{
					Name: entry.Name(),
					Path: pdfPath,
					Lang: lang,
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
<html lang="{{.Lang}}">
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
        .font-controls {
            display: flex;
            align-items: center;
            gap: 0.3rem;
        }
        .font-btn {
            background: #16213e;
            color: #fff;
            border: 1px solid #0f3460;
            border-radius: 6px;
            padding: 0.25rem 0.6rem;
            font-size: 0.85rem;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.2s;
            line-height: 1;
        }
        .font-btn:hover { background: #0f3460; }
        .font-label {
            font-size: 0.7rem;
            color: #8ecae6;
            min-width: 30px;
            text-align: center;
        }
    </style>
</head>
<body>
    <nav class="navbar">
        <a href="/">&#8592; Voltar à lista</a>
        <div class="nav-center">
            <a class="nav-btn {{if le .Page 1}}disabled{{end}}"
               href="/convert?pdf={{.PdfPath}}&page={{.PrevPage}}&zoom={{.Zoom}}&fmt={{.Fmt}}&lang={{.Lang}}">
                &#9664; Anterior
            </a>
            <span class="page-info">Página {{.Page}} de {{.TotalPages}}</span>
            <a class="nav-btn {{if ge .Page .TotalPages}}disabled{{end}}"
               href="/convert?pdf={{.PdfPath}}&page={{.NextPage}}&zoom={{.Zoom}}&fmt={{.Fmt}}&lang={{.Lang}}">
                Próxima &#9654;
            </a>
        </div>
        <div class="font-controls">
            <button class="font-btn" onclick="changeFontSize(-1)" title="Diminuir fonte">A&minus;</button>
            <span class="font-label" id="font-delta">0</span>
            <button class="font-btn" onclick="changeFontSize(1)" title="Aumentar fonte">A+</button>
        </div>
    </nav>
    <div class="content">
        {{.HTMLContent}}
    </div>
    <script>
        // Salvar a página atual no localStorage
        (function() {
            try {
                var data = JSON.parse(localStorage.getItem('pdf-bookmarks') || '{}');
                data['{{.PdfPath}}'] = {
                    page: {{.Page}},
                    total: {{.TotalPages}},
                    timestamp: new Date().toISOString()
                };
                localStorage.setItem('pdf-bookmarks', JSON.stringify(data));
            } catch(e) {}
        })();

        // Controle de tamanho de fonte (A- / A+)
        // Funciona em dois modos:
        //   1. Se existem <font size="...">: ajusta o atributo size de cada um
        //   2. Senão: ajusta o font-size CSS do container de conteúdo
        var fontDelta = 0;
        var fontDeltaKey = 'pdf-font-delta';
        var fonts = document.querySelectorAll('.content font[size]');
        var originalSizes = [];
        var hasFontTags = fonts.length > 0;

        if (hasFontTags) {
            fonts.forEach(function(el) {
                originalSizes.push(parseInt(el.getAttribute('size'), 10) || 3);
            });
        }

        function applyFontDelta(delta) {
            fontDelta = Math.max(-8, Math.min(20, delta));
            if (hasFontTags) {
                fonts.forEach(function(el, i) {
                    var newSize = Math.max(1, originalSizes[i] + fontDelta);
                    el.setAttribute('size', newSize);
                });
            } else {
                var basePx = 16;
                var newPx = Math.max(8, basePx + fontDelta * 2);
                document.querySelector('.content').style.fontSize = newPx + 'px';
            }
            document.getElementById('font-delta').textContent =
                (fontDelta >= 0 ? '+' : '') + fontDelta;
            try { localStorage.setItem(fontDeltaKey, fontDelta.toString()); } catch(e) {}
        }

        function changeFontSize(step) {
            applyFontDelta(fontDelta + step);
        }

        // Restaurar delta salvo
        (function() {
            try {
                var saved = parseInt(localStorage.getItem(fontDeltaKey), 10);
                if (!isNaN(saved)) applyFontDelta(saved);
            } catch(e) {}
        })();
    </script>
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
	Lang        string
}

// handleDelete exclui um PDF do disco e do banco de dados.
//
// DELETE /delete?pdf=samples/teste.pdf
func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido. Use DELETE.", http.StatusMethodNotAllowed)
		return
	}

	pdfPath := r.URL.Query().Get("pdf")
	if pdfPath == "" {
		http.Error(w, "Parâmetro 'pdf' é obrigatório.", http.StatusBadRequest)
		return
	}

	// Segurança: garantir que o arquivo está dentro da pasta samples
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		http.Error(w, "Caminho inválido.", http.StatusBadRequest)
		return
	}
	absSamples, _ := filepath.Abs(samplesDir)
	if !strings.HasPrefix(absPath, absSamples+string(os.PathSeparator)) {
		http.Error(w, "Apenas arquivos na pasta samples podem ser excluídos.", http.StatusForbidden)
		return
	}

	// Remover arquivo do disco
	if err := os.Remove(pdfPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Erro ao remover arquivo %s: %v", pdfPath, err)
		http.Error(w, "Erro ao remover arquivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Remover do banco de dados
	if db != nil {
		if err := db.Delete(pdfPath); err != nil {
			log.Printf("Erro ao remover metadados de %s: %v", pdfPath, err)
		}
	}

	log.Printf("Excluído: %s", pdfPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	// Resolver idioma: query param > banco de dados > default
	lang := query.Get("lang")
	if lang == "" && db != nil {
		lang = db.GetLanguage(pdfPath)
	}
	if lang == "" {
		lang = "en"
	}

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
		Lang:        lang,
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
