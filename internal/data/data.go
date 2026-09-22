package data

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed xsd/* tabelas/*
var EmbeddedFS embed.FS

// Risco representa um item da Tabela 24 do eSocial.
type Risco struct {
	Codigo        string `json:"codigo"`
	Nome          string `json:"nome"`
	Categoria     string `json:"categoria"`
	Classificacao string `json:"classificacao"`
}

type riscosWrapper struct {
	Fonte  string  `json:"fonte"`
	Riscos []Risco `json:"riscos"`
}

// CBO representa uma ocupação da base oficial.
type CBO struct {
	Codigo    string `json:"codigo"`
	Titulo    string `json:"titulo"`
	Descricao string `json:"descricao,omitempty"`
}

// CarregarRiscosEsocial lê a base oficial da Tabela 24 embutida no binário.
func CarregarRiscosEsocial() ([]Risco, error) {
	bytes, err := EmbeddedFS.ReadFile("tabelas/riscos_esocial.json")
	if err != nil {
		return nil, fmt.Errorf("falha ao ler riscos_esocial.json: %w", err)
	}
	var wrapper riscosWrapper
	if err := json.Unmarshal(bytes, &wrapper); err != nil {
		return nil, fmt.Errorf("falha ao decodificar riscos_esocial.json: %w", err)
	}
	return wrapper.Riscos, nil
}

type cbosWrapper struct {
	Fonte     string `json:"fonte"`
	Ocupacoes []CBO  `json:"ocupacoes"`
}

// CarregarCBOs lê a tabela oficial de CBOs embutida no binário.
func CarregarCBOs() ([]CBO, error) {
	bytes, err := EmbeddedFS.ReadFile("tabelas/cbo_ocupacoes.json")
	if err != nil {
		return nil, fmt.Errorf("falha ao ler cbo_ocupacoes.json: %w", err)
	}
	var wrapper cbosWrapper
	if err := json.Unmarshal(bytes, &wrapper); err != nil {
		return nil, fmt.Errorf("falha ao decodificar cbo_ocupacoes.json: %w", err)
	}
	return wrapper.Ocupacoes, nil
}

// ObterXSD lê o conteúdo de um arquivo de esquema XSD embutido.
func ObterXSD(nome string) ([]byte, error) {
	caminho := fmt.Sprintf("xsd/%s.xsd", nome)
	bytes, err := EmbeddedFS.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler esquema %s: %w", caminho, err)
	}
	return bytes, nil
}
