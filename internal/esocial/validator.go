package esocial

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/forg3/esocial-emissor-livre/internal/data"
)

var (
	xsdTempDir     string
	xsdInitOnce    sync.Once
	xsdInitErr     error
	xmllintDisponivel bool
	checkXmllintOnce sync.Once
)

const dummySignatureXML = `  <Signature xmlns="http://www.w3.org/2000/09/xmldsig#">
    <SignedInfo>
      <CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315" />
      <SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256" />
      <Reference URI="">
        <Transforms>
          <Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature" />
        </Transforms>
        <DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256" />
        <DigestValue>MA==</DigestValue>
      </Reference>
    </SignedInfo>
    <SignatureValue>MA==</SignatureValue>
  </Signature>`

// inicializarDiretorioXSD extrai os arquivos XSD embutidos para um diretório temporário
// para que includes e imports relativos (como tipos.xsd e xmldsig-core-schema.xsd) funcionem.
func inicializarDiretorioXSD() {
	dir, err := os.MkdirTemp("", "esocial_xsd_*")
	if err != nil {
		xsdInitErr = fmt.Errorf("falha ao criar pasta temporária para esquemas XSD: %w", err)
		return
	}
	xsdTempDir = dir

	arquivos := []string{
		"tipos.xsd",
		"xmldsig-core-schema.xsd",
		"evtInfoEmpregador.xsd",
		"evtExpRisco.xsd",
		"evtCAT.xsd",
		"evtMonit.xsd",
	}

	for _, arq := range arquivos {
		conteudo, err := data.EmbeddedFS.ReadFile("xsd/" + arq)
		if err != nil {
			xsdInitErr = fmt.Errorf("falha ao ler XSD embutido %s: %w", arq, err)
			return
		}
		dest := filepath.Join(dir, arq)
		if err := os.WriteFile(dest, conteudo, 0644); err != nil {
			xsdInitErr = fmt.Errorf("falha ao gravar XSD temporário %s: %w", dest, err)
			return
		}
	}
}

func verificarXmllint() bool {
	checkXmllintOnce.Do(func() {
		_, err := exec.LookPath("xmllint")
		xmllintDisponivel = (err == nil)
	})
	return xmllintDisponivel
}

// ObterNomeXSD mapeia o código do evento (ex: "S-2240") para o nome do arquivo XSD.
func ObterNomeXSD(tipoEvento string) (string, error) {
	tipoLimpo := strings.ToUpper(strings.TrimSpace(tipoEvento))
	switch tipoLimpo {
	case "S-1000", "1000", "EVTINFOEMPREGADOR":
		return "evtInfoEmpregador.xsd", nil
	case "S-2240", "2240", "EVTEXPRISCO":
		return "evtExpRisco.xsd", nil
	case "S-2210", "2210", "EVTCAT":
		return "evtCAT.xsd", nil
	case "S-2220", "2220", "EVTMONIT":
		return "evtMonit.xsd", nil
	default:
		return "", fmt.Errorf("tipo de evento eSocial desconhecido: %s", tipoEvento)
	}
}

// ValidarXSD valida o conteúdo XML do evento contra os schemas oficiais do eSocial S-1.3.
// Se o XML não contiver a tag <Signature>, uma assinatura dummy é adicionada temporariamente
// apenas para a conferência estrutural do XSD (visto que o XSD oficial exige assinatura digital).
func ValidarXSD(xmlBytes []byte, tipoEvento string) error {
	if len(bytes.TrimSpace(xmlBytes)) == 0 {
		return errors.New("conteúdo XML vazio para validação")
	}

	// 1. Validação estrutural de sintaxe XML nativa em Go
	var envelope struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(xmlBytes, &envelope); err != nil {
		return fmt.Errorf("XML malformado: %w", err)
	}
	if envelope.XMLName.Local != "eSocial" {
		return fmt.Errorf("elemento raiz deve ser <eSocial>, encontrado <%s>", envelope.XMLName.Local)
	}

	nomeXSD, err := ObterNomeXSD(tipoEvento)
	if err != nil {
		return err
	}

	// Prepara XML para validação: se não tiver Signature, anexa antes de fechar </eSocial>
	xmlValidacao := xmlBytes
	if !bytes.Contains(xmlBytes, []byte("<Signature")) {
		fechamento := []byte("</eSocial>")
		idx := bytes.LastIndex(xmlBytes, fechamento)
		if idx != -1 {
			var buf bytes.Buffer
			buf.Write(xmlBytes[:idx])
			buf.WriteString("\n")
			buf.WriteString(dummySignatureXML)
			buf.WriteString("\n</eSocial>")
			xmlValidacao = buf.Bytes()
		}
	}

	// Se xmllint estiver disponível, executa a validação W3C rigorosa
	if verificarXmllint() {
		xsdInitOnce.Do(inicializarDiretorioXSD)
		if xsdInitErr != nil {
			return xsdInitErr
		}

		caminhoXSD := filepath.Join(xsdTempDir, nomeXSD)

		tmpXML, err := os.CreateTemp("", "esocial_val_*.xml")
		if err != nil {
			return fmt.Errorf("falha ao criar arquivo XML temporário: %w", err)
		}
		defer os.Remove(tmpXML.Name())

		if _, err := tmpXML.Write(xmlValidacao); err != nil {
			tmpXML.Close()
			return fmt.Errorf("falha ao escrever XML temporário: %w", err)
		}
		tmpXML.Close()

		cmd := exec.Command("xmllint", "--noout", "--schema", caminhoXSD, tmpXML.Name())
		out, err := cmd.CombinedOutput()
		if err != nil {
			msgErro := strings.TrimSpace(string(out))
			// Remove o caminho do arquivo temporário para melhor leitura
			msgErro = strings.ReplaceAll(msgErro, tmpXML.Name()+":", "Linha ")
			msgErro = strings.ReplaceAll(msgErro, caminhoXSD+":", "")
			return fmt.Errorf("erro de conformidade com o esquema oficial S-1.3 (%s): %s", nomeXSD, msgErro)
		}
	}

	return nil
}
