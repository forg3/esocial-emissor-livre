package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

// Sprint 4: cadastro de múltiplos perfis, ativação e sincronização com a configuração global.
func TestPerfisMultiEmpresaAtivacaoESincronizacao(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)
	csrf := srv.TokenCSRF()

	// 1. Página de perfis abre autenticada
	reqPagina := httptest.NewRequest("GET", "/perfis", nil)
	reqPagina.Host = "localhost:8000"
	reqPagina.AddCookie(cookie)
	recPagina := httptest.NewRecorder()
	mux.ServeHTTP(recPagina, reqPagina)
	if recPagina.Code != http.StatusOK {
		t.Fatalf("GET /perfis falhou: %d", recPagina.Code)
	}
	if !strings.Contains(recPagina.Body.String(), "Empresas e Certificados") {
		t.Errorf("página de perfis não renderizou o título esperado")
	}

	criarPerfil := func(razao, cnpj string) {
		form := url.Values{
			"razao_social": {razao},
			"cnpj":         {cnpj},
			"ambiente":     {"2"},
			"csrf_token":   {csrf},
		}
		req := httptest.NewRequest("POST", "/perfis", strings.NewReader(form.Encode()))
		req.Host = "localhost:8000"
		req.AddCookie(cookie)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-CSRF-Token", csrf)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST /perfis (%s) falhou: %d", razao, rec.Code)
		}
	}

	// 2. Dois perfis distintos (matriz e filial)
	criarPerfil("EMPRESA MATRIZ LTDA", "12345678000190")
	criarPerfil("EMPRESA FILIAL LTDA", "98765432000155")

	perfis, err := db.ListarPerfis()
	if err != nil {
		t.Fatalf("falha ao listar perfis: %v", err)
	}
	if len(perfis) != 2 {
		t.Fatalf("esperados 2 perfis, obtidos %d", len(perfis))
	}

	// 3. O primeiro perfil cadastrado fica ativo e sincroniza a configuração global
	ativo, err := db.ObterPerfilAtivo()
	if err != nil || ativo == nil {
		t.Fatalf("nenhum perfil ativo após o cadastro: %v", err)
	}
	if ativo.CNPJ != "12345678000190" {
		t.Errorf("perfil ativo deveria ser a matriz, obtido %s", ativo.CNPJ)
	}
	cfg, _ := db.ObterConfiguracao()
	if cfg.CNPJ != "12345678000190" {
		t.Errorf("configuração global deveria refletir o perfil ativo, obtido %s", cfg.CNPJ)
	}

	// 4. Ativa o segundo perfil e confere a sincronização
	var filial storage.PerfilEmpresa
	for _, p := range perfis {
		if p.CNPJ == "98765432000155" {
			filial = p
		}
	}
	reqAtivar := httptest.NewRequest("POST", "/perfis/"+filial.ID+"/ativar", nil)
	reqAtivar.Host = "localhost:8000"
	reqAtivar.AddCookie(cookie)
	reqAtivar.Header.Set("X-CSRF-Token", csrf)
	recAtivar := httptest.NewRecorder()
	mux.ServeHTTP(recAtivar, reqAtivar)
	if recAtivar.Code != http.StatusOK {
		t.Fatalf("POST /perfis/{id}/ativar falhou: %d", recAtivar.Code)
	}

	cfgDepois, _ := db.ObterConfiguracao()
	if cfgDepois.CNPJ != "98765432000155" || cfgDepois.RazaoSocial != "EMPRESA FILIAL LTDA" {
		t.Errorf("configuração global não sincronizou com o novo perfil ativo: %+v", cfgDepois)
	}
	ativos, _ := db.ListarPerfis()
	contagemAtivos := 0
	for _, p := range ativos {
		if p.Ativo {
			contagemAtivos++
		}
	}
	if contagemAtivos != 1 {
		t.Errorf("deveria existir exatamente 1 perfil ativo, obtidos %d", contagemAtivos)
	}

	// 5. O perfil ativo não pode ser excluído
	reqDelAtivo := httptest.NewRequest("DELETE", "/perfis/"+filial.ID, nil)
	reqDelAtivo.Host = "localhost:8000"
	reqDelAtivo.AddCookie(cookie)
	reqDelAtivo.Header.Set("X-CSRF-Token", csrf)
	recDelAtivo := httptest.NewRecorder()
	mux.ServeHTTP(recDelAtivo, reqDelAtivo)
	if !strings.Contains(recDelAtivo.Body.String(), "Não é possível excluir o perfil ativo") {
		t.Errorf("exclusão do perfil ativo deveria ser bloqueada")
	}

	// 6. Perfil inativo pode ser excluído
	var matriz storage.PerfilEmpresa
	for _, p := range ativos {
		if p.CNPJ == "12345678000190" {
			matriz = p
		}
	}
	reqDel := httptest.NewRequest("DELETE", "/perfis/"+matriz.ID, nil)
	reqDel.Host = "localhost:8000"
	reqDel.AddCookie(cookie)
	reqDel.Header.Set("X-CSRF-Token", csrf)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("DELETE /perfis/{id} falhou: %d", recDel.Code)
	}
	restantes, _ := db.ListarPerfis()
	if len(restantes) != 1 {
		t.Errorf("esperado 1 perfil restante, obtidos %d", len(restantes))
	}
}

// Sprint 4: CNPJ inválido é rejeitado com mensagem clara.
func TestPerfilRejeitaCNPJInvalido(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	form := url.Values{"razao_social": {"EMPRESA X"}, "cnpj": {"123"}, "csrf_token": {srv.TokenCSRF()}}
	req := httptest.NewRequest("POST", "/perfis", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "CNPJ inválido") {
		t.Errorf("CNPJ inválido deveria ser rejeitado com mensagem")
	}
	perfis, _ := db.ListarPerfis()
	if len(perfis) != 0 {
		t.Errorf("nenhum perfil deveria ter sido gravado, obtidos %d", len(perfis))
	}
}

// Sprint 4: a procuração eletrônica é persistida junto ao perfil.
func TestPerfilComProcuradorEletronico(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cookie := autenticar(t, mux, srv)

	form := url.Values{
		"razao_social":    {"ESCRITORIO CONTABIL LTDA"},
		"cnpj":            {"11222333000181"},
		"procurador_nome": {"CONTABILIDADE MODELO LTDA"},
		"procurador_doc":  {"12.345.678/0001-90"},
		"csrf_token":      {srv.TokenCSRF()},
	}
	req := httptest.NewRequest("POST", "/perfis", strings.NewReader(form.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", srv.TokenCSRF())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /perfis falhou: %d", rec.Code)
	}

	perfis, _ := db.ListarPerfis()
	if len(perfis) != 1 {
		t.Fatalf("esperado 1 perfil, obtidos %d", len(perfis))
	}
	if perfis[0].ProcuradorNome != "CONTABILIDADE MODELO LTDA" {
		t.Errorf("nome do procurador não persistido: %q", perfis[0].ProcuradorNome)
	}
	if perfis[0].ProcuradorDoc != "12345678000190" {
		t.Errorf("documento do procurador deveria ser normalizado para dígitos, obtido %q", perfis[0].ProcuradorDoc)
	}
}

// Sprint 4: rotas de perfis exigem sessão e token anti-CSRF.
func TestPerfisProtegidosPorAutenticacaoECSRF(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/perfis", nil)
	req.Host = "localhost:8000"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("GET /perfis sem sessão deveria redirecionar, obtido %d", rec.Code)
	}

	cookie := autenticar(t, mux, srv)
	reqPost := httptest.NewRequest("POST", "/perfis", strings.NewReader("razao_social=X&cnpj=12345678000190"))
	reqPost.Host = "localhost:8000"
	reqPost.AddCookie(cookie)
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recPost := httptest.NewRecorder()
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusForbidden {
		t.Errorf("POST /perfis sem token anti-CSRF deveria retornar 403, obtido %d", recPost.Code)
	}
	_ = srv
}
