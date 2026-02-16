// Package store fornece um banco de dados simples baseado em arquivo JSON
// para armazenar metadados dos PDFs (idioma original, data de upload, etc.).
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// PDFMeta contém os metadados de um PDF.
type PDFMeta struct {
	Filename   string    `json:"filename"`
	Language   string    `json:"language"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// Store é o banco de dados baseado em arquivo JSON.
type Store struct {
	path string
	mu   sync.RWMutex
	data map[string]PDFMeta // chave: caminho relativo do PDF (ex: "samples/teste.pdf")
}

// New cria ou carrega um Store a partir do arquivo especificado.
func New(path string) (*Store, error) {
	s := &Store{
		path: path,
		data: make(map[string]PDFMeta),
	}

	// Se o arquivo já existe, carregar os dados
	if _, err := os.Stat(path); err == nil {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler store: %w", err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &s.data); err != nil {
				return nil, fmt.Errorf("erro ao parsear store: %w", err)
			}
		}
	}

	return s, nil
}

// Set salva os metadados de um PDF.
func (s *Store) Set(pdfPath string, meta PDFMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[pdfPath] = meta
	return s.save()
}

// Get retorna os metadados de um PDF, ou ok=false se não existir.
func (s *Store) Get(pdfPath string) (PDFMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, ok := s.data[pdfPath]
	return meta, ok
}

// GetLanguage retorna o idioma de um PDF, ou string vazia se não existir.
func (s *Store) GetLanguage(pdfPath string) string {
	meta, ok := s.Get(pdfPath)
	if !ok {
		return ""
	}
	return meta.Language
}

// Delete remove os metadados de um PDF do banco de dados.
func (s *Store) Delete(pdfPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, pdfPath)
	return s.save()
}

// All retorna todos os metadados.
func (s *Store) All() map[string]PDFMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[string]PDFMeta, len(s.data))
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}

// save persiste os dados no arquivo JSON.
func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar store: %w", err)
	}
	if err := os.WriteFile(s.path, raw, 0644); err != nil {
		return fmt.Errorf("erro ao salvar store: %w", err)
	}
	return nil
}
