package web_test

// Correções do pente fino de 26/09/2026: XML importado passa por validação; evento
// rejeitado não é assinado; a assinatura A1 real existe na interface.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

func postar(t *testing.T, h http.Handler, rota string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", rota, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestImportarASOMalformadoNaoEntraComoPronto(t *testing.T) {
	_, db, h, limpar := prepararServidorTeste(t)
	defer limpar()

	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	fw, _ := mw.CreateFormFile("arquivo_xml_aso", "aso.xml")
	_, _ = fw.Write([]byte(`<?xml version="1.0"?><!DOCTYPE r [<!ENTITY x SYSTEM "file:///etc/passwd">]><eSocial><evtMonit><ideVinculo><cpfTrab>&x;</cpfTrab></ideVinculo></evtMonit></eSocial>`))
	mw.Close()
	req := httptest.NewRequest("POST", "/eventos/s2220/importar-xml", &corpo)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	h.ServeHTTP(httptest.NewRecorder(), req)

	eventos, _ := db.ListarEventos()
	for _, e := range eventos {
		if e.Status == "pronto" {
			t.Fatalf("XML malformado entrou na fila como 'pronto' (evento %s)", e.ID)
		}
	}
}

func TestEventoRejeitadoNaoEAssinado(t *testing.T) {
	_, db, h, limpar := prepararServidorTeste(t)
	defer limpar()
	ev := &storage.Evento{ID: "ID-REJEITADO", Tipo: "S-2240", Status: "rejeitado",
		XMLGerado: "<eSocial/>", CriadoEm: time.Now(), AtualizadoEm: time.Now()}
	_ = db.SalvarEvento(ev)

	postar(t, h, "/fila/ID-REJEITADO/assinar", url.Values{})
	postar(t, h, "/fila/lote/assinar", url.Values{})

	depois, _ := db.ObterEvento("ID-REJEITADO")
	if depois.Status != "rejeitado" || depois.XMLAssinado != "" {
		t.Fatalf("evento rejeitado foi assinado: status %q", depois.Status)
	}
}

func TestAssinaturaA1RealPelaInterface(t *testing.T) {
	_, db, h, limpar := prepararServidorTeste(t)
	defer limpar()

	caminho, senha := gerarCertificadoTeste(t, t.TempDir())
	cfg, _ := db.ObterConfiguracao()
	cfg.CertificadoPath = caminho
	_ = db.SalvarConfiguracao(cfg)

	colab := &storage.Colaborador{ID: "colab-a1", Nome: "Técnico", CPF: "11144477735", Matricula: "M1", Status: "ativo"}
	_ = db.SalvarColaborador(colab)
	postar(t, h, "/eventos/s2240", url.Values{"colaborador_id": {colab.ID}, "dt_inicio": {"2026-01-01"},
		"codigo_risco": {"09.01.001"}, "nome_risco": {"Ausência de risco"}, "tipo_avaliacao": {"2"},
		"nome_resp": {"Eng. Técnico"}, "cpf_resp": {"12312312312"}, "num_registro": {"12345"}})
	eventos, _ := db.ListarEventos()
	if len(eventos) == 0 {
		t.Fatal("evento S-2240 não foi gerado")
	}
	id := eventos[0].ID
	postar(t, h, "/fila/"+id+"/validar", url.Values{})
	postar(t, h, "/fila/"+id+"/assinar", url.Values{"senha_a1": {senha}})

	ev, _ := db.ObterEvento(id)
	if ev.Status != "assinado" {
		t.Fatalf("status esperado 'assinado', obtido %q (%s)", ev.Status, ev.MensagemRetorno)
	}
	if strings.Contains(strings.ToUpper(ev.MensagemRetorno), "SIMULAD") {
		t.Fatalf("com certificado e senha a assinatura deveria ser real: %s", ev.MensagemRetorno)
	}
	if err := crypto.ValidarAssinaturaXML([]byte(ev.XMLAssinado)); err != nil {
		t.Fatalf("assinatura XMLDSig inválida: %v", err)
	}
}
