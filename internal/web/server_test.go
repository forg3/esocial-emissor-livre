package web_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/storage"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

func prepararServidorTeste(t *testing.T) (*web.Servidor, *storage.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "esocial-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Abrir(dbPath)
	if err != nil {
		t.Fatalf("falha ao abrir sqlite de teste: %v", err)
	}

	srv, err := web.NovoServidor(db)
	if err != nil {
		t.Fatalf("falha ao instanciar servidor web: %v", err)
	}

	limpeza := func() {
		db.Fechar()
		os.RemoveAll(tmpDir)
	}

	return srv, db, limpeza
}

func TestRotasEstaticas(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	// Testa CSS
	req := httptest.NewRequest("GET", "/static/css/app.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado status 200 para app.css, obtido: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Canteiro Prime") {
		t.Errorf("app.css não contém menção ao design system Canteiro Prime")
	}

	// Testa JS (HTMX)
	reqJS := httptest.NewRequest("GET", "/static/js/htmx.min.js", nil)
	recJS := httptest.NewRecorder()
	handler.ServeHTTP(recJS, reqJS)

	if recJS.Code != http.StatusOK {
		t.Errorf("esperado status 200 para htmx.min.js, obtido: %d", recJS.Code)
	}
}

func TestDashboard(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado status 200 no dashboard, obtido: %d - erro: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "CANTEIRO PRIME") {
		t.Errorf("dashboard não contém a marca Canteiro Prime")
	}
	if !strings.Contains(body, "Painel de Controle SST") {
		t.Errorf("dashboard não contém o título do painel")
	}
	if !strings.Contains(body, "kpi-grid") {
		t.Errorf("dashboard não contém o grid de KPIs")
	}
}

func TestDashboardHTMXPartial(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "conteudo")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado status 200 no HTMX dashboard, obtido: %d", rec.Code)
	}

	body := rec.Body.String()
	// No swap HTMX parcial de conteúdo, não deve ter a casca HTML completa
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("resposta parcial HTMX não deveria incluir a tag <!DOCTYPE html>")
	}
	if !strings.Contains(body, "Painel de Controle SST") {
		t.Errorf("resposta HTMX deve conter o conteúdo do dashboard")
	}
}

func TestConfiguracao(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	// 1. GET /configuracao
	req := httptest.NewRequest("GET", "/configuracao", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /configuracao falhou: %d", rec.Code)
	}

	// 2. POST /configuracao (salvar empresa)
	form := url.Values{}
	form.Set("razao_social", "Construtora Canteiro Prime Ltda")
	form.Set("cnpj", "12.345.678/0001-90")
	form.Set("ambiente", "2")

	reqPost := httptest.NewRequest("POST", "/configuracao", strings.NewReader(form.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recPost := httptest.NewRecorder()
	handler.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusOK {
		t.Errorf("POST /configuracao falhou: %d", recPost.Code)
	}

	if !strings.Contains(recPost.Body.String(), "Construtora Canteiro Prime Ltda") {
		t.Errorf("razão social não foi refletida na resposta")
	}

	// 3. Teste do certificado
	reqTeste := httptest.NewRequest("POST", "/configuracao/testar", nil)
	recTeste := httptest.NewRecorder()
	handler.ServeHTTP(recTeste, reqTeste)

	if recTeste.Code != http.StatusOK {
		t.Errorf("POST /configuracao/testar falhou: %d", recTeste.Code)
	}
}

func TestColaboradoresECSV(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	// 1. Download do modelo CSV
	reqModelo := httptest.NewRequest("GET", "/colaboradores/modelo-csv", nil)
	recModelo := httptest.NewRecorder()
	handler.ServeHTTP(recModelo, reqModelo)

	if recModelo.Code != http.StatusOK {
		t.Errorf("GET /colaboradores/modelo-csv falhou: %d", recModelo.Code)
	}
	if !strings.Contains(recModelo.Body.String(), "nome,cpf,matricula") {
		t.Errorf("modelo CSV não possui cabeçalho padrão")
	}

	// 2. Cadastro manual de colaborador
	form := url.Values{}
	form.Set("nome", "João da Silva")
	form.Set("cpf", "11122233344")
	form.Set("matricula", "MAT-001")
	form.Set("cargo", "Eletricista")
	form.Set("cbo", "951105")
	form.Set("data_admissao", "2023-01-10")
	form.Set("setor", "Manutenção")

	reqNovo := httptest.NewRequest("POST", "/colaboradores", strings.NewReader(form.Encode()))
	reqNovo.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recNovo := httptest.NewRecorder()
	handler.ServeHTTP(recNovo, reqNovo)

	if recNovo.Code != http.StatusOK {
		t.Errorf("POST /colaboradores falhou: %d", recNovo.Code)
	}
	if !strings.Contains(recNovo.Body.String(), "João da Silva") {
		t.Errorf("colaborador cadastrado não listado na resposta")
	}

	// 3. Importação via CSV
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("arquivo_csv", "teste.csv")
	if err != nil {
		t.Fatalf("falha ao criar form file: %v", err)
	}
	csvContent := `nome,cpf,matricula,cargo,cbo,data_admissao,setor
"Maria Oliveira","55566677788","MAT-002","Soldadora","991305","2023-02-01","Produção"
`
	part.Write([]byte(csvContent))
	w.Close()

	reqCSV := httptest.NewRequest("POST", "/colaboradores/importar-csv", &b)
	reqCSV.Header.Set("Content-Type", w.FormDataContentType())
	recCSV := httptest.NewRecorder()
	handler.ServeHTTP(recCSV, reqCSV)

	if recCSV.Code != http.StatusOK {
		t.Errorf("POST /colaboradores/importar-csv falhou: %d", recCSV.Code)
	}
	if !strings.Contains(recCSV.Body.String(), "Maria Oliveira") {
		t.Errorf("trabalhadora importada por CSV não listada")
	}

	// 4. Busca instantânea HTMX
	reqBusca := httptest.NewRequest("GET", "/colaboradores/tabela?q=maria", nil)
	recBusca := httptest.NewRecorder()
	handler.ServeHTTP(recBusca, reqBusca)

	if recBusca.Code != http.StatusOK {
		t.Errorf("GET /colaboradores/tabela com filtro falhou: %d", recBusca.Code)
	}
	if !strings.Contains(recBusca.Body.String(), "Maria Oliveira") {
		t.Errorf("busca por 'maria' falhou em trazer resultado")
	}
}

func TestEventosSSTCicloDeVidaFila(t *testing.T) {
	srv, db, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	// Cadastra um colaborador inicial
	colab := &storage.Colaborador{
		ID:        "colab-teste-1",
		Nome:      "Engenheiro Lucas",
		CPF:       "99988877766",
		Matricula: "MAT-99",
		Cargo:     "Engenheiro de Segurança",
		Status:    "ativo",
	}
	_ = db.SalvarColaborador(colab)

	// 1. Gera Evento S-2240
	form2240 := url.Values{}
	form2240.Set("colaborador_id", colab.ID)
	form2240.Set("dt_inicio", "2026-01-01")
	form2240.Set("codigo_risco", "09.01.001")
	form2240.Set("nome_risco", "Ausência de risco")
	form2240.Set("tipo_avaliacao", "2")
	form2240.Set("nome_resp", "Eng. Técnico")
	form2240.Set("cpf_resp", "12312312312")
	form2240.Set("num_registro", "12345")

	req2240 := httptest.NewRequest("POST", "/eventos/s2240", strings.NewReader(form2240.Encode()))
	req2240.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec2240 := httptest.NewRecorder()
	handler.ServeHTTP(rec2240, req2240)

	if rec2240.Code != http.StatusOK {
		t.Errorf("POST /eventos/s2240 falhou: %d", rec2240.Code)
	}

	eventos, _ := db.ListarEventos()
	if len(eventos) == 0 {
		t.Fatalf("nenhum evento foi gravado no banco")
	}
	evtID := eventos[0].ID

	// 2. Validar XSD
	reqVal := httptest.NewRequest("POST", "/fila/"+evtID+"/validar", nil)
	recVal := httptest.NewRecorder()
	handler.ServeHTTP(recVal, reqVal)
	if recVal.Code != http.StatusOK {
		t.Errorf("POST /fila/{id}/validar falhou: %d", recVal.Code)
	}

	// 3. Assinar evento
	reqAss := httptest.NewRequest("POST", "/fila/"+evtID+"/assinar", nil)
	recAss := httptest.NewRecorder()
	handler.ServeHTTP(recAss, reqAss)
	if recAss.Code != http.StatusOK {
		t.Errorf("POST /fila/{id}/assinar falhou: %d", recAss.Code)
	}

	evtAtualizado, _ := db.ObterEvento(evtID)
	if evtAtualizado.Status != "assinado" {
		t.Errorf("status esperado 'assinado', obtido '%s'", evtAtualizado.Status)
	}

	// 4. Transmitir ao eSocial
	reqTrans := httptest.NewRequest("POST", "/fila/"+evtID+"/transmitir", nil)
	recTrans := httptest.NewRecorder()
	handler.ServeHTTP(recTrans, reqTrans)
	if recTrans.Code != http.StatusOK {
		t.Errorf("POST /fila/{id}/transmitir falhou: %d", recTrans.Code)
	}

	evtTransmitido, _ := db.ObterEvento(evtID)
	if evtTransmitido.Status != "aceito" || evtTransmitido.Recibo == "" {
		t.Errorf("status esperado 'aceito' com recibo, obtido status '%s' e recibo '%s'", evtTransmitido.Status, evtTransmitido.Recibo)
	}

	// 5. Download do XML
	reqXML := httptest.NewRequest("GET", "/fila/"+evtID+"/xml", nil)
	recXML := httptest.NewRecorder()
	handler.ServeHTTP(recXML, reqXML)

	if recXML.Code != http.StatusOK {
		t.Errorf("GET /fila/{id}/xml falhou: %d", recXML.Code)
	}
	if !strings.Contains(recXML.Body.String(), "<eSocial") {
		t.Errorf("conteúdo do XML baixado não contém tag raiz eSocial")
	}

	// 6. Modal de detalhes e logs
	reqDet := httptest.NewRequest("GET", "/fila/"+evtID+"/detalhes", nil)
	recDet := httptest.NewRecorder()
	handler.ServeHTTP(recDet, reqDet)

	if recDet.Code != http.StatusOK {
		t.Errorf("GET /fila/{id}/detalhes falhou: %d", recDet.Code)
	}
	if !strings.Contains(recDet.Body.String(), evtID) {
		t.Errorf("modal de detalhes não contém o ID do evento")
	}
}

func TestAPIsBuscaRiscosECBOs(t *testing.T) {
	srv, _, cleanup := prepararServidorTeste(t)
	defer cleanup()

	handler := srv.Rotas()

	// Busca Riscos
	reqRisco := httptest.NewRequest("GET", "/api/riscos/busca?q=ruido", nil)
	recRisco := httptest.NewRecorder()
	handler.ServeHTTP(recRisco, reqRisco)

	if recRisco.Code != http.StatusOK {
		t.Errorf("GET /api/riscos/busca falhou: %d", recRisco.Code)
	}
	if !strings.Contains(recRisco.Body.String(), "busca-item") {
		t.Errorf("busca por 'ruido' deveria retornar itens de risco da Tabela 24")
	}

	// Busca CBO
	reqCBO := httptest.NewRequest("GET", "/api/cbos/busca?q=eletricista", nil)
	recCBO := httptest.NewRecorder()
	handler.ServeHTTP(recCBO, reqCBO)

	if recCBO.Code != http.StatusOK {
		t.Errorf("GET /api/cbos/busca falhou: %d", recCBO.Code)
	}
	if !strings.Contains(recCBO.Body.String(), "busca-item") {
		t.Errorf("busca por 'eletricista' deveria retornar itens de CBO")
	}
}
