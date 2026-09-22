package esocial

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var regexPlaceholder = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

// GerarXMLEventoGenerico constrói o arquivo XML oficial no padrão eSocial S-1.3
// para qualquer um dos 36 eventos do catálogo, preenchendo o identificador de 36 caracteres,
// ideEvento, ideEmpregador e os campos específicos informados no mapa de dados.
func GerarXMLEventoGenerico(codigo string, dados map[string]string) ([]byte, error) {
	evt := ObterEventoPorCodigo(codigo)
	if evt == nil {
		return nil, fmt.Errorf("evento '%s' não localizado no catálogo oficial do eSocial", codigo)
	}

	if dados == nil {
		dados = make(map[string]string)
	}

	// Normaliza busca de chaves case-insensitive no mapa de dados
	obterDado := func(chaves ...string) string {
		for _, k := range chaves {
			if v, ok := dados[k]; ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
			if v, ok := dados[strings.ToLower(k)]; ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
			if v, ok := dados[strings.ToUpper(k)]; ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	// 1. Identificação do Empregador
	tpInscStr := obterDado("tpInsc", "TP_INSC", "tipoInscricao")
	if tpInscStr == "" {
		tpInscStr = "1" // Padrão CNPJ
	}
	tpInsc, _ := strconv.Atoi(tpInscStr)
	if tpInsc < 1 || tpInsc > 2 {
		tpInsc = 1
	}

	nrInsc := LimparDocumento(obterDado("nrInsc", "NR_INSC", "cnpj", "cpf"))
	if nrInsc == "" {
		nrInsc = "12345678000195" // Exemplo padrão
	}

	// 2. Identificação do Evento
	tpAmb := obterDado("tpAmb", "TP_AMB", "ambiente")
	if tpAmb == "" {
		tpAmb = "2" // Padrão Produção Restrita (Homologação)
	}

	procEmi := obterDado("procEmi", "PROC_EMI")
	if procEmi == "" {
		procEmi = "1" // 1 - Aplicativo do empregador
	}

	verProc := obterDado("verProc", "VER_PROC")
	if verProc == "" {
		verProc = VersaoAplicativo // "EmissorLivre-1.0" (máx 20 caracteres)
	}

	indRetif := obterDado("indRetif", "IND_RETIF")
	if indRetif == "" {
		indRetif = "1" // 1 - Original
	}

	seqStr := obterDado("sequencial", "seq")
	seq := 1
	if seqStr != "" {
		if s, err := strconv.Atoi(seqStr); err == nil && s > 0 {
			seq = s
		}
	}

	// 3. Geração determinística do ID oficial (36 caracteres)
	var id string
	if idFornecido := obterDado("Id", "id", "ID"); idFornecido != "" {
		id = idFornecido
	} else {
		id = GerarIDEvento(tpInsc, nrInsc, seq)
	}

	// 4. Preenchimento de Placeholders
	template := evt.TemplateXML

	// Mapa de substituições prioritárias
	subs := map[string]string{
		"ID":        id,
		"TP_AMB":    tpAmb,
		"PROC_EMI":  procEmi,
		"VER_PROC":  verProc,
		"TP_INSC":   fmt.Sprintf("%d", tpInsc),
		"NR_INSC":   nrInsc,
		"IND_RETIF": indRetif,
	}

	// Mescla dados específicos informados pelo chamador
	for k, v := range dados {
		subs[strings.ToUpper(k)] = v
		// Suporta formatos snake_case e camelCase
		subs[k] = v
	}

	// Substituições padrão para campos obrigatórios comuns caso não informados
	defaultsComuns := map[string]string{
		"INI_VALID":           time.Now().Format("2006-01"),
		"DT_INI_CONDICAO":     time.Now().Format("2006-01-02"),
		"DT_ACID":             time.Now().Format("2006-01-02"),
		"DT_ASO":              time.Now().Format("2006-01-02"),
		"DT_ADM":              time.Now().Format("2006-01-02"),
		"DT_NASCTO":           "1990-01-01",
		"DT_ALTERACAO":        time.Now().Format("2006-01-02"),
		"DT_INICIO":           time.Now().Format("2006-01-02"),
		"DT_TERM":             time.Now().Format("2006-01-02"),
		"DT_DESLIG":           time.Now().Format("2006-01-02"),
		"PER_APUR":            time.Now().Format("2006-01"),
		"CLASS_TRIB":          "01",
		"IND_DES_FOLHA":       "0",
		"IND_OPT_REG_ELETRON": "1",
		"LOCAL_AMB":           "1",
		"DSC_SETOR":           "Setor Operacional",
		"TP_INSC_AMB":         fmt.Sprintf("%d", tpInsc),
		"NR_INSC_AMB":         nrInsc,
		"DSC_ATIV_DES":        "Atividades gerais do cargo",
		"COD_AG_NOC":          "09.01.001",
		"CPF_RESP":            "12345678909",
		"TP_ACID":             "1",
		"HR_ACID":             "0900",
		"TP_CAT":              "1",
		"IND_CAT_OBITO":       "N",
		"IND_COMUN_POLICIA":   "N",
		"COD_SIT_GERADORA":    "303000000",
		"INICIAT_CAT":         "1",
		"TP_LOCAL":            "1",
		"DSC_LOGRAD":          "Rua Principal",
		"NR_LOGRAD":           "100",
		"COD_PARTE_ATING":     "752000000",
		"LATERALIDADE":        "0",
		"COD_AGNT_CAUSADOR":   "302000000",
		"DT_ATENDIMENTO":      time.Now().Format("2006-01-02"),
		"HR_ATENDIMENTO":      "1000",
		"IND_INTERNACAO":      "N",
		"DUR_TRAT":            "1",
		"IND_AFAST":           "N",
		"DSC_LESAO":           "702000000",
		"COD_CID":             "S61",
		"NM_EMIT":             "Médico Examinador",
		"IDE_OC":              "1",
		"NR_OC":               "123456",
		"UF_OC":               "SP",
		"TP_EXAME_OCUP":       "1",
		"RES_ASO":             "1",
		"NM_MED":              "Médico do Trabalho",
		"NR_CRM":              "123456",
		"UF_CRM":              "SP",
		"CPF_TRAB":            "12345678909",
		"NM_TRAB":             "Trabalhador Exemplo",
		"SEXO":                "M",
		"RACA_COR":            "1",
		"EST_CIV":             "1",
		"GRAU_INSTR":          "07",
		"MATRICULA":           "MAT-0001",
		"COD_CARGO":           "CARGO-01",
		"COD_CBO":             "411010",
		"COD_CATEG":           "101",
		"NAT_ATIVIDADE":       "1",
		"TP_REG_TRAB":         "1",
		"TP_REG_PREV":         "1",
		"MTV_DESLIG":          "02",
		"COD_RUBR":            "1000",
		"IDE_TAB_RUBR":        "TAB01",
		"DSC_RUBR":            "Salário Base",
		"NAT_RUBR":            "1000",
		"TP_RUBR":             "1",
		"COD_INC_CP":          "11",
		"COD_INC_IRRF":        "11",
		"COD_INC_FGTS":        "11",
		"COD_LOTACAO":         "LOT01",
		"TP_LOTACAO":          "01",
		"FPAS":                "507",
		"COD_TERCS":           "0079",
		"CNAE_PREP":           "6201501",
		"TP_INSC_ESTAB":       fmt.Sprintf("%d", tpInsc),
		"NR_INSC_ESTAB":       nrInsc,
		"TEM_REMUN":           "S",
		"TEM_PGTOS":           "S",
		"CPF_BENEF":           "12345678909",
		"NM_BENEF":            "Beneficiário Exemplo",
		"DT_PGTO":             time.Now().Format("2006-01-02"),
		"TP_PGTO":             "1",
		"PER_REF":             time.Now().Format("2006-01"),
		"IDE_DM_DEV":          "DEV01",
		"VR_LIQ":              "2500.00",
		"VR_BC_CP":            "2500.00",
		"IND_SUBST_PATR":      "1",
		"PERC_RED_CONTRIB":    "0.00",
		"TP_PROC":             "1",
		"NR_PROC":             "000000120265020001",
		"IND_AUTORIA":         "1",
		"ORIGEM_PROC":         "1",
		"NR_PROC_TRAB":        "000000120265020001",
		"DT_SENT":             time.Now().Format("2006-01-02"),
		"UF_VARA":             "SP",
		"COD_MUNIC":           "3550308",
		"ID_VARA":             "01",
		"NR_BENEFICIO":        "BEN-2026-001",
		"DT_INI":              time.Now().Format("2006-01-02"),
		"TP_BENEFICIO":        "01",
		"TP_EVENTO_EXCLUIR":   "S-2200",
		"NR_REC_EVT":          "1.1.0000000000000000001",
		"DT_INI_AFAST":        time.Now().Format("2006-01-02"),
		"COD_MOT_AFAST":       "01",
		"DT_INI_CESSAO":       time.Now().Format("2006-01-02"),
		"CNPJ_CESS":           "98765432000180",
		"RESP_REMUN":          "1",
		"TP_REINT":            "1",
		"DT_EFET_RETORNO":     time.Now().Format("2006-01-02"),
		"DT_EFEITO":           time.Now().Format("2006-01-02"),
	}

	for k, v := range defaultsComuns {
		if _, ok := subs[k]; !ok {
			subs[k] = v
		}
	}

	// Executa a substituição de placeholders
	xmlFinal := regexPlaceholder.ReplaceAllStringFunc(template, func(match string) string {
		tag := match[2 : len(match)-2]
		if valor, ok := subs[tag]; ok {
			return valor
		}
		if valor, ok := subs[strings.ToUpper(tag)]; ok {
			return valor
		}
		return ""
	})

	resultado := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + strings.TrimSpace(xmlFinal) + "\n"

	// Validação de sintaxe XML nativa
	var envelope struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(resultado), &envelope); err != nil {
		return nil, fmt.Errorf("XML gerado para %s possui sintaxe malformada: %w", codigo, err)
	}

	if envelope.XMLName.Local != "eSocial" {
		return nil, errors.New("o elemento raiz do XML gerado deve ser <eSocial>")
	}

	return bytes.TrimSpace([]byte(resultado)), nil
}
