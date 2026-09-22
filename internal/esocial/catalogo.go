package esocial

import (
	"strings"
	"sync"
)

// Grupos de responsabilidade técnica no eSocial
const (
	GrupoSESMT              = "SESMT"
	GrupoClinicaOcupacional = "Clínica Ocupacional"
	GrupoRHDP               = "RH / DP"
	GrupoContabilidadeFolha = "Contabilidade e Folha"
	GrupoJuridico           = "Jurídico"
	GrupoRPPSSetorPublico   = "RPPS / Setor Público"
)

// Categorias de eventos no eSocial
const (
	CategoriaTabelas       = "Tabelas"
	CategoriaNaoPeriodicos = "Não Periódicos"
	CategoriaPeriodicos    = "Periódicos"
)

// GrupoResponsabilidade representa o grupo técnico de competência.
type GrupoResponsabilidade = string

// EventoCatalogo reúne os metadados técnicos, operacionais e o template XML
// de um evento oficial do leiaute S-1.3 do eSocial.
type EventoCatalogo struct {
	Codigo      string `json:"codigo"`       // Ex: "S-1000", "S-2240"
	Nome        string `json:"nome"`         // Nome oficial do MOS
	Categoria   string `json:"categoria"`    // "Tabelas", "Não Periódicos", "Periódicos"
	Grupo       string `json:"grupo"`        // Grupo de responsabilidade
	Responsavel string `json:"responsavel"`  // Cargo/Papel do responsável
	Descricao   string `json:"descricao"`    // Propósito legal
	Prazo       string `json:"prazo"`        // Regra de prazo oficial S-1.3
	TagRaiz     string `json:"tag_raiz"`     // Tag principal do evento
	Namespace   string `json:"namespace"`    // Namespace oficial S-1.3
	TemplateXML string `json:"template_xml"` // Template esqueleto XML
}

var (
	catalogoOnce sync.Once
	catalogo     []EventoCatalogo
	mapaCatalogo map[string]EventoCatalogo
)

func inicializarCatalogo() {
	catalogo = []EventoCatalogo{
		// =========================================================================
		// GRUPO 1: SESMT
		// =========================================================================
		{
			Codigo:      "S-2240",
			Nome:        "Condições Ambientais do Trabalho - Agentes Nocivos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoSESMT,
			Responsavel: "Engenheiro de Segurança do Trabalho / TST",
			Descricao:   "Informa as condições de prestação de serviços pelo trabalhador, ambientes, atividades e exposição a agentes nocivos da Tabela 24 para aposentadoria especial.",
			Prazo:       "Até o dia 15 do mês subsequente ao início da atividade ou de qualquer alteração nas condições ambientais.",
			TagRaiz:     "evtExpRisco",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00">
  <evtExpRisco Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <infoExpRisco>
      <dtIniCondicao>{{DT_INI_CONDICAO}}</dtIniCondicao>
      <infoAmb>
        <localAmb>{{LOCAL_AMB}}</localAmb>
        <dscSetor>{{DSC_SETOR}}</dscSetor>
        <tpInsc>{{TP_INSC_AMB}}</tpInsc>
        <nrInsc>{{NR_INSC_AMB}}</nrInsc>
      </infoAmb>
      <infoAtiv>
        <dscAtivDes>{{DSC_ATIV_DES}}</dscAtivDes>
      </infoAtiv>
      <agNoc>
        <codAgNoc>{{COD_AG_NOC}}</codAgNoc>
      </agNoc>
      <respReg>
        <cpfResp>{{CPF_RESP}}</cpfResp>
      </respReg>
    </infoExpRisco>
  </evtExpRisco>
</eSocial>`,
		},

		// =========================================================================
		// GRUPO 2: CLÍNICA OCUPACIONAL
		// =========================================================================
		{
			Codigo:      "S-2210",
			Nome:        "Comunicação de Acidente de Trabalho (CAT)",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoClinicaOcupacional,
			Responsavel: "Médico do Trabalho / Clínica de Medicina Ocupacional",
			Descricao:   "Comunica a ocorrência de acidente de trabalho, de trajeto ou doença profissional, com detalhes da lesão e atestado médico emitido.",
			Prazo:       "Até o primeiro dia útil seguinte ao da ocorrência e, em caso de morte, de imediato.",
			TagRaiz:     "evtCAT",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00">
  <evtCAT Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <cat>
      <dtAcid>{{DT_ACID}}</dtAcid>
      <tpAcid>{{TP_ACID}}</tpAcid>
      <hrAcid>{{HR_ACID}}</hrAcid>
      <tpCat>{{TP_CAT}}</tpCat>
      <indCatObito>{{IND_CAT_OBITO}}</indCatObito>
      <indComunPolicia>{{IND_COMUN_POLICIA}}</indComunPolicia>
      <codSitGeradora>{{COD_SIT_GERADORA}}</codSitGeradora>
      <iniciatCAT>{{INICIAT_CAT}}</iniciatCAT>
      <localAcidente>
        <tpLocal>{{TP_LOCAL}}</tpLocal>
        <dscLograd>{{DSC_LOGRAD}}</dscLograd>
        <nrLograd>{{NR_LOGRAD}}</nrLograd>
      </localAcidente>
      <parteAtingida>
        <codParteAting>{{COD_PARTE_ATING}}</codParteAting>
        <lateralidade>{{LATERALIDADE}}</lateralidade>
      </parteAtingida>
      <agenteCausador>
        <codAgntCausador>{{COD_AGNT_CAUSADOR}}</codAgntCausador>
      </agenteCausador>
      <atestado>
        <dtAtendimento>{{DT_ATENDIMENTO}}</dtAtendimento>
        <hrAtendimento>{{HR_ATENDIMENTO}}</hrAtendimento>
        <indInternacao>{{IND_INTERNACAO}}</indInternacao>
        <durTrat>{{DUR_TRAT}}</durTrat>
        <indAfast>{{IND_AFAST}}</indAfast>
        <dscLesao>{{DSC_LESAO}}</dscLesao>
        <codCID>{{COD_CID}}</codCID>
        <emitente>
          <nmEmit>{{NM_EMIT}}</nmEmit>
          <ideOC>{{IDE_OC}}</ideOC>
          <nrOC>{{NR_OC}}</nrOC>
          <ufOC>{{UF_OC}}</ufOC>
        </emitente>
      </atestado>
    </cat>
  </evtCAT>
</eSocial>`,
		},
		{
			Codigo:      "S-2220",
			Nome:        "Monitoramento da Saúde do Trabalhador (ASO)",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoClinicaOcupacional,
			Responsavel: "Médico do Trabalho / Examinador",
			Descricao:   "Registra os exames médicos ocupacionais (admissional, periódico, retorno, mudança de risco e demissional) e os exames complementares do PCMSO.",
			Prazo:       "Até o dia 15 do mês subsequente à emissão do respectivo Atestado de Saúde Ocupacional (ASO).",
			TagRaiz:     "evtMonit",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00">
  <evtMonit Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <exMedOcup>
      <tpExameOcup>{{TP_EXAME_OCUP}}</tpExameOcup>
      <aso>
        <dtAso>{{DT_ASO}}</dtAso>
        <resAso>{{RES_ASO}}</resAso>
        <medico>
          <nmMed>{{NM_MED}}</nmMed>
          <nrCRM>{{NR_CRM}}</nrCRM>
          <ufCRM>{{UF_CRM}}</ufCRM>
        </medico>
      </aso>
    </exMedOcup>
  </evtMonit>
</eSocial>`,
		},

		// =========================================================================
		// GRUPO 3: RH / DP
		// =========================================================================
		{
			Codigo:      "S-2190",
			Nome:        "Registro Preliminar de Trabalhador",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Registro simplificado e prévio para garantir o cumprimento do prazo de admissão quando faltam dados completos do funcionário.",
			Prazo:       "Até o final do dia imediatamente anterior ao início da prestação dos serviços.",
			TagRaiz:     "evtAdmPrelim",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtAdmPrelim/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAdmPrelim/v_S_01_03_00">
  <evtAdmPrelim Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoRegPrelim>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <dtNascto>{{DT_NASCTO}}</dtNascto>
      <dtAdm>{{DT_ADM}}</dtAdm>
      <matricula>{{MATRICULA}}</matricula>
      <codCateg>{{COD_CATEG}}</codCateg>
      <natAtividade>{{NAT_ATIVIDADE}}</natAtividade>
    </infoRegPrelim>
  </evtAdmPrelim>
</eSocial>`,
		},
		{
			Codigo:      "S-2200",
			Nome:        "Cadastramento Inicial do Vínculo e Admissão/Ingresso de Trabalhador",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Formaliza a admissão completa do empregado com todos os dados pessoais, endereço, contrato, remuneração, cargo, CBO e horário.",
			Prazo:       "Até o dia anterior ao início da prestação de serviços ou até o dia 15 do mês subsequente caso tenha havido S-2190 prévio.",
			TagRaiz:     "evtAdmissao",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtAdmissao/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAdmissao/v_S_01_03_00">
  <evtAdmissao Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <trabalhador>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <nmTrab>{{NM_TRAB}}</nmTrab>
      <sexo>{{SEXO}}</sexo>
      <racaCor>{{RACA_COR}}</racaCor>
      <estCiv>{{EST_CIV}}</estCiv>
      <grauInstr>{{GRAU_INSTR}}</grauInstr>
      <dtNascto>{{DT_NASCTO}}</dtNascto>
    </trabalhador>
    <vinculo>
      <matricula>{{MATRICULA}}</matricula>
      <tpRegTrab>{{TP_REG_TRAB}}</tpRegTrab>
      <tpRegPrev>{{TP_REG_PREV}}</tpRegPrev>
      <cadIni>N</cadIni>
      <infoRegimeTrab>
        <infoCLT>
          <dtAdm>{{DT_ADM}}</dtAdm>
          <tpAdmissao>1</tpAdmissao>
          <indAdmissao>1</indAdmissao>
          <tpRegJor>1</tpRegJor>
          <natAtividade>1</natAtividade>
        </infoCLT>
      </infoRegimeTrab>
      <infoContrato>
        <codCargo>{{COD_CARGO}}</codCargo>
        <codCBO>{{COD_CBO}}</codCBO>
      </infoContrato>
    </vinculo>
  </evtAdmissao>
</eSocial>`,
		},
		{
			Codigo:      "S-2205",
			Nome:        "Alteração de Dados Cadastrais do Trabalhador",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Comunica alteração em dados pessoais do trabalhador como nome civil/social, estado civil, endereço ou escolaridade.",
			Prazo:       "Até o dia 15 do mês subsequente à ocorrência da alteração.",
			TagRaiz:     "evtAltCadastral",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtAltCadastral/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAltCadastral/v_S_01_03_00">
  <evtAltCadastral Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideTrabalhador>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
    </ideTrabalhador>
    <alteracao>
      <dtAlteracao>{{DT_ALTERACAO}}</dtAlteracao>
      <dadosTrabalhador>
        <nmTrab>{{NM_TRAB}}</nmTrab>
        <sexo>{{SEXO}}</sexo>
        <racaCor>{{RACA_COR}}</racaCor>
        <estCiv>{{EST_CIV}}</estCiv>
        <grauInstr>{{GRAU_INSTR}}</grauInstr>
      </dadosTrabalhador>
    </alteracao>
  </evtAltCadastral>
</eSocial>`,
		},
		{
			Codigo:      "S-2206",
			Nome:        "Alteração de Contrato de Trabalho / Relação Estatutária",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Comunica alterações contratuais como mudança de cargo, CBO, salário, jornada de trabalho, local de trabalho ou setor.",
			Prazo:       "Até o dia 15 do mês subsequente ao da ocorrência da alteração contratual.",
			TagRaiz:     "evtAltContratual",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtAltContratual/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAltContratual/v_S_01_03_00">
  <evtAltContratual Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <altContratual>
      <dtAlteracao>{{DT_ALTERACAO}}</dtAlteracao>
      <infoContrato>
        <codCargo>{{COD_CARGO}}</codCargo>
        <codCBO>{{COD_CBO}}</codCBO>
      </infoContrato>
    </altContratual>
  </evtAltContratual>
</eSocial>`,
		},
		{
			Codigo:      "S-2230",
			Nome:        "Afastamento Temporário",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Informa o início e término de afastamentos temporários dos colaboradores (auxílio por incapacidade temporária, licença-maternidade, férias, etc.).",
			Prazo:       "Até o dia 15 do mês subsequente à ocorrência do afastamento ou término.",
			TagRaiz:     "evtAfastTemp",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtAfastTemp/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAfastTemp/v_S_01_03_00">
  <evtAfastTemp Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <infoAfastamento>
      <iniAfastamento>
        <dtIniAfast>{{DT_INI_AFAST}}</dtIniAfast>
        <codMotAfast>{{COD_MOT_AFAST}}</codMotAfast>
      </iniAfastamento>
    </infoAfastamento>
  </evtAfastTemp>
</eSocial>`,
		},
		{
			Codigo:      "S-2231",
			Nome:        "Cessão / Exercício em Outro Órgão",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Informa o início e término de cessão ou exercício de trabalhador em outro órgão ou entidade pública.",
			Prazo:       "Até o dia 15 do mês subsequente à data da cessão ou do retorno.",
			TagRaiz:     "evtCessao",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCessao/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCessao/v_S_01_03_00">
  <evtCessao Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <infoCessao>
      <iniCessao>
        <dtIniCessao>{{DT_INI_CESSAO}}</dtIniCessao>
        <cnpjCess>{{CNPJ_CESS}}</cnpjCess>
        <respRemun>{{RESP_REMUN}}</respRemun>
      </iniCessao>
    </infoCessao>
  </evtCessao>
</eSocial>`,
		},
		{
			Codigo:      "S-2298",
			Nome:        "Reintegração / Outros Provimentos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Informa a reintegração de empregado ao serviço decorrente de decisão judicial, administrativa ou anistia legal.",
			Prazo:       "Até o dia 15 do mês seguinte ao do retorno efetivo do trabalhador.",
			TagRaiz:     "evtReintegr",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtReintegr/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtReintegr/v_S_01_03_00">
  <evtReintegr Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <infoReintegr>
      <tpReint>{{TP_REINT}}</tpReint>
      <dtEfetRetorno>{{DT_EFET_RETORNO}}</dtEfetRetorno>
      <dtEfeito>{{DT_EFEITO}}</dtEfeito>
    </infoReintegr>
  </evtReintegr>
</eSocial>`,
		},
		{
			Codigo:      "S-2299",
			Nome:        "Desligamento",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Encerra o vínculo de trabalho do empregado comunicando a data, motivo da rescisão, aviso prévio e verbas rescisórias.",
			Prazo:       "Até 10 dias seguintes ao do término do contrato ou antes do envio da folha de pagamento do mês.",
			TagRaiz:     "evtDeslig",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtDeslig/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtDeslig/v_S_01_03_00">
  <evtDeslig Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideVinculo>
    <infoDeslig>
      <mtvDeslig>{{MTV_DESLIG}}</mtvDeslig>
      <dtDeslig>{{DT_DESLIG}}</dtDeslig>
      <indPagtoAPI>N</indPagtoAPI>
    </infoDeslig>
  </evtDeslig>
</eSocial>`,
		},
		{
			Codigo:      "S-2300",
			Nome:        "Início de Trabalhador Sem Vínculo de Emprego/Estatutário (TSVE)",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Registra o início de atividades de trabalhadores sem vínculo empregatício como estagiários, diretores não empregados, avulsos e cooperados.",
			Prazo:       "Até o dia 15 do mês subsequente ao do início da prestação de serviços.",
			TagRaiz:     "evtTSVEInicio",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTSVEInicio/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTSVEInicio/v_S_01_03_00">
  <evtTSVEInicio Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <trabalhador>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <nmTrab>{{NM_TRAB}}</nmTrab>
      <dtNascto>{{DT_NASCTO}}</dtNascto>
    </trabalhador>
    <infoTSVEInicio>
      <cadIni>N</cadIni>
      <matricula>{{MATRICULA}}</matricula>
      <codCateg>{{COD_CATEG}}</codCateg>
      <dtInicio>{{DT_INICIO}}</dtInicio>
    </infoTSVEInicio>
  </evtTSVEInicio>
</eSocial>`,
		},
		{
			Codigo:      "S-2306",
			Nome:        "Trabalhador Sem Vínculo de Emprego/Estatutário - Alteração Contratual",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Comunica alterações de informações contratuais do trabalhador sem vínculo (TSVE).",
			Prazo:       "Até o dia 15 do mês subsequente ao da ocorrência da alteração contratual.",
			TagRaiz:     "evtTSVEAltContr",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTSVEAltContr/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTSVEAltContr/v_S_01_03_00">
  <evtTSVEAltContr Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideTrabSemVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideTrabSemVinculo>
    <infoTSVEAlteracao>
      <dtAlteracao>{{DT_ALTERACAO}}</dtAlteracao>
    </infoTSVEAlteracao>
  </evtTSVEAltContr>
</eSocial>`,
		},
		{
			Codigo:      "S-2399",
			Nome:        "Trabalhador Sem Vínculo de Emprego/Estatutário - Término",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Comunica o término das atividades de trabalhador sem vínculo de emprego ou estatutário (ex: fim de estágio).",
			Prazo:       "Até o dia 15 do mês seguinte ao término das atividades do TSVE.",
			TagRaiz:     "evtTSVETermino",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTSVETermino/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTSVETermino/v_S_01_03_00">
  <evtTSVETermino Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideTrabSemVinculo>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <matricula>{{MATRICULA}}</matricula>
    </ideTrabSemVinculo>
    <infoTSVETermino>
      <dtTerm>{{DT_TERM}}</dtTerm>
    </infoTSVETermino>
  </evtTSVETermino>
</eSocial>`,
		},
		{
			Codigo:      "S-3000",
			Nome:        "Exclusão de Eventos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRHDP,
			Responsavel: "Analista de DP / RH",
			Descricao:   "Torna sem efeito e exclui do repositório nacional do eSocial um evento enviado indevidamente por engano.",
			Prazo:       "A qualquer momento em que seja constatada a necessidade de cancelar o evento anterior.",
			TagRaiz:     "evtExclusao",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtExclusao/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExclusao/v_S_01_03_00">
  <evtExclusao Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoExclusao>
      <tpEvento>{{TP_EVENTO_EXCLUIR}}</tpEvento>
      <nrRecEvt>{{NR_REC_EVT}}</nrRecEvt>
      <ideTrabalhador>
        <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      </ideTrabalhador>
    </infoExclusao>
  </evtExclusao>
</eSocial>`,
		},

		// =========================================================================
		// GRUPO 4: CONTABILIDADE E FOLHA
		// =========================================================================
		{
			Codigo:      "S-1000",
			Nome:        "Informações do Empregador / Contribuinte / Órgão Público",
			Categoria:   CategoriaTabelas,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Cadastra o contribuinte e define a classificação tributária, desoneração da folha, cooperativas e registro eletrônico.",
			Prazo:       "Antes de qualquer outro evento do eSocial e até o dia 15 do mês subsequente à alteração.",
			TagRaiz:     "evtInfoEmpregador",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtInfoEmpregador/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtInfoEmpregador/v_S_01_03_00">
  <evtInfoEmpregador Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoEmpregador>
      <inclusao>
        <idePeriodo>
          <iniValid>{{INI_VALID}}</iniValid>
        </idePeriodo>
        <infoCadastro>
          <classTrib>{{CLASS_TRIB}}</classTrib>
          <indDesFolha>{{IND_DES_FOLHA}}</indDesFolha>
          <indOptRegEletron>{{IND_OPT_REG_ELETRON}}</indOptRegEletron>
        </infoCadastro>
      </inclusao>
    </infoEmpregador>
  </evtInfoEmpregador>
</eSocial>`,
		},
		{
			Codigo:      "S-1005",
			Nome:        "Tabela de Estabelecimentos, Obras ou Unidades de Órgãos Públicos",
			Categoria:   CategoriaTabelas,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Cadastra matriz, filiais, obras de construção civil e unidades vinculadas com CNAE preponderante, alíquota GILRAT e FAP.",
			Prazo:       "Antes de enviar eventos que referenciem o estabelecimento e até o dia 15 do mês seguinte ao de início da atividade.",
			TagRaiz:     "evtTabEstab",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTabEstab/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTabEstab/v_S_01_03_00">
  <evtTabEstab Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoEstab>
      <inclusao>
        <ideEstab>
          <tpInsc>{{TP_INSC_ESTAB}}</tpInsc>
          <nrInsc>{{NR_INSC_ESTAB}}</nrInsc>
          <iniValid>{{INI_VALID}}</iniValid>
        </ideEstab>
        <dadosEstab>
          <cnaePrep>{{CNAE_PREP}}</cnaePrep>
        </dadosEstab>
      </inclusao>
    </infoEstab>
  </evtTabEstab>
</eSocial>`,
		},
		{
			Codigo:      "S-1010",
			Nome:        "Tabela de Rubricas",
			Categoria:   CategoriaTabelas,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Cadastra todas as verbas da folha de pagamento com incidência tributária para INSS, IRRF e FGTS.",
			Prazo:       "Antes do envio dos eventos periódicos de remuneração (S-1200) e desligamento (S-2299).",
			TagRaiz:     "evtTabRubrica",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTabRubrica/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTabRubrica/v_S_01_03_00">
  <evtTabRubrica Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoRubrica>
      <inclusao>
        <ideRubrica>
          <codRubr>{{COD_RUBR}}</codRubr>
          <ideTabRubr>{{IDE_TAB_RUBR}}</ideTabRubr>
          <iniValid>{{INI_VALID}}</iniValid>
        </ideRubrica>
        <dadosRubrica>
          <dscRubr>{{DSC_RUBR}}</dscRubr>
          <natRubr>{{NAT_RUBR}}</natRubr>
          <tpRubr>{{TP_RUBR}}</tpRubr>
          <codIncCP>{{COD_INC_CP}}</codIncCP>
          <codIncIRRF>{{COD_INC_IRRF}}</codIncIRRF>
          <codIncFGTS>{{COD_INC_FGTS}}</codIncFGTS>
        </dadosRubrica>
      </inclusao>
    </infoRubrica>
  </evtTabRubrica>
</eSocial>`,
		},
		{
			Codigo:      "S-1020",
			Nome:        "Tabela de Lotações Tributárias",
			Categoria:   CategoriaTabelas,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Classifica as lotações tributárias da empresa e seus códigos FPAS e Terceiros/Outras Entidades.",
			Prazo:       "Antes do envio de qualquer evento de remuneração de trabalhadores vinculados à lotação.",
			TagRaiz:     "evtTabLotacao",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTabLotacao/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTabLotacao/v_S_01_03_00">
  <evtTabLotacao Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoLotacao>
      <inclusao>
        <ideLotacao>
          <codLotacao>{{COD_LOTACAO}}</codLotacao>
          <iniValid>{{INI_VALID}}</iniValid>
        </ideLotacao>
        <dadosLotacao>
          <tpLotacao>{{TP_LOTACAO}}</tpLotacao>
          <fpasLotacao>
            <fpas>{{FPAS}}</fpas>
            <codTercs>{{COD_TERCS}}</codTercs>
          </fpasLotacao>
        </dadosLotacao>
      </inclusao>
    </infoLotacao>
  </evtTabLotacao>
</eSocial>`,
		},
		{
			Codigo:      "S-1200",
			Nome:        "Remuneração de Trabalhador vinculado ao RGPS",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista de Folha de Pagamento",
			Descricao:   "Contém todas as rubricas salariais e bases de cálculo da remuneração de cada trabalhador no mês de competência.",
			Prazo:       "Até o dia 15 do mês seguinte ao do período de apuração (ou dia útil anterior).",
			TagRaiz:     "evtRemun",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtRemun/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtRemun/v_S_01_03_00">
  <evtRemun Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideTrabalhador>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <dmDev>
        <ideDmDev>{{IDE_DM_DEV}}</ideDmDev>
        <codCateg>{{COD_CATEG}}</codCateg>
      </dmDev>
    </ideTrabalhador>
  </evtRemun>
</eSocial>`,
		},
		{
			Codigo:      "S-1210",
			Nome:        "Pagamentos de Rendimentos do Trabalho",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Informa a data efetiva de pagamento e o valor líquido pago ao trabalhador para apuração do Imposto de Renda Retido na Fonte (IRRF).",
			Prazo:       "Até o dia 15 do mês subsequente ao mês em que o pagamento foi efetivamente realizado.",
			TagRaiz:     "evtPgtos",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtPgtos/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtPgtos/v_S_01_03_00">
  <evtPgtos Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideBenef>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
      <infoPgto>
        <dtPgto>{{DT_PGTO}}</dtPgto>
        <tpPgto>{{TP_PGTO}}</tpPgto>
        <perRef>{{PER_REF}}</perRef>
        <ideDmDev>{{IDE_DM_DEV}}</ideDmDev>
        <vrLiq>{{VR_LIQ}}</vrLiq>
      </infoPgto>
    </ideBenef>
  </evtPgtos>
</eSocial>`,
		},
		{
			Codigo:      "S-1260",
			Nome:        "Comercialização da Produção Rural Pessoa Física",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal",
			Descricao:   "Prestação de informações sobre a comercialização da produção rural realizada por produtor rural pessoa física ou segurado especial.",
			Prazo:       "Até o dia 15 do mês subsequente ao da comercialização da produção.",
			TagRaiz:     "evtComProd",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtComProd/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtComProd/v_S_01_03_00">
  <evtComProd Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoComProd>
      <ideEstab>
        <nrInscEstabRef>{{NR_INSC_ESTAB}}</nrInscEstabRef>
      </ideEstab>
    </infoComProd>
  </evtComProd>
</eSocial>`,
		},
		{
			Codigo:      "S-1270",
			Nome:        "Contratação de Trabalhadores Avulsos Não Portuários",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista de Folha",
			Descricao:   "Informa a remuneração paga a trabalhadores avulsos não portuários contratados por intermédio de sindicato.",
			Prazo:       "Até o dia 15 do mês subsequente ao da prestação dos serviços.",
			TagRaiz:     "evtContratAvNP",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtContratAvNP/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtContratAvNP/v_S_01_03_00">
  <evtContratAvNP Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <remunAvNP>
      <tpInsc>{{TP_INSC_ESTAB}}</tpInsc>
      <nrInsc>{{NR_INSC_ESTAB}}</nrInsc>
      <codLotacao>{{COD_LOTACAO}}</codLotacao>
      <vrBcCp00>{{VR_BC_CP}}</vrBcCp00>
    </remunAvNP>
  </evtContratAvNP>
</eSocial>`,
		},
		{
			Codigo:      "S-1280",
			Nome:        "Informações Complementares aos Eventos Periódicos",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista Fiscal de Folha",
			Descricao:   "Prestação de informações sobre desoneração da folha de pagamento e percentual de isenção de empresas do Simples Concomitantes.",
			Prazo:       "Até o dia 15 do mês seguinte ao do período de apuração.",
			TagRaiz:     "evtInfoComplPer",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtInfoComplPer/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtInfoComplPer/v_S_01_03_00">
  <evtInfoComplPer Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoSubstPatr>
      <indSubstPatr>{{IND_SUBST_PATR}}</indSubstPatr>
      <percRedContrib>{{PERC_RED_CONTRIB}}</percRedContrib>
    </infoSubstPatr>
  </evtInfoComplPer>
</eSocial>`,
		},
		{
			Codigo:      "S-1298",
			Nome:        "Reabertura dos Eventos Periódicos",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista de Folha",
			Descricao:   "Reabre a folha de pagamento de determinado mês já encerrado para permitir retificações de remunerações ou pagamentos.",
			Prazo:       "Sempre que for necessário retificar ou incluir informações na folha já encerrada pelo S-1299.",
			TagRaiz:     "evtReabreEvPer",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtReabreEvPer/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtReabreEvPer/v_S_01_03_00">
  <evtReabreEvPer Id="{{ID}}">
    <ideEvento>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
  </evtReabreEvPer>
</eSocial>`,
		},
		{
			Codigo:      "S-1299",
			Nome:        "Fechamento dos Eventos Periódicos",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoContabilidadeFolha,
			Responsavel: "Contador / Analista de Folha",
			Descricao:   "Informa o encerramento da folha de pagamento do mês para consolidação das contribuições previdenciárias e geração da DCTFWeb.",
			Prazo:       "Até o dia 15 do mês seguinte ao do período de apuração.",
			TagRaiz:     "evtFechaEvPer",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtFechaEvPer/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtFechaEvPer/v_S_01_03_00">
  <evtFechaEvPer Id="{{ID}}">
    <ideEvento>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoFech>
      <evtRemun>{{TEM_REMUN}}</evtRemun>
      <evtPgtos>{{TEM_PGTOS}}</evtPgtos>
      <evtComProd>N</evtComProd>
      <evtContratAvNP>N</evtContratAvNP>
      <evtInfoComplPer>N</evtInfoComplPer>
    </infoFech>
  </evtFechaEvPer>
</eSocial>`,
		},

		// =========================================================================
		// GRUPO 5: JURÍDICO
		// =========================================================================
		{
			Codigo:      "S-1070",
			Nome:        "Tabela de Processos Administrativos / Judiciais",
			Categoria:   CategoriaTabelas,
			Grupo:       GrupoJuridico,
			Responsavel: "Advogado Trabalhista / Departamento Jurídico",
			Descricao:   "Cadastra decisões e liminares em processos judiciais ou administrativos que suspendem a exigibilidade de tributos e FGTS.",
			Prazo:       "Até o dia 15 do mês subsequente ao da contestação ou antes do envio de eventos que referenciem o processo.",
			TagRaiz:     "evtTabProcesso",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTabProcesso/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTabProcesso/v_S_01_03_00">
  <evtTabProcesso Id="{{ID}}">
    <ideEvento>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoProcesso>
      <inclusao>
        <ideProcesso>
          <tpProc>{{TP_PROC}}</tpProc>
          <nrProc>{{NR_PROC}}</nrProc>
          <iniValid>{{INI_VALID}}</iniValid>
        </ideProcesso>
        <dadosProc>
          <indAutoria>{{IND_AUTORIA}}</indAutoria>
        </dadosProc>
      </inclusao>
    </infoProcesso>
  </evtTabProcesso>
</eSocial>`,
		},
		{
			Codigo:      "S-2500",
			Nome:        "Processo Trabalhista",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoJuridico,
			Responsavel: "Advogado Trabalhista / Jurídico",
			Descricao:   "Informa os dados de processos trabalhistas transitados em julgado, acordos homologados judicialmente ou acordos perante CCP/NINTER.",
			Prazo:       "Até o dia 15 do mês subsequente ao do trânsito em julgado da decisão ou homologação do acordo.",
			TagRaiz:     "evtProcTrab",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtProcTrab/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtProcTrab/v_S_01_03_00">
  <evtProcTrab Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <infoProcesso>
      <origem>{{ORIGEM_PROC}}</origem>
      <nrProcTrab>{{NR_PROC_TRAB}}</nrProcTrab>
      <dadosCompl>
        <infoProcJud>
          <dtSent>{{DT_SENT}}</dtSent>
          <ufVara>{{UF_VARA}}</ufVara>
          <codMunic>{{COD_MUNIC}}</codMunic>
          <idVara>{{ID_VARA}}</idVara>
        </infoProcJud>
      </dadosCompl>
    </infoProcesso>
    <ideTrab>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
    </ideTrab>
  </evtProcTrab>
</eSocial>`,
		},
		{
			Codigo:      "S-2501",
			Nome:        "Informações de Tributos Decorrentes de Processo Trabalhista",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoJuridico,
			Responsavel: "Advogado Trabalhista / Jurídico",
			Descricao:   "Informa as contribuições previdenciárias e o Imposto de Renda Retido na Fonte apurados nas decisões judiciais de processos trabalhistas.",
			Prazo:       "Até o dia 15 do mês subsequente ao do pagamento da condenação/acordo judicial.",
			TagRaiz:     "evtTribProcTrab",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtTribProcTrab/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTribProcTrab/v_S_01_03_00">
  <evtTribProcTrab Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideProc>
      <nrProcTrab>{{NR_PROC_TRAB}}</nrProcTrab>
      <perRef>{{PER_REF}}</perRef>
    </ideProc>
  </evtTribProcTrab>
</eSocial>`,
		},
		{
			Codigo:      "S-2555",
			Nome:        "Solicitação de Consolidação das Informações de Tributos Decorrentes de Processo Trabalhista",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoJuridico,
			Responsavel: "Advogado Trabalhista / Jurídico",
			Descricao:   "Solicita a totalização e fechamento dos débitos tributários decorrentes de processos trabalhistas para transmissão à DCTFWeb.",
			Prazo:       "Até o dia 15 do mês subsequente ao do pagamento da obrigação.",
			TagRaiz:     "evtConsolidContProc",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtConsolidContProc/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtConsolidContProc/v_S_01_03_00">
  <evtConsolidContProc Id="{{ID}}">
    <ideEvento>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
  </evtConsolidContProc>
</eSocial>`,
		},

		// =========================================================================
		// GRUPO 6: RPPS / SETOR PÚBLICO
		// =========================================================================
		{
			Codigo:      "S-1202",
			Nome:        "Remuneração de Servidor vinculado a RPPS",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de Recursos Humanos do Setor Público",
			Descricao:   "Informa as verbas e bases de cálculo da remuneração de servidores públicos vinculados a Regime Próprio de Previdência Social.",
			Prazo:       "Até o dia 15 do mês seguinte ao do período de apuração.",
			TagRaiz:     "evtRmnRPPS",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtRmnRPPS/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtRmnRPPS/v_S_01_03_00">
  <evtRmnRPPS Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideTrabalhador>
      <cpfTrab>{{CPF_TRAB}}</cpfTrab>
      <dmDev>
        <ideDmDev>{{IDE_DM_DEV}}</ideDmDev>
        <codCateg>{{COD_CATEG}}</codCateg>
      </dmDev>
    </ideTrabalhador>
  </evtRmnRPPS>
</eSocial>`,
		},
		{
			Codigo:      "S-1207",
			Nome:        "Benefícios - Entes Públicos",
			Categoria:   CategoriaPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de RPPS / Órgão Previdenciário Público",
			Descricao:   "Registra os proventos de aposentadoria e pensão pagos a beneficiários de Regimes Próprios de Previdência Social (RPPS).",
			Prazo:       "Até o dia 15 do mês seguinte ao do período de apuração.",
			TagRaiz:     "evtBenPrRP",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtBenPrRP/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtBenPrRP/v_S_01_03_00">
  <evtBenPrRP Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <perApur>{{PER_APUR}}</perApur>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideBenef>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
      <dmDev>
        <ideDmDev>{{IDE_DM_DEV}}</ideDmDev>
        <nrBeneficio>{{NR_BENEFICIO}}</nrBeneficio>
      </dmDev>
    </ideBenef>
  </evtBenPrRP>
</eSocial>`,
		},
		{
			Codigo:      "S-2400",
			Nome:        "Cadastro de Beneficiário - Entes Públicos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de RPPS / Setor Público",
			Descricao:   "Cadastramento inicial do beneficiário de aposentadoria ou pensão gerida pelo ente público ou instituto de previdência RPPS.",
			Prazo:       "Até o dia 15 do mês subsequente ao do início do benefício ou antes do envio de eventos periódicos.",
			TagRaiz:     "evtCdBenefIn",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCdBenefIn/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCdBenefIn/v_S_01_03_00">
  <evtCdBenefIn Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <beneficiario>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
      <nmBenef>{{NM_BENEF}}</nmBenef>
      <dtNascto>{{DT_NASCTO}}</dtNascto>
    </beneficiario>
  </evtCdBenefIn>
</eSocial>`,
		},
		{
			Codigo:      "S-2405",
			Nome:        "Alteração de Dados Cadastrais do Beneficiário - Entes Públicos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de RPPS / Setor Público",
			Descricao:   "Comunica alteração cadastral de beneficiário de RPPS (nome, endereço, estado civil).",
			Prazo:       "Até o dia 15 do mês subsequente à alteração cadastral.",
			TagRaiz:     "evtCdBenefAlt",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCdBenefAlt/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCdBenefAlt/v_S_01_03_00">
  <evtCdBenefAlt Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideBeneficiario>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
    </ideBeneficiario>
    <alteracao>
      <dtAlteracao>{{DT_ALTERACAO}}</dtAlteracao>
      <dadosBeneficiario>
        <nmBenef>{{NM_BENEF}}</nmBenef>
      </dadosBeneficiario>
    </alteracao>
  </evtCdBenefAlt>
</eSocial>`,
		},
		{
			Codigo:      "S-2410",
			Nome:        "Cadastro de Benefício - Entes Públicos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de RPPS / Setor Público",
			Descricao:   "Cadastramento e concessão do benefício previdenciário (aposentadoria, pensão militar ou civil) no âmbito do RPPS.",
			Prazo:       "Até o dia 15 do mês subsequente à concessão do benefício.",
			TagRaiz:     "evtCdBenInpr",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCdBenInpr/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCdBenInpr/v_S_01_03_00">
  <evtCdBenInpr Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <beneficiario>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
      <nrBeneficio>{{NR_BENEFICIO}}</nrBeneficio>
    </beneficiario>
    <infoBenInicio>
      <cadIni>N</cadIni>
      <dtIni>{{DT_INI}}</dtIni>
      <tpBeneficio>{{TP_BENEFICIO}}</tpBeneficio>
    </infoBenInicio>
  </evtCdBenInpr>
</eSocial>`,
		},
		{
			Codigo:      "S-2416",
			Nome:        "Alteração do Benefício - Entes Públicos",
			Categoria:   CategoriaNaoPeriodicos,
			Grupo:       GrupoRPPSSetorPublico,
			Responsavel: "Gestor de RPPS / Setor Público",
			Descricao:   "Registra alterações nos dados do benefício previdenciário pago pelo órgão público ou RPPS.",
			Prazo:       "Até o dia 15 do mês seguinte ao da alteração no benefício.",
			TagRaiz:     "evtCdBenAlt",
			Namespace:   "http://www.esocial.gov.br/schema/evt/evtCdBenAlt/v_S_01_03_00",
			TemplateXML: `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCdBenAlt/v_S_01_03_00">
  <evtCdBenAlt Id="{{ID}}">
    <ideEvento>
      <indRetif>{{IND_RETIF}}</indRetif>
      <tpAmb>{{TP_AMB}}</tpAmb>
      <procEmi>{{PROC_EMI}}</procEmi>
      <verProc>{{VER_PROC}}</verProc>
    </ideEvento>
    <ideEmpregador>
      <tpInsc>{{TP_INSC}}</tpInsc>
      <nrInsc>{{NR_INSC}}</nrInsc>
    </ideEmpregador>
    <ideBeneficio>
      <cpfBenef>{{CPF_BENEF}}</cpfBenef>
      <nrBeneficio>{{NR_BENEFICIO}}</nrBeneficio>
    </ideBeneficio>
    <infoBenAlteracao>
      <dtAlteracao>{{DT_ALTERACAO}}</dtAlteracao>
    </infoBenAlteracao>
  </evtCdBenAlt>
</eSocial>`,
		},
	}

	mapaCatalogo = make(map[string]EventoCatalogo, len(catalogo)*2)
	for _, evt := range catalogo {
		mapaCatalogo[evt.Codigo] = evt
		// Indexa variações como "S1000", "1000", maiúsculas e minúsculas
		codLimpo := strings.ToUpper(strings.ReplaceAll(evt.Codigo, "-", ""))
		mapaCatalogo[codLimpo] = evt
		mapaCatalogo[strings.ToLower(evt.Codigo)] = evt
		mapaCatalogo[strings.ToLower(codLimpo)] = evt
	}
}

// ObterCatalogo retorna a lista completa de todos os eventos catalogados no eSocial S-1.3.
func ObterCatalogo() []EventoCatalogo {
	catalogoOnce.Do(inicializarCatalogo)
	return catalogo
}

// ObterEventoPorCodigo localiza um evento no catálogo pelo código (ex: "S-2240", "S2240", "2240").
func ObterEventoPorCodigo(codigo string) *EventoCatalogo {
	catalogoOnce.Do(inicializarCatalogo)
	codNorm := strings.ToUpper(strings.TrimSpace(codigo))
	if evt, ok := mapaCatalogo[codNorm]; ok {
		return &evt
	}
	// Tenta prefixar com "S-" se o usuário passar apenas o número (ex: "2240")
	if !strings.HasPrefix(codNorm, "S") {
		if evt, ok := mapaCatalogo["S-"+codNorm]; ok {
			return &evt
		}
	}
	return nil
}

// ListarPorGrupo filtra os eventos por um dos 6 grupos técnicos de responsabilidade:
// "SESMT", "Clínica Ocupacional", "RH / DP", "Contabilidade e Folha", "Jurídico", "RPPS / Setor Público".
func ListarPorGrupo(grupo string) []EventoCatalogo {
	catalogoOnce.Do(inicializarCatalogo)
	grupoNorm := strings.ToLower(strings.TrimSpace(grupo))
	var res []EventoCatalogo
	for _, evt := range catalogo {
		if strings.ToLower(evt.Grupo) == grupoNorm {
			res = append(res, evt)
		}
	}
	return res
}

// ListarGrupos retorna os nomes oficiais dos grupos de responsabilidade técnica.
func ListarGrupos() []string {
	return []string{
		GrupoSESMT,
		GrupoClinicaOcupacional,
		GrupoRHDP,
		GrupoContabilidadeFolha,
		GrupoJuridico,
		GrupoRPPSSetorPublico,
	}
}

// ObterGruposCatalogo é um alias compatível para ListarGrupos.
func ObterGruposCatalogo() []string {
	return ListarGrupos()
}

// ObterEventoCatalogo é um alias para ObterEventoPorCodigo.
func ObterEventoCatalogo(codigo string) *EventoCatalogo {
	return ObterEventoPorCodigo(codigo)
}

// FiltrarCatalogo busca eventos por grupo e/ou termo de busca.
func FiltrarCatalogo(grupo, busca string) []EventoCatalogo {
	todos := ObterCatalogo()
	grupoNorm := strings.ToLower(strings.TrimSpace(grupo))
	buscaNorm := strings.ToLower(strings.TrimSpace(busca))

	var res []EventoCatalogo
	for _, ev := range todos {
		if grupoNorm != "" && grupoNorm != "todos" {
			if strings.ToLower(ev.Grupo) != grupoNorm && !strings.Contains(strings.ToLower(ev.Grupo), grupoNorm) {
				continue
			}
		}
		if buscaNorm != "" {
			if !strings.Contains(strings.ToLower(ev.Codigo), buscaNorm) &&
				!strings.Contains(strings.ToLower(ev.Nome), buscaNorm) &&
				!strings.Contains(strings.ToLower(ev.Descricao), buscaNorm) {
				continue
			}
		}
		res = append(res, ev)
	}
	return res
}

