# Clavy — Documentação

Aplicação Go para converter PDFs em HTML interativo, com suporte a dois modos de uso: **linha de comando (CLI)** e **servidor HTTP com interface web**.

---

## Sumário

- [Visão Geral](#visão-geral)
- [Pré-requisitos](#pré-requisitos)
- [Instalação](#instalação)
- [Modos de Uso](#modos-de-uso)
  - [Modo CLI](#modo-cli)
  - [Modo Servidor](#modo-servidor)
- [Funcionalidades](#funcionalidades)
- [Arquitetura](#arquitetura)
- [Endpoints HTTP](#endpoints-http)
- [Estrutura do Projeto](#estrutura-do-projeto)
- [Banco de Dados](#banco-de-dados)
- [Limitações](#limitações)
- [Melhorias Planejadas](#melhorias-planejadas)

---

## Visão Geral

O **Clavy** utiliza o [Poppler](https://poppler.freedesktop.org/) (`pdftohtml`) como engine de conversão, invocado via `os/exec`. O HTML gerado é autocontido — todas as imagens são embutidas como data URIs em base64, sem necessidade de servidor de assets.

```
┌──────────────┐   os/exec   ┌─────────────────┐
│   Go App     │ ──────────► │  pdftohtml      │
│              │             │  (Poppler)       │
│  1. Recebe   │ ◄────────── │  Gera HTML +    │
│     PDF      │   .html     │  imagens         │
│  2. Converte │             └─────────────────┘
│  3. Serve/   │
│     salva    │
└──────────────┘
```

**Destaques:**
- Zero dependências Go externas (apenas stdlib)
- HTML gerado é completamente autocontido (imagens em base64)
- Interface web responsiva com temas e controle de fonte
- Persistência de leitura via `localStorage`
- Detecção automática de idioma pelo Chrome para tradução

---

## Pré-requisitos

### Go (>= 1.21)

```bash
go version
# Se necessário:
brew install go          # macOS
```

### Poppler

```bash
# macOS
brew install poppler

# Ubuntu/Debian
sudo apt install poppler-utils

# Fedora/RHEL
sudo dnf install poppler-utils

# Arch Linux
sudo pacman -S poppler
```

> **Atenção:** Use `brew install poppler`, não `brew install pdftohtml`. O segundo instala o Xpdf, que tem flags incompatíveis com este projeto.

Verifique a instalação:

```bash
pdftohtml -v
pdfinfo -v
```

---

## Instalação

```bash
# Clonar o repositório
git clone https://github.com/sebastiaofortes/clavy.git
cd clavy

# Compilar
go build -o clavy .

# Verificar dependências
./clavy -check
```

---

## Modos de Uso

### Modo CLI

Converte um PDF para um arquivo HTML sem iniciar o servidor.

```bash
./clavy -pdf <arquivo.pdf> [opções]
```

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-pdf` | — | Caminho para o arquivo PDF (obrigatório) |
| `-page` | `0` | Página a converter (`0` = todas) |
| `-zoom` | `1.5` | Fator de zoom |
| `-fmt` | `png` | Formato das imagens: `png` ou `jpg` |
| `-output` | stdout | Arquivo de saída HTML |
| `-check` | — | Verifica dependências e sai |

**Exemplos:**

```bash
# Converter a página 1 e salvar
./clavy -pdf samples/livro.pdf -page 1 -output output/pagina1.html

# Converter com zoom maior e formato JPG
./clavy -pdf samples/livro.pdf -page 3 -zoom 2.0 -fmt jpg -output output/p3.html

# Converter todas as páginas para stdout
./clavy -pdf samples/livro.pdf

# Verificar dependências instaladas
./clavy -check
```

### Modo Servidor

Inicia uma interface web completa para gerenciar e ler PDFs.

```bash
./clavy -serve 8080
```

Acesse no navegador: `http://localhost:8080`

**Fluxo de uso:**

1. Na página inicial, clique em **+ Upload PDF**
2. Selecione o arquivo PDF e o idioma de origem
3. Clique em **Enviar PDF**
4. Selecione o livro na lista para começar a ler

---

## Funcionalidades

### Visualizador de PDFs
- Navegação página a página (Anterior / Próxima)
- Ajuste de tamanho da fonte (A− / A+)
- Três temas: **Branco**, **Creme** e **Escuro**
- Todas as preferências são salvas automaticamente no `localStorage`

### Retomada de leitura
O sistema armazena a última página lida de cada PDF no `localStorage` do navegador. Ao abrir um livro novamente, a leitura retoma de onde parou.

### Tradução automática
Ao acessar uma página, o Chrome detecta o idioma do documento (definido no upload) e oferece tradução automática. Selecione "Traduzir todas as páginas" para traduzir a leitura completa.

### Gerenciamento de PDFs
- **Upload**: envio de PDF com seleção de idioma (12 idiomas suportados)
- **Exclusão**: remove o arquivo PDF e seus metadados

---

## Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│                   FRONTEND (HTML/JS/CSS)                 │
│    /          /upload         /convert                   │
│  (lista)    (formulário)    (visualizador)               │
└────────────────────────────┬────────────────────────────┘
                             │ HTTP
┌────────────────────────────┴────────────────────────────┐
│                    BACKEND (Go)                          │
│  ┌──────────────┬─────────────────┬──────────────────┐  │
│  │ converter/   │ store/          │ main.go           │  │
│  │ converter.go │ store.go        │ handlers HTTP     │  │
│  │              │                 │ templates         │  │
│  │ Convert()    │ Get() / Set()   │ handleList        │  │
│  │ PageCount()  │ Persistência    │ handleUpload      │  │
│  │ embedImages()│ JSON            │ handleConvert     │  │
│  └──────────────┴─────────────────┴──────────────────┘  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │    Poppler (pdftohtml / pdfinfo) via os/exec     │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Fluxo de conversão

```
GET /convert?pdf=samples/livro.pdf&page=3&zoom=1.5
       │
       ▼
handleConvert()
       │
       ├── converter.Convert()
       │       ├── exec: pdftohtml -noframes -nodrm -q -f 3 -l 3 -zoom 1.5 -fmt png
       │       ├── Lê HTML gerado em diretório temporário
       │       └── embedImages() → converte <img src="..."> para data URIs base64
       │
       ├── converter.PageCount()
       │       └── exec: pdfinfo → extrai total de páginas
       │
       └── viewerTemplate.Execute() → HTML final com navbar + controles
```

### Packages

| Package | Responsabilidade |
|---------|-----------------|
| `main` | CLI, servidor HTTP, templates HTML/JS/CSS, handlers |
| `converter` | Invocação do Poppler, processamento de imagens, contagem de páginas |
| `store` | Leitura e escrita do banco de dados JSON, thread-safety via `sync.RWMutex` |

---

## Endpoints HTTP

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/` | Lista os PDFs disponíveis |
| `GET` | `/upload` | Exibe o formulário de upload |
| `POST` | `/upload` | Recebe o PDF e salva metadados |
| `GET` | `/convert` | Converte e exibe uma página do PDF |
| `DELETE` | `/delete` | Remove um PDF e seus metadados |
| `GET` | `/health` | Retorna status do servidor |

### Parâmetros de `/convert`

| Parâmetro | Obrigatório | Padrão | Descrição |
|-----------|-------------|--------|-----------|
| `pdf` | sim | — | Caminho relativo do arquivo PDF |
| `page` | não | `1` | Número da página |
| `zoom` | não | `1.5` | Fator de zoom |
| `fmt` | não | `png` | Formato das imagens (`png` ou `jpg`) |
| `lang` | não | — | Idioma do documento (BCP 47) |

**Exemplo:**
```
GET /convert?pdf=samples/livro.pdf&page=5&zoom=1.3&fmt=png&lang=en
```

---

## Estrutura do Projeto

```
clavy/
├── main.go                    # CLI, servidor HTTP, templates, handlers
├── go.mod                     # Módulo Go (sem dependências externas)
├── converter/
│   └── converter.go           # Engine de conversão via Poppler
├── store/
│   └── store.go               # Banco de dados JSON com thread-safety
├── data/                      # Banco de dados local (gitignored)
│   └── books.json             # Metadados dos PDFs (gerado automaticamente)
├── samples/                   # PDFs armazenados (gitignored)
├── output/                    # HTMLs gerados via CLI (gitignored)
├── IMPLEMENTATION.md          # Guia de implementação com Poppler
├── PDF2HTMLEX-IMPLEMENTATION.md  # Plano para suporte ao pdf2htmlEX
└── DOCS.md                    # Este arquivo
```

---

## Banco de Dados

Os metadados dos PDFs são persistidos em `data/books.json`. O arquivo é criado automaticamente na primeira execução do servidor.

**Formato:**

```json
{
  "samples/livro.pdf": {
    "filename": "livro.pdf",
    "language": "pt",
    "uploaded_at": "2026-02-21T11:24:08.561232-03:00"
  }
}
```

| Campo | Descrição |
|-------|-----------|
| `filename` | Nome original do arquivo |
| `language` | Código BCP 47 do idioma (ex: `pt`, `en`, `es`) |
| `uploaded_at` | Data e hora do upload |

O acesso ao banco de dados é thread-safe via `sync.RWMutex`.

---

## Limitações

| Limitação | Detalhes |
|-----------|---------|
| **PDFs escaneados** | Não extrai texto de imagens (seria necessário OCR com Tesseract) |
| **Fidelidade visual** | Boa, mas não é pixel-perfect — layouts complexos com colunas podem ficar distorcidos |
| **Fontes customizadas** | Substituídas por fontes genéricas do sistema |
| **DRM** | Arquivos com restrições severas podem falhar (a flag `-nodrm` contorna casos simples) |
| **Timeout** | Conversões têm limite de 30 segundos por padrão |

---

## Melhorias Planejadas

### Suporte ao pdf2htmlEX (via Docker)

Está planejado o suporte a uma segunda engine de conversão: o **pdf2htmlEX**, que oferece fidelidade pixel-perfect ao preservar fontes, layout e imagens com qualidade superior.

| Engine | Fidelidade | Velocidade | Dependência |
|--------|-----------|-----------|-------------|
| Poppler (`pdftohtml`) | Boa | ~200ms | Binário local |
| pdf2htmlEX | Excelente (pixel-perfect) | ~2–5s | Docker |

O plano de implementação completo está em [PDF2HTMLEX-IMPLEMENTATION.md](PDF2HTMLEX-IMPLEMENTATION.md).

### Outras melhorias

- Cache de páginas já convertidas (evitar reconversão desnecessária)
- Suporte a OCR via Tesseract para PDFs escaneados
- Testes automatizados com PDFs de exemplo
- Containerização com Docker para deploy em produção
