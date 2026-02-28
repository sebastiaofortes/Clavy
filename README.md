# Clavy — Documentação

Clavy é uma aplicação desenvolvida para visualizar PDFs em formato de páginas HTML interativas, permitindo sua tradução pelo navegador.
A aplicação conta também com suporte a dois modos de uso: **linha de comando (CLI)** e **servidor HTTP com interface web**.

---

## Sumário

- [Manual de Uso](#manual-de-uso)
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

## Manual de Uso

Siga os passos abaixo e você estará lendo e traduzindo PDFs em minutos.

### Passo 1 — Abrir o Clavy no navegador

Com o servidor já rodando (veja a seção [Instalação](#instalação)), abra o **Google Chrome** e acesse:

```
http://localhost:8080
```

Você verá a lista de livros disponíveis (inicialmente vazia).

---

### Passo 2 — Enviar um PDF

1. Clique no botão **"+ Upload PDF"** no canto superior direito da tela.
2. Na tela de upload, clique em **"Escolher arquivo"** e selecione o PDF do seu computador.
3. No campo **"Idioma do documento"**, escolha o idioma em que o PDF está escrito (ex: *Inglês*, *Espanhol*, *Francês*). Isso permite que o Chrome ofereça tradução automática depois.
4. Clique em **"Enviar PDF"**.
5. Aguarde o upload. O livro aparecerá na lista principal.

> **Dica:** O arquivo pode ter até **100 MB**. PDFs maiores podem demorar um pouco mais para carregar.

---

### Passo 3 — Abrir e ler o PDF

1. Na lista principal, clique no nome do livro que você acabou de enviar.
2. O visualizador abrirá na **primeira página** do documento.
3. Use os botões de navegação para avançar ou voltar páginas:
   - **← Anterior** — página anterior
   - **Próxima →** — próxima página
4. Ajuste o tamanho do texto com os botões **A−** (menor) e **A+** (maior).
5. Mude o tema de leitura clicando em **Branco**, **Creme** ou **Escuro** conforme sua preferência.

> **Dica:** O Clavy salva automaticamente a última página lida. Na próxima vez que abrir o mesmo livro, ele retomará de onde você parou.

> **Observação:** A informação da última página lida fica guardada no armazenamento local do navegador (`localStorage`). Se você limpar o histórico ou os dados do navegador, essa informação será perdida e a leitura voltará à página 1.

---

### Passo 4 — Traduzir o documento com o Chrome

O Chrome detecta automaticamente o idioma do documento (o mesmo que você informou no upload) e exibe uma barra de tradução.

1. Quando a barra de tradução aparecer no topo do Chrome, clique em **"Traduzir"**.
2. Se a barra não aparecer automaticamente, clique com o botão direito em qualquer parte da página e selecione **"Traduzir para o português"**.
3. Para traduzir todas as páginas sem precisar clicar toda vez, escolha a opção **"Traduzir sempre de [idioma]"** no menu de opções da barra de tradução (ícone ⚙️ ou os três pontinhos ao lado do botão Traduzir).

> **Atenção:** A tradução automática funciona melhor no **Google Chrome**. Em outros navegadores (Firefox, Safari, Edge) o comportamento pode ser diferente.

---

### Passo 5 — Remover um PDF

Para excluir um livro da lista:

1. Volte para a página inicial (`http://localhost:8080`).
2. Ao lado do livro que deseja remover, clique no botão **"Excluir"** (ícone de lixeira).
3. Confirme a exclusão. O arquivo e seus metadados serão apagados permanentemente.

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