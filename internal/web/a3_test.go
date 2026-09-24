package web_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
)

// Sprint 6: a camada criptográfica A3 assina XMLDSig com um Signer arbitrário
// (é exatamente o contrato usado por tokens PKCS#11, onde a chave não sai do hardware).
func TestAssinaturaComCertificadoA3ViaSigner(t *testing.T) {
	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("falha ao gerar chave: %v", err)
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	modelo := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "EMPRESA TOKEN LTDA:11222333000181",
			Organization: []string{"EMPRESA TOKEN LTDA"},
			Country:      []string{"BR"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &modelo, &modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("falha ao criar certificado: %v", err)
	}
	folha, _ := x509.ParseCertificate(der)

	// Certificado A3 montado a partir de um Signer (mesmo contrato do token PKCS#11)
	certA3 := crypto.NovoCertificadoA3(chave, folha, nil)
	if certA3.Tipo() != "A3" {
		t.Errorf("tipo esperado A3, obtido %q", certA3.Tipo())
	}
	if certA3.CNPJ() != "11222333000181" {
		t.Errorf("CNPJ extraído incorretamente do certificado A3: %q", certA3.CNPJ())
	}

	xmlEvento := []byte(`<?xml version="1.0" encoding="UTF-8"?><eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00"><evtExpRisco Id="ID1000000000000002026010112000000001"><ideVinculo><cpfTrab>11122233344</cpfTrab></ideVinculo></evtExpRisco></eSocial>`)
	assinado, err := crypto.AssinarXML(xmlEvento, certA3)
	if err != nil {
		t.Fatalf("falha ao assinar com certificado A3: %v", err)
	}
	if !strings.Contains(string(assinado), "<Signature") {
		t.Errorf("XML assinado com A3 não contém o envelope XMLDSig")
	}
	if strings.Contains(string(assinado), "SIMULACAO") {
		t.Errorf("assinatura A3 não pode conter marcador de simulação")
	}
}

// Sprint 6: o token A3 é exposto ao operador com mensagem honesta quando o driver
// PKCS#11 não está incluído na compilação (build padrão).
func TestTokenA3SemDriverInformaComoHabilitar(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	form := url.Values{
		"modulo_pkcs11": {"/usr/lib/opensc-pkcs11.so"},
		"pin_a3":        {"1234"},
		"csrf_token":    {csrf},
	}
	req := httptest.NewRequest("POST", "/configuracao/token-a3", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /configuracao/token-a3 falhou: %d", rec.Code)
	}
	corpo := rec.Body.String()
	if !crypto.PKCS11Disponivel() {
		if !strings.Contains(corpo, "pkcs11") {
			t.Errorf("resposta deveria indicar como habilitar o driver PKCS#11: %s", corpo)
		}
		if !strings.Contains(corpo, "go build -tags pkcs11") {
			t.Errorf("resposta deveria trazer o comando de build com a tag pkcs11")
		}
	} else if !strings.Contains(corpo, "token") {
		t.Errorf("com driver PKCS#11 o teste deveria tentar abrir o token: %s", corpo)
	}
}

// Sprint 6: sem módulo informado, o teste orienta o operador.
func TestTokenA3ExigeModulo(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	form := url.Values{"modulo_pkcs11": {""}, "csrf_token": {srv.TokenCSRF()}}
	req := httptest.NewRequest("POST", "/configuracao/token-a3", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "módulo PKCS#11") {
		t.Errorf("mensagem deveria orientar o preenchimento do módulo PKCS#11")
	}
}

// Sprint 6: o CPF/CNPJ do titular é extraído corretamente de certificados A3.
func TestCertificadoA3ExtraiTitular(t *testing.T) {
	chave, _ := rsa.GenerateKey(rand.Reader, 2048)
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	modelo := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "PESSOA FISICA TESTE:11222333000181"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour * 24),
		BasicConstraintsValid: true,
	}
	der, _ := x509.CreateCertificate(rand.Reader, &modelo, &modelo, &chave.PublicKey, chave)
	folha, _ := x509.ParseCertificate(der)

	cert := crypto.NovoCertificadoA3(chave, folha, nil)
	if cert.CNPJ() != "11222333000181" {
		t.Errorf("documento do titular deveria ser extraído do certificado, obtido %q", cert.CNPJ())
	}
	if cert.RazaoSocial() != "PESSOA FISICA TESTE" {
		t.Errorf("titular incorreto: %q", cert.RazaoSocial())
	}
	if cert.ChavePrivada() == nil {
		t.Errorf("chave privada do certificado A3 não disponível para assinatura")
	}
	_ = hex.EncodeToString([]byte{})
	if cert.ChavePrivada() == nil {
		t.Errorf("assinador do token indisponível")
	}
}
