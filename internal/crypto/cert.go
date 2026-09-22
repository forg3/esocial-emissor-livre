package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	xpkcs12 "golang.org/x/crypto/pkcs12"
	gopkcs12 "software.sslmate.com/src/go-pkcs12"
)

var (
	// ErrCertificadoNaoConfigurado indica ausência de certificado configurado no gerenciador.
	ErrCertificadoNaoConfigurado = errors.New("nenhum certificado digital configurado no sistema")
	// ErrChavePrivadaNaoRSA indica que o certificado não contém uma chave privada RSA.
	ErrChavePrivadaNaoRSA = errors.New("a chave privada do certificado digital não é do tipo RSA")
	// ErrCertificadoExpirado indica que a data atual ultrapassou a validade do certificado.
	ErrCertificadoExpirado = errors.New("o certificado digital está expirado")
	// ErrCertificadoAindaNaoValido indica que o certificado ainda não atingiu o início da sua vigência.
	ErrCertificadoAindaNaoValido = errors.New("o certificado digital ainda não é válido")
)

// Tipos de certificados suportados
const (
	TipoCertificadoA1 = "A1"
	TipoCertificadoA3 = "A3"
)

// OIDs oficiais ICP-Brasil
var (
	// OID 2.16.76.1.3.3: Informações da Pessoa Jurídica no e-CNPJ ICP-Brasil
	oidDadosPJ = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 3}
	// OID 2.16.76.1.3.14: CNPJ direto do titular
	oidCNPJ = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 14}
	// OID 2.16.76.1.3.1: CPF do titular da pessoa física (e-CPF)
	oidCPF = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 1}
)

// Certificado é a interface universal para manipulação de certificados digitais
// no eSocial, abstraindo implementações em memória (A1) e tokens/smartcards (A3/PKCS#11).
type Certificado interface {
	// ChavePrivada retorna o crypto.Signer para operações de assinatura criptográfica.
	ChavePrivada() crypto.Signer
	// ChavePrivadaRSA retorna a chave RSA direta se disponível em memória (comum em A1).
	ChavePrivadaRSA() *rsa.PrivateKey
	// CertificadoFolha retorna o certificado X.509 do titular.
	CertificadoFolha() *x509.Certificate
	// Cadeia retorna a lista de certificados intermediários e de autoridade certificadora (se disponíveis).
	Cadeia() []*x509.Certificate
	// TLSCertificate prepara e retorna o tls.Certificate configurado para mTLS no cliente SOAP.
	TLSCertificate() (tls.Certificate, error)
	// CNPJ retorna o número do CNPJ extraído do certificado (somente dígitos).
	CNPJ() string
	// RazaoSocial retorna a Razão Social ou Nome do Titular identificado no certificado.
	RazaoSocial() string
	// ValidoDe retorna o início do período de vigência.
	ValidoDe() time.Time
	// ValidoAte retorna o fim do período de vigência.
	ValidoAte() time.Time
	// EstaValido verifica se a data atual está dentro do período de vigência.
	EstaValido() bool
	// Tipo retorna a classificação do certificado ("A1" ou "A3").
	Tipo() string
}

// CertificadoA1 implementa a interface Certificado para certificados de arquivo em memória.
type CertificadoA1 struct {
	chavePrivada *rsa.PrivateKey
	folha        *x509.Certificate
	cadeia       []*x509.Certificate
	cnpj         string
	razaoSocial  string
}

// NovoCertificadoA1 cria uma nova instância de CertificadoA1.
func NovoCertificadoA1(privKey *rsa.PrivateKey, folha *x509.Certificate, cadeia []*x509.Certificate) *CertificadoA1 {
	cnpj, razaoSocial := ExtrairDadosTitular(folha)
	return &CertificadoA1{
		chavePrivada: privKey,
		folha:        folha,
		cadeia:       cadeia,
		cnpj:         cnpj,
		razaoSocial:  razaoSocial,
	}
}

func (c *CertificadoA1) ChavePrivada() crypto.Signer {
	return c.chavePrivada
}

func (c *CertificadoA1) ChavePrivadaRSA() *rsa.PrivateKey {
	return c.chavePrivada
}

func (c *CertificadoA1) CertificadoFolha() *x509.Certificate {
	return c.folha
}

func (c *CertificadoA1) Cadeia() []*x509.Certificate {
	return c.cadeia
}

func (c *CertificadoA1) TLSCertificate() (tls.Certificate, error) {
	if c.folha == nil || c.chavePrivada == nil {
		return tls.Certificate{}, errors.New("certificado ou chave privada ausentes no CertificadoA1")
	}

	rawCerts := make([][]byte, 0, len(c.cadeia)+1)
	rawCerts = append(rawCerts, c.folha.Raw)
	for _, cert := range c.cadeia {
		rawCerts = append(rawCerts, cert.Raw)
	}

	return tls.Certificate{
		Certificate: rawCerts,
		PrivateKey:  c.chavePrivada,
		Leaf:        c.folha,
	}, nil
}

func (c *CertificadoA1) CNPJ() string {
	return c.cnpj
}

func (c *CertificadoA1) RazaoSocial() string {
	return c.razaoSocial
}

func (c *CertificadoA1) ValidoDe() time.Time {
	if c.folha == nil {
		return time.Time{}
	}
	return c.folha.NotBefore
}

func (c *CertificadoA1) ValidoAte() time.Time {
	if c.folha == nil {
		return time.Time{}
	}
	return c.folha.NotAfter
}

func (c *CertificadoA1) EstaValido() bool {
	if c.folha == nil {
		return false
	}
	agora := time.Now()
	return agora.After(c.folha.NotBefore) && agora.Before(c.folha.NotAfter)
}

func (c *CertificadoA1) Tipo() string {
	return TipoCertificadoA1
}

// CertificadoA3 implementa a interface Certificado para hardware tokens, HSM ou PKCS#11.
type CertificadoA3 struct {
	signer      crypto.Signer
	folha       *x509.Certificate
	cadeia      []*x509.Certificate
	cnpj        string
	razaoSocial string
}

// NovoCertificadoA3 cria uma instância de CertificadoA3 recebendo um crypto.Signer arbitrário
// (como os fornecidos por bibliotecas PKCS#11 ou wrappers de hardware criptográfico).
func NovoCertificadoA3(signer crypto.Signer, folha *x509.Certificate, cadeia []*x509.Certificate) *CertificadoA3 {
	cnpj, razaoSocial := ExtrairDadosTitular(folha)
	return &CertificadoA3{
		signer:      signer,
		folha:       folha,
		cadeia:      cadeia,
		cnpj:        cnpj,
		razaoSocial: razaoSocial,
	}
}

func (c *CertificadoA3) ChavePrivada() crypto.Signer {
	return c.signer
}

func (c *CertificadoA3) ChavePrivadaRSA() *rsa.PrivateKey {
	if rsaKey, ok := c.signer.(*rsa.PrivateKey); ok {
		return rsaKey
	}
	return nil
}

func (c *CertificadoA3) CertificadoFolha() *x509.Certificate {
	return c.folha
}

func (c *CertificadoA3) Cadeia() []*x509.Certificate {
	return c.cadeia
}

func (c *CertificadoA3) TLSCertificate() (tls.Certificate, error) {
	if c.folha == nil || c.signer == nil {
		return tls.Certificate{}, errors.New("certificado folha ou signer ausentes no CertificadoA3")
	}

	rawCerts := make([][]byte, 0, len(c.cadeia)+1)
	rawCerts = append(rawCerts, c.folha.Raw)
	for _, cert := range c.cadeia {
		rawCerts = append(rawCerts, cert.Raw)
	}

	return tls.Certificate{
		Certificate: rawCerts,
		PrivateKey:  c.signer,
		Leaf:        c.folha,
	}, nil
}

func (c *CertificadoA3) CNPJ() string {
	return c.cnpj
}

func (c *CertificadoA3) RazaoSocial() string {
	return c.razaoSocial
}

func (c *CertificadoA3) ValidoDe() time.Time {
	if c.folha == nil {
		return time.Time{}
	}
	return c.folha.NotBefore
}

func (c *CertificadoA3) ValidoAte() time.Time {
	if c.folha == nil {
		return time.Time{}
	}
	return c.folha.NotAfter
}

func (c *CertificadoA3) EstaValido() bool {
	if c.folha == nil {
		return false
	}
	agora := time.Now()
	return agora.After(c.folha.NotBefore) && agora.Before(c.folha.NotAfter)
}

func (c *CertificadoA3) Tipo() string {
	return TipoCertificadoA3
}

// CarregarA1Arquivo lê os bytes do arquivo .pfx / .p12 no caminho informado e decodifica o certificado A1.
func CarregarA1Arquivo(caminho string, senha string) (Certificado, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo de certificado %s: %w", caminho, err)
	}
	return CarregarA1(dados, senha)
}

// CarregarA1 decodifica os dados em formato PKCS#12 (.pfx / .p12) usando golang.org/x/crypto/pkcs12
// e software.sslmate.com/src/go-pkcs12, extraindo a chave privada RSA, o certificado folha e a cadeia.
func CarregarA1(pfxDados []byte, senha string) (Certificado, error) {
	var privKeyInterface crypto.PrivateKey
	var folha *x509.Certificate
	var cadeia []*x509.Certificate

	// Tentativa 1: DecodeChain via software.sslmate.com/src/go-pkcs12 (suporta PBKDF2 moderno, SHA-256 e legados)
	k, c, chain, errChain := gopkcs12.DecodeChain(pfxDados, senha)
	if errChain == nil && k != nil && c != nil {
		privKeyInterface = k
		folha = c
		cadeia = chain
	} else {
		// Tentativa 2: golang.org/x/crypto/pkcs12.Decode (padrão frozen de x/crypto)
		kDec, cDec, errDec := xpkcs12.Decode(pfxDados, senha)
		if errDec == nil && kDec != nil && cDec != nil {
			privKeyInterface = kDec
			folha = cDec

			// Tenta extrair a cadeia intermediária via ToPEM do x/crypto
			if blocks, errPEM := xpkcs12.ToPEM(pfxDados, senha); errPEM == nil {
				for _, block := range blocks {
					if block.Type == "CERTIFICATE" {
						parsed, errP := x509.ParseCertificate(block.Bytes)
						if errP == nil && !parsed.Equal(folha) {
							cadeia = append(cadeia, parsed)
						}
					}
				}
			}
		} else {
			// Se ambos falharam, tenta ToPEM para decodificar blocos brutos
			blocks, errPEM := xpkcs12.ToPEM(pfxDados, senha)
			if errPEM != nil {
				if errChain != nil {
					return nil, fmt.Errorf("falha ao decodificar PKCS#12: %w (tentativa legada: %v)", errChain, errDec)
				}
				return nil, fmt.Errorf("falha ao decodificar PKCS#12: %w", errDec)
			}

			for _, b := range blocks {
				switch b.Type {
				case "PRIVATE KEY", "RSA PRIVATE KEY":
					if privKeyInterface == nil {
						if pk, err := x509.ParsePKCS8PrivateKey(b.Bytes); err == nil {
							privKeyInterface = pk
						} else if pkRsa, err := x509.ParsePKCS1PrivateKey(b.Bytes); err == nil {
							privKeyInterface = pkRsa
						}
					}
				case "CERTIFICATE":
					parsed, err := x509.ParseCertificate(b.Bytes)
					if err == nil {
						if folha == nil && !parsed.IsCA {
							folha = parsed
						} else {
							cadeia = append(cadeia, parsed)
						}
					}
				}
			}
		}
	}

	if folha == nil {
		return nil, errors.New("nenhum certificado X.509 encontrado no arquivo PKCS#12")
	}

	if privKeyInterface == nil {
		return nil, errors.New("nenhuma chave privada encontrada no arquivo PKCS#12")
	}

	rsaKey, ok := privKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: recebido %T", ErrChavePrivadaNaoRSA, privKeyInterface)
	}

	return NovoCertificadoA1(rsaKey, folha, cadeia), nil
}

// ExtrairDadosTitular analisa os campos e extensões do certificado X.509
// e extrai o CNPJ (se presente) e a Razão Social/Nome do titular.
func ExtrairDadosTitular(cert *x509.Certificate) (cnpj string, razaoSocial string) {
	if cert == nil {
		return "", ""
	}

	// 1. Tentar extrair CNPJ das extensões oficiais da ICP-Brasil
	cnpj = extrairCNPJDasExtensoes(cert)

	// 2. Extrair Razão Social e CNPJ a partir do Subject (Common Name / Organization)
	cn := cert.Subject.CommonName
	org := ""
	if len(cert.Subject.Organization) > 0 {
		org = cert.Subject.Organization[0]
	}

	// Padrão típico ICP-Brasil: "RAZAO SOCIAL OU NOME:00000000000191"
	if strings.Contains(cn, ":") {
		partes := strings.Split(cn, ":")
		if razaoSocial == "" {
			razaoSocial = strings.TrimSpace(partes[0])
		}
		if cnpj == "" && len(partes) > 1 {
			candidato := ApenasDigitos(partes[1])
			if ValidarCNPJ(candidato) {
				cnpj = candidato
			}
		}
	}

	// Se ainda não encontrou CNPJ, busca padrão de 14 dígitos no CN
	if cnpj == "" {
		re := regexp.MustCompile(`\b\d{14}\b`)
		if match := re.FindString(cn); match != "" && ValidarCNPJ(match) {
			cnpj = match
		}
	}

	// Se não encontrou no CN, tenta na Organization ou SerialNumber
	if cnpj == "" {
		re := regexp.MustCompile(`\b\d{14}\b`)
		if match := re.FindString(cert.Subject.SerialNumber); match != "" && ValidarCNPJ(match) {
			cnpj = match
		}
		if match := re.FindString(org); match != "" && ValidarCNPJ(match) {
			cnpj = match
		}
	}

	// Define Razão Social final
	if razaoSocial == "" {
		if org != "" {
			razaoSocial = org
		} else if cn != "" {
			razaoSocial = cn
		}
	}

	return cnpj, razaoSocial
}

// extrairCNPJDasExtensoes inspeciona as extensões ASN.1 do certificado procurando os OIDs ICP-Brasil.
func extrairCNPJDasExtensoes(cert *x509.Certificate) string {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oidCNPJ) {
			cnpj := extrairDigitosCNPJ(ext.Value)
			if ValidarCNPJ(cnpj) {
				return cnpj
			}
		}

		if ext.Id.Equal(oidDadosPJ) {
			// OID 2.16.76.1.3.3 geralmente encapsula dados cadastrais em ASN.1
			// onde uma das sequências de 14 dígitos é o CNPJ
			cnpj := extrairDigitosCNPJ(ext.Value)
			if ValidarCNPJ(cnpj) {
				return cnpj
			}
		}
	}

	return ""
}

// extrairDigitosCNPJ vasculha uma sequência de bytes em busca de qualquer sequência de 14 dígitos válidos como CNPJ.
func extrairDigitosCNPJ(val []byte) string {
	// Tenta decodificar como ASN.1 string
	var strVal string
	if _, err := asn1.Unmarshal(val, &strVal); err == nil {
		val = []byte(strVal)
	}

	texto := string(val)
	re := regexp.MustCompile(`\d{14}`)
	matches := re.FindAllString(texto, -1)
	for _, m := range matches {
		if ValidarCNPJ(m) {
			return m
		}
	}
	return ""
}

// ApenasDigitos remove qualquer caractere não numérico da string.
func ApenasDigitos(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ValidarCNPJ realiza a validação algorítmica do CNPJ segundo o cálculo oficial de módulo 11.
func ValidarCNPJ(cnpj string) bool {
	cnpj = ApenasDigitos(cnpj)
	if len(cnpj) != 14 {
		return false
	}

	// Rejeita sequências conhecidas de dígitos repetidos
	todosIguais := true
	for i := 1; i < 14; i++ {
		if cnpj[i] != cnpj[0] {
			todosIguais = false
			break
		}
	}
	if todosIguais {
		return false
	}

	// Primeiro dígito verificador
	pesos1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	soma1 := 0
	for i := 0; i < 12; i++ {
		soma1 += int(cnpj[i]-'0') * pesos1[i]
	}
	resto1 := soma1 % 11
	d1 := 0
	if resto1 >= 2 {
		d1 = 11 - resto1
	}
	if int(cnpj[12]-'0') != d1 {
		return false
	}

	// Segundo dígito verificador
	pesos2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	soma2 := 0
	for i := 0; i < 13; i++ {
		soma2 += int(cnpj[i]-'0') * pesos2[i]
	}
	resto2 := soma2 % 11
	d2 := 0
	if resto2 >= 2 {
		d2 = 11 - resto2
	}

	return int(cnpj[13]-'0') == d2
}

// FormatarCNPJ formata uma string numérica de 14 dígitos no padrão XX.XXX.XXX/XXXX-XX.
func FormatarCNPJ(cnpj string) string {
	limpo := ApenasDigitos(cnpj)
	if len(limpo) != 14 {
		return cnpj
	}
	return fmt.Sprintf("%s.%s.%s/%s-%s",
		limpo[0:2], limpo[2:5], limpo[5:8], limpo[8:12], limpo[12:14])
}

// GerenciadorCertificado mantém uma referência em memória com controle de concorrência thread-safe
// para o certificado digital ativo na sessão local da aplicação.
type GerenciadorCertificado struct {
	mu   sync.RWMutex
	cert Certificado
}

// NovoGerenciadorCertificado cria um gerenciador de certificado thread-safe.
func NovoGerenciadorCertificado() *GerenciadorCertificado {
	return &GerenciadorCertificado{}
}

// CarregarA1 decodifica o PKCS#12 informado e o armazena como certificado ativo.
func (g *GerenciadorCertificado) CarregarA1(pfxDados []byte, senha string) (Certificado, error) {
	cert, err := CarregarA1(pfxDados, senha)
	if err != nil {
		return nil, err
	}
	g.Definir(cert)
	return cert, nil
}

// CarregarA1Arquivo lê o arquivo .pfx / .p12 do disco e o armazena como certificado ativo.
func (g *GerenciadorCertificado) CarregarA1Arquivo(caminho string, senha string) (Certificado, error) {
	cert, err := CarregarA1Arquivo(caminho, senha)
	if err != nil {
		return nil, err
	}
	g.Definir(cert)
	return cert, nil
}

// Definir armazena a instância informada de Certificado com proteção de escrita.
func (g *GerenciadorCertificado) Definir(cert Certificado) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cert = cert
}

// Obter retorna o certificado ativo ou ErrCertificadoNaoConfigurado com proteção de leitura.
func (g *GerenciadorCertificado) Obter() (Certificado, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.cert == nil {
		return nil, ErrCertificadoNaoConfigurado
	}
	return g.cert, nil
}

// TemCertificado informa atomicamente se há um certificado carregado na sessão.
func (g *GerenciadorCertificado) TemCertificado() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cert != nil
}

// Limpar remove o certificado da memória, liberando referências e zerando chaves.
func (g *GerenciadorCertificado) Limpar() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cert = nil
}
