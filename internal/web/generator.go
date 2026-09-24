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

// ParametrosS2240 reúne os dados necessários para compor o S-2240.
type ParametrosS2240 struct {
	ID                 string
	Ambiente           int
	CNPJ               string
	CPFTrabalhador     string
	Matricula          string
	DataInicio         string
	DescAtividade      string
	LocalAmbiente      string
	CodigoRisco        string
	NomeRisco          string
	TipoAvaliacao      string // 1 - Quantitativa, 2 - Qualitativa
	Intensidade        string
	UtilizaEPC         string // 0 - Não se aplica, 1 - Não utilizado, 2 - Utilizado
	EfficazEPC         string // S/N
	UtilizaEPI         string // 0 - Não se aplica, 1 - Não utilizado, 2 - Utilizado
	CAEPI              string
	EficazEPI          string // S/N
	MedicaoProtecao    string // S/N
	CondicaoFunc       string // S/N
	UsoIninterrupto    string // S/N
	PrazoValidade      string // S/N
	PeriodicidadeTroca string // S/N
	Higienizacao       string // S/N
	NomeResp           string
	CPFResp            string
	OrgaoClasse        string
	NumRegistro        string
	UFRegistro         string
}

// GerarXMLS2240 produz o documento XML no leiaute S-1.3 oficial.
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
	if p.TipoAvaliacao == "" {
		p.TipoAvaliacao = "2"
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtExpRisco Id="%s">`+"\n", p.ID)
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.0.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")
	sb.WriteString("    <ideTrabalhador>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	sb.WriteString("    </ideTrabalhador>\n")
	sb.WriteString("    <infoExpRisco>\n")
	fmt.Fprintf(&sb, "      <dtIniCondicao>%s</dtIniCondicao>\n", validarDataISO(p.DataInicio))
	sb.WriteString("      <infoAmb>\n")
	sb.WriteString("        <localAmb>1</localAmb>\n")
	sb.WriteString("        <dscSetor>Geral Operacional</dscSetor>\n")
	fmt.Fprintf(&sb, "        <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "        <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("      </infoAmb>\n")
	sb.WriteString("      <infoAtiv>\n")
	descAtiv := escapeXML(p.DescAtividade)
	if descAtiv == "" {
		descAtiv = "Atividades operacionais pertinentes ao cargo."
	}
	fmt.Fprintf(&sb, "        <dscAtivDes>%s</dscAtivDes>\n", descAtiv)
	sb.WriteString("      </infoAtiv>\n")
	sb.WriteString("      <agenteNoc>\n")
	fmt.Fprintf(&sb, "        <codAgNoc>%s</codAgNoc>\n", escapeXML(p.CodigoRisco))
	fmt.Fprintf(&sb, "        <dscAgNoc>%s</dscAgNoc>\n", escapeXML(p.NomeRisco))
	fmt.Fprintf(&sb, "        <tpAval>%s</tpAval>\n", validarDominio(p.TipoAvaliacao, "2", "1"))
	if p.TipoAvaliacao == "1" && p.Intensidade != "" {
		fmt.Fprintf(&sb, "        <intConc>%s</intConc>\n", escapeXML(p.Intensidade))
	}
	// EPI / EPC
	if p.UtilizaEPC == "" {
		p.UtilizaEPC = "0"
	}
	fmt.Fprintf(&sb, "        <epcEpi>\n")
	fmt.Fprintf(&sb, "          <utilizEPC>%s</utilizEPC>\n", validarDominio(p.UtilizaEPC, "0", "1", "2"))
	if p.UtilizaEPC == "2" {
		eficaz := p.EfficazEPC
		if eficaz == "" {
			eficaz = "S"
		}
		fmt.Fprintf(&sb, "          <eficEpc>%s</eficEpc>\n", validarDominio(eficaz, "S", "N"))
	}
	if p.UtilizaEPI == "" {
		p.UtilizaEPI = "0"
	}
	fmt.Fprintf(&sb, "          <utilizEPI>%s</utilizEPI>\n", validarDominio(p.UtilizaEPI, "0", "1", "2"))
	if p.UtilizaEPI == "2" {
		sb.WriteString("          <epi>\n")
		ca := p.CAEPI
		if ca == "" {
			ca = "00000"
		}
		fmt.Fprintf(&sb, "            <docAval>%s</docAval>\n", escapeXML(ca))
		sb.WriteString("            <eficEpi>S</eficEpi>\n")
		sb.WriteString("            <medProtecao>S</medProtecao>\n")
		sb.WriteString("            <condFuncto>S</condFuncto>\n")
		sb.WriteString("            <usoInint>S</usoInint>\n")
		sb.WriteString("            <przValid>S</przValid>\n")
		sb.WriteString("            <periodicTroca>S</periodicTroca>\n")
		sb.WriteString("            <higienizacao>S</higienizacao>\n")
		sb.WriteString("          </epi>\n")
	}
	sb.WriteString("        </epcEpi>\n")
	sb.WriteString("      </agenteNoc>\n")
	// Responsável Técnico
	sb.WriteString("      <respReg>\n")
	fmt.Fprintf(&sb, "        <cpfResp>%s</cpfResp>\n", cpfRespLimpo)
	orgao := p.OrgaoClasse
	if orgao == "" {
		orgao = "CREA"
	}
	fmt.Fprintf(&sb, "        <ideOC>%s</ideOC>\n", escapeXML(orgao))
	numReg := p.NumRegistro
	if numReg == "" {
		numReg = "000000"
	}
	fmt.Fprintf(&sb, "        <dscOC>%s</dscOC>\n", escapeXML(numReg))
	uf := p.UFRegistro
	if uf == "" {
		uf = "SP"
	}
	fmt.Fprintf(&sb, "        <ufOC>%s</ufOC>\n", validarUF(uf))
	sb.WriteString("      </respReg>\n")
	sb.WriteString("    </infoExpRisco>\n")
	sb.WriteString("  </evtExpRisco>\n")
	sb.WriteString("</eSocial>")

	return sb.String()
}

// ParametrosS2210 reúne os dados para o evento de Acidente de Trabalho.
type ParametrosS2210 struct {
	ID             string
	Ambiente       int
	CNPJ           string
	CPFTrabalhador string
	Matricula      string
	DtAcidente     string
	HrAcidente     string
	TpAcidente     string // 1 - Típico, 2 - Doença, 3 - Trajeto
	HouveAfast     string // S/N
	DtAfast        string
	HouveObito     string // S/N
	DtObito        string
	ComunPolicia   string // S/N
	LocalAcidente  string // 1 - Estabelecimento, etc.
	DescLocal      string
	ParteCorpo     string
	AgenteCausador string
	DtAtendimento  string
	HrAtendimento  string
	CID10          string
	DescLesao      string
	NomeMedico     string
	CRMMedico      string
	UFMedico       string
}

// GerarXMLS2210 produz o documento XML para o CAT.
func GerarXMLS2210(p ParametrosS2210) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpjLimpo := limpaDigitos(p.CNPJ)
	cpfLimpo := limpaDigitos(p.CPFTrabalhador)
	if p.DtAcidente == "" {
		p.DtAcidente = time.Now().Format("2006-01-02")
	}
	if p.HrAcidente == "" {
		p.HrAcidente = "0800"
	}
	p.HrAcidente = strings.ReplaceAll(p.HrAcidente, ":", "")
	if len(p.HrAcidente) > 4 {
		p.HrAcidente = p.HrAcidente[:4]
	}
	if p.TpAcidente == "" {
		p.TpAcidente = "1"
	}
	if p.DtAtendimento == "" {
		p.DtAtendimento = p.DtAcidente
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtCAT Id="%s">`+"\n", p.ID)
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.0.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")
	sb.WriteString("    <ideTrabalhador>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	sb.WriteString("    </ideTrabalhador>\n")
	sb.WriteString("    <cat>\n")
	fmt.Fprintf(&sb, "      <dtAcid>%s</dtAcid>\n", validarDataISO(p.DtAcidente))
	fmt.Fprintf(&sb, "      <tpAcid>%s</tpAcid>\n", validarDominio(p.TpAcidente, "1", "2", "3"))
	fmt.Fprintf(&sb, "      <hrAcid>%s</hrAcid>\n", validarHora(p.HrAcidente))
	sb.WriteString("      <hrsTrabAntesAcid>0200</hrsTrabAntesAcid>\n")
	tpCat := "1" // Inicial
	fmt.Fprintf(&sb, "      <tpCat>%s</tpCat>\n", tpCat)
	afast := "N"
	if strings.ToUpper(p.HouveAfast) == "S" || p.HouveAfast == "sim" {
		afast = "S"
	}
	fmt.Fprintf(&sb, "      <houveAfast>%s</houveAfast>\n", afast)
	fmt.Fprintf(&sb, "      <indCatObito>%s</indCatObito>\n", func() string {
		if strings.ToUpper(p.HouveObito) == "S" || p.HouveObito == "sim" {
			return "S"
		}
		return "N"
	}())
	fmt.Fprintf(&sb, "      <indComunPolicia>%s</indComunPolicia>\n", func() string {
		if strings.ToUpper(p.ComunPolicia) == "S" || p.ComunPolicia == "sim" {
			return "S"
		}
		return "N"
	}())
	sb.WriteString("      <localAcidente>\n")
	sb.WriteString("        <tpLocal>1</tpLocal>\n")
	descLoc := escapeXML(p.DescLocal)
	if descLoc == "" {
		descLoc = "Instalações da empresa."
	}
	fmt.Fprintf(&sb, "        <dscLocal>%s</dscLocal>\n", descLoc)
	sb.WriteString("      </localAcidente>\n")
	sb.WriteString("      <parteAtingida>\n")
	codParte := "752000000" // Dedos da mão como default se não informado
	if p.ParteCorpo != "" {
		codParte = limpaDigitos(p.ParteCorpo)
		if len(codParte) < 9 {
			codParte = "752000000"
		}
	}
	fmt.Fprintf(&sb, "        <codParteAting>%s</codParteAting>\n", codParte)
	sb.WriteString("      </parteAtingida>\n")
	sb.WriteString("      <agenteCausador>\n")
	codAgente := "302010100" // Máquinas / ferramentas manuais
	if p.AgenteCausador != "" {
		codAgente = limpaDigitos(p.AgenteCausador)
		if len(codAgente) < 9 {
			codAgente = "302010100"
		}
	}
	fmt.Fprintf(&sb, "        <codAgntCausador>%s</codAgntCausador>\n", codAgente)
	sb.WriteString("      </agenteCausador>\n")
	sb.WriteString("      <atestado>\n")
	fmt.Fprintf(&sb, "        <dtAtendimento>%s</dtAtendimento>\n", validarDataISO(p.DtAtendimento))
	cid := p.CID10
	if cid == "" {
		cid = "S61.0" // Ferimento de dedos
	}
	fmt.Fprintf(&sb, "        <codCID>%s</codCID>\n", escapeXML(cid))
	descLes := escapeXML(p.DescLesao)
	if descLes == "" {
		descLes = "Contusão / Escoriação superficial."
	}
	fmt.Fprintf(&sb, "        <dscLesao>%s</dscLesao>\n", descLes)
	sb.WriteString("        <emitente>\n")
	nomeMed := escapeXML(p.NomeMedico)
	if nomeMed == "" {
		nomeMed = "Dr. Médico Examinador"
	}
	fmt.Fprintf(&sb, "          <nmEmit>%s</nmEmit>\n", nomeMed)
	sb.WriteString("          <ideOC>1</ideOC>\n")
	crm := p.CRMMedico
	if crm == "" {
		crm = "123456"
	}
	fmt.Fprintf(&sb, "          <nrOC>%s</nrOC>\n", escapeXML(crm))
	ufMed := p.UFMedico
	if ufMed == "" {
		ufMed = "SP"
	}
	fmt.Fprintf(&sb, "          <ufOC>%s</ufOC>\n", validarUF(ufMed))
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
	TipoExame      string // 0 - Admissional, 1 - Periódico, 2 - Retorno, 3 - Mudança Função, 9 - Demissional
	DataASO        string
	ResultadoASO   string // 1 - Apto, 2 - Inapto
	NomeMedico     string
	CRMMedico      string
	UFMedico       string
	NomeCoord      string
	CRMCoord       string
	UFCoord        string
	ObsASO         string
}

// GerarXMLS2220 produz o documento XML de Monitoramento da Saúde do Trabalhador.
func GerarXMLS2220(p ParametrosS2220) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpjLimpo := limpaDigitos(p.CNPJ)
	cpfLimpo := limpaDigitos(p.CPFTrabalhador)
	if p.DataASO == "" {
		p.DataASO = time.Now().Format("2006-01-02")
	}
	if p.TipoExame == "" {
		p.TipoExame = "1" // Periódico
	}
	if p.ResultadoASO == "" {
		p.ResultadoASO = "1" // Apto
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00">` + "\n")
	fmt.Fprintf(&sb, `  <evtMonit Id="%s">`+"\n", p.ID)
	sb.WriteString("    <ideEvento>\n")
	sb.WriteString("      <indRetif>1</indRetif>\n")
	fmt.Fprintf(&sb, "      <tpAmb>%d</tpAmb>\n", p.Ambiente)
	sb.WriteString("      <procEmi>1</procEmi>\n")
	sb.WriteString("      <verProc>1.0.0</verProc>\n")
	sb.WriteString("    </ideEvento>\n")
	sb.WriteString("    <ideEmpregador>\n")
	sb.WriteString("      <tpInsc>1</tpInsc>\n")
	fmt.Fprintf(&sb, "      <nrInsc>%s</nrInsc>\n", cnpjLimpo)
	sb.WriteString("    </ideEmpregador>\n")
	sb.WriteString("    <ideTrabalhador>\n")
	fmt.Fprintf(&sb, "      <cpfTrab>%s</cpfTrab>\n", cpfLimpo)
	sb.WriteString("    </ideTrabalhador>\n")
	sb.WriteString("    <aso>\n")
	fmt.Fprintf(&sb, "      <dtAso>%s</dtAso>\n", validarDataISO(p.DataASO))
	fmt.Fprintf(&sb, "      <tpExameOcup>%s</tpExameOcup>\n", validarDominio(p.TipoExame, "0", "1", "2", "3", "9"))
	sb.WriteString("      <exame>\n")
	fmt.Fprintf(&sb, "        <dtExm>%s</dtExm>\n", validarDataISO(p.DataASO))
	sb.WriteString("        <procRealizado>0295</procRealizado>\n")
	sb.WriteString("      </exame>\n")
	sb.WriteString("      <medico>\n")
	crm := p.CRMMedico
	if crm == "" {
		crm = "100000"
	}
	fmt.Fprintf(&sb, "        <crm>%s</crm>\n", escapeXML(crm))
	uf := p.UFMedico
	if uf == "" {
		uf = "SP"
	}
	fmt.Fprintf(&sb, "        <ufCRM>%s</ufCRM>\n", validarUF(uf))
	sb.WriteString("      </medico>\n")
	sb.WriteString("    </aso>\n")
	sb.WriteString("  </evtMonit>\n")
	sb.WriteString("</eSocial>")

	return sb.String()
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
