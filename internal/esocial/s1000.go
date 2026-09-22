package esocial

import (
	"encoding/xml"
	"fmt"
)

const NamespaceS1000 = "http://www.esocial.gov.br/schema/evt/evtInfoEmpregador/v_S_01_03_00"

// EventoS1000 representa o envelope raiz do evento S-1000 (Informações do Empregador/Contribuinte).
type EventoS1000 struct {
	XMLName           xml.Name          `xml:"eSocial"`
	Xmlns             string            `xml:"xmlns,attr"`
	EvtInfoEmpregador EvtInfoEmpregador `xml:"evtInfoEmpregador"`
}

// EvtInfoEmpregador contém o identificador e os dados cadastrais do empregador.
type EvtInfoEmpregador struct {
	Id             string             `xml:"Id,attr"`
	IdeEvento      IdeEventoTab       `xml:"ideEvento"`
	IdeEmpregador  IdeEmpregador      `xml:"ideEmpregador"`
	InfoEmpregador InfoEmpregadorS1000 `xml:"infoEmpregador"`
}

// InfoEmpregadorS1000 contempla as operações de inclusão, alteração ou exclusão cadastral.
type InfoEmpregadorS1000 struct {
	Inclusao  *InclusaoS1000  `xml:"inclusao,omitempty"`
	Alteracao *AlteracaoS1000 `xml:"alteracao,omitempty"`
	Exclusao  *ExclusaoS1000  `xml:"exclusao,omitempty"`
}

// InclusaoS1000 detalha a inclusão inicial de dados cadastrais no eSocial.
type InclusaoS1000 struct {
	IdePeriodo   IdePeriodo         `xml:"idePeriodo"`
	InfoCadastro InfoCadastroS1000   `xml:"infoCadastro"`
}

// AlteracaoS1000 detalha a alteração de dados cadastrais e eventual nova validade.
type AlteracaoS1000 struct {
	IdePeriodo   IdePeriodo        `xml:"idePeriodo"`
	InfoCadastro InfoCadastroS1000  `xml:"infoCadastro"`
	NovaValidade *IdePeriodo        `xml:"novaValidade,omitempty"`
}

// ExclusaoS1000 detalha a exclusão de um período de validade do cadastro.
type ExclusaoS1000 struct {
	IdePeriodo IdePeriodo `xml:"idePeriodo"`
}

// IdePeriodo define o início e fim da vigência das informações (formato AAAA-MM).
type IdePeriodo struct {
	IniValid string `xml:"iniValid"`
	FimValid string `xml:"fimValid,omitempty"`
}

// InfoCadastroS1000 engloba as definições tributárias, porte e regras de escrituração da empresa.
type InfoCadastroS1000 struct {
	ClassTrib           string                `xml:"classTrib"` // Tabela 08 (ex: "01", "02", "03", "99")
	IndCoop             *int                  `xml:"indCoop,omitempty"` // 0=Não é, 1=Trabalho, 2=Produção, 3=Outras
	IndConstr           *int                  `xml:"indConstr,omitempty"` // 0=Não é, 1=Construtora
	IndDesFolha         int                   `xml:"indDesFolha"` // 0=Não aplicável, 1=Desonerada Lei 12.546/2011, 2=Município
	IndOpcCP            *int                  `xml:"indOpcCP,omitempty"` // 1=Comercialização, 2=Folha de pagamento
	IndPorte            string                `xml:"indPorte,omitempty"` // S=ME/EPP, N=Demais
	IndOptRegEletron    int                   `xml:"indOptRegEletron"` // 0=Não optou, 1=Optou pelo registro eletrônico de empregados
	CnpjEFR             string                `xml:"cnpjEFR,omitempty"` // CNPJ do Ente Federativo Responsável
	DadosIsencao        *DadosIsencaoS1000    `xml:"dadosIsencao,omitempty"`
	InfoOrgInternacional *InfoOrgInternacional `xml:"infoOrgInternacional,omitempty"`
}

// DadosIsencaoS1000 detalha informações de entidades beneficentes de assistência social isentas.
type DadosIsencaoS1000 struct {
	IdeMinLei    int    `xml:"ideMinLei"`
	NrCertif     string `xml:"nrCertif"`
	DtEmisCertif string `xml:"dtEmisCertif"` // AAAA-MM-DD
	DtVencCertif string `xml:"dtVencCertif"` // AAAA-MM-DD
	NrProtRenov  string `xml:"nrProtRenov,omitempty"`
	DtProtRenov  string `xml:"dtProtRenov,omitempty"`
	DtDou        string `xml:"dtDou,omitempty"`
	PagDou       int    `xml:"pagDou,omitempty"`
}

// InfoOrgInternacional para organismos internacionais em acordo de isenção de multa.
type InfoOrgInternacional struct {
	IndAcordoIsenMulta int `xml:"indAcordoIsenMulta"` // 0=Sem acordo, 1=Com acordo
}

// NovoEventoS1000Inclusao cria uma estrutura pronta para inclusão do evento S-1000 com dados essenciais.
func NovoEventoS1000Inclusao(
	id string,
	tpAmb int,
	tpInsc int,
	nrInsc string,
	iniValid string,
	classTrib string,
	indDesFolha int,
	indOptRegEletron int,
) *EventoS1000 {
	docLimpo := LimparDocumento(nrInsc)
	// Para S-1000 o nrInsc é a raiz do CNPJ (8 posições) ou CPF completo (11 posições) ou CNPJ completo (14 posições)
	if tpInsc == TpInscCNPJ && len(docLimpo) > 14 {
		docLimpo = docLimpo[:14]
	}

	zeroCoop := 0
	zeroConstr := 0

	return &EventoS1000{
		Xmlns: NamespaceS1000,
		EvtInfoEmpregador: EvtInfoEmpregador{
			Id: id,
			IdeEvento: IdeEventoTab{
				TpAmb:   tpAmb,
				ProcEmi: ProcEmiAppEmpregador,
				VerProc: VersaoAplicativo,
			},
			IdeEmpregador: IdeEmpregador{
				TpInsc: tpInsc,
				NrInsc: docLimpo,
			},
			InfoEmpregador: InfoEmpregadorS1000{
				Inclusao: &InclusaoS1000{
					IdePeriodo: IdePeriodo{
						IniValid: iniValid,
					},
					InfoCadastro: InfoCadastroS1000{
						ClassTrib:        classTrib,
						IndCoop:          &zeroCoop,
						IndConstr:        &zeroConstr,
						IndDesFolha:      indDesFolha,
						IndOptRegEletron: indOptRegEletron,
					},
				},
			},
		},
	}
}

// GerarXMLS1000 serializa o evento S-1000 para XML estritamente de acordo com o leiaute S-1.3.
func GerarXMLS1000(evento *EventoS1000) ([]byte, error) {
	if evento == nil {
		return nil, fmt.Errorf("evento S-1000 não pode ser nulo")
	}
	evento.Xmlns = NamespaceS1000
	return GerarXMLEvento(evento)
}
