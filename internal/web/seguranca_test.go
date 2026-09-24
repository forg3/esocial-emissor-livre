package web_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/esocial"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

// prepararServidorSeguranca cria o servidor com autenticação conhecida e devolve o
// handler completo (com middlewares) SEM sessão, para testar os controles de acesso.
func prepararServidorSeguranca(t *testing.T) (*web.Servidor, *storage.DB, http.Handler, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "esocial-seg-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}
	db, err := storage.Abrir(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("falha ao abrir sqlite: %v", err)
	}
	srv, err := web.NovoServidorComOpcoes(db, web.OpcoesAuth{Senha: senhaTeste, Diretorio: tmpDir})
	if err != nil {
		t.Fatalf("falha ao instanciar servidor: %v", err)
	}
	configurarEmpresaTeste(t, db)
	return srv, db, srv.Rotas(), func() {
		db.Fechar()
		os.RemoveAll(tmpDir)
	}
}

func autenticar(t *testing.T, mux http.Handler, srv *web.Servidor) *http.Cookie {
	t.Helper()
	form := url.Values{"senha": {senhaTeste}, "csrf_token": {srv.TokenCSRF()}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "esocial_sessao" {
			return c
		}
	}
	t.Fatalf("login falhou com status %d", rec.Code)
	return nil
}

// C-01: nenhuma rota sensível responde sem sessão válida.
func TestRotasExigemAutenticacao(t *testing.T) {
	_, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	getRotas := []string{"/", "/dashboard", "/fila", "/colaboradores", "/configuracao", "/eventos",
		"/fila/ID1/xml", "/fila/ID1/detalhes", "/api/riscos/busca?q=ruido", "/api/cbos/busca?q=x"}
	for _, rota := range getRotas {
		req := httptest.NewRequest("GET", rota, nil)
		req.Host = "localhost:8000"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Errorf("GET %s deveria redirecionar para /login, obtido %d", rota, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/login" {
			t.Errorf("GET %s deveria redirecionar para /login, obtido %q", rota, loc)
		}
	}

	postRotas := []string{"/configuracao", "/configuracao/testar", "/colaboradores",
		"/eventos/s2240", "/fila/lote/assinar", "/fila/lote/transmitir"}
	for _, rota := range postRotas {
		req := httptest.NewRequest("POST", rota, strings.NewReader(""))
		req.Host = "localhost:8000"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("POST %s sem sessão deveria retornar 401, obtido %d", rota, rec.Code)
		}
	}

	// Requisição HTMX sem sessão recebe HX-Redirect
	reqHTMX := httptest.NewRequest("GET", "/fila", nil)
	reqHTMX.Host = "localhost:8000"
	reqHTMX.Header.Set("HX-Request", "true")
	recHTMX := httptest.NewRecorder()
	mux.ServeHTTP(recHTMX, reqHTMX)
	if recHTMX.Code != http.StatusUnauthorized || recHTMX.Header().Get("HX-Redirect") != "/login" {
		t.Errorf("HTMX sem sessão deveria retornar 401 + HX-Redirect, obtido %d/%q",
			recHTMX.Code, recHTMX.Header().Get("HX-Redirect"))
	}
}

// A-01: escrita exige token anti-CSRF e origem compatível.
func TestCSRFExigidoEmEscrita(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	// 1. Sem token -> 403
	req := httptest.NewRequest("POST", "/colaboradores", strings.NewReader("nome=X&cpf=11122233344"))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST sem token anti-CSRF deveria retornar 403, obtido %d", rec.Code)
	}

	// 2. Token inválido -> 403
	form := url.Values{"nome": {"X"}, "cpf": {"11122233344"}, "csrf_token": {"token-errado"}}
	req2 := httptest.NewRequest("POST", "/colaboradores", strings.NewReader(form.Encode()))
	req2.Host = "localhost:8000"
	req2.AddCookie(cookie)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Errorf("POST com token inválido deveria retornar 403, obtido %d", rec2.Code)
	}

	// 3. Origem externa -> 403
	form3 := url.Values{"nome": {"X"}, "cpf": {"11122233344"}, "csrf_token": {srv.TokenCSRF()}}
	req3 := httptest.NewRequest("POST", "/colaboradores", strings.NewReader(form3.Encode()))
	req3.Host = "localhost:8000"
	req3.AddCookie(cookie)
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req3.Header.Set("Origin", "https://evil.example")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusForbidden {
		t.Errorf("POST com Origin externa deveria retornar 403, obtido %d", rec3.Code)
	}

	// 4. Token válido no cabeçalho (padrão HTMX) -> aceito
	req4 := httptest.NewRequest("POST", "/colaboradores", strings.NewReader("nome=Valido&cpf=11122233344"))
	req4.Host = "localhost:8000"
	req4.AddCookie(cookie)
	req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req4.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Errorf("POST com token válido deveria ser aceito, obtido %d", rec4.Code)
	}
}

// C-01: cabeçalho Host fora da allowlist é rejeitado (anti DNS rebinding).
func TestHostNaoAutorizadoRejeitado(t *testing.T) {
	_, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMisdirectedRequest {
		t.Errorf("Host externo deveria retornar 421, obtido %d", rec.Code)
	}
}

// B-03: cabeçalhos de segurança presentes em todas as respostas.
func TestCabecalhosDeSeguranca(t *testing.T) {
	_, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/login", nil)
	req.Host = "localhost:8000"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	esperados := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "no-referrer",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for cabecalho, valor := range esperados {
		if obtido := rec.Header().Get(cabecalho); obtido != valor {
			t.Errorf("cabeçalho %s esperado %q, obtido %q", cabecalho, valor, obtido)
		}
	}
	csp := rec.Header().Get("Content-Security-Policy")
	for _, diretiva := range []string{"default-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "base-uri 'none'"} {
		if !strings.Contains(csp, diretiva) {
			t.Errorf("CSP deveria conter %q, obtido %q", diretiva, csp)
		}
	}
}

// A-03: nome de arquivo malicioso do certificado não é mais refletido como HTML.
func TestXSSNomeArquivoCertificadoEscapado(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	cfg, _ := db.ObterConfiguracao()
	cfg.TipoCertificado = "A1"
	cfg.CertificadoPath = `./dados/certificados/<img src=x onerror=alert(1)>.pfx`
	if err := db.SalvarConfiguracao(cfg); err != nil {
		t.Fatalf("falha ao salvar configuração: %v", err)
	}

	req := httptest.NewRequest("POST", "/configuracao/testar", strings.NewReader("senha_a1=x"))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<img src=x onerror=alert(1)>") {
		t.Errorf("nome de arquivo malicioso foi refletido sem escape (XSS): %s", body)
	}
	if !strings.Contains(body, "&lt;img src=x onerror=alert(1)&gt;") {
		t.Errorf("esperado o nome do arquivo escapado na resposta, obtido: %s", body)
	}
}

// B-02: ID de evento fora do padrão oficial é substituído.
func TestIDEventoInvalidoSubstituido(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	xmlConteudo := `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtInfoEmpregador/v_S_01_03_00">
  <evtInfoEmpregador Id="ID_INVALIDO_123456789">
    <ideEvento><tpAmb>2</tpAmb><procEmi>1</procEmi><verProc>1.0.0</verProc></ideEvento>
    <ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678000190</nrInsc></ideEmpregador>
  </evtInfoEmpregador>
</eSocial>`

	form := url.Values{"codigo_evento": {"S-1000"}, "xml_conteudo": {xmlConteudo}}
	req := httptest.NewRequest("POST", "/eventos/salvar-generico", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /eventos/salvar-generico falhou: %d", rec.Code)
	}

	eventos, _ := db.ListarEventos()
	if len(eventos) == 0 {
		t.Fatalf("nenhum evento gravado")
	}
	idOficial := regexp.MustCompile(`^ID[0-9]{34}$`)
	for _, e := range eventos {
		if !idOficial.MatchString(e.ID) {
			t.Errorf("evento gravado com ID fora do padrão oficial: %q", e.ID)
		}
	}
	if !strings.Contains(rec.Body.String(), "não segue o padrão oficial") {
		t.Errorf("resposta deveria avisar sobre o ID inválido substituído")
	}
}

// M-02: upload acima do limite é recusado com mensagem clara.
func TestUploadAcimaDoLimite(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("arquivo_pfx", "grande.pfx")
	_, _ = part.Write(bytes.Repeat([]byte("A"), 3<<20)) // 3 MB > limite de 2 MB
	_ = w.WriteField("tipo_certificado", "A1")
	w.Close()

	req := httptest.NewRequest("POST", "/configuracao/certificado", &b)
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("resposta inesperada: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "excede o limite") {
		t.Errorf("esperada mensagem de limite excedido, obtido: %s", rec.Body.String())
	}
}

// M-01: campos de formulário não escapados não conseguem mais injetar XML.
func TestGeradorXMLEscapaCamposMaliciosos(t *testing.T) {
	payload := `09.01.001</codAgNoc><codAgNoc>INJETADO`

	xml2240 := web.GerarXMLS2240(web.ParametrosS2240{
		ID:             "ID1000000000000002026010112000000001",
		Ambiente:       2,
		CNPJ:           "12345678000190",
		CPFTrabalhador: "11122233344",
		CodigoRisco:    payload,
		NomeRisco:      payload,
		DataInicio:     "2026-01-01</dtIniCondicao><x>",
		UFRegistro:     "SP</ufOC><x>",
		OrgaoClasse:    "4</ideOC><x>",
		DscSetor:       "Setor</dscSetor><x>",
		NumRegistro:    "123</dscOC><x>",
	})
	if strings.Contains(xml2240, "<codAgNoc>INJETADO") || strings.Contains(xml2240, "<x>") {
		t.Errorf("XML S-2240 contém injeção de estrutura:\n%s", xml2240)
	}
	if !strings.Contains(xml2240, "&lt;/codAgNoc&gt;") {
		t.Errorf("esperado escape do payload no S-2240:\n%s", xml2240)
	}

	xml2210 := web.GerarXMLS2210(web.ParametrosS2210{
		ID:             "ID1000000000000002026010112000000002",
		Ambiente:       2,
		CNPJ:           "12345678000190",
		CPFTrabalhador: "11122233344",
		DtAcidente:     "2026-01-01</dtAcid><x>",
		HrAcidente:     "08:00</hrAcid><x>",
		DscLocal:       "Local</dscLocal><x>",
		DscLograd:      "Rua</dscLograd><x>",
		NrLograd:       "1</nrLograd><x>",
		Bairro:         "Bairro</bairro><x>",
		NomeMedico:     "Med</nmEmit><x>",
		CRMMedico:      "1</nrOC><x>",
		Observacao:     "Obs</observacao><x>",
		CID10:          "S61.0</codCID><x>",
		UFMedico:       "SP</ufOC><x>",
	})
	if strings.Contains(xml2210, "<x>") {
		t.Errorf("XML S-2210 contém injeção de estrutura:\n%s", xml2210)
	}

	xml2220 := web.GerarXMLS2220(web.ParametrosS2220{
		ID:             "ID1000000000000002026010112000000003",
		Ambiente:       2,
		CNPJ:           "12345678000190",
		CPFTrabalhador: "11122233344",
		DataASO:        "2026-01-01</dtAso><x>",
		NomeMedico:     "Med</nmMed><x>",
		CRMMedico:      "123456",
		UFMedico:       "XX</ufCRM><x>",
	})
	if strings.Contains(xml2220, "<x>") {
		t.Errorf("XML S-2220 contém injeção de estrutura:\n%s", xml2220)
	}
	if !strings.Contains(xml2220, "<ufCRM>SP</ufCRM>") {
		t.Errorf("UF inválida deveria cair no padrão SP:\n%s", xml2220)
	}
}

// A-02: assinatura simulada é explicitamente marcada e a transmissão não gera recibo.
func TestAssinaturaSimuladaMarcadaENenhumReciboFabricado(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?><eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00"><evtExpRisco Id="ID1000000000000002026010112000000001"></evtExpRisco></eSocial>`
	assinado := web.SimularAssinatura(xml, "EMPRESA TESTE LTDA")
	if !strings.Contains(assinado, "ASSINATURA SIMULADA") {
		t.Errorf("envelope simulado deveria estar explicitamente marcado:\n%s", assinado)
	}
	if strings.Contains(assinado, "SIMULATED_RSA_SHA256_SIGNATURE") {
		t.Errorf("assinatura simulada antiga (que se passava por real) ainda presente")
	}

	msg := web.MensagemTransmissaoSimulada(1, "S-2240")
	if !strings.Contains(msg, "SIMULAÇÃO") || !strings.Contains(msg, "nenhum dado foi enviado") {
		t.Errorf("mensagem de transmissão simulada deve explicitar a ausência de envio: %s", msg)
	}
	if strings.Contains(msg, "Recibo emitido pelo Serpro") {
		t.Errorf("mensagem não pode afirmar emissão de recibo oficial: %s", msg)
	}
}

// M-03: validação passa a usar o schema oficial quando o tipo possui XSD embutido.
func TestValidacaoUsaSchemaOficial(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	casos := []struct {
		nome           string
		id             string
		xml            string
		esperaRejeicao bool
		trechoMsg      string
	}{
		{
			nome: "evento aderente ao schema",
			id:   "ID1000000000000002026010112000000001",
			// Amostra construída conforme evtExpRisco.xsd (S-1.3): ideVinculo, agNoc,
			// epcEpi e respReg com vocabulários válidos. A assinatura dummy é
			// adicionada pelo próprio validador.
			xml: `<?xml version="1.0" encoding="UTF-8"?>` +
				`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00">` +
				`<evtExpRisco Id="ID1000000000000002026010112000000001">` +
				`<ideEvento><indRetif>1</indRetif><tpAmb>2</tpAmb><procEmi>1</procEmi><verProc>1.0.0</verProc></ideEvento>` +
				`<ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678000190</nrInsc></ideEmpregador>` +
				`<ideVinculo><cpfTrab>11122233344</cpfTrab><matricula>MAT-001</matricula></ideVinculo>` +
				`<infoExpRisco><dtIniCondicao>2026-01-01</dtIniCondicao>` +
				`<infoAmb><localAmb>1</localAmb><dscSetor>Geral</dscSetor><tpInsc>1</tpInsc><nrInsc>12345678000190</nrInsc></infoAmb>` +
				`<infoAtiv><dscAtivDes>Atividades operacionais</dscAtivDes></infoAtiv>` +
				`<agNoc><codAgNoc>09.01.001</codAgNoc><dscAgNoc>Ausencia de agente nocivo</dscAgNoc><tpAval>2</tpAval>` +
				`<epcEpi><utilizEPC>0</utilizEPC><utilizEPI>0</utilizEPI></epcEpi></agNoc>` +
				`<respReg><cpfResp>11122233344</cpfResp><ideOC>1</ideOC><dscOC>000000</dscOC><nrOC>123456</nrOC><ufOC>SP</ufOC></respReg>` +
				`</infoExpRisco></evtExpRisco></eSocial>`,
			esperaRejeicao: false,
			trechoMsg:      "schema",
		},
		{
			nome:           "evento sem campos obrigatorios do schema",
			id:             "ID1000000000000002026010112000000010",
			xml:            `<?xml version="1.0" encoding="UTF-8"?><eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtExpRisco/v_S_01_03_00"><evtExpRisco Id="ID1000000000000002026010112000000010"><ideEvento><indRetif>1</indRetif><tpAmb>2</tpAmb><procEmi>1</procEmi><verProc>1.0.0</verProc></ideEvento></evtExpRisco></eSocial>`,
			esperaRejeicao: true,
			trechoMsg:      "schema oficial",
		},
	}

	xmllint := esocial.XmllintDisponivel()

	for _, caso := range casos {
		if caso.esperaRejeicao && !xmllint {
			// Sem xmllint o validador só executa a checagem sintática nativa, portanto
			// não há reprovação por schema a verificar neste ambiente.
			t.Log("xmllint ausente: caso de reprovação por schema ignorado")
			continue
		}
		evento := &storage.Evento{
			ID: caso.id, Tipo: "S-2240", Ambiente: 2, Status: "pronto", XMLGerado: caso.xml,
		}
		if err := db.SalvarEvento(evento); err != nil {
			t.Fatalf("falha ao gravar evento: %v", err)
		}

		req := httptest.NewRequest("POST", "/fila/"+caso.id+"/validar", nil)
		req.Host = "localhost:8000"
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s: POST validar falhou: %d", caso.nome, rec.Code)
		}
		atualizado, _ := db.ObterEvento(caso.id)
		if caso.esperaRejeicao {
			if atualizado.Status != "rejeitado" {
				t.Errorf("%s: esperada rejeição, obtido status %q (%s)", caso.nome, atualizado.Status, atualizado.MensagemRetorno)
			}
		} else if atualizado.Status == "rejeitado" {
			t.Errorf("%s: evento válido foi rejeitado (%s)", caso.nome, atualizado.MensagemRetorno)
		}
		if !strings.Contains(atualizado.MensagemRetorno, caso.trechoMsg) {
			t.Errorf("%s: mensagem deveria conter %q, obtido: %s", caso.nome, caso.trechoMsg, atualizado.MensagemRetorno)
		}
	}
}

// B-04: exclusão inexistente informa erro em vez de sucesso falso.
func TestExclusaoInexistenteInformaErro(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	req := httptest.NewRequest("DELETE", "/fila/ID_INEXISTENTE", nil)
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "não encontrado") {
		t.Errorf("exclusão de evento inexistente deveria informar erro, obtido: %s", rec.Body.String())
	}
}

// B-01: chave privada gravada com permissão restrita (0600).
func TestPermissaoDoArquivoDeCertificado(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	// O handler grava em ./dados/certificados relativo ao diretório de trabalho;
	// o teste roda em diretório temporário para não sujar o repositório.
	t.Chdir(t.TempDir())

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("arquivo_pfx", "certificado.pfx")
	_, _ = part.Write([]byte("conteudo-fake"))
	_ = w.WriteField("tipo_certificado", "A1")
	w.Close()

	req := httptest.NewRequest("POST", "/configuracao/certificado", &b)
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	info, err := os.Stat(filepath.Join("dados", "certificados", "certificado.pfx"))
	if err != nil {
		t.Fatalf("arquivo de certificado não foi gravado: %v (resposta: %d)", err, rec.Code)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissão esperada 0600 para a chave privada, obtida %o", perm)
	}

	dirInfo, err := os.Stat(filepath.Join("dados", "certificados"))
	if err != nil {
		t.Fatalf("diretório de certificados não encontrado: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("permissão esperada 0700 para o diretório de certificados, obtida %o", perm)
	}
}

// B-05: a função safeHTML (bypass de escaping) não existe mais.
func TestSemFuncaoSafeHTML(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	req := httptest.NewRequest("GET", "/configuracao", nil)
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /configuracao falhou: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "safeHTML") {
		t.Errorf("referência residual à função safeHTML removida")
	}
}
