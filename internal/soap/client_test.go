package soap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
)

// Helper para gerar um Certificado de teste
func gerarCertificadoMock(t *testing.T) crypto.Certificado {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("falha ao gerar RSA: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "EMPRESA TESTE:11222333000181",
			Organization: []string{"EMPRESA TESTE"},
		},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("falha ao criar cert: %v", err)
	}

	folha, _ := x509.ParseCertificate(certBytes)
	return crypto.NovoCertificadoA1(privKey, folha, nil)
}

func TestURLsAmbiente(t *testing.T) {
	cert := gerarCertificadoMock(t)
	cliente, err := NovoClienteSOAP(cert)
	if err != nil {
		t.Fatalf("NovoClienteSOAP falhou: %v", err)
	}

	// Produção (1)
	if cliente.ObterURLEnvio(AmbienteProducao) != URLProducaoEnvio {
		t.Errorf("URL Produção Envio incorreta: %s", cliente.ObterURLEnvio(AmbienteProducao))
	}
	if cliente.ObterURLConsulta(AmbienteProducao) != URLProducaoConsulta {
		t.Errorf("URL Produção Consulta incorreta: %s", cliente.ObterURLConsulta(AmbienteProducao))
	}

	// Produção Restrita (2)
	if cliente.ObterURLEnvio(AmbienteProducaoRestrita) != URLRestritaEnvio {
		t.Errorf("URL Restrita Envio incorreta: %s", cliente.ObterURLEnvio(AmbienteProducaoRestrita))
	}
	if cliente.ObterURLConsulta(AmbienteProducaoRestrita) != URLRestritaConsulta {
		t.Errorf("URL Restrita Consulta incorreta: %s", cliente.ObterURLConsulta(AmbienteProducaoRestrita))
	}
}

func TestEnviarLoteSucessoMock(t *testing.T) {
	respostaXMLMock := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <EnviarLoteEventosResponse xmlns="http://www.esocial.gov.br/servicos/empregador/lote/eventos/envio/v1_1_0">
      <EnviarLoteEventosResult>
        <eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/envio/retornoEnvio/v1_1_0">
          <retornoEnvioLoteEventos>
            <status>
              <cdResposta>201</cdResposta>
              <descResposta>Lote recebido com sucesso.</descResposta>
            </status>
            <dadosRecepcaoLote>
              <dhRecepcao>2026-09-22T12:00:00</dhRecepcao>
              <protocoloEnvio>1.2.202609.0000000000000000001</protocoloEnvio>
            </dadosRecepcaoLote>
          </retornoEnvioLoteEventos>
        </eSocial>
      </EnviarLoteEventosResult>
    </EnviarLoteEventosResponse>
  </soap:Body>
</soap:Envelope>`

	servidorMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("esperava método POST, recebido: %s", r.Method)
		}
		if r.Header.Get("SOAPAction") != SOAPActionEnvioLote {
			t.Errorf("SOAPAction incorreta: %s", r.Header.Get("SOAPAction"))
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respostaXMLMock))
	}))
	defer servidorMock.Close()

	cert := gerarCertificadoMock(t)
	cliente, err := NovoClienteSOAP(cert, ComURLPersonalizada(servidorMock.URL, servidorMock.URL))
	if err != nil {
		t.Fatalf("falha ao criar cliente: %v", err)
	}

	xmlEvento := `<eSocial><evtExpRisco Id="ID1000000000000002026092212000000001"><ideEvento/></evtExpRisco></eSocial>`
	resp, err := cliente.EnviarLote(context.Background(), []byte(xmlEvento), AmbienteProducaoRestrita)
	if err != nil {
		t.Fatalf("EnviarLote falhou: %v", err)
	}

	if !resp.Sucesso {
		t.Errorf("esperava sucesso=true, obtido=false (código %d)", resp.CodigoResposta)
	}

	if resp.CodigoResposta != 201 {
		t.Errorf("código resposta esperado 201, obtido %d", resp.CodigoResposta)
	}

	if resp.ProtocoloEnvio != "1.2.202609.0000000000000000001" {
		t.Errorf("protocolo incorreto: %s", resp.ProtocoloEnvio)
	}
}

func TestConsultarLoteSucessoMock(t *testing.T) {
	respostaXMLMock := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ConsultarLoteEventosResponse xmlns="http://www.esocial.gov.br/servicos/empregador/lote/eventos/consulta/v1_1_0">
      <ConsultarLoteEventosResult>
        <eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/consulta/retornoProcessamento/v1_3_0">
          <retornoProcessamentoLoteEventos>
            <status>
              <cdResposta>201</cdResposta>
              <descResposta>Lote processado com sucesso.</descResposta>
            </status>
            <dadosRecepcaoLote>
              <protocoloEnvio>1.2.202609.0000000000000000001</protocoloEnvio>
            </dadosRecepcaoLote>
            <retornoEventos>
              <evento Id="ID1000000000000002026092212000000001">
                <retornoEvento>
                  <eSocial>
                    <retornoEvento>
                      <recibo>
                        <nrRecibo>1.2.0000000000000000001</nrRecibo>
                      </recibo>
                    </retornoEvento>
                  </eSocial>
                </retornoEvento>
              </evento>
            </retornoEventos>
          </retornoProcessamentoLoteEventos>
        </eSocial>
      </ConsultarLoteEventosResult>
    </ConsultarLoteEventosResponse>
  </soap:Body>
</soap:Envelope>`

	servidorMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("SOAPAction") != SOAPActionConsultaLote {
			t.Errorf("SOAPAction incorreta: %s", r.Header.Get("SOAPAction"))
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respostaXMLMock))
	}))
	defer servidorMock.Close()

	cert := gerarCertificadoMock(t)
	cliente, err := NovoClienteSOAP(cert, ComURLPersonalizada(servidorMock.URL, servidorMock.URL))
	if err != nil {
		t.Fatalf("falha ao criar cliente: %v", err)
	}

	resp, err := cliente.ConsultarLote(context.Background(), "1.2.202609.0000000000000000001", AmbienteProducaoRestrita)
	if err != nil {
		t.Fatalf("ConsultarLote falhou: %v", err)
	}

	if !resp.Sucesso {
		t.Errorf("esperava sucesso=true, obtido=false")
	}

	if len(resp.Eventos) != 1 {
		t.Fatalf("esperava 1 evento no recibo, obtidos %d", len(resp.Eventos))
	}

	if resp.Eventos[0].NumeroRecibo != "1.2.0000000000000000001" {
		t.Errorf("número de recibo incorreto: %s", resp.Eventos[0].NumeroRecibo)
	}
}

func TestSOAPFaultTratamento(t *testing.T) {
	faultMock := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <soap:Fault>
      <faultcode>soap:Client</faultcode>
      <faultstring>Certificado do transmissor inválido ou revogado</faultstring>
    </soap:Fault>
  </soap:Body>
</soap:Envelope>`

	servidorMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(faultMock))
	}))
	defer servidorMock.Close()

	cert := gerarCertificadoMock(t)
	cliente, err := NovoClienteSOAP(cert, ComURLPersonalizada(servidorMock.URL, servidorMock.URL))
	if err != nil {
		t.Fatalf("falha ao criar cliente: %v", err)
	}

	_, err = cliente.EnviarLote(context.Background(), []byte("<eSocial/>"), AmbienteProducaoRestrita)
	if err == nil {
		t.Error("esperava erro por SOAP Fault, mas obteve nil")
	}

	if !strings.Contains(err.Error(), "Certificado do transmissor") {
		t.Errorf("mensagem de erro inesperada: %v", err)
	}
}
