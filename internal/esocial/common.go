package esocial

import (
	"bytes"
	"encoding/xml"
)

const (
	// VersaoLeiaute define a versão oficial S-1.3 suportada
	VersaoLeiaute = "S-1.3"

	// VersaoAplicativo padrão para o emissor livre (máx 20 caracteres conforme TS_verProc)
	VersaoAplicativo = "EmissorLivre-1.0"

	// Ambientes eSocial
	AmbienteProducao        = 1
	AmbienteProducaoRestrita = 2

	// Processo de Emissão
	ProcEmiAppEmpregador = 1 // 1 - Aplicativo do empregador

	// Tipos de Inscrição
	TpInscCNPJ = 1
	TpInscCPF  = 2
	TpInscCAEPF = 3
	TpInscCNO   = 4
)

// IdeEventoTab representa a identificação de evento de tabela (como S-1000).
type IdeEventoTab struct {
	TpAmb   int    `xml:"tpAmb"`
	ProcEmi int    `xml:"procEmi"`
	VerProc string `xml:"verProc"`
}

// IdeEventoTrab representa a identificação de eventos não periódicos de SST (S-2210, S-2220, S-2240).
type IdeEventoTrab struct {
	IndRetif int    `xml:"indRetif"`           // 1 - Original, 2 - Retificação
	NrRecibo string `xml:"nrRecibo,omitempty"` // Obrigatório se IndRetif = 2
	TpAmb    int    `xml:"tpAmb"`              // 1 - Produção, 2 - Produção restrita
	ProcEmi  int    `xml:"procEmi"`            // 1 - Aplicativo do empregador
	VerProc  string `xml:"verProc"`            // Versão do aplicativo
}

// IdeEmpregador identifica o empregador/contribuinte declarante.
type IdeEmpregador struct {
	TpInsc int    `xml:"tpInsc"`
	NrInsc string `xml:"nrInsc"`
}

// IdeVinculoSST identifica o colaborador/trabalhador e seu vínculo/contrato no eSocial.
type IdeVinculoSST struct {
	CpfTrab   string `xml:"cpfTrab"`
	Matricula string `xml:"matricula,omitempty"`
	CodCateg  string `xml:"codCateg,omitempty"` // Informado somente se TSVE sem matrícula
}

// GerarXMLEvento serializa uma estrutura para XML com cabeçalho padrão UTF-8 e indentação limpa.
func GerarXMLEvento(v any) ([]byte, error) {
	buf := bytes.NewBufferString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
