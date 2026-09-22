package esocial

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

const NamespaceS2220 = "http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00"

// Tipos de Exame Ocupacional
const (
	ExameAdmissional       = 0
	ExamePeriodico         = 1
	ExameRetornoTrabalho   = 2
	ExameMudancaFuncao     = 3
	ExameMonitoracaoPontual = 4
	ExameDemissional       = 9
)

// Resultados do ASO
const (
	ResultadoApto   = 1
	ResultadoInapto = 2
)

// EventoS2220 envelope raiz para o Monitoramento da Saúde do Trabalhador (ASO).
type EventoS2220 struct {
	XMLName  xml.Name `xml:"eSocial"`
	Xmlns    string   `xml:"xmlns,attr"`
	EvtMonit EvtMonit `xml:"evtMonit"`
}

// EvtMonit congrega dados cadastrais, vínculo e o exame médico ocupacional.
type EvtMonit struct {
	Id           string        `xml:"Id,attr"`
	IdeEvento    IdeEventoTrab `xml:"ideEvento"`
	IdeEmpregador IdeEmpregador `xml:"ideEmpregador"`
	IdeVinculo   IdeVinculoSST `xml:"ideVinculo"`
	ExMedOcup    ExMedOcup     `xml:"exMedOcup"`
}

// ExMedOcup detalha o tipo de exame ocupacional, o ASO emitido e o coordenador do PCMSO.
type ExMedOcup struct {
	TpExameOcup int         `xml:"tpExameOcup"`           // 0, 1, 2, 3, 4, 9
	ASO         DadosASO    `xml:"aso"`                   // Atestado de Saúde Ocupacional
	RespMonit   *RespMonit  `xml:"respMonit,omitempty"`   // Responsável/coordenador do PCMSO
}

// DadosASO registra a emissão, resultado, exames complementares e o médico emitente do ASO.
type DadosASO struct {
	DtAso   string       `xml:"dtAso"`             // AAAA-MM-DD
	ResAso  *int         `xml:"resAso,omitempty"`  // 1 - Apto, 2 - Inapto
	Exame   []ExameItem  `xml:"exame,omitempty"`   // Até 99 exames complementares
	Medico  MedicoASO    `xml:"medico"`            // Médico emitente do ASO
}

// ExameItem detalha cada avaliação clínica e exame complementar realizado (Tabela 27 do eSocial).
type ExameItem struct {
	DtExm         string `xml:"dtExm"`                   // AAAA-MM-DD
	ProcRealizado string `xml:"procRealizado"`           // 4 dígitos (Tabela 27)
	ObsProc       string `xml:"obsProc,omitempty"`       // Observação sobre o procedimento
	OrdExame      *int   `xml:"ordExame,omitempty"`      // 1 - Inicial, 2 - Sequencial
	IndResult     *int   `xml:"indResult,omitempty"`     // 1 - Normal, 2 - Alterado, 3 - Estável, 4 - Agravamento
}

// MedicoASO identifica o médico do trabalho ou examinador emitente do ASO.
type MedicoASO struct {
	NmMed string `xml:"nmMed"`           // Nome do médico
	NrCRM string `xml:"nrCRM,omitempty"` // Número no CRM
	UfCRM string `xml:"ufCRM,omitempty"` // Sigla da UF do CRM
}

// RespMonit identifica o médico responsável ou coordenador do PCMSO da empresa.
type RespMonit struct {
	CpfResp string `xml:"cpfResp,omitempty"` // CPF do coordenador
	NmResp  string `xml:"nmResp"`            // Nome do coordenador
	NrCRM   string `xml:"nrCRM"`             // Número de inscrição no CRM
	UfCRM   string `xml:"ufCRM"`             // Sigla da UF de expedição do CRM
}

// GerarXMLS2220 serializa o evento S-2220 para XML nos padrões estritos do eSocial S-1.3.
func GerarXMLS2220(evento *EventoS2220) ([]byte, error) {
	if evento == nil {
		return nil, fmt.Errorf("evento S-2220 não pode ser nulo")
	}
	evento.Xmlns = NamespaceS2220
	return GerarXMLEvento(evento)
}

// ParseS2220 decodifica e valida o XML de um evento S-2220, suportando arquivos
// enviados por clínicas externas de medicina e segurança do trabalho.
func ParseS2220(xmlData []byte) (*EventoS2220, error) {
	if len(bytes.TrimSpace(xmlData)) == 0 {
		return nil, errors.New("arquivo XML vazio")
	}

	var evento EventoS2220
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	if err := dec.Decode(&evento); err != nil {
		return nil, fmt.Errorf("falha ao decodificar XML S-2220: %w", err)
	}

	// Validações essenciais de integridade
	if strings.TrimSpace(evento.EvtMonit.IdeVinculo.CpfTrab) == "" {
		return nil, errors.New("XML S-2220 inválido: CPF do trabalhador (cpfTrab) não informado")
	}

	if strings.TrimSpace(evento.EvtMonit.ExMedOcup.ASO.DtAso) == "" {
		return nil, errors.New("XML S-2220 inválido: data de emissão do ASO (dtAso) não informada")
	}

	if strings.TrimSpace(evento.EvtMonit.ExMedOcup.ASO.Medico.NmMed) == "" {
		return nil, errors.New("XML S-2220 inválido: médico emitente (nmMed) não informado")
	}

	return &evento, nil
}
