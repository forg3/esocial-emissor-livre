package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrElementoAssinavelNaoEncontrado ocorre se o XML não tiver nenhum evento com Id
	ErrElementoAssinavelNaoEncontrado = errors.New("nenhum elemento de evento assinável com atributo 'Id' foi encontrado no XML")
	// ErrCertificadoInvalidoParaAssinatura ocorre quando o certificado não pode ser usado
	ErrCertificadoInvalidoParaAssinatura = errors.New("o certificado fornecido é inválido para assinatura digital")
	// ErrAssinaturaJaPresente ocorre se o XML já possuir uma tag <Signature>
	ErrAssinaturaJaPresente = errors.New("o XML já se encontra assinado com a tag <Signature>")
	// ErrTagSignatureAusente ocorre ao tentar validar um XML não assinado
	ErrTagSignatureAusente = errors.New("a tag <Signature> não foi encontrada no documento XML")
	// ErrDigestInvalido indica que o conteúdo do XML foi adulterado após a assinatura
	ErrDigestInvalido = errors.New("o DigestValue não confere: a integridade do evento foi violada")
	// ErrAssinaturaInvalida indica que o SignatureValue é criptograficamente inválido
	ErrAssinaturaInvalida = errors.New("a assinatura digital RSA-SHA256 não pôde ser validada com o certificado anexado")
)

// AssinarXML assina digitalmente um evento XML do eSocial de acordo com o padrão estrito:
// - Inclusive C14N (http://www.w3.org/TR/2001/REC-xml-c14n-20010315)
// - Digest SHA-256 (http://www.w3.org/2001/04/xmlenc#sha256)
// - Transforms: Enveloped Signature e C14N
// - SignatureMethod RSA-SHA256 (http://www.w3.org/2001/04/xmldsig-more#rsa-sha256)
// - Inserção de <Signature> dentro de <eSocial> após a tag do evento
func AssinarXML(xmlBytes []byte, cert Certificado) ([]byte, error) {
	if cert == nil || cert.ChavePrivada() == nil || cert.CertificadoFolha() == nil {
		return nil, ErrCertificadoInvalidoParaAssinatura
	}

	// 1. Constrói o DOM do XML original
	root, err := parseXMLToDOM(xmlBytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar documento XML para assinatura: %w", err)
	}

	// 2. Localiza o elemento do evento que possui o atributo Id (ex: evtExpRisco, evtCAT, etc.)
	eventoNode := root.LocalizarPrimeiroEvento()
	if eventoNode == nil {
		return nil, ErrElementoAssinavelNaoEncontrado
	}

	idValor := eventoNode.ObterValorID()
	if idValor == "" {
		return nil, ErrElementoAssinavelNaoEncontrado
	}

	// 3. Canonicalização C14N do elemento do evento assinado (com herança de namespaces do pai)
	eventoC14N, err := CanonicalizarC14N(eventoNode, true)
	if err != nil {
		return nil, fmt.Errorf("falha ao canonicalizar evento: %w", err)
	}

	// 4. Digest SHA-256 do elemento canonicalizado
	hashEvento := sha256.Sum256(eventoC14N)
	digestValue := base64.StdEncoding.EncodeToString(hashEvento[:])

	// 5. Monta o bloco <SignedInfo> com o URI #Id e o DigestValue
	signedInfoRaw := fmt.Sprintf(
		`<SignedInfo xmlns="%s">`+
			`<CanonicalizationMethod Algorithm="%s"></CanonicalizationMethod>`+
			`<SignatureMethod Algorithm="%s"></SignatureMethod>`+
			`<Reference URI="#%s">`+
			`<Transforms>`+
			`<Transform Algorithm="%s"></Transform>`+
			`<Transform Algorithm="%s"></Transform>`+
			`</Transforms>`+
			`<DigestMethod Algorithm="%s"></DigestMethod>`+
			`<DigestValue>%s</DigestValue>`+
			`</Reference>`+
			`</SignedInfo>`,
		NamespaceXMLDSig,
		CanonicalizationMethodC14N,
		SignatureMethodRSASHA256,
		idValor,
		TransformEnveloped,
		TransformC14N,
		DigestMethodSHA256,
		digestValue,
	)

	// 6. Canonicaliza o <SignedInfo>
	signedInfoDOM, err := parseXMLToDOM([]byte(signedInfoRaw))
	if err != nil {
		return nil, fmt.Errorf("falha ao analisar SignedInfo para canonicalização: %w", err)
	}

	signedInfoC14N, err := CanonicalizarC14N(signedInfoDOM, false)
	if err != nil {
		return nil, fmt.Errorf("falha ao canonicalizar SignedInfo: %w", err)
	}

	// 7. Calcula o hash SHA-256 do SignedInfo canonicalizado e assina com RSA
	hashSignedInfo := sha256.Sum256(signedInfoC14N)
	sigBytes, err := cert.ChavePrivada().Sign(rand.Reader, hashSignedInfo[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("falha na operação criptográfica RSA-SHA256: %w", err)
	}
	signatureValue := base64.StdEncoding.EncodeToString(sigBytes)

	// 8. Codifica o certificado folha em Base64 para a tag <X509Certificate>
	x509CertBase64 := base64.StdEncoding.EncodeToString(cert.CertificadoFolha().Raw)

	// 9. Monta a estrutura da tag <Signature> completa
	signatureBlock := fmt.Sprintf(
		`<Signature xmlns="%s">`+
			`<SignedInfo>`+
			`<CanonicalizationMethod Algorithm="%s"/>`+
			`<SignatureMethod Algorithm="%s"/>`+
			`<Reference URI="#%s">`+
			`<Transforms>`+
			`<Transform Algorithm="%s"/>`+
			`<Transform Algorithm="%s"/>`+
			`</Transforms>`+
			`<DigestMethod Algorithm="%s"/>`+
			`<DigestValue>%s</DigestValue>`+
			`</Reference>`+
			`</SignedInfo>`+
			`<SignatureValue>%s</SignatureValue>`+
			`<KeyInfo>`+
			`<X509Data>`+
			`<X509Certificate>%s</X509Certificate>`+
			`</X509Data>`+
			`</KeyInfo>`+
			`</Signature>`,
		NamespaceXMLDSig,
		CanonicalizationMethodC14N,
		SignatureMethodRSASHA256,
		idValor,
		TransformEnveloped,
		TransformC14N,
		DigestMethodSHA256,
		digestValue,
		signatureValue,
		x509CertBase64,
	)

	// 10. Insere a tag <Signature> antes do fechamento da tag </eSocial>
	xmlFinal, err := inserirAssinaturaNoXML(xmlBytes, signatureBlock)
	if err != nil {
		return nil, err
	}

	return xmlFinal, nil
}

// inserirAssinaturaNoXML adiciona o bloco <Signature> antes da tag final de fechamento de </eSocial>.
func inserirAssinaturaNoXML(xmlBytes []byte, signatureXML string) ([]byte, error) {
	strXML := string(xmlBytes)

	// Se já contiver assinatura, substitui ou remove anterior
	if strings.Contains(strXML, "<Signature ") || strings.Contains(strXML, "<Signature>") || strings.Contains(strXML, ":Signature>") || strings.Contains(strXML, ":Signature ") {
		reSig := regexp.MustCompile(`(?s)<(?:\w+:)?Signature[\s>].*?</(?:\w+:)?Signature>`)
		strXML = reSig.ReplaceAllString(strXML, "")
	}

	// Localiza o fechamento da tag </eSocial> ou raiz
	idx := strings.LastIndex(strXML, "</eSocial>")
	if idx != -1 {
		novo := strXML[:idx] + signatureXML + strXML[idx:]
		return []byte(novo), nil
	}

	// Se não for tag eSocial, insere antes da última tag de fechamento </...>
	reLastTag := regexp.MustCompile(`</\s*[^>]+>\s*$`)
	loc := reLastTag.FindStringIndex(strXML)
	if len(loc) == 2 {
		novo := strXML[:loc[0]] + signatureXML + strXML[loc[0]:]
		return []byte(novo), nil
	}

	return nil, errors.New("não foi possível localizar a tag de fechamento para inserção da assinatura")
}

// ValidarAssinaturaXML extrai a assinatura e o certificado X.509 presentes no XML,
// verifica a integridade do elemento assinado (Digest SHA-256) e valida criptograficamente
// a assinatura digital RSA-SHA256.
func ValidarAssinaturaXML(xmlBytes []byte) error {
	strXML := string(xmlBytes)
	if !strings.Contains(strXML, "<Signature") && !strings.Contains(strXML, ":Signature") {
		return ErrTagSignatureAusente
	}

	root, err := parseXMLToDOM(xmlBytes)
	if err != nil {
		return fmt.Errorf("falha ao analisar XML assinado: %w", err)
	}

	// Localiza a tag Signature
	sigNode := root.LocalizarElementoPorTag("Signature")
	if sigNode == nil {
		return ErrTagSignatureAusente
	}

	// Extrai o nó SignedInfo
	signedInfoNode := sigNode.LocalizarElementoPorTag("SignedInfo")
	if signedInfoNode == nil {
		return errors.New("elemento <SignedInfo> ausente na assinatura")
	}

	// Extrai Reference e URI
	refNode := signedInfoNode.LocalizarElementoPorTag("Reference")
	if refNode == nil {
		return errors.New("elemento <Reference> ausente em <SignedInfo>")
	}

	var uri string
	for _, a := range refNode.Attrs {
		if a.Local == "URI" || a.Name == "URI" {
			uri = a.Value
			break
		}
	}
	if uri == "" {
		return errors.New("atributo URI ausente em <Reference>")
	}
	idAlvo := strings.TrimPrefix(uri, "#")

	// Extrai DigestValue esperado
	digestNode := refNode.LocalizarElementoPorTag("DigestValue")
	if digestNode == nil {
		return errors.New("elemento <DigestValue> ausente em <Reference>")
	}
	digestEsperado := strings.TrimSpace(digestNode.ObterTexto())

	// Extrai SignatureValue
	sigValNode := sigNode.LocalizarElementoPorTag("SignatureValue")
	if sigValNode == nil {
		return errors.New("elemento <SignatureValue> ausente na assinatura")
	}
	signatureBase64 := strings.TrimSpace(sigValNode.ObterTexto())
	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("SignatureValue Base64 inválido: %w", err)
	}

	// Extrai X509Certificate
	certNode := sigNode.LocalizarElementoPorTag("X509Certificate")
	if certNode == nil {
		return errors.New("elemento <X509Certificate> ausente em <KeyInfo>")
	}
	certBase64 := strings.TrimSpace(certNode.ObterTexto())
	// Remove eventuais espaços/quebras de linha
	certBase64 = strings.ReplaceAll(certBase64, "\n", "")
	certBase64 = strings.ReplaceAll(certBase64, "\r", "")
	certBase64 = strings.ReplaceAll(certBase64, " ", "")

	certBytes, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return fmt.Errorf("X509Certificate Base64 inválido: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("falha ao analisar certificado X.509 da assinatura: %w", err)
	}

	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("a chave pública contida no certificado não é RSA")
	}

	// Localiza o elemento alvo referenciado pelo Id
	alvoNode := root.LocalizarElementoPorID(idAlvo)
	if alvoNode == nil {
		return fmt.Errorf("elemento assinado com Id '%s' não encontrado", idAlvo)
	}

	// Canonicaliza o elemento assinado com C14N
	alvoC14N, err := CanonicalizarC14N(alvoNode, true)
	if err != nil {
		return fmt.Errorf("falha ao canonicalizar elemento assinado: %w", err)
	}

	// Confere o Digest SHA-256
	hashAlvo := sha256.Sum256(alvoC14N)
	digestCalculado := base64.StdEncoding.EncodeToString(hashAlvo[:])

	if digestCalculado != digestEsperado {
		return fmt.Errorf("%w: calculado=%s, esperado=%s", ErrDigestInvalido, digestCalculado, digestEsperado)
	}

	// Canonicaliza o SignedInfo
	// Para garantir canonicalização correta de SignedInfo com namespace padrão
	signedInfoC14N, err := CanonicalizarC14N(signedInfoNode, true)
	if err != nil {
		return fmt.Errorf("falha ao canonicalizar SignedInfo para validação: %w", err)
	}

	// Valida a assinatura RSA-SHA256
	hashSignedInfo := sha256.Sum256(signedInfoC14N)
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashSignedInfo[:], sigBytes)
	if err != nil {
		// Tenta canonicalização sem ser raiz de subconjunto (caso o SignedInfo já possua xmlns próprio)
		signedInfoC14NAlt, errAlt := CanonicalizarC14N(signedInfoNode, false)
		if errAlt == nil {
			hashAlt := sha256.Sum256(signedInfoC14NAlt)
			if rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashAlt[:], sigBytes) == nil {
				return nil
			}
		}
		return fmt.Errorf("%w: %v", ErrAssinaturaInvalida, err)
	}

	return nil
}

// LocalizarElementoPorTag busca recursivamente o primeiro elemento com o nome informado (ignorando prefixo).
func (n *domNode) LocalizarElementoPorTag(tagName string) *domNode {
	if n == nil {
		return nil
	}
	nomeLocal := n.Name
	if idx := strings.Index(nomeLocal, ":"); idx != -1 {
		nomeLocal = nomeLocal[idx+1:]
	}
	if nomeLocal == tagName {
		return n
	}
	for _, child := range n.Children {
		if child.Type == domNodeElement {
			if found := child.LocalizarElementoPorTag(tagName); found != nil {
				return found
			}
		}
	}
	return nil
}

// ObterTexto retorna a concatenação de todos os nós de texto filhos imediatos do elemento.
func (n *domNode) ObterTexto() string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	for _, child := range n.Children {
		if child.Type == domNodeText {
			sb.WriteString(child.Text)
		}
	}
	return sb.String()
}
