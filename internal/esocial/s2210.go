package esocial

import (
	"encoding/xml"
	"fmt"
)

const NamespaceS2210 = "http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00"

// EventoS2210 representa o envelope raiz do evento S-2210 (Comunicação de Acidente de Trabalho - CAT).
type EventoS2210 struct {
	XMLName xml.Name `xml:"eSocial"`
	Xmlns   string   `xml:"xmlns,attr"`
	EvtCAT  EvtCAT   `xml:"evtCAT"`
}

// EvtCAT congrega a identificação do evento, empregador, trabalhador e os dados do acidente.
type EvtCAT struct {
	Id           string        `xml:"Id,attr"`
	IdeEvento    IdeEventoTrab `xml:"ideEvento"`
	IdeEmpregador IdeEmpregador `xml:"ideEmpregador"`
	IdeVinculo   IdeVinculoSST `xml:"ideVinculo"`
	CAT          DadosCAT      `xml:"cat"`
}

// DadosCAT detalha o acidente de trabalho, partes atingidas, agente causador e o atestado médico.
type DadosCAT struct {
	DtAcid           string          `xml:"dtAcid"`                     // AAAA-MM-DD
	TpAcid           int             `xml:"tpAcid"`                     // 1 - Típico, 2 - Doença, 3 - Trajeto
	HrAcid           string          `xml:"hrAcid,omitempty"`           // HHMM
	HrsTrabAntesAcid string          `xml:"hrsTrabAntesAcid,omitempty"` // HHMM
	TpCat            int             `xml:"tpCat"`                      // 1 - Inicial, 2 - Reabertura, 3 - Comunicação de Óbito
	IndCatObito      string          `xml:"indCatObito"`                // S ou N
	DtObito          string          `xml:"dtObito,omitempty"`          // AAAA-MM-DD (se indCatObito=S)
	IndComunPolicia  string          `xml:"indComunPolicia"`            // S ou N
	CodSitGeradora   string          `xml:"codSitGeradora"`             // 9 dígitos (Tabela 15)
	IniciatCAT       int             `xml:"iniciatCAT"`                 // 1 - Empregador, 2 - Ordem judicial, 3 - Órgão fiscalizador
	ObsCAT           string          `xml:"obsCAT,omitempty"`           // Observação
	UltDiaTrab       string          `xml:"ultDiaTrab,omitempty"`       // AAAA-MM-DD
	HouveAfast       string          `xml:"houveAfast,omitempty"`       // S ou N
	LocalAcidente    LocalAcidente   `xml:"localAcidente"`              // Local onde ocorreu o acidente
	ParteAtingida    ParteAtingida   `xml:"parteAtingida"`              // Parte do corpo atingida (Tabela 13)
	AgenteCausador   AgenteCausador  `xml:"agenteCausador"`             // Agente causador (Tabela 14/15)
	Atestado         AtestadoMedico  `xml:"atestado"`                   // Atestado médico de atendimento
	CatOrigem        *CatOrigem      `xml:"catOrigem,omitempty"`        // Para reabertura ou óbito
}

// LocalAcidente especifica o endereço e enquadramento do local do infortúnio.
type LocalAcidente struct {
	TpLocal      int            `xml:"tpLocal"`                // 1 a 6, 9
	DscLocal     string         `xml:"dscLocal,omitempty"`     // Pátio, rampa, etc.
	TpLograd     string         `xml:"tpLograd,omitempty"`     // R, AV, PCA, etc.
	DscLograd    string         `xml:"dscLograd"`              // Nome do logradouro
	NrLograd     string         `xml:"nrLograd"`               // Número do logradouro
	Complemento  string         `xml:"complemento,omitempty"`  // Complemento
	Bairro       string         `xml:"bairro,omitempty"`       // Bairro
	Cep          string         `xml:"cep,omitempty"`          // 8 dígitos numéricos
	CodMunic     string         `xml:"codMunic,omitempty"`     // 7 dígitos IBGE
	Uf           string         `xml:"uf,omitempty"`           // Sigla da UF
	Pais         string         `xml:"pais,omitempty"`         // Código do país (se exterior)
	CodPostal    string         `xml:"codPostal,omitempty"`    // Código postal (se exterior)
	IdeLocalAcid *IdeLocalAcid  `xml:"ideLocalAcid,omitempty"` // Identificação do estabelecimento
}

// IdeLocalAcid identifica o CNPJ/CNO do estabelecimento onde ocorreu o acidente.
type IdeLocalAcid struct {
	TpInsc int    `xml:"tpInsc"` // 1 - CNPJ, 3 - CAEPF, 4 - CNO
	NrInsc string `xml:"nrInsc"` // Inscrição do contratante/estabelecimento
}

// ParteAtingida especifica o órgão lesado e sua lateralidade.
type ParteAtingida struct {
	CodParteAting string `xml:"codParteAting"` // 9 dígitos (Tabela 13)
	Lateralidade  int    `xml:"lateralidade"`  // 0 - Não aplicável, 1 - Esquerda, 2 - Direita, 3 - Ambas
}

// AgenteCausador especifica a máquina, ferramenta ou substância causadora da lesão.
type AgenteCausador struct {
	CodAgntCausador string `xml:"codAgntCausador"` // 9 dígitos (Tabela 14/15)
}

// AtestadoMedico registra o primeiro atendimento médico da vítima.
type AtestadoMedico struct {
	DtAtendimento string         `xml:"dtAtendimento"`           // AAAA-MM-DD
	HrAtendimento string         `xml:"hrAtendimento"`           // HHMM
	IndInternacao string         `xml:"indInternacao"`           // S ou N
	DurTrat       int            `xml:"durTrat"`                 // Duração estimada do tratamento em dias
	IndAfast      string         `xml:"indAfast"`                // S ou N
	DscLesao      string         `xml:"dscLesao"`                // 9 dígitos (Tabela 17 da natureza da lesão)
	DscCompLesao  string         `xml:"dscCompLesao,omitempty"`  // Descrição complementar
	DiagProvavel  string         `xml:"diagProvavel,omitempty"`  // Diagnóstico provável
	CodCID        string         `xml:"codCID"`                  // Código CID-10
	Observacao    string         `xml:"observacao,omitempty"`    // Observação médica
	Emitente      EmitenteMedico `xml:"emitente"`                // Médico que emitiu o atestado
}

// EmitenteMedico identifica o profissional de saúde declarante e seu CRM.
type EmitenteMedico struct {
	NmEmit string `xml:"nmEmit"`           // Nome do médico ou dentista
	IdeOC  int    `xml:"ideOC"`            // 1 - CRM, 2 - CRO, 3 - RMS
	NrOC   string `xml:"nrOC"`             // Número de inscrição no conselho
	UfOC   string `xml:"ufOC,omitempty"`   // Sigla da UF de expedição do CRM
}

// CatOrigem vincula à CAT originária em reabertura ou óbito subsequente.
type CatOrigem struct {
	NrRecCatOrig string `xml:"nrRecCatOrig"` // Recibo da CAT original
}

// GerarXMLS2210 serializa o evento S-2210 para XML segundo os esquemas oficiais S-1.3.
func GerarXMLS2210(evento *EventoS2210) ([]byte, error) {
	if evento == nil {
		return nil, fmt.Errorf("evento S-2210 não pode ser nulo")
	}
	evento.Xmlns = NamespaceS2210
	return GerarXMLEvento(evento)
}
