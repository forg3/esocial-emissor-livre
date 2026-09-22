package crypto

import (
	"bytes"
	"strings"
	"testing"
)

// XML de exemplo do evento S-2240 (Condições Ambientais do Trabalho)
const xmlS2240Exemplo = `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00">
    <evtExpRisco Id="ID1000000000000002026092212000000001">
        <ideEvento>
            <indRetif>1</indRetif>
            <tpAmb>2</tpAmb>
            <procEmi>1</procEmi>
            <verProc>1.0.0</verProc>
        </ideEvento>
        <ideEmpregador>
            <tpInsc>1</tpInsc>
            <nrInsc>11222333</nrInsc>
        </ideEmpregador>
        <ideVinculo>
            <cpfTrab>12345678901</cpfTrab>
            <matricula>MAT-001</matricula>
        </ideVinculo>
        <infoExpRisco>
            <dtIniCondicao>2026-09-01</dtIniCondicao>
            <infoAmb>
                <localAmb>1</localAmb>
                <dscSetor>Oficina Central</dscSetor>
                <tpInsc>1</tpInsc>
                <nrInsc>11222333000181</nrInsc>
            </infoAmb>
            <ativDes>
                <dscAtivDes>Manutenção preventiva e corretiva de maquinário.</dscAtivDes>
            </ativDes>
            <agNoc>
                <codAgNoc>01.01.001</codAgNoc>
                <dscAgNoc>Ruído contínuo ou intermitente acima dos limites de tolerância.</dscAgNoc>
                <tpAval>1</tpAval>
            </agNoc>
        </infoExpRisco>
    </evtExpRisco>
</eSocial>`

// XML de exemplo do evento S-2210 (Comunicação de Acidente de Trabalho - CAT)
const xmlS2210Exemplo = `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCAT/v_S_01_03_00">
    <evtCAT Id="ID1000000000000002026092212000000002">
        <ideEvento>
            <indRetif>1</indRetif>
            <tpAmb>2</tpAmb>
            <procEmi>1</procEmi>
            <verProc>1.0.0</verProc>
        </ideEvento>
        <ideEmpregador>
            <tpInsc>1</tpInsc>
            <nrInsc>11222333</nrInsc>
        </ideEmpregador>
        <ideTrabalhador>
            <cpfTrab>98765432100</cpfTrab>
        </ideTrabalhador>
        <cat>
            <dtAcid>2026-09-20</dtAcid>
            <tpAcid>1</tpAcid>
            <hrAcid>1430</hrAcid>
            <hrsTrabAntesAcid>0600</hrsTrabAntesAcid>
            <tpCat>1</tpCat>
            <localAcidente>
                <tpLocal>1</tpLocal>
                <dscLocal>Linha de Montagem A</dscLocal>
            </localAcidente>
        </cat>
    </evtCAT>
</eSocial>`

func TestAssinaturaEValidacaoXML(t *testing.T) {
	pfxBytes, _, _ := gerarPFXTeste(t, "senha123", "EMPRESA DE TESTE LTDA:11222333000181", "EMPRESA DE TESTE LTDA")
	cert, err := CarregarA1(pfxBytes, "senha123")
	if err != nil {
		t.Fatalf("falha ao carregar certificado para assinatura: %v", err)
	}

	// 1. Assina o evento S-2240
	xmlAssinado, err := AssinarXML([]byte(xmlS2240Exemplo), cert)
	if err != nil {
		t.Fatalf("AssinarXML falhou: %v", err)
	}

	if len(xmlAssinado) == 0 {
		t.Fatal("XML assinado retornou vazio")
	}

	strAssinado := string(xmlAssinado)

	// 2. Verifica a presença das tags obrigatórias do padrão eSocial
	elementosObrigatorios := []string{
		`<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"`,
		`<CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"`,
		`<SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"`,
		`<Reference URI="#ID1000000000000002026092212000000001">`,
		`<Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"`,
		`<Transform Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"`,
		`<DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"`,
		`<DigestValue>`,
		`<SignatureValue>`,
		`<KeyInfo>`,
		`<X509Data>`,
		`<X509Certificate>`,
		`</eSocial>`,
	}

	for _, elem := range elementosObrigatorios {
		if !strings.Contains(strAssinado, elem) {
			t.Errorf("XML assinado não contém o elemento obrigatório: %s", elem)
		}
	}

	// 3. Validação criptográfica do XML assinado (deve passar com sucesso)
	if err := ValidarAssinaturaXML(xmlAssinado); err != nil {
		t.Fatalf("ValidarAssinaturaXML falhou no XML recém assinado: %v", err)
	}

	// 4. Teste de violação de integridade (Digest inválido)
	// Se adulterarmos qualquer dado dentro do evento, a validação DEVE falhar
	xmlAdulterado := bytes.Replace(xmlAssinado, []byte("Oficina Central"), []byte("Oficina Alterada"), 1)
	err = ValidarAssinaturaXML(xmlAdulterado)
	if err == nil {
		t.Error("esperava falha de validação após alterar dados do evento assinado, mas obteve nil")
	}
	if !strings.Contains(err.Error(), "DigestValue") && !strings.Contains(err.Error(), "integridade") {
		t.Errorf("esperava erro de DigestValue/integridade, obteve: %v", err)
	}

	// 5. Teste de violação da assinatura (SignatureValue adulterado)
	xmlSigAdulterada := bytes.Replace(xmlAssinado, []byte("<SignatureValue>"), []byte("<SignatureValue>AAAA"), 1)
	err = ValidarAssinaturaXML(xmlSigAdulterada)
	if err == nil {
		t.Error("esperava falha de validação após adulterar SignatureValue, mas obteve nil")
	}
}

func TestAssinaturaEventoCAT(t *testing.T) {
	pfxBytes, _, _ := gerarPFXTeste(t, "senhaCAT", "EMPRESA METALURGICA LTDA:11222333000181", "EMPRESA METALURGICA LTDA")
	cert, err := CarregarA1(pfxBytes, "senhaCAT")
	if err != nil {
		t.Fatalf("falha ao carregar certificado: %v", err)
	}

	xmlAssinado, err := AssinarXML([]byte(xmlS2210Exemplo), cert)
	if err != nil {
		t.Fatalf("AssinarXML falhou para evento CAT: %v", err)
	}

	if err := ValidarAssinaturaXML(xmlAssinado); err != nil {
		t.Fatalf("ValidarAssinaturaXML falhou para evento CAT: %v", err)
	}
}

func TestCanonicalizacaoC14N(t *testing.T) {
	// XML de teste com atributos fora de ordem, comentários e espaços
	xmlEntrada := `<root b="segundo" a="primeiro"><!-- Comentario ignorado --><vazio/><texto>  conteúdo com   espaços  </texto></root>`

	c14n, err := CanonicalizarXML([]byte(xmlEntrada), "")
	if err != nil {
		t.Fatalf("CanonicalizarXML falhou: %v", err)
	}

	resultado := string(c14n)

	// No C14N:
	// 1. Atributos devem estar ordenados: a="primeiro" antes de b="segundo"
	// 2. Comentários devem ser omitidos
	// 3. Tag <vazio/> deve ser transformada em <vazio></vazio>
	// 4. Espaços no texto devem ser preservados
	if !strings.Contains(resultado, `<root a="primeiro" b="segundo">`) {
		t.Errorf("Atributos não foram ordenados corretamente no C14N: %s", resultado)
	}

	if strings.Contains(resultado, "Comentario") {
		t.Error("Comentários não foram removidos na canonicalização C14N")
	}

	if !strings.Contains(resultado, "<vazio></vazio>") {
		t.Errorf("Tag vazia não convertida para <vazio></vazio>: %s", resultado)
	}

	if !strings.Contains(resultado, "<texto>  conteúdo com   espaços  </texto>") {
		t.Errorf("Conteúdo de texto com espaços não preservado: %s", resultado)
	}
}

func TestAssinaturaXMLErroSemId(t *testing.T) {
	xmlSemId := `<eSocial><evtSemId><ideEvento/></evtSemId></eSocial>`
	pfxBytes, _, _ := gerarPFXTeste(t, "123", "TESTE:11222333000181", "TESTE")
	cert, _ := CarregarA1(pfxBytes, "123")

	_, err := AssinarXML([]byte(xmlSemId), cert)
	if err == nil {
		t.Error("esperava erro ao tentar assinar XML sem atributo Id")
	}
}

func TestAssinarXMLCertificadoNulo(t *testing.T) {
	_, err := AssinarXML([]byte(xmlS2240Exemplo), nil)
	if err != ErrCertificadoInvalidoParaAssinatura {
		t.Errorf("esperava ErrCertificadoInvalidoParaAssinatura, obtido: %v", err)
	}
}

func TestReassinaturaXML(t *testing.T) {
	pfxBytes, _, _ := gerarPFXTeste(t, "123", "TESTE REASSINAR:11222333000181", "TESTE REASSINAR")
	cert, _ := CarregarA1(pfxBytes, "123")

	// Primeira assinatura
	assinado1, err := AssinarXML([]byte(xmlS2240Exemplo), cert)
	if err != nil {
		t.Fatalf("primeira assinatura falhou: %v", err)
	}

	// Segunda assinatura sobre o mesmo XML já assinado (deve substituir e validar normalmente)
	assinado2, err := AssinarXML(assinado1, cert)
	if err != nil {
		t.Fatalf("reassinatura falhou: %v", err)
	}

	if err := ValidarAssinaturaXML(assinado2); err != nil {
		t.Fatalf("validação da reassinatura falhou: %v", err)
	}

	// Não deve haver tags duplicadas de Signature
	count := strings.Count(string(assinado2), "<Signature ")
	if count != 1 {
		t.Errorf("esperava exatamente 1 bloco Signature, encontrado %d", count)
	}
}

func TestValidarAssinaturaSemSignature(t *testing.T) {
	err := ValidarAssinaturaXML([]byte(xmlS2240Exemplo))
	if err != ErrTagSignatureAusente {
		t.Errorf("esperava ErrTagSignatureAusente, obtido: %v", err)
	}
}

func TestCanonicalizarXMLComIdInexistente(t *testing.T) {
	_, err := CanonicalizarXML([]byte(xmlS2240Exemplo), "ID_QUE_NAO_EXISTE")
	if err == nil {
		t.Error("esperava erro ao canonicalizar Id inexistente, obteve nil")
	}
}
