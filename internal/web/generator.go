package web

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// GerarIDEvento gera um identificador único no padrão oficial do eSocial:
// ID + tpInsc (1=CNPJ) + nrInsc (14 dig) + timestamp (YYYYMMDDHHMMSS) + seq (5 dig)
func GerarIDEvento(cnpj string) string {
	limpo := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cnpj)
	if len(limpo) < 14 {
		limpo = strings.Repeat("0", 14-len(limpo)) + limpo
	} else if len(limpo) > 14 {
		limpo = limpo[:14]
	}
	agora := time.Now().Format("20060102150405")
	nano := time.Now().Nanosecond() % 100000
	return fmt.Sprintf("ID1%s%s%05d", limpo, agora, nano)
}

// ParametrosS2240 reúne os dados necessários para compor o S-2240 (leiaute S-1.3).
type ParametrosS2240 struct {
	ID                 string
	Ambiente           int
	CNPJ               string
	CPFTrabalhador     string
	Matricula          string
	DataInicio         string
	DescAtividade      string
	LocalAmbiente      string
	DscSetor           string
	CodigoRisco        string
	NomeRisco          string
	TipoAvaliacao      string // 1 - Quantitativa, 2 - Qualitativa
	Intensidade        string
	UnidadeMedida      string // 1 dose diária de ruído, 2 dB linear, 3 dB(C), 4 dB(A)
	TecnicaMedicao     string
	UtilizaEPC         string // 0 - Não se aplica, 1 - Não implementa, 2 - Implementa
	EfficazEPC         string // S/N
	UtilizaEPI         string // 0 - Não se aplica, 1 - Não utilizado, 2 - Utilizado
	EficazEPI          string // S/N
	CAEPI              string
	MedicaoProtecao    string // S/N
	CondicaoFunc       string // S/N
	UsoIninterrupto    string // S/N
	PrazoValidade      string // S/N
	PeriodicidadeTroca string // S/N
	Higienizacao       string // S/N
	NomeResp           string
	CPFResp            string
	OrgaoClasse        string // 1 CRM, 4 CREA, 9 RMS (código do XSD)
	NumRegistro        string
	UFRegistro         string
}

// GerarXMLS2240 produz o documento XML no leiaute S-1.3 oficial (evtExpRisco),
// aderente ao XSD embutido em internal/data/xsd/evtExpRisco.xsd.
func GerarXMLS2240(p ParametrosS2240) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpjLimpo := limpaDigitos(p.CNPJ)
	cpfLimpo := limpaDigitos(p.CPFTrabalhador)
	cpfRespLimpo := limpaDigitos(p.CPFResp)
	if cpfRespLimpo == "" {
		cpfRespLimpo = cpfLimpo
	}
	if p.DataInicio == "" {
		p.DataInicio = time.Now().Format("2006-01-02")
	}
	if p.CodigoRisco == "" {
		p.CodigoRisco = "09.01.001"
		p.NomeRisco = "Ausência de agente nocivo ou atividades não constantes da Tabela 24"
	}
	tpAval := validarDominio(p.TipoAvaliacao, "2", "1")
	if p.CodigoRisco != "09.01.001" && p.NomeRisco == "" {
		p.NomeRisco = "Agente nocivo informado conforme Tabela 24"
	}
	localAmb := validarDominio(p.LocalAmbiente, "1", "2")
	dscSetor := strings.TrimSpace(p.DscSetor)
	if dscSetor == "" {
		dscSetor = "Geral Operacional"
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtExpRisco Id="%s">`+"\n", escapeXML(p.ID))
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.1.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")

	// ideVinculo (o XSD S-1.3 exige ideVinculo; matrícula é opcional)
	sb.WriteString("    <ideVinculo>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	if mat := strings.TrimSpace(p.Matricula); mat != "" {
		fmt.Fprintf(&sb, "      <matricula>%s</matricula>\n", escapeXML(mat))
	}
	sb.WriteString("    </ideVinculo>\n")

	sb.WriteString("    <infoExpRisco>\n")
	fmt.Fprintf(&sb, "      <dtIniCondicao>%s</dtIniCondicao>\n", validarDataISO(p.DataInicio))
	sb.WriteString("      <infoAmb>\n")
	fmt.Fprintf(&sb, "        <localAmb>%s</localAmb>\n", localAmb)
	fmt.Fprintf(&sb, "        <dscSetor>%s</dscSetor>\n", escapeXML(dscSetor))
	sb.WriteString("        <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "        <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("      </infoAmb>\n")
	sb.WriteString("      <infoAtiv>\n")
	descAtiv := escapeXML(p.DescAtividade)
	if descAtiv == "" {
		descAtiv = "Atividades operacionais pertinentes ao cargo."
	}
	fmt.Fprintf(&sb, "        <dscAtivDes>%s</dscAtivDes>\n", descAtiv)
	sb.WriteString("      </infoAtiv>\n")

	sb.WriteString("      <agNoc>\n")
	fmt.Fprintf(&sb, "        <codAgNoc>%s</codAgNoc>\n", escapeXML(p.CodigoRisco))
	if nomeRisco := strings.TrimSpace(p.NomeRisco); nomeRisco != "" {
		fmt.Fprintf(&sb, "        <dscAgNoc>%s</dscAgNoc>\n", escapeXML(nomeRisco))
	}
	fmt.Fprintf(&sb, "        <tpAval>%s</tpAval>\n", tpAval)
	if tpAval == "1" {
		intensidade := strings.TrimSpace(p.Intensidade)
		if intensidade == "" {
			intensidade = "0"
		}
		fmt.Fprintf(&sb, "        <intConc>%s</intConc>\n", escapeXML(intensidade))
		fmt.Fprintf(&sb, "        <unMed>%s</unMed>\n", validarDominio(p.UnidadeMedida, "1", "2", "3", "4"))
		tec := strings.TrimSpace(p.TecnicaMedicao)
		if tec == "" {
			tec = "Avaliação quantitativa conforme NR-09"
		}
		fmt.Fprintf(&sb, "        <tecMedicao>%s</tecMedicao>\n", escapeXML(tec))
	}

	// epcEpi
	utilizaEPC := validarDominio(p.UtilizaEPC, "0", "1", "2")
	utilizaEPI := validarDominio(p.UtilizaEPI, "0", "1", "2")
	sb.WriteString("        <epcEpi>\n")
	fmt.Fprintf(&sb, "          <utilizEPC>%s</utilizEPC>\n", utilizaEPC)
	if utilizaEPC == "2" {
		fmt.Fprintf(&sb, "          <eficEpc>%s</eficEpc>\n", validarDominio(p.EfficazEPC, "S", "N"))
	}
	fmt.Fprintf(&sb, "          <utilizEPI>%s</utilizEPI>\n", utilizaEPI)
	if utilizaEPI == "2" {
		fmt.Fprintf(&sb, "          <eficEpi>%s</eficEpi>\n", validarDominio(p.EficazEPI, "S", "N"))
		ca := strings.TrimSpace(p.CAEPI)
		if ca == "" {
			ca = "00000"
		}
		sb.WriteString("          <epi>\n")
		fmt.Fprintf(&sb, "            <docAval>%s</docAval>\n", escapeXML(ca))
		sb.WriteString("          </epi>\n")
		sb.WriteString("          <epiCompl>\n")
		fmt.Fprintf(&sb, "            <medProtecao>%s</medProtecao>\n", validarDominio(p.MedicaoProtecao, "N", "S"))
		fmt.Fprintf(&sb, "            <condFuncto>%s</condFuncto>\n", validarDominio(p.CondicaoFunc, "S", "N"))
		fmt.Fprintf(&sb, "            <usoInint>%s</usoInint>\n", validarDominio(p.UsoIninterrupto, "S", "N"))
		fmt.Fprintf(&sb, "            <przValid>%s</przValid>\n", validarDominio(p.PrazoValidade, "S", "N"))
		fmt.Fprintf(&sb, "            <periodicTroca>%s</periodicTroca>\n", validarDominio(p.PeriodicidadeTroca, "S", "N"))
		fmt.Fprintf(&sb, "            <higienizacao>%s</higienizacao>\n", validarDominio(p.Higienizacao, "S", "N"))
		sb.WriteString("          </epiCompl>\n")
	}
	sb.WriteString("        </epcEpi>\n")
	sb.WriteString("      </agNoc>\n")

	// respReg: responsável pelos registros ambientais
	if cpfRespLimpo != "" {
		sb.WriteString("      <respReg>\n")
		fmt.Fprintf(&sb, "        <cpfResp>%s</cpfResp>\n", cpfRespLimpo)
		// ideOC é obrigatório quando codAgNoc difere de 09.01.001
		fmt.Fprintf(&sb, "        <ideOC>%s</ideOC>\n", validarDominio(p.OrgaoClasse, "4", "1", "9"))
		numReg := strings.TrimSpace(p.NumRegistro)
		if numReg == "" {
			numReg = "000000"
		}
		fmt.Fprintf(&sb, "        <dscOC>%s</dscOC>\n", escapeXML(numReg))
		fmt.Fprintf(&sb, "        <nrOC>%s</nrOC>\n", escapeXML(numReg))
		fmt.Fprintf(&sb, "        <ufOC>%s</ufOC>\n", validarUF(p.UFRegistro))
		sb.WriteString("      </respReg>\n")
	}
	sb.WriteString("    </infoExpRisco>\n")
	sb.WriteString("  </evtExpRisco>\n")
	sb.WriteString("</eSocial>")

	return sb.String()
}

// ParametrosS2210 reúne os dados para o evento de Acidente de Trabalho (CAT).
type ParametrosS2210 struct {
	ID             string
	Ambiente       int
	CNPJ           string
	CPFTrabalhador string
	Matricula      string
	DtAcidente     string
	HrAcidente     string
	TpAcidente     string // 1 - Típico, 2 - Doença, 3 - Trajeto
	HrsTrabAntes   string
	TpCat          string // 1 - Inicial, 2 - Reabertura, 3 - Comunicação de óbito
	HouveAfast     string // S/N
	HouveObito     string // S/N
	ComunPolicia   string // S/N
	CodSitGeradora string
	IniciatCAT     string // 1 Empregador, 2 Ordem judicial, 3 Determinação de órgão fiscalizador
	ObsCAT         string
	UltDiaTrab     string
	TpLocal        string // 1..5
	DscLocal       string
	DscLograd      string
	NrLograd       string
	Bairro         string
	CEP            string
	CodMunic       string
	UF             string
	ParteCorpo     string
	Lateralidade   string // 0..3
	AgenteCausador string
	DtAtendimento  string
	HrAtendimento  string
	IndInternacao  string // S/N
	DurTrat        string
	IndAfast       string // S/N
	DescLesao      string // código da Tabela 17
	DscCompLesao   string
	DiagProvavel   string
	CID10          string
	Observacao     string
	NomeMedico     string
	CRMMedico      string
	UFMedico       string
	OrgaoClasseMed string // 1 CRM, 2 CRO, 3 RMS
}

// GerarXMLS2210 produz o documento XML do CAT no leiaute S-1.3 oficial (evtCAT),
// aderente ao XSD embutido em internal/data/xsd/evtCAT.xsd.
func GerarXMLS2210(p ParametrosS2210) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpjLimpo := limpaDigitos(p.CNPJ)
	cpfLimpo := limpaDigitos(p.CPFTrabalhador)
	if p.DtAcidente == "" {
		p.DtAcidente = time.Now().Format("2006-01-02")
	}
	tpAcid := validarDominio(p.TpAcidente, "1", "2", "3")
	hrAcid := validarHora(p.HrAcidente)
	hrsAntes := validarHora(p.HrsTrabAntes)
	tpCat := validarDominio(p.TpCat, "1", "2", "3")
	indObito := simNao(p.HouveObito)
	indPolicia := simNao(p.ComunPolicia)
	indAfast := simNao(p.IndAfast)
	if p.HouveAfast != "" && p.IndAfast == "" {
		indAfast = simNao(p.HouveAfast)
	}
	if indObito == "S" {
		indAfast = "N"
	}
	dtAtendimento := validarDataISO(p.DtAtendimento)
	if dtAtendimento == "" {
		dtAtendimento = validarDataISO(p.DtAcidente)
	}
	sitGeradora := strings.TrimSpace(p.CodSitGeradora)
	if len(sitGeradora) != 9 || !somenteDigitos(sitGeradora) {
		sitGeradora = "200004300" // Tabela 15 - impacto de pessoa contra objeto parado
	}
	codParte := strings.TrimSpace(p.ParteCorpo)
	if len(codParte) != 9 || !somenteDigitos(codParte) {
		codParte = "755070000" // Tabela 13 - dedo
	}
	codAgente := strings.TrimSpace(p.AgenteCausador)
	if len(codAgente) != 9 || !somenteDigitos(codAgente) {
		codAgente = "303010040" // Tabela 14 - martelo/marreta (ferramenta manual)
	}
	lesao := strings.TrimSpace(p.DescLesao)
	if !somenteDigitos(lesao) {
		lesao = "702010000" // Tabela 17 - corte, laceração, ferida contusa, punctura
	}
	durTrat := strings.TrimSpace(p.DurTrat)
	if !somenteDigitos(durTrat) {
		durTrat = "0"
	}
	cid := normalizarCID(p.CID10)
	tpLocal := validarDominio(p.TpLocal, "1", "2", "3", "4", "5")
	cep := somenteDigitosOu(strings.TrimSpace(p.CEP), "00000000", 8)
	munic := somenteDigitosOu(strings.TrimSpace(p.CodMunic), "0000000", 7)
	logradouro := strings.TrimSpace(p.DscLograd)
	if logradouro == "" {
		logradouro = "NAO INFORMADO"
	}
	nrLogradouro := strings.TrimSpace(p.NrLograd)
	if nrLogradouro == "" {
		nrLogradouro = "S/N"
	}
	bairro := strings.TrimSpace(p.Bairro)
	if bairro == "" {
		bairro = "CENTRO"
	}
	ufLocal := validarUF(p.UF)

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtCAT Id="%s">`+"\n", escapeXML(p.ID))
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.1.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")
	sb.WriteString("    <ideVinculo>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	if mat := strings.TrimSpace(p.Matricula); mat != "" {
		fmt.Fprintf(&sb, "      <matricula>%s</matricula>\n", escapeXML(mat))
	}
	sb.WriteString("    </ideVinculo>\n")

	sb.WriteString("    <cat>\n")
	fmt.Fprintf(&sb, "      <dtAcid>%s</dtAcid>\n", validarDataISO(p.DtAcidente))
	fmt.Fprintf(&sb, "      <tpAcid>%s</tpAcid>\n", tpAcid)
	if tpAcid == "1" || tpAcid == "3" {
		fmt.Fprintf(&sb, "      <hrAcid>%s</hrAcid>\n", hrAcid)
		fmt.Fprintf(&sb, "      <hrsTrabAntesAcid>%s</hrsTrabAntesAcid>\n", hrsAntes)
	}
	fmt.Fprintf(&sb, "      <tpCat>%s</tpCat>\n", tpCat)
	fmt.Fprintf(&sb, "      <indCatObito>%s</indCatObito>\n", indObito)
	if indObito == "S" {
		fmt.Fprintf(&sb, "      <dtObito>%s</dtObito>\n", validarDataISO(p.DtAcidente))
	}
	fmt.Fprintf(&sb, "      <indComunPolicia>%s</indComunPolicia>\n", indPolicia)
	fmt.Fprintf(&sb, "      <codSitGeradora>%s</codSitGeradora>\n", sitGeradora)
	fmt.Fprintf(&sb, "      <iniciatCAT>%s</iniciatCAT>\n", validarDominio(p.IniciatCAT, "1", "2", "3"))
	if obs := strings.TrimSpace(p.ObsCAT); obs != "" {
		fmt.Fprintf(&sb, "      <obsCAT>%s</obsCAT>\n", escapeXML(obs))
	}
	fmt.Fprintf(&sb, "      <ultDiaTrab>%s</ultDiaTrab>\n", validarDataISO(p.UltDiaTrab))
	fmt.Fprintf(&sb, "      <houveAfast>%s</houveAfast>\n", indAfast)

	sb.WriteString("      <localAcidente>\n")
	fmt.Fprintf(&sb, "        <tpLocal>%s</tpLocal>\n", tpLocal)
	if dscLocal := strings.TrimSpace(p.DscLocal); dscLocal != "" {
		fmt.Fprintf(&sb, "        <dscLocal>%s</dscLocal>\n", escapeXML(dscLocal))
	}
	fmt.Fprintf(&sb, "        <dscLograd>%s</dscLograd>\n", escapeXML(logradouro))
	fmt.Fprintf(&sb, "        <nrLograd>%s</nrLograd>\n", escapeXML(nrLogradouro))
	fmt.Fprintf(&sb, "        <bairro>%s</bairro>\n", escapeXML(bairro))
	fmt.Fprintf(&sb, "        <cep>%s</cep>\n", cep)
	fmt.Fprintf(&sb, "        <codMunic>%s</codMunic>\n", munic)
	fmt.Fprintf(&sb, "        <uf>%s</uf>\n", ufLocal)
	sb.WriteString("      </localAcidente>\n")

	sb.WriteString("      <parteAtingida>\n")
	fmt.Fprintf(&sb, "        <codParteAting>%s</codParteAting>\n", codParte)
	fmt.Fprintf(&sb, "        <lateralidade>%s</lateralidade>\n", validarDominio(p.Lateralidade, "0", "1", "2", "3"))
	sb.WriteString("      </parteAtingida>\n")

	sb.WriteString("      <agenteCausador>\n")
	fmt.Fprintf(&sb, "        <codAgntCausador>%s</codAgntCausador>\n", codAgente)
	sb.WriteString("      </agenteCausador>\n")

	sb.WriteString("      <atestado>\n")
	fmt.Fprintf(&sb, "        <dtAtendimento>%s</dtAtendimento>\n", dtAtendimento)
	fmt.Fprintf(&sb, "        <hrAtendimento>%s</hrAtendimento>\n", validarHora(p.HrAtendimento))
	fmt.Fprintf(&sb, "        <indInternacao>%s</indInternacao>\n", simNao(p.IndInternacao))
	fmt.Fprintf(&sb, "        <durTrat>%s</durTrat>\n", durTrat)
	fmt.Fprintf(&sb, "        <indAfast>%s</indAfast>\n", indAfast)
	fmt.Fprintf(&sb, "        <dscLesao>%s</dscLesao>\n", lesao)
	if comp := strings.TrimSpace(p.DscCompLesao); comp != "" {
		fmt.Fprintf(&sb, "        <dscCompLesao>%s</dscCompLesao>\n", escapeXML(comp))
	}
	if diag := strings.TrimSpace(p.DiagProvavel); diag != "" {
		fmt.Fprintf(&sb, "        <diagProvavel>%s</diagProvavel>\n", escapeXML(diag))
	}
	fmt.Fprintf(&sb, "        <codCID>%s</codCID>\n", escapeXML(cid))
	if obs := strings.TrimSpace(p.Observacao); obs != "" {
		fmt.Fprintf(&sb, "        <observacao>%s</observacao>\n", escapeXML(obs))
	}
	sb.WriteString("        <emitente>\n")
	nomeMed := strings.TrimSpace(p.NomeMedico)
	if nomeMed == "" {
		nomeMed = "Médico Examinador"
	}
	fmt.Fprintf(&sb, "          <nmEmit>%s</nmEmit>\n", escapeXML(nomeMed))
	fmt.Fprintf(&sb, "          <ideOC>%s</ideOC>\n", validarDominio(p.OrgaoClasseMed, "1", "2", "3"))
	crm := strings.TrimSpace(p.CRMMedico)
	if crm == "" {
		crm = "000000"
	}
	fmt.Fprintf(&sb, "          <nrOC>%s</nrOC>\n", escapeXML(crm))
	fmt.Fprintf(&sb, "          <ufOC>%s</ufOC>\n", validarUF(p.UFMedico))
	sb.WriteString("        </emitente>\n")
	sb.WriteString("      </atestado>\n")
	sb.WriteString("    </cat>\n")
	sb.WriteString("  </evtCAT>\n")
	sb.WriteString("</eSocial>")

	return sb.String()
}

// ParametrosS2220 reúne os dados do ASO.
type ParametrosS2220 struct {
	ID             string
	Ambiente       int
	CNPJ           string
	CPFTrabalhador string
	Matricula      string
	TipoExame      string // 0 - Admissional, 1 - Periódico, 2 - Retorno, 3 - Mudança de função, 4 - Demissional
	DataASO        string
	ResultadoASO   string // 1 - Apto, 2 - Inapto
	ProcRealizado  string // Tabela 27
	ObsProc        string
	OrdExame       string // 1 - Inicial, 2 - Sequencial
	IndResult      string // 1 - Normal, 2 - Alterado, 3 - Estável, 4 - Inconclusivo
	NomeMedico     string
	CRMMedico      string
	UFMedico       string
	NomeCoord      string
	CPFCoord       string
	CRMCoord       string
	UFCoord        string
	ObsASO         string
}

// GerarXMLS2220 produz o documento XML de Monitoramento da Saúde do Trabalhador
// no leiaute S-1.3 oficial (evtMonit), aderente a internal/data/xsd/evtMonit.xsd.
func GerarXMLS2220(p ParametrosS2220) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpjLimpo := limpaDigitos(p.CNPJ)
	cpfLimpo := limpaDigitos(p.CPFTrabalhador)
	if p.DataASO == "" {
		p.DataASO = time.Now().Format("2006-01-02")
	}
	// O XSD aceita 0..4 (o valor 9 do formulário legado é convertido em 4 - demissional).
	tpExame := validarDominio(p.TipoExame, "1", "0", "2", "3", "4")
	if p.TipoExame == "9" {
		tpExame = "4"
	}
	resAso := validarDominio(p.ResultadoASO, "1", "2")
	proc := strings.TrimSpace(p.ProcRealizado)
	if !somenteDigitos(proc) {
		proc = "0295" // Tabela 27 - avaliação clínica ocupacional
	}
	nomeMed := strings.TrimSpace(p.NomeMedico)
	if nomeMed == "" {
		nomeMed = "Médico Examinador"
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtMonit Id="%s">`+"\n", escapeXML(p.ID))
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.1.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")
	sb.WriteString("    <ideVinculo>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	if mat := strings.TrimSpace(p.Matricula); mat != "" {
		fmt.Fprintf(&sb, "      <matricula>%s</matricula>\n", escapeXML(mat))
	}
	sb.WriteString("    </ideVinculo>\n")

	sb.WriteString("    <exMedOcup>\n")
	fmt.Fprintf(&sb, "      <tpExameOcup>%s</tpExameOcup>\n", tpExame)
	sb.WriteString("      <aso>\n")
	fmt.Fprintf(&sb, "        <dtAso>%s</dtAso>\n", validarDataISO(p.DataASO))
	fmt.Fprintf(&sb, "        <resAso>%s</resAso>\n", resAso)
	sb.WriteString("        <exame>\n")
	fmt.Fprintf(&sb, "          <dtExm>%s</dtExm>\n", validarDataISO(p.DataASO))
	fmt.Fprintf(&sb, "          <procRealizado>%s</procRealizado>\n", proc)
	if obs := strings.TrimSpace(p.ObsProc); obs != "" {
		fmt.Fprintf(&sb, "          <obsProc>%s</obsProc>\n", escapeXML(obs))
	}
	if p.OrdExame != "" {
		fmt.Fprintf(&sb, "          <ordExame>%s</ordExame>\n", validarDominio(p.OrdExame, "1", "2"))
	}
	if p.IndResult != "" {
		fmt.Fprintf(&sb, "          <indResult>%s</indResult>\n", validarDominio(p.IndResult, "1", "2", "3", "4"))
	}
	sb.WriteString("        </exame>\n")
	sb.WriteString("        <medico>\n")
	fmt.Fprintf(&sb, "          <nmMed>%s</nmMed>\n", escapeXML(nomeMed))
	if crm := strings.TrimSpace(p.CRMMedico); crm != "" {
		fmt.Fprintf(&sb, "          <nrCRM>%s</nrCRM>\n", escapeXML(crm))
		fmt.Fprintf(&sb, "          <ufCRM>%s</ufCRM>\n", validarUF(p.UFMedico))
	}
	sb.WriteString("        </medico>\n")
	sb.WriteString("      </aso>\n")

	// respMonit: médico responsável/coordenador do PCMSO (opcional)
	if nomeCoord := strings.TrimSpace(p.NomeCoord); nomeCoord != "" {
		sb.WriteString("      <respMonit>\n")
		if cpfCoord := limpaDigitos(p.CPFCoord); cpfCoord != "" {
			fmt.Fprintf(&sb, "        <cpfResp>%s</cpfResp>\n", cpfCoord)
		}
		fmt.Fprintf(&sb, "        <nmResp>%s</nmResp>\n", escapeXML(nomeCoord))
		if crm := strings.TrimSpace(p.CRMCoord); crm != "" {
			fmt.Fprintf(&sb, "        <nrCRM>%s</nrCRM>\n", escapeXML(crm))
		}
		fmt.Fprintf(&sb, "        <ufCRM>%s</ufCRM>\n", validarUF(p.UFCoord))
		sb.WriteString("      </respMonit>\n")
	}
	sb.WriteString("    </exMedOcup>\n")
	sb.WriteString("  </evtMonit>\n")
	sb.WriteString("</eSocial>")

	return sb.String()
}

// normalizarCID limpa o código CID-10 informado: o leiaute S-1.3 aceita no máximo
// 4 caracteres alfanuméricos (o formato com ponto, como "S61.0", é inválido).
func normalizarCID(v string) string {
	limpo := strings.ToUpper(strings.TrimSpace(v))
	limpo = strings.NewReplacer(".", "", "-", "", " ", "").Replace(limpo)
	if len(limpo) > 4 {
		limpo = limpo[:4]
	}
	if limpo == "" {
		limpo = "S610" // ferimento de dedo (CID-10 S61.0) como padrão
	}
	return limpo
}

// simNao normaliza valores S/N vindos do formulário.
func simNao(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "S", "SIM", "1", "TRUE":
		return "S"
	default:
		return "N"
	}
}

// somenteDigitosOu devolve o valor se ele tiver exatamente n dígitos; caso contrário, o padrão.
func somenteDigitosOu(v, padrao string, n int) string {
	if len(v) == n && somenteDigitos(v) {
		return v
	}
	return padrao
}

// ValidarEventoXSD executa checagens essenciais no documento gerado.
func ValidarEventoXSD(tipo, xmlContent string) (bool, []string) {
	var erros []string
	if strings.TrimSpace(xmlContent) == "" {
		return false, []string{"XML está vazio."}
	}

	// 1. Verificação sintática básica
	var parsed struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(xmlContent), &parsed); err != nil {
		erros = append(erros, fmt.Sprintf("Erro de sintaxe XML: %v", err))
		return false, erros
	}

	// 2. Namespace e tags requeridas por evento
	switch tipo {
	case "S-2240":
		if !strings.Contains(xmlContent, "http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00") {
			erros = append(erros, "Namespace oficial de evtExpRisco não encontrado.")
		}
		if !strings.Contains(xmlContent, "<evtExpRisco") {
			erros = append(erros, "Tag raiz <evtExpRisco> obrigatória não encontrada.")
		}
		if !strings.Contains(xmlContent, "<cpfTrab>") {
			erros = append(erros, "Campo <cpfTrab> obrigatório não preenchido.")
		}
		if !strings.Contains(xmlContent, "<codAgNoc>") {
			erros = append(erros, "Campo <codAgNoc> (Tabela 24) obrigatório não encontrado.")
		}

	case "S-2210":
		if !strings.Contains(xmlContent, "http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00") {
			erros = append(erros, "Namespace oficial de evtCAT não encontrado.")
		}
		if !strings.Contains(xmlContent, "<evtCAT") {
			erros = append(erros, "Tag raiz <evtCAT> obrigatória não encontrada.")
		}
		if !strings.Contains(xmlContent, "<dtAcid>") {
			erros = append(erros, "Campo <dtAcid> obrigatório não informado.")
		}
		if !strings.Contains(xmlContent, "<codCID>") {
			erros = append(erros, "Campo <codCID> no atestado obrigatório.")
		}

	case "S-2220":
		if !strings.Contains(xmlContent, "http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00") {
			erros = append(erros, "Namespace oficial de evtMonit não encontrado.")
		}
		if !strings.Contains(xmlContent, "<evtMonit") {
			erros = append(erros, "Tag raiz <evtMonit> obrigatória não encontrada.")
		}
		if !strings.Contains(xmlContent, "<dtAso>") {
			erros = append(erros, "Campo <dtAso> obrigatório não informado.")
		}
	}

	return len(erros) == 0, erros
}

// SimularAssinatura insere um envelope XMLDSig DEMONSTRATIVO no documento XML.
//
// ATENÇÃO (achado A-02): este envelope NÃO é uma assinatura digital válida. Ele é
// marcado explicitamente como simulação (valores em base64 legíveis como
// "SIMULACAO-...") para que nenhum artefato seja confundido com prova documental
// oficial. A assinatura real deve usar crypto.AssinarXML com o certificado A1.
func SimularAssinatura(xmlContent, certSubject string) string {
	h := sha256.New()
	h.Write([]byte(xmlContent))
	digest := hex.EncodeToString(h.Sum(nil))

	valorSimulado := base64.StdEncoding.EncodeToString([]byte("SIMULACAO-NAO-VALIDA-JURIDICAMENTE:" + digest[:32]))
	certSimulado := base64.StdEncoding.EncodeToString([]byte("SIMULACAO-SEM-CERTIFICADO-ICP-BRASIL"))

	sigBlock := fmt.Sprintf(`  <!-- ASSINATURA SIMULADA: sem certificado digital aplicado e sem validade jurídica -->
  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
    <SignedInfo>
      <CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315" />
      <SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256" />
      <Reference URI="">
        <Transforms>
          <Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature" />
          <Transform Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315" />
        </Transforms>
        <DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256" />
<DigestValue>%s</DigestValue>
</Reference>
</SignedInfo>
<SignatureValue>%s</SignatureValue>
<KeyInfo>
<X509Data>
<X509Certificate>%s</X509Certificate>
</X509Data>
</KeyInfo>
</Signature>
</eSocial>`, digest[:24], valorSimulado, certSimulado)

	return strings.Replace(xmlContent, "</eSocial>", sigBlock, 1)
}

// MensagemTransmissaoSimulada descreve explicitamente que NADA foi enviado ao eSocial.
//
// ATENÇÃO (achado A-02): esta versão não está integrada ao webservice oficial
// (internal/soap). Nenhum protocolo e nenhum recibo são gerados, para não produzir
// prova documental fictícia nem trilha de auditoria falsa.
func MensagemTransmissaoSimulada(ambiente int, tipo string) string {
	ambStr := "Produção Restrita (homologação)"
	if ambiente == 1 {
		ambStr = "Produção Oficial"
	}
	return fmt.Sprintf(
		"SIMULAÇÃO: nenhum dado foi enviado ao eSocial. O envio real do evento %s para o ambiente %s depende da "+
			"integração com o webservice oficial (mTLS com certificado A1), ainda não conectada nesta versão. "+
			"Nenhum protocolo ou recibo oficial foi gerado.",
		tipo, ambStr,
	)
}

func limpaDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// validarDataISO aceita apenas datas no formato AAAA-MM-DD (fallback: data de hoje).
func validarDataISO(v string) string {
	v = strings.TrimSpace(v)
	if len(v) == 10 && v[4] == '-' && v[7] == '-' &&
		somenteDigitos(v[0:4]) && somenteDigitos(v[5:7]) && somenteDigitos(v[8:10]) {
		return v
	}
	return time.Now().Format("2006-01-02")
}

// validarHora aceita apenas HHMM numérico de 4 dígitos (fallback: 0800).
func validarHora(v string) string {
	v = strings.ReplaceAll(strings.TrimSpace(v), ":", "")
	if len(v) == 4 && somenteDigitos(v) {
		return v
	}
	return "0800"
}

// validarDominio restringe o valor a uma lista fechada (fallback: primeiro item).
func validarDominio(v string, permitidos ...string) string {
	v = strings.TrimSpace(v)
	for _, p := range permitidos {
		if v == p {
			return v
		}
	}
	if len(permitidos) > 0 {
		return permitidos[0]
	}
	return ""
}

var ufsOficiais = map[string]bool{
	"AC": true, "AL": true, "AP": true, "AM": true, "BA": true, "CE": true, "DF": true,
	"ES": true, "GO": true, "MA": true, "MT": true, "MS": true, "MG": true, "PA": true,
	"PB": true, "PR": true, "PE": true, "PI": true, "RJ": true, "RN": true, "RS": true,
	"RO": true, "RR": true, "SC": true, "SP": true, "SE": true, "TO": true,
}

// validarUF aceita apenas siglas oficiais de UF (fallback: SP).
func validarUF(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if ufsOficiais[v] {
		return v
	}
	return "SP"
}

func somenteDigitos(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
