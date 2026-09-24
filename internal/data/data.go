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

// ItemTabela representa um código/descrição de uma tabela oficial do eSocial.
type ItemTabela struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
}

type itensWrapper struct {
	Fonte string       `json:"fonte"`
	Itens []ItemTabela `json:"itens"`
}

func carregarItens(arquivo string) ([]ItemTabela, error) {
	bytes, err := EmbeddedFS.ReadFile("tabelas/" + arquivo)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler %s: %w", arquivo, err)
	}
	var wrapper itensWrapper
	if err := json.Unmarshal(bytes, &wrapper); err != nil {
		return nil, fmt.Errorf("falha ao decodificar %s: %w", arquivo, err)
	}
	return wrapper.Itens, nil
}

// CarregarPartesCorpo lê a Tabela 13 (parte do corpo atingida).
func CarregarPartesCorpo() ([]ItemTabela, error) {
	return carregarItens("tabela13_partes_corpo.json")
}

// CarregarAgentesCausadores lê a Tabela 14 (agente causador do acidente).
func CarregarAgentesCausadores() ([]ItemTabela, error) {
	return carregarItens("tabela14_agentes_causadores.json")
}

// CarregarSituacoesGeradoras lê a Tabela 15 (situação geradora do acidente).
func CarregarSituacoesGeradoras() ([]ItemTabela, error) {
	return carregarItens("tabela15_situacoes.json")
}

// CarregarNaturezasLesao lê a Tabela 17 (descrição da natureza da lesão).
func CarregarNaturezasLesao() ([]ItemTabela, error) {
	return carregarItens("tabela17_lesoes.json")
}

// CarregarProcedimentosDiagnosticos lê a Tabela 27 (procedimentos diagnósticos do ASO).
func CarregarProcedimentosDiagnosticos() ([]ItemTabela, error) {
	return carregarItens("tabela27_procedimentos.json")
}
