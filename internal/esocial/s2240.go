package esocial

import (
	"encoding/xml"
	"fmt"
)

const NamespaceS2240 = "http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00"

// EventoS2240 representa a declaração de Condições Ambientais do Trabalho - Agentes Nocivos.
type EventoS2240 struct {
	XMLName     xml.Name    `xml:"eSocial"`
	Xmlns       string      `xml:"xmlns,attr"`
	EvtExpRisco EvtExpRisco `xml:"evtExpRisco"`
}

// EvtExpRisco agrupa os blocos de identificação e informações de exposição ao risco ambiental.
type EvtExpRisco struct {
	Id           string        `xml:"Id,attr"`
	IdeEvento    IdeEventoTrab `xml:"ideEvento"`
	IdeEmpregador IdeEmpregador `xml:"ideEmpregador"`
	IdeVinculo   IdeVinculoSST `xml:"ideVinculo"`
	InfoExpRisco InfoExpRisco  `xml:"infoExpRisco"`
}

// InfoExpRisco detalha datas, ambientes, atividades, agentes nocivos e responsáveis pelos registros.
type InfoExpRisco struct {
	DtIniCondicao string      `xml:"dtIniCondicao"` // AAAA-MM-DD
	DtFimCondicao string      `xml:"dtFimCondicao,omitempty"` // Opcional (obrigatório avulso)
	InfoAmb       []InfoAmb   `xml:"infoAmb"`                 // 1 a 9 ambientes
	InfoAtiv      InfoAtiv    `xml:"infoAtiv"`                // Atividades exercidas
	AgNoc         []AgNoc     `xml:"agNoc"`                   // 1 a 999 agentes nocivos
	RespReg       []RespReg   `xml:"respReg"`                 // 1 a 99 responsáveis técnicos
	Obs           *ObsS2240   `xml:"obs,omitempty"`           // Observações complementares
}

// InfoAmb representa o local físico/administrativo onde o colaborador trabalha.
type InfoAmb struct {
	LocalAmb int    `xml:"localAmb"` // 1 - Estabelecimento do próprio empregador, 2 - Terceiros
	DscSetor string `xml:"dscSetor"` // Ex: "Oficina Mecânica", "Produção Farmacêutica"
	TpInsc   int    `xml:"tpInsc"`   // 1 - CNPJ, 3 - CAEPF, 4 - CNO
	NrInsc   string `xml:"nrInsc"`   // Número de inscrição onde está o ambiente
}

// InfoAtiv descreve a atividade realizada no ambiente.
type InfoAtiv struct {
	DscAtivDes string `xml:"dscAtivDes"` // Verbos no infinitivo, ex: "Operar torno mecânico..."
}

// AgNoc representa um agente nocivo ao qual o trabalhador está exposto (Tabela 24 do eSocial).
type AgNoc struct {
	CodAgNoc   string   `xml:"codAgNoc"`             // Tabela 24: "09.01.001" (ausência) ou código com pontos
	DscAgNoc   string   `xml:"dscAgNoc,omitempty"`   // Obrigatório para certos códigos genéricos
	TpAval     *int     `xml:"tpAval,omitempty"`     // 1 - Quantitativo, 2 - Qualitativo (obrigatório se diferente de 09.01.001)
	IntConc    string   `xml:"intConc,omitempty"`    // Intensidade ou concentração (se tpAval=1)
	LimTol     string   `xml:"limTol,omitempty"`     // Limite de tolerância (se aplicável)
	UnMed      *int     `xml:"unMed,omitempty"`      // Unidade de medida (1..30)
	TecMedicao string   `xml:"tecMedicao,omitempty"` // Técnica utilizada na medição
	NrProcJud  string   `xml:"nrProcJud,omitempty"`  // Processo judicial
	EpcEpi     *EpcEpi  `xml:"epcEpi,omitempty"`     // Informações de EPC e EPI
}

// EpcEpi detalha medidas de proteção coletiva e individual.
type EpcEpi struct {
	UtilizEPC int       `xml:"utilizEPC"`           // 0 - Não se aplica, 1 - Não implementa, 2 - Implementa
	EficEpc   string    `xml:"eficEpc,omitempty"`   // S, N (obrigatório se utilizEPC=2)
	UtilizEPI int       `xml:"utilizEPI"`           // 0 - Não se aplica, 1 - Não utilizado, 2 - Utilizado
	EficEpi   string    `xml:"eficEpi,omitempty"`   // S, N (obrigatório se utilizEPI=2)
	Epi       []EpiItem `xml:"epi,omitempty"`       // Lista de EPIs com Certificado de Aprovação (CA)
	EpiCompl  *EpiCompl `xml:"epiCompl,omitempty"`  // Requisitos NR-06 / NR-09
}

// EpiItem identifica o EPI pelo Certificado de Aprovação (CA).
type EpiItem struct {
	DocAval string `xml:"docAval"` // Certificado de Aprovação (CA) ou doc de avaliação do EPI
}

// EpiCompl contém as 6 perguntas obrigatórias das NRs 06 e 09 quando há uso de EPI eficaz.
type EpiCompl struct {
	MedProtecao   string `xml:"medProtecao"`   // S ou N: Tentadas medidas de proteção coletiva/administrativa?
	CondFuncto    string `xml:"condFuncto"`    // S ou N: Observadas condições de funcionamento ao longo do tempo?
	UsoInint      string `xml:"usoInint"`      // S ou N: Observado o uso ininterrupto do EPI?
	PrzValid      string `xml:"przValid"`      // S ou N: Observado o prazo de validade do CA na compra?
	PeriodicTroca string `xml:"periodicTroca"` // S ou N: Observada periodicidade de troca com recibo assinado?
	Higienizacao  string `xml:"higienizacao"`  // S ou N: Observada a higienização conforme orientação técnica?
}

// RespReg identifica o médico ou engenheiro de segurança responsável pelos registros ambientais.
type RespReg struct {
	CpfResp string `xml:"cpfResp"`           // CPF do responsável técnico
	IdeOC   *int   `xml:"ideOC,omitempty"`   // 1 - CRM, 4 - CREA, 9 - Outros
	DscOC   string `xml:"dscOC,omitempty"`   // Sigla caso ideOC=9
	NrOC    string `xml:"nrOC,omitempty"`    // Número do registro no conselho de classe
	UfOC    string `xml:"ufOC,omitempty"`    // Sigla da UF do órgão de classe
}

// ObsS2240 contém anotações adicionais relativas aos registros ambientais.
type ObsS2240 struct {
	ObsCompl string `xml:"obsCompl"`
}

// GerarXMLS2240 serializa o evento S-2240 para XML no leiaute estrito S-1.3.
func GerarXMLS2240(evento *EventoS2240) ([]byte, error) {
	if evento == nil {
		return nil, fmt.Errorf("evento S-2240 não pode ser nulo")
	}
	evento.Xmlns = NamespaceS2240
	return GerarXMLEvento(evento)
}
