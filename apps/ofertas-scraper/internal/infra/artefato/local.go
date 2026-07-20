package artefato

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/domain"
)

// LocalStore writes Artefatos under ARTEFATO_ROOT (ADR 0021).
type LocalStore struct {
	Root string
}

func NewLocalStore(root string) *LocalStore {
	return &LocalStore{Root: root}
}

func (s *LocalStore) dir(doc domain.Documento, tentativa string) string {
	return filepath.Join(s.Root, string(doc.FonteID), doc.Dia, doc.Filename, tentativa)
}

func (s *LocalStore) AttemptPath(doc domain.Documento, tentativa string) string {
	return s.dir(doc, tentativa)
}

func (s *LocalStore) ensure(doc domain.Documento, tentativa string) (string, error) {
	dir := s.dir(doc, tentativa)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *LocalStore) SavePDF(_ context.Context, doc domain.Documento, tentativa string, pdf []byte) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "original.pdf"), pdf, 0o644)
}

func (s *LocalStore) SaveImages(_ context.Context, doc domain.Documento, tentativa string, images []domain.PageImage) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	imgDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return err
	}
	for _, img := range images {
		name := fmt.Sprintf("page-%03d.jpg", img.Page)
		if err := os.WriteFile(filepath.Join(imgDir, name), img.JPEG, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *LocalStore) SaveRawExtrator(_ context.Context, doc domain.Documento, tentativa string, raw []byte) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "extrator-raw.json"), raw, 0o644)
}

func (s *LocalStore) SaveUsoExtrator(_ context.Context, doc domain.Documento, tentativa string, uso domain.UsoExtrator) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(uso, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "uso-extrator.json"), b, 0o644)
}

func (s *LocalStore) SaveValidated(_ context.Context, doc domain.Documento, tentativa string, ofertas []domain.Oferta, falhas []domain.FalhaExtracao) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	payload := struct {
		Ofertas []domain.Oferta        `json:"ofertas"`
		Falhas  []domain.FalhaExtracao `json:"falhas"`
	}{Ofertas: ofertas, Falhas: falhas}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "validated.json"), b, 0o644)
}

func (s *LocalStore) SaveConteudoIdentico(_ context.Context, doc domain.Documento, tentativa string, priorID domain.DocumentoID) error {
	dir, err := s.ensure(doc, tentativa)
	if err != nil {
		return err
	}
	payload := struct {
		ConteudoIdenticoA domain.DocumentoID `json:"conteudoIdenticoA"`
		Fingerprint       string             `json:"fingerprint,omitempty"`
	}{ConteudoIdenticoA: priorID, Fingerprint: doc.Fingerprint}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "conteudo-identico.json"), b, 0o644)
}
