package web_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gopkcs12 "software.sslmate.com/src/go-pkcs12"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

// gerarCertificadoTeste cria um .pfx autoassinado para exercitar o fluxo mTLS.
func gerarCertificadoTeste(t *testing.T, dir string) (caminho, senha string) {
	t.Helper()

	chave, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("falha ao gerar chave: %v", err)
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	modelo := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "EMPRESA MODELO TESTE LTDA:12345678000190",
			Organization: []string{"EMPRESA MODELO TESTE LTDA"},
			Country:      []string{"BR"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &modelo, &modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("falha ao criar certificado: %v", err)
	}
	certificado, _ := x509.ParseCertificate(der)

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	senha = "senha-" + hex.EncodeToString(b)

	pfx, err := gopkcs12.Modern.Encode(chave, certificado, nil, senha)
	if err != nil {
		t.Fatalf("falha ao gerar PFX: %v", err)
	}
	caminho = filepath.Join(dir, "perfil-teste.pfx")
	if err := os.WriteFile(caminho, pfx, 0600); err != nil {
		t.Fatalf("falha ao gravar PFX: %v", err)
	}
	return caminho, senha
}

// mockESocial simula o webservice oficial (envio e consulta de lote).
func mockESocial(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		corpo := ""
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		requisicao := buf.String()

		switch {
		case strings.Contains(requisicao, "ConsultarLoteEventos"):
			corpo = `<?xml version="1.0" encoding="utf-8"?>
<ConsultarLoteEventosResponse xmlns="http://www.esocial.gov.br/servicos/empregador/lote/eventos/consulta/v1_1_0">
  <ConsultarLoteEventosResult>
    <eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/envio/consulta/v1_1_0">
      <retornoProcessamentoLoteEventos>
        <status><cdResposta>201</cdResposta><descResposta>Sucesso.</descResposta></status>
        <dadosRecepcaoLote><protocoloEnvio>1.2.202601.0000000000000001</protocoloEnvio></dadosRecepcaoLote>
        <retornoEventos>
          <evento Id="ID1000000000000002026010112000000001">
            <retornoEvento>
              <recibo><nrRecibo>1.2.202601.0000000000000009</nrRecibo></recibo>
              <status><cdResposta>201</cdResposta><descResposta>Sucesso.</descResposta></status>
            </retornoEvento>
          </evento>
        </retornoEventos>
      </retornoProcessamentoLoteEventos>
    </eSocial>
  </ConsultarLoteEventosResult>
</ConsultarLoteEventosResponse>`
		default:
			corpo = `<?xml version="1.0" encoding="utf-8"?>
<EnviarLoteEventosResponse xmlns="http://www.esocial.gov.br/servicos/empregador/lote/eventos/envio/v1_1_0">
  <EnviarLoteEventosResult>
    <eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/envio/v1_1_0">
      <retornoEnvioLoteEventos>
        <status><cdResposta>201</cdResposta><descResposta>Lote recebido com sucesso.</descResposta></status>
        <dadosRecepcaoLote><protocoloEnvio>1.2.202601.0000000000000001</protocoloEnvio><dhRecepcao>2026-01-05T15:00:00</dhRecepcao></dadosRecepcaoLote>
      </retornoEnvioLoteEventos>
    </eSocial>
  </EnviarLoteEventosResult>
</EnviarLoteEventosResponse>`
		}
		_, _ = w.Write([]byte(corpo))
	}))
}

// Sprint 5: fluxo de transmissão real com webservice simulado (mTLS + protocolo + recibo).
func TestTransmissaoRealComWebserviceSimulado(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	// Certificado A1 e webservice simulado
	dirCert := t.TempDir()
	caminhoPFX, senha := gerarCertificadoTeste(t, dirCert)
	servidorWS := mockESocial(t)
	defer servidorWS.Close()
	t.Setenv("ESOCIAL_WS_ENVIO", servidorWS.URL)
	t.Setenv("ESOCIAL_WS_CONSULTA", servidorWS.URL)

	// Configuração em modo real com o certificado carregado
	cfg, _ := db.ObterConfiguracao()
	cfg.CertificadoPath = caminhoPFX
	cfg.ModoTransmissao = "real"
	cfg.Ambiente = 2
	if err := db.SalvarConfiguracao(cfg); err != nil {
		t.Fatalf("falha ao configurar modo real: %v", err)
	}

	// Evento assinado de verdade com o certificado A1 (XMLDSig)
	cert, err := crypto.CarregarA1Arquivo(caminhoPFX, senha)
	if err != nil {
		t.Fatalf("falha ao carregar certificado: %v", err)
	}
	idEvento := "ID1000000000000002026010112000000001"
	xmlEvento := []byte(`<?xml version="1.0" encoding="UTF-8"?><eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00"><evtExpRisco Id="` + idEvento + `"><ideEvento><indRetif>1</indRetif><tpAmb>2</tpAmb><procEmi>1</procEmi><verProc>1.1.0</verProc></ideEvento><ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678000190</nrInsc></ideEmpregador><ideVinculo><cpfTrab>11122233344</cpfTrab></ideVinculo></evtExpRisco></eSocial>`)
	xmlAssinado, err := crypto.AssinarXML(xmlEvento, cert)
	if err != nil {
		t.Fatalf("falha ao assinar evento: %v", err)
	}

	evento := &storage.Evento{
		ID: idEvento, Tipo: "S-2240", Ambiente: 2, Status: "assinado",
		XMLGerado: string(xmlAssinado), XMLAssinado: string(xmlAssinado),
	}
	if err := db.SalvarEvento(evento); err != nil {
		t.Fatalf("falha ao gravar evento: %v", err)
	}

	// Transmissão real
	form := url.Values{"senha_a1": {senha}, "csrf_token": {csrf}}
	req := httptest.NewRequest("POST", "/fila/"+idEvento+"/transmitir", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /fila/{id}/transmitir falhou: %d", rec.Code)
	}
	atualizado, _ := db.ObterEvento(idEvento)
	if atualizado.Status != "transmitido" {
		t.Errorf("status esperado 'transmitido', obtido %q (mensagem: %s)", atualizado.Status, atualizado.MensagemRetorno)
	}
	if atualizado.Protocolo != "1.2.202601.0000000000000001" {
		t.Errorf("protocolo oficial não persistido: %q", atualizado.Protocolo)
	}
	if !strings.Contains(atualizado.MensagemRetorno, "Lote recebido com sucesso") {
		t.Errorf("mensagem oficial não persistida: %q", atualizado.MensagemRetorno)
	}

	// Consulta do recibo real
	reqRecibo := httptest.NewRequest("POST", "/fila/"+idEvento+"/recibo", strings.NewReader(form.Encode()))
	reqRecibo.Host = "localhost:8000"
	reqRecibo.AddCookie(cookie)
	reqRecibo.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqRecibo.Header.Set("X-CSRF-Token", csrf)
	recRecibo := httptest.NewRecorder()
	mux.ServeHTTP(recRecibo, reqRecibo)
	if recRecibo.Code != http.StatusOK {
		t.Fatalf("POST /fila/{id}/recibo falhou: %d", recRecibo.Code)
	}
	comRecibo, _ := db.ObterEvento(idEvento)
	if comRecibo.Recibo != "1.2.202601.0000000000000009" {
		t.Errorf("recibo oficial não persistido: %q (mensagem: %s)", comRecibo.Recibo, comRecibo.MensagemRetorno)
	}
	if comRecibo.Status != "aceito" {
		t.Errorf("status esperado 'aceito' após consulta, obtido %q", comRecibo.Status)
	}
}

// Sprint 5: modo real exige certificado e senha; sem eles a transmissão é recusada.
func TestTransmissaoRealExigeCertificadoESenha(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	cfg, _ := db.ObterConfiguracao()
	cfg.ModoTransmissao = "real"
	cfg.CertificadoPath = ""
	_ = db.SalvarConfiguracao(cfg)

	evento := &storage.Evento{ID: "ID1000000000000002026010112000000002", Tipo: "S-2240", Ambiente: 2, Status: "assinado",
		XMLGerado: "<eSocial/>", XMLAssinado: "<eSocial/>"}
	_ = db.SalvarEvento(evento)

	req := httptest.NewRequest("POST", "/fila/"+evento.ID+"/transmitir", strings.NewReader("senha_a1=x"))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "nenhum certificado A1 configurado") {
		t.Errorf("transmissão real sem certificado deveria ser recusada: %s", rec.Body.String())
	}
	inalterado, _ := db.ObterEvento(evento.ID)
	if inalterado.Status == "transmitido" || inalterado.Status == "aceito" {
		t.Errorf("status não deveria avançar sem certificado: %q", inalterado.Status)
	}
}

// Sprint 5: assinatura simulada é recusada no envio real.
func TestTransmissaoRealRecusaAssinaturaSimulada(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	dirCert := t.TempDir()
	caminhoPFX, senha := gerarCertificadoTeste(t, dirCert)
	cfg, _ := db.ObterConfiguracao()
	cfg.CertificadoPath = caminhoPFX
	cfg.ModoTransmissao = "real"
	_ = db.SalvarConfiguracao(cfg)

	evento := &storage.Evento{
		ID: "ID1000000000000002026010112000000003", Tipo: "S-2240", Ambiente: 2, Status: "assinado",
		XMLGerado:   "<eSocial></eSocial>",
		XMLAssinado: web.SimularAssinatura("<eSocial></eSocial>", "EMPRESA TESTE"),
	}
	_ = db.SalvarEvento(evento)

	form := url.Values{"senha_a1": {senha}, "csrf_token": {csrf}}
	req := httptest.NewRequest("POST", "/fila/"+evento.ID+"/transmitir", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "assinatura SIMULADA") {
		t.Errorf("envio real com assinatura simulada deveria ser recusado: %s", rec.Body.String())
	}
}

// Sprint 5: o modo de transmissão é persistido e exige certificado para virar "real".
func TestModoTransmissaoPersistido(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	// Sem certificado, o modo real é recusado
	form := url.Values{"modo_transmissao": {"real"}, "csrf_token": {csrf}}
	req := httptest.NewRequest("POST", "/configuracao/modo-transmissao", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "Carregue um certificado A1 antes") {
		t.Errorf("modo real sem certificado deveria ser recusado")
	}

	// Com certificado presente, o modo real é aceito
	dirCert := t.TempDir()
	caminhoPFX, _ := gerarCertificadoTeste(t, dirCert)
	cfg, _ := db.ObterConfiguracao()
	cfg.CertificadoPath = caminhoPFX
	_ = db.SalvarConfiguracao(cfg)

	req2 := httptest.NewRequest("POST", "/configuracao/modo-transmissao", strings.NewReader(form.Encode()))
	req2.Host = "localhost:8000"
	req2.AddCookie(cookie)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("X-CSRF-Token", csrf)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if !strings.Contains(rec2.Body.String(), "Modo de transmissão REAL habilitado") {
		t.Errorf("modo real deveria ser habilitado com certificado presente")
	}

	cfgFinal, _ := db.ObterConfiguracao()
	if cfgFinal.ModoTransmissao != "real" {
		t.Errorf("modo de transmissão não persistido: %q", cfgFinal.ModoTransmissao)
	}

	// Voltar ao simulado
	formSimulado := url.Values{"modo_transmissao": {"simulado"}, "csrf_token": {csrf}}
	req3 := httptest.NewRequest("POST", "/configuracao/modo-transmissao", strings.NewReader(formSimulado.Encode()))
	req3.Host = "localhost:8000"
	req3.AddCookie(cookie)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req3.Header.Set("X-CSRF-Token", csrf)
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	cfgSimulado, _ := db.ObterConfiguracao()
	if cfgSimulado.ModoTransmissao != "simulado" {
		t.Errorf("modo deveria voltar para simulado, obtido %q", cfgSimulado.ModoTransmissao)
	}
}
