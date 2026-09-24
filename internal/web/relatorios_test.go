package web_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

const retornoS5001Teste = `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtBasesTrab/v_S_01_03_00">
  <evtBasesTrab Id="ID1000000000000002026020112000000001">
    <ideEvento><perApur>2026-01</perApur></ideEvento>
    <ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678</nrInsc></ideEmpregador>
    <ideTrabalhador><cpfTrab>11122233344</cpfTrab><matricula>MAT-001</matricula></ideTrabalhador>
    <infoBases>
      <vrBcCpMensal>1500.00</vrBcCpMensal>
      <vrCpDescPR>165.00</vrCpDescPR>
    </infoBases>
  </evtBasesTrab>
</eSocial>`

// Sprint 3: importação e consolidação dos totalizadores S-5001/S-5011.
func TestRelatoriosImportacaoConsolidacaoEExportacao(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	// 1. A página de relatórios abre autenticada
	reqPagina := httptest.NewRequest("GET", "/relatorios", nil)
	reqPagina.Host = "localhost:8000"
	reqPagina.AddCookie(cookie)
	recPagina := httptest.NewRecorder()
	mux.ServeHTTP(recPagina, reqPagina)
	if recPagina.Code != http.StatusOK {
		t.Fatalf("GET /relatorios falhou: %d", recPagina.Code)
	}
	if !strings.Contains(recPagina.Body.String(), "Auditoria de Retorno") {
		t.Errorf("página de relatórios não renderizou o título esperado")
	}

	// 2. Importa um retorno S-5001 via upload multipart
	var corpo bytes.Buffer
	escritor := multipart.NewWriter(&corpo)
	parte, err := escritor.CreateFormFile("arquivo_retorno", "retorno-s5001.xml")
	if err != nil {
		t.Fatalf("falha ao montar multipart: %v", err)
	}
	_, _ = parte.Write([]byte(retornoS5001Teste))
	escritor.Close()

	reqImport := httptest.NewRequest("POST", "/relatorios/importar", &corpo)
	reqImport.Host = "localhost:8000"
	reqImport.AddCookie(cookie)
	reqImport.Header.Set("Content-Type", escritor.FormDataContentType())
	reqImport.Header.Set("X-CSRF-Token", csrf)
	recImport := httptest.NewRecorder()
	mux.ServeHTTP(recImport, reqImport)
	if recImport.Code != http.StatusOK {
		t.Fatalf("POST /relatorios/importar falhou: %d", recImport.Code)
	}
	if !strings.Contains(recImport.Body.String(), "totalizador(es) registrado(s)") {
		t.Errorf("mensagem de sucesso da importação não encontrada: %s", recImport.Body.String())
	}

	// 3. O totalizador foi persistido com os valores somados (1500,00 + 165,00)
	retornos, err := db.ListarRetornosTotalizacao("2026-01")
	if err != nil {
		t.Fatalf("falha ao listar totalizadores: %v", err)
	}
	if len(retornos) != 1 {
		t.Fatalf("esperado 1 totalizador persistido, obtido %d", len(retornos))
	}
	if retornos[0].TotalCentavos != 166500 {
		t.Errorf("total esperado 166500 centavos, obtido %d", retornos[0].TotalCentavos)
	}
	if retornos[0].CPFTrab != "11122233344" || retornos[0].Matricula != "MAT-001" {
		t.Errorf("identificação do trabalhador incorreta: %+v", retornos[0])
	}

	// 4. A consolidação por período aparece na página
	reqConsolidado := httptest.NewRequest("GET", "/relatorios?periodo=2026-01", nil)
	reqConsolidado.Host = "localhost:8000"
	reqConsolidado.AddCookie(cookie)
	recConsolidado := httptest.NewRecorder()
	mux.ServeHTTP(recConsolidado, reqConsolidado)
	corpoPagina := recConsolidado.Body.String()
	for _, esperado := range []string{"2026-01", "R$ 1.665,00", "S-5001"} {
		if !strings.Contains(corpoPagina, esperado) {
			t.Errorf("consolidação deveria conter %q", esperado)
		}
	}

	// 5. Exportação CSV com memórias de cálculo
	reqCSV := httptest.NewRequest("GET", "/relatorios/exportar?periodo=2026-01", nil)
	reqCSV.Host = "localhost:8000"
	reqCSV.AddCookie(cookie)
	recCSV := httptest.NewRecorder()
	mux.ServeHTTP(recCSV, reqCSV)
	if recCSV.Code != http.StatusOK {
		t.Fatalf("GET /relatorios/exportar falhou: %d", recCSV.Code)
	}
	if ct := recCSV.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("exportação deveria ser CSV, obtido %q", ct)
	}
	csv := recCSV.Body.String()
	for _, esperado := range []string{"periodo_apuracao", "2026-01", "vrBcCpMensal=1500.00", "1665.00"} {
		if !strings.Contains(csv, esperado) {
			t.Errorf("CSV deveria conter %q; conteúdo: %s", esperado, csv)
		}
	}

	// 6. Exclusão do totalizador
	reqDel := httptest.NewRequest("DELETE", "/relatorios/"+retornos[0].ID, nil)
	reqDel.Host = "localhost:8000"
	reqDel.AddCookie(cookie)
	reqDel.Header.Set("X-CSRF-Token", csrf)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /relatorios/{id} falhou: %d", recDel.Code)
	}
	restantes, _ := db.ListarRetornosTotalizacao("")
	if len(restantes) != 0 {
		t.Errorf("totalizador deveria ter sido removido, restaram %d", len(restantes))
	}
}

// Sprint 3: o retorno exige autenticação e token anti-CSRF como as demais rotas.
func TestRelatoriosProtegidosPorAutenticacaoECSRF(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/relatorios", nil)
	req.Host = "localhost:8000"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("GET /relatorios sem sessão deveria redirecionar para /login, obtido %d", rec.Code)
	}

	cookie := autenticar(t, mux, srv)
	reqPost := httptest.NewRequest("POST", "/relatorios/importar", strings.NewReader(""))
	reqPost.Host = "localhost:8000"
	reqPost.AddCookie(cookie)
	recPost := httptest.NewRecorder()
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusForbidden {
		t.Errorf("POST sem token anti-CSRF deveria retornar 403, obtido %d", recPost.Code)
	}
	_ = srv
}

// Sprint 3: totalizadores por trabalhador são consolidados corretamente no resumo.
func TestResumoRetornosPorPeriodo(t *testing.T) {
	_, db, _, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	itens := []storage.RetornoTotalizacao{
		{ID: "r1", Tipo: "evtBasesTrab", PerApur: "2026-01", CPFTrab: "11122233344", TotalCentavos: 100000},
		{ID: "r2", Tipo: "evtBasesTrab", PerApur: "2026-01", CPFTrab: "55566677788", TotalCentavos: 50000},
		{ID: "r3", Tipo: "evtFGTS", PerApur: "2026-01", TotalCentavos: 8000},
		{ID: "r4", Tipo: "evtCS", PerApur: "2026-02", TotalCentavos: 200000},
	}
	for i := range itens {
		if err := db.SalvarRetornoTotalizacao(&itens[i]); err != nil {
			t.Fatalf("falha ao salvar totalizador %s: %v", itens[i].ID, err)
		}
	}

	resumo, err := db.ObterResumoRetornos()
	if err != nil {
		t.Fatalf("falha ao consolidar: %v", err)
	}
	if len(resumo) != 2 {
		t.Fatalf("esperados 2 períodos, obtidos %d", len(resumo))
	}

	porPeriodo := map[string]storage.ResumoRetornoTotalizacao{}
	for _, r := range resumo {
		porPeriodo[r.PerApur] = r
	}
	janeiro := porPeriodo["2026-01"]
	if janeiro.Registros != 3 || janeiro.Trabalhadores != 2 {
		t.Errorf("janeiro deveria ter 3 registros e 2 trabalhadores: %+v", janeiro)
	}
	if janeiro.PrevidenciarioCentavos != 150000 || janeiro.FGTSCentavos != 8000 || janeiro.TotalCentavos != 158000 {
		t.Errorf("consolidação de janeiro incorreta: %+v", janeiro)
	}
	if fevereiro := porPeriodo["2026-02"]; fevereiro.PrevidenciarioCentavos != 200000 {
		t.Errorf("consolidação de fevereiro incorreta: %+v", fevereiro)
	}
}

// Sprint 3: banco de dados de retorno é criado com permissão restrita.
func TestPermissaoBancoComTabelaDeRetornos(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Abrir(filepath.Join(dir, "retornos.db"))
	if err != nil {
		t.Fatalf("falha ao abrir banco: %v", err)
	}
	defer db.Fechar()

	info, err := os.Stat(filepath.Join(dir, "retornos.db"))
	if err != nil {
		t.Fatalf("banco não encontrado: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissão esperada 0600, obtida %o", perm)
	}
	if err := db.SalvarRetornoTotalizacao(&storage.RetornoTotalizacao{ID: "x", Tipo: "evtCS", Valores: map[string]string{"vrCR": "1.00"}, TotalCentavos: 100}); err != nil {
		t.Errorf("tabela de retornos indisponível: %v", err)
	}
}
