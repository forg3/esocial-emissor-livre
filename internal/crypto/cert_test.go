package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gopkcs12 "software.sslmate.com/src/go-pkcs12"
)

// Helper para gerar um certificado e-CNPJ ICP-Brasil de teste autoassinado
func gerarPFXTeste(t *testing.T, senha string, cn string, razaoSocial string) ([]byte, *rsa.PrivateKey, *x509.Certificate) {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("erro ao gerar chave RSA: %v", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("erro ao gerar número de série: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{razaoSocial},
			Country:      []string{"BR"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("erro ao criar certificado x509: %v", err)
	}

	parsedCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		t.Fatalf("erro ao analisar certificado x509: %v", err)
	}

	pfxData, err := gopkcs12.Encode(rand.Reader, privKey, parsedCert, nil, senha)
	if err != nil {
		t.Fatalf("erro ao codificar PKCS#12: %v", err)
	}

	return pfxData, privKey, parsedCert
}

func TestCarregarCertificadoA1(t *testing.T) {
	senha := "senhaSegura123"
	cnpjTeste := "11222333000181" // CNPJ válido com dígitos verificadores
	razaoTeste := "EMPRESA DE TESTE LTDA"
	cnTeste := razaoTeste + ":" + cnpjTeste

	pfxBytes, origKey, origCert := gerarPFXTeste(t, senha, cnTeste, razaoTeste)

	certA1, err := CarregarA1(pfxBytes, senha)
	if err != nil {
		t.Fatalf("CarregarA1 falhou: %v", err)
	}

	if certA1 == nil {
		t.Fatal("certA1 retornado é nulo")
	}

	if certA1.Tipo() != TipoCertificadoA1 {
		t.Errorf("Tipo esperado %s, obtido %s", TipoCertificadoA1, certA1.Tipo())
	}

	if certA1.ChavePrivadaRSA() == nil {
		t.Error("ChavePrivadaRSA retornou nulo")
	}

	if certA1.ChavePrivada() == nil {
		t.Error("ChavePrivada retornou nulo")
	}

	if !origKey.Equal(certA1.ChavePrivadaRSA()) {
		t.Error("Chave privada extraída difere da chave gerada")
	}

	if !origCert.Equal(certA1.CertificadoFolha()) {
		t.Error("Certificado folha extraído difere do gerado")
	}

	if certA1.CNPJ() != cnpjTeste {
		t.Errorf("CNPJ esperado %s, obtido %s", cnpjTeste, certA1.CNPJ())
	}

	if certA1.RazaoSocial() != razaoTeste {
		t.Errorf("Razão Social esperada %s, obtida %s", razaoTeste, certA1.RazaoSocial())
	}

	if !certA1.EstaValido() {
		t.Error("Certificado deveria estar válido")
	}

	tlsCert, err := certA1.TLSCertificate()
	if err != nil {
		t.Fatalf("TLSCertificate falhou: %v", err)
	}

	if len(tlsCert.Certificate) == 0 {
		t.Error("tlsCert não contém dados de certificado")
	}
}

func TestCarregarA1Arquivo(t *testing.T) {
	senha := "testeArquivo456"
	cnpj := "00000000000191"
	razao := "BANCO DO BRASIL S.A."
	cn := razao + ":" + cnpj

	pfxBytes, _, _ := gerarPFXTeste(t, senha, cn, razao)

	tmpDir := t.TempDir()
	caminhoArquivo := filepath.Join(tmpDir, "certificado_teste.pfx")
	if err := os.WriteFile(caminhoArquivo, pfxBytes, 0600); err != nil {
		t.Fatalf("falha ao gravar arquivo temporário: %v", err)
	}

	cert, err := CarregarA1Arquivo(caminhoArquivo, senha)
	if err != nil {
		t.Fatalf("CarregarA1Arquivo falhou: %v", err)
	}

	if cert.CNPJ() != cnpj {
		t.Errorf("CNPJ esperado %s, obtido %s", cnpj, cert.CNPJ())
	}

	if cert.RazaoSocial() != razao {
		t.Errorf("Razão social esperada %s, obtida %s", razao, cert.RazaoSocial())
	}

	// Testa senha incorreta
	_, err = CarregarA1Arquivo(caminhoArquivo, "senhaIncorreta")
	if err == nil {
		t.Error("esperava erro ao fornecer senha incorreta, mas obteve nil")
	}
}

func TestGerenciadorCertificadoConcorrencia(t *testing.T) {
	gerenciador := NovoGerenciadorCertificado()

	if gerenciador.TemCertificado() {
		t.Error("gerenciador recém-criado não deveria ter certificado")
	}

	_, err := gerenciador.Obter()
	if err != ErrCertificadoNaoConfigurado {
		t.Errorf("esperava ErrCertificadoNaoConfigurado, obtido: %v", err)
	}

	pfxBytes, _, _ := gerarPFXTeste(t, "1234", "EMPRESA CONCORRENTE LTDA:11222333000181", "EMPRESA CONCORRENTE LTDA")
	cert, err := CarregarA1(pfxBytes, "1234")
	if err != nil {
		t.Fatalf("falha ao carregar A1: %v", err)
	}

	gerenciador.Definir(cert)

	if !gerenciador.TemCertificado() {
		t.Error("gerenciador deveria ter certificado após Definir()")
	}

	// Teste de concorrência com 50 goroutines
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%3 == 0 {
				_ = gerenciador.TemCertificado()
			} else if id%3 == 1 {
				c, err := gerenciador.Obter()
				if err != nil || c == nil {
					t.Errorf("goroutine %d falhou ao obter certificado: %v", id, err)
				}
			} else {
				gerenciador.Definir(cert)
			}
		}(i)
	}
	wg.Wait()

	gerenciador.Limpar()
	if gerenciador.TemCertificado() {
		t.Error("após Limpar(), gerenciador não deveria ter certificado")
	}
}

func TestValidacaoCNPJ(t *testing.T) {
	testes := []struct {
		cnpj   string
		valido bool
	}{
		{"11.222.333/0001-81", true},
		{"11222333000181", true},
		{"00.000.000/0001-91", true},
		{"00000000000191", true},
		{"12.345.678/0001-95", true},
		{"11.111.111/1111-11", false}, // Dígitos todos iguais
		{"00000000000000", false},     // Dígitos todos iguais
		{"11222333000182", false},     // Dígito verificador errado
		{"123", false},                // Tamanho insuficiente
		{"", false},
	}

	for _, tc := range testes {
		resultado := ValidarCNPJ(tc.cnpj)
		if resultado != tc.valido {
			t.Errorf("ValidarCNPJ(%q) = %v; esperado %v", tc.cnpj, resultado, tc.valido)
		}
	}
}

func TestFormatarCNPJ(t *testing.T) {
	formatado := FormatarCNPJ("11222333000181")
	esperado := "11.222.333/0001-81"
	if formatado != esperado {
		t.Errorf("FormatarCNPJ = %s, esperado %s", formatado, esperado)
	}
}

func TestCertificadoA3Abstracao(t *testing.T) {
	// Cria chave e cert
	pfxBytes, origKey, origCert := gerarPFXTeste(t, "1234", "TITULAR A3:11222333000181", "TITULAR A3")
	certA1, err := CarregarA1(pfxBytes, "1234")
	if err != nil {
		t.Fatalf("falha ao carregar A1 para teste A3: %v", err)
	}

	// Instancia CertificadoA3 recebendo o crypto.Signer (aqui a chave RSA faz o papel da interface PKCS#11)
	certA3 := NovoCertificadoA3(origKey, origCert, certA1.Cadeia())

	if certA3.Tipo() != TipoCertificadoA3 {
		t.Errorf("Tipo esperado %s, obtido %s", TipoCertificadoA3, certA3.Tipo())
	}

	if certA3.CNPJ() != "11222333000181" {
		t.Errorf("CNPJ esperado 11222333000181, obtido %s", certA3.CNPJ())
	}

	if !certA3.EstaValido() {
		t.Error("Certificado A3 deveria estar válido")
	}

	tlsCert, err := certA3.TLSCertificate()
	if err != nil {
		t.Fatalf("falha ao gerar TLSCertificate para A3: %v", err)
	}

	if tlsCert.PrivateKey == nil {
		t.Error("PrivateKey não deveria ser nula em TLSCertificate do A3")
	}

	if certA3.ChavePrivadaRSA() == nil {
		t.Error("ChavePrivadaRSA deveria retornar a chave RSA para este teste")
	}

	if certA3.ValidoDe().IsZero() || certA3.ValidoAte().IsZero() {
		t.Error("ValidoDe e ValidoAte não devem ser zerados")
	}
}

func TestExtracaoDadosTitularVariantes(t *testing.T) {
	// Caso 1: Apenas CN no formato "NOME:CNPJ"
	cert1 := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "EMPRESA ALFA LTDA:11222333000181",
		},
	}
	cnpj1, razao1 := ExtrairDadosTitular(cert1)
	if cnpj1 != "11222333000181" || razao1 != "EMPRESA ALFA LTDA" {
		t.Errorf("Caso 1 falhou: cnpj=%s, razao=%s", cnpj1, razao1)
	}

	// Caso 2: Organization preenchida e CNPJ no CN sem ":"
	cert2 := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "11222333000181",
			Organization: []string{"EMPRESA BETA S.A."},
		},
	}
	cnpj2, razao2 := ExtrairDadosTitular(cert2)
	if cnpj2 != "11222333000181" || razao2 != "EMPRESA BETA S.A." {
		t.Errorf("Caso 2 falhou: cnpj=%s, razao=%s", cnpj2, razao2)
	}

	// Caso 3: Certificado nulo
	cnpj3, razao3 := ExtrairDadosTitular(nil)
	if cnpj3 != "" || razao3 != "" {
		t.Errorf("Caso 3 falhou: cnpj=%s, razao=%s", cnpj3, razao3)
	}
}
