package web

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/data"
	"github.com/forg3/esocial-emissor-livre/internal/esocial"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// Limites de tamanho de upload/corpo (achado M-02).
const (
	limiteUploadPFX = 2 << 20  // 2 MB
	limiteUploadXML = 5 << 20  // 5 MB
	limiteUploadCSV = 10 << 20 // 10 MB
)

// Servidor gerencia os handlers HTTP e templates do sistema.
type Servidor struct {
	db        *storage.DB
	templates map[string]*template.Template
	riscos    []data.Risco
	cbos      []data.CBO
	auth      *autenticacao

	// Tabelas oficiais do eSocial usadas pelos formulários de SST.
	partesCorpo       []data.ItemTabela
	agentesCausadores []data.ItemTabela
	situacoesGeradora []data.ItemTabela
	naturezasLesao    []data.ItemTabela
	procedimentos     []data.ItemTabela
}

// NovoServidor inicializa as dependências com autenticação padrão (senha aleatória
// persistida em ./dados/auth.json, exibida por SenhaInicial).
func NovoServidor(db *storage.DB) (*Servidor, error) {
	return NovoServidorComOpcoes(db, OpcoesAuth{})
}

// NovoServidorComOpcoes inicializa as dependências, lê as bases oficiais, compila
// os templates e monta a camada de autenticação local.
func NovoServidorComOpcoes(db *storage.DB, opts OpcoesAuth) (*Servidor, error) {
	riscos, err := data.CarregarRiscosEsocial()
	if err != nil {
		// Loga mas não impede inicialização se houver fallback
		fmt.Printf("Aviso ao carregar riscos: %v\n", err)
	}

	cbos, err := data.CarregarCBOs()
	if err != nil {
		fmt.Printf("Aviso ao carregar CBOs: %v\n", err)
	}

	auth, err := novoServicoAuth(opts)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar autenticação local: %w", err)
	}

	partesCorpo, err := data.CarregarPartesCorpo()
	if err != nil {
		fmt.Printf("Aviso ao carregar Tabela 13 (partes do corpo): %v\n", err)
	}
	agentes, err := data.CarregarAgentesCausadores()
	if err != nil {
		fmt.Printf("Aviso ao carregar Tabela 14 (agentes causadores): %v\n", err)
	}
	situacoes, err := data.CarregarSituacoesGeradoras()
	if err != nil {
		fmt.Printf("Aviso ao carregar Tabela 15 (situações geradoras): %v\n", err)
	}
	lesoes, err := data.CarregarNaturezasLesao()
	if err != nil {
		fmt.Printf("Aviso ao carregar Tabela 17 (naturezas da lesão): %v\n", err)
	}
	procedimentos, err := data.CarregarProcedimentosDiagnosticos()
	if err != nil {
		fmt.Printf("Aviso ao carregar Tabela 27 (procedimentos diagnósticos): %v\n", err)
	}

	srv := &Servidor{
		db:                db,
		templates:         make(map[string]*template.Template),
		riscos:            riscos,
		cbos:              cbos,
		auth:              auth,
		partesCorpo:       partesCorpo,
		agentesCausadores: agentes,
		situacoesGeradora: situacoes,
		naturezasLesao:    lesoes,
		procedimentos:     procedimentos,
	}

	if err := srv.carregarTemplates(); err != nil {
		return nil, fmt.Errorf("falha ao compilar templates: %w", err)
	}

	return srv, nil
}

// SenhaInicial devolve a senha gerada automaticamente na primeira execução
// (string vazia quando o operador informou a própria senha).
func (s *Servidor) SenhaInicial() string { return s.auth.SenhaInicial() }

// TokenCSRF expõe o token anti-CSRF para uso em clientes/testes.
func (s *Servidor) TokenCSRF() string { return s.auth.csrf }

func (s *Servidor) carregarTemplates() error {
	funcMap := template.FuncMap{
		"formatarData": func(t time.Time) string {
			if t.IsZero() {
				return "--"
			}
			return t.Format("02/01/2006 15:04")
		},
		"formatarDataSimples": func(dataStr string) string {
			if len(dataStr) == 10 {
				partes := strings.Split(dataStr, "-")
				if len(partes) == 3 {
					return partes[2] + "/" + partes[1] + "/" + partes[0]
				}
			}
			return dataStr
		},
		"formatarCPF": func(cpf string) string {
			digs := limpaDigitos(cpf)
			if len(digs) == 11 {
				return fmt.Sprintf("%s.%s.%s-%s", digs[0:3], digs[3:6], digs[6:9], digs[9:11])
			}
			return cpf
		},
		"formatarCNPJ": func(cnpj string) string {
			digs := limpaDigitos(cnpj)
			if len(digs) == 14 {
				return fmt.Sprintf("%s.%s.%s/%s-%s", digs[0:2], digs[2:5], digs[5:8], digs[8:12], digs[12:14])
			}
			return cnpj
		},
		"csrfToken": func() string {
			return s.auth.csrf
		},
		"cspNonce": func() string {
			return s.auth.cspNonce
		},
		"statusBadgeClass": func(status string) string {
			switch status {
			case "aceito":
				return "ok"
			case "assinado":
				return "info"
			case "simulado":
				return "warn"
			case "transmitido":
				return "warn"
			case "rejeitado":
				return "err"
			default:
				return "neutral"
			}
		},
		"statusDescricao": func(status string) string {
			switch status {
			case "pronto":
				return "Pronto p/ Assinar"
			case "assinado":
				return "Assinado (simulado)"
			case "simulado":
				return "Simulado (não transmitido)"
			case "transmitido":
				return "Aguardando eSocial"
			case "aceito":
				return "Aceito (Recibo OK)"
			case "rejeitado":
				return "Rejeitado / Ajustar"
			default:
				return status
			}
		},
		"iconeEvento": func(codigo, grupo string) string {
			switch {
			case codigo == "S-2240":
				return "⚠️"
			case codigo == "S-2210":
				return "🚨"
			case codigo == "S-2220":
				return "🩺"
			case codigo == "S-1005":
				return "🛡️"
			case strings.Contains(grupo, "SESMT"):
				return "🛡️"
			case strings.Contains(grupo, "Clínica"):
				return "🩺"
			case strings.Contains(grupo, "RH"):
				return "👥"
			case strings.Contains(grupo, "Folha") || strings.Contains(grupo, "Contabilidade"):
				return "💰"
			case strings.Contains(grupo, "Jurídico"):
				return "⚖️"
			case strings.Contains(grupo, "RPPS"):
				return "🏛️"
			default:
				return "📄"
			}
		},
		"isObrigSST": func(codigo string) bool {
			return codigo == "S-2240" || codigo == "S-2220" || codigo == "S-2210" || codigo == "S-1005"
		},
	}

	paginas := []string{
		"dashboard",
		"certificado",
		"colaboradores",
		"evento_s2240",
		"evento_s2210",
		"evento_s2220",
		"fila",
		"menu_eventos",
		"editor_generico",
	}

	for _, pag := range paginas {
		t := template.New("layout.html").Funcs(funcMap)
		// Inclui layout, a página específica e todos os partials
		arquivos := []string{
			"templates/layout.html",
			fmt.Sprintf("templates/%s.html", pag),
			"templates/partials/tabela_colaboradores.html",
			"templates/partials/tabela_fila.html",
			"templates/partials/resultado_riscos.html",
			"templates/partials/resultado_cbos.html",
			"templates/partials/detalhes_evento.html",
			"templates/partials/catalogo_eventos.html",
			"templates/partials/teste_certificado.html",
		}

		parsed, err := t.ParseFS(templatesFS, arquivos...)
		if err != nil {
			return fmt.Errorf("erro no template %s: %w", pag, err)
		}
		s.templates[pag] = parsed
	}

	// Template autônomo da tela de login (não usa layout.html)
	login, err := template.New("login.html").Funcs(funcMap).ParseFS(templatesFS, "templates/login.html")
	if err != nil {
		return fmt.Errorf("erro no template de login: %w", err)
	}
	s.templates["login"] = login

	return nil
}

// Rotas configura e retorna o ServeMux do servidor HTTP, já com os middlewares de
// segurança: cabeçalhos (B-03), validação de Host (C-01), autenticação (C-01) e
// proteção anti-CSRF (A-01).
func (s *Servidor) Rotas() http.Handler {
	mux := http.NewServeMux()

	// Autenticação
	mux.HandleFunc("GET /login", s.handleLoginForm)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("GET /logout", s.handleLogout)

	// Arquivos estáticos (CSS, JS, Fontes) servidos de embed.FS
	subStatic, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(subStatic))))

	// 1. Dashboard
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /dashboard", s.handleDashboard)

	// 2. Configuração / Certificado
	mux.HandleFunc("GET /configuracao", s.handleConfiguracao)
	mux.HandleFunc("POST /configuracao", s.handleSalvarConfiguracao)
	mux.HandleFunc("POST /configuracao/certificado", s.handleUploadCertificado)
	mux.HandleFunc("POST /configuracao/testar", s.handleTestarCertificado)

	// 3. Colaboradores
	mux.HandleFunc("GET /colaboradores", s.handleColaboradores)
	mux.HandleFunc("GET /colaboradores/tabela", s.handleTabelaColaboradores)
	mux.HandleFunc("POST /colaboradores", s.handleSalvarColaborador)
	mux.HandleFunc("DELETE /colaboradores/{id}", s.handleExcluirColaborador)
	mux.HandleFunc("GET /colaboradores/modelo-csv", s.handleDownloadModeloCSV)
	mux.HandleFunc("POST /colaboradores/importar-csv", s.handleImportarCSV)

	// 4. Catálogo de Eventos S-1.3 e Editores
	mux.HandleFunc("GET /eventos", s.handleMenuEventos)
	mux.HandleFunc("GET /eventos/catalogo-filtro", s.handleCatalogoFiltro)
	mux.HandleFunc("GET /eventos/novo/{codigo}", s.handleNovoEventoGenerico)
	mux.HandleFunc("POST /eventos/salvar-generico", s.handleSalvarEventoGenerico)
	mux.HandleFunc("GET /eventos/s2240", s.handleEditorS2240)
	mux.HandleFunc("POST /eventos/s2240", s.handleSalvarS2240)
	mux.HandleFunc("GET /eventos/s2210", s.handleEditorS2210)
	mux.HandleFunc("POST /eventos/s2210", s.handleSalvarS2210)
	mux.HandleFunc("GET /eventos/s2220", s.handleEditorS2220)
	mux.HandleFunc("POST /eventos/s2220", s.handleSalvarS2220)
	mux.HandleFunc("POST /eventos/s2220/importar-xml", s.handleImportarXMLASO)
	mux.HandleFunc("GET /eventos/{codigo}", s.handleVerEventoCatalogo)

	// 5. Central de Transmissão / Fila
	mux.HandleFunc("GET /fila", s.handleFila)
	mux.HandleFunc("GET /fila/tabela", s.handleTabelaFila)
	mux.HandleFunc("GET /fila/{id}/xml", s.handleDownloadXML)
	mux.HandleFunc("GET /fila/{id}/detalhes", s.handleDetalhesEvento)
	mux.HandleFunc("POST /fila/{id}/validar", s.handleValidarEvento)
	mux.HandleFunc("POST /fila/{id}/assinar", s.handleAssinarEvento)
	mux.HandleFunc("POST /fila/{id}/transmitir", s.handleTransmitirEvento)
	mux.HandleFunc("POST /fila/{id}/recibo", s.handleConsultarRecibo)
	mux.HandleFunc("DELETE /fila/{id}", s.handleExcluirEvento)
	mux.HandleFunc("POST /fila/lote/assinar", s.handleAssinarLote)
	mux.HandleFunc("POST /fila/lote/transmitir", s.handleTransmitirLote)

	// 6. APIs auxiliares de busca (Autocomplete)
	mux.HandleFunc("GET /api/riscos/busca", s.handleBuscaRiscos)
	mux.HandleFunc("GET /api/cbos/busca", s.handleBuscaCBOs)

	// Cadeia de middlewares: cabeçalhos -> host -> autenticação -> CSRF -> rotas
	var handler http.Handler = mux
	handler = s.middlewareCSRF(handler)
	handler = s.middlewareAuth(handler)
	handler = s.middlewareHost(handler)
	handler = s.middlewareCabecalhos(handler)
	return handler
}

// Helper para renderizar página ou parcial HTMX
func (s *Servidor) render(w http.ResponseWriter, r *http.Request, nomePagina string, dados any) {
	tmpl, ok := s.templates[nomePagina]
	if !ok {
		http.Error(w, "Template não encontrado: "+nomePagina, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Se for requisição HTMX para trocar o conteúdo principal
	if r.Header.Get("HX-Request") == "true" && (r.Header.Get("HX-Target") == "conteudo" || r.Header.Get("HX-Boosted") == "true") {
		if err := tmpl.ExecuteTemplate(w, "content", dados); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Requisição direta / completa do navegador
	if err := tmpl.ExecuteTemplate(w, "layout.html", dados); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// -------------------------------------------------------------
// HANDLERS: DASHBOARD
// -------------------------------------------------------------

// GrupoCatalogoView estrutura um grupo de responsabilidade e seus eventos para renderização.
type GrupoCatalogoView struct {
	Nome    string
	Eventos []esocial.EventoCatalogo
}

// obterGruposCatalogo organiza os eventos em grupos ordenados conforme filtros.
func (s *Servidor) obterGruposCatalogo(grupoFiltro, busca string) []GrupoCatalogoView {
	filtrados := esocial.FiltrarCatalogo(grupoFiltro, busca)
	gruposOficiais := esocial.ObterGruposCatalogo()

	mapa := make(map[esocial.GrupoResponsabilidade][]esocial.EventoCatalogo)
	for _, evt := range filtrados {
		mapa[evt.Grupo] = append(mapa[evt.Grupo], evt)
	}

	var resultado []GrupoCatalogoView
	for _, grp := range gruposOficiais {
		if evts, ok := mapa[grp]; ok && len(evts) > 0 {
			resultado = append(resultado, GrupoCatalogoView{
				Nome:    string(grp),
				Eventos: evts,
			})
		}
	}
	return resultado
}

type DadosViewDashboard struct {
	Titulo          string
	MenuAtivo       string
	Config          *storage.Configuracao
	KPI             *storage.ResumoKPI
	EventosRecentes []storage.Evento
	Grupos          []GrupoCatalogoView
	MensagemFlash   string
	FlashErro       bool
}

func (s *Servidor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	kpi, _ := s.db.ObterResumoKPI()
	if kpi == nil {
		kpi = &storage.ResumoKPI{}
	}

	eventos, _ := s.db.ListarEventos()
	recentes := eventos
	if len(recentes) > 8 {
		recentes = recentes[:8]
	}

	grupos := s.obterGruposCatalogo("todos", "")

	s.render(w, r, "dashboard", DadosViewDashboard{
		Titulo:          "Dashboard",
		MenuAtivo:       "dashboard",
		Config:          cfg,
		KPI:             kpi,
		EventosRecentes: recentes,
		Grupos:          grupos,
	})
}

// -------------------------------------------------------------
// HANDLERS: CONFIGURAÇÃO / CERTIFICADO
// -------------------------------------------------------------

type DadosViewConfiguracao struct {
	Titulo        string
	MenuAtivo     string
	Config        *storage.Configuracao
	MensagemFlash string
	FlashErro     bool
}

func (s *Servidor) handleConfiguracao(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	s.render(w, r, "certificado", DadosViewConfiguracao{
		Titulo:    "Certificado & Empresa",
		MenuAtivo: "configuracao",
		Config:    cfg,
	})
}

func (s *Servidor) handleSalvarConfiguracao(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	cfg.RazaoSocial = strings.TrimSpace(r.FormValue("razao_social"))
	cfg.CNPJ = limpaDigitos(r.FormValue("cnpj"))
	if amb, err := strconv.Atoi(r.FormValue("ambiente")); err == nil {
		cfg.Ambiente = amb
	}
	cfg.AtualizadoEm = time.Now()

	if err := s.db.SalvarConfiguracao(cfg); err != nil {
		s.render(w, r, "certificado", DadosViewConfiguracao{
			Titulo:        "Certificado & Empresa",
			MenuAtivo:     "configuracao",
			Config:        cfg,
			MensagemFlash: "Erro ao salvar dados da empresa: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	s.render(w, r, "certificado", DadosViewConfiguracao{
		Titulo:        "Certificado & Empresa",
		MenuAtivo:     "configuracao",
		Config:        cfg,
		MensagemFlash: "Configurações da empresa atualizadas com sucesso!",
	})
}

func (s *Servidor) handleUploadCertificado(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	cfg.TipoCertificado = r.FormValue("tipo_certificado")

	var msgSucesso = "Configuração do certificado gravada com sucesso!"
	var msgErro = ""

	if cfg.TipoCertificado == "A1" {
		// Limite de corpo e de arquivo para o upload do .pfx (achado M-02).
		r.Body = http.MaxBytesReader(w, r.Body, limiteUploadPFX)
		arquivo, header, err := r.FormFile("arquivo_pfx")
		if err != nil {
			msg := "Falha ao receber o arquivo de certificado."
			if _, excedeu := err.(*http.MaxBytesError); excedeu {
				msg = "Arquivo de certificado excede o limite de 2 MB permitido."
			}
			s.render(w, r, "certificado", DadosViewConfiguracao{
				Titulo:        "Certificado & Empresa",
				MenuAtivo:     "configuracao",
				Config:        cfg,
				MensagemFlash: msg,
				FlashErro:     true,
			})
			return
		}
		if header != nil {
			defer arquivo.Close()

			if header.Size > limiteUploadPFX {
				s.render(w, r, "certificado", DadosViewConfiguracao{
					Titulo:        "Certificado & Empresa",
					MenuAtivo:     "configuracao",
					Config:        cfg,
					MensagemFlash: "Arquivo de certificado excede o limite de 2 MB permitido.",
					FlashErro:     true,
				})
				return
			}

			destinoDir := "./dados/certificados"
			_ = os.MkdirAll(destinoDir, 0700)
			_ = os.Chmod(destinoDir, 0700) // corrige permissões preexistentes (achado B-01)
			caminhoDest := filepath.Join(destinoDir, header.Filename)

			// 0600: a chave privada não deve ser legível por outros usuários locais (achado B-01).
			destFile, err := os.OpenFile(caminhoDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			if err == nil {
				_, errCopy := io.Copy(destFile, arquivo)
				destFile.Close()
				if errCopy != nil {
					msgErro = "Falha ao gravar o arquivo de certificado: " + errCopy.Error()
					_ = s.db.SalvarConfiguracao(cfg)
					s.render(w, r, "certificado", DadosViewConfiguracao{
						Titulo:        "Certificado & Empresa",
						MenuAtivo:     "configuracao",
						Config:        cfg,
						MensagemFlash: msgErro,
						FlashErro:     true,
					})
					return
				}
				_ = os.Chmod(caminhoDest, 0600)
				cfg.CertificadoPath = caminhoDest

				senha := r.FormValue("senha_a1")
				if senha != "" {
					cert, errCert := crypto.CarregarA1Arquivo(caminhoDest, senha)
					if errCert == nil {
						cfg.CertificadoValidoAte = cert.ValidoAte()
						if cfg.RazaoSocial == "" && cert.RazaoSocial() != "" {
							cfg.RazaoSocial = cert.RazaoSocial()
						}
						if cfg.CNPJ == "" && cert.CNPJ() != "" {
							cfg.CNPJ = cert.CNPJ()
						}
						msgSucesso = fmt.Sprintf("Certificado A1 carregado e validado com sucesso! Titular: %s (Validade: %s)", cert.RazaoSocial(), cert.ValidoAte().Format("02/01/2006"))
					} else {
						msgErro = "Certificado salvo, porém a senha informada não conferiu: " + errCert.Error()
					}
				} else {
					cfg.CertificadoValidoAte = time.Now().AddDate(1, 0, 0)
				}
			}
		}
	}

	_ = s.db.SalvarConfiguracao(cfg)

	dados := DadosViewConfiguracao{
		Titulo:        "Certificado & Empresa",
		MenuAtivo:     "configuracao",
		Config:        cfg,
		MensagemFlash: msgSucesso,
	}
	if msgErro != "" {
		dados.MensagemFlash = msgErro
		dados.FlashErro = true
	}

	s.render(w, r, "certificado", dados)
}

// DadosTesteCertificado alimenta o partial teste_certificado.html (saída sempre escapada).
type DadosTesteCertificado struct {
	Classe   string // err | warn | ok
	Titulo   string
	Mensagem string
	Itens    []ItemTesteCertificado
	Nota     string
}

// ItemTesteCertificado é uma linha rótulo/valor do resultado do teste.
type ItemTesteCertificado struct {
	Rotulo string
	Valor  string
}

// renderTesteCertificado escreve o resultado via html/template (escaping contextual),
// eliminando a montagem manual de HTML por fmt.Fprintf (achado A-03).
func (s *Servidor) renderTesteCertificado(w http.ResponseWriter, dados DadosTesteCertificado) {
	tmpl, ok := s.templates["certificado"]
	if !ok {
		http.Error(w, "Template de certificado indisponível", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "teste_certificado", dados); err != nil {
		http.Error(w, "Falha ao renderizar resultado do teste", http.StatusInternalServerError)
	}
}

func (s *Servidor) handleTestarCertificado(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()

	if cfg.TipoCertificado == "A1" {
		if cfg.CertificadoPath == "" {
			s.renderTesteCertificado(w, DadosTesteCertificado{
				Classe: "err",
				Titulo: "Atenção",
				Mensagem: "Nenhum arquivo de certificado A1 foi carregado ainda. " +
					"Faça o upload do arquivo .pfx ou .p12 acima.",
			})
			return
		}
		if _, err := os.Stat(cfg.CertificadoPath); os.IsNotExist(err) {
			s.renderTesteCertificado(w, DadosTesteCertificado{
				Classe:   "err",
				Titulo:   "Arquivo não encontrado",
				Mensagem: "O arquivo informado não foi localizado no disco local.",
				Itens:    []ItemTesteCertificado{{Rotulo: "Arquivo", Valor: filepath.Base(cfg.CertificadoPath)}},
			})
			return
		}

		senha := strings.TrimSpace(r.FormValue("senha_a1"))
		if senha == "" {
			s.renderTesteCertificado(w, DadosTesteCertificado{
				Classe:   "warn",
				Titulo:   "Arquivo A1 presente",
				Mensagem: `Digite a senha do certificado no campo acima e clique em "Testar Validade" para validar a chave privada RSA e o teste criptográfico.`,
				Itens:    []ItemTesteCertificado{{Rotulo: "Arquivo", Valor: filepath.Base(cfg.CertificadoPath)}},
			})
			return
		}

		if !s.permitirTesteCertificado(r) {
			w.Header().Set("Retry-After", "60")
			s.renderTesteCertificado(w, DadosTesteCertificado{
				Classe:   "err",
				Titulo:   "Muitas tentativas",
				Mensagem: "Limite de validações de senha atingido. Aguarde um minuto antes de tentar novamente.",
			})
			return
		}

		cert, err := crypto.CarregarA1Arquivo(cfg.CertificadoPath, senha)
		if err != nil {
			s.renderTesteCertificado(w, DadosTesteCertificado{
				Classe:   "err",
				Titulo:   "Falha na validação do Certificado A1",
				Mensagem: "A senha informada está incorreta ou o arquivo está corrompido.",
				Itens:    []ItemTesteCertificado{{Rotulo: "Detalhe técnico", Valor: err.Error()}},
			})
			return
		}

		cfg.CertificadoValidoAte = cert.ValidoAte()
		if cfg.RazaoSocial == "" && cert.RazaoSocial() != "" {
			cfg.RazaoSocial = cert.RazaoSocial()
		}
		if cfg.CNPJ == "" && cert.CNPJ() != "" {
			cfg.CNPJ = cert.CNPJ()
		}
		_ = s.db.SalvarConfiguracao(cfg)

		xmlTeste := []byte(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtMonit/v_S_01_03_00"><evtMonit Id="ID1000000000000000000000000001"><ideEmpregador><tpInsc>1</tpInsc><nrInsc>00000000000000</nrInsc></ideEmpregador></evtMonit></eSocial>`)
		_, errSign := crypto.AssinarXML(xmlTeste, cert)
		testeCripto := "Assinatura XMLDSig SHA-256 executada com sucesso (chave privada do certificado)."
		classe := "ok"
		if errSign != nil {
			testeCripto = fmt.Sprintf("Alerta na assinatura de teste: %v", errSign)
			classe = "warn"
		}

		diasRestantes := int(time.Until(cert.ValidoAte()).Hours() / 24)
		emissor := "Autoridade Certificadora ICP-Brasil"
		if cert.CertificadoFolha() != nil && cert.CertificadoFolha().Issuer.CommonName != "" {
			emissor = cert.CertificadoFolha().Issuer.CommonName
		}

		ambiente := map[int]string{1: "Produção Oficial (Governo Federal)", 2: "Produção Restrita (Testes / Homologação)"}[cfg.Ambiente]

		s.renderTesteCertificado(w, DadosTesteCertificado{
			Classe:   classe,
			Titulo:   "Certificado Digital A1 operacional e válido",
			Mensagem: "Chave privada lida com sucesso e teste criptográfico concluído localmente.",
			Itens: []ItemTesteCertificado{
				{Rotulo: "Titular", Valor: cert.RazaoSocial()},
				{Rotulo: "Documento identificado", Valor: cert.CNPJ()},
				{Rotulo: "Emissor", Valor: emissor},
				{Rotulo: "Vigência", Valor: fmt.Sprintf("%s até %s (%d dias restantes)", cert.ValidoDe().Format("02/01/2006"), cert.ValidoAte().Format("02/01/2006"), diasRestantes)},
				{Rotulo: "Criptografia", Valor: testeCripto},
				{Rotulo: "Ambiente ativo", Valor: ambiente},
			},
			Nota: "A assinatura dos eventos é executada localmente; a transmissão ao eSocial depende do webservice oficial (ver aviso de modo simulação).",
		})
	} else if cfg.TipoCertificado == "A3" {
		s.renderTesteCertificado(w, DadosTesteCertificado{
			Classe:   "ok",
			Titulo:   "Modo Certificado A3 selecionado",
			Mensagem: "Interface de hardware Token USB / SmartCard (PKCS#11). O PIN de segurança será solicitado diretamente na transmissão dos lotes.",
			Nota:     "Certifique-se de que os drivers do fabricante do token estejam instalados no seu sistema operacional.",
		})
	} else {
		s.renderTesteCertificado(w, DadosTesteCertificado{
			Classe:   "err",
			Titulo:   "Atenção",
			Mensagem: "Nenhum tipo de certificado digital foi configurado ainda.",
		})
	}
}

// -------------------------------------------------------------
// HANDLERS: COLABORADORES
// -------------------------------------------------------------

type DadosViewColaboradores struct {
	Titulo        string
	MenuAtivo     string
	Config        *storage.Configuracao
	Colaboradores []storage.Colaborador
	MensagemFlash string
	FlashErro     bool
}

func (s *Servidor) handleColaboradores(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	lista, _ := s.db.ListarColaboradores()

	s.render(w, r, "colaboradores", DadosViewColaboradores{
		Titulo:        "Colaboradores",
		MenuAtivo:     "colaboradores",
		Config:        cfg,
		Colaboradores: lista,
	})
}

func (s *Servidor) handleTabelaColaboradores(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	todos, _ := s.db.ListarColaboradores()

	var filtrados []storage.Colaborador
	for _, c := range todos {
		if q == "" ||
			strings.Contains(strings.ToLower(c.Nome), q) ||
			strings.Contains(c.CPF, q) ||
			strings.Contains(strings.ToLower(c.Matricula), q) ||
			strings.Contains(strings.ToLower(c.Cargo), q) {
			filtrados = append(filtrados, c)
		}
	}

	tmpl := s.templates["colaboradores"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "tabela_colaboradores", DadosViewColaboradores{
		Colaboradores: filtrados,
	})
}

func (s *Servidor) handleSalvarColaborador(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()

	c := &storage.Colaborador{
		ID:           fmt.Sprintf("colab-%d", time.Now().UnixNano()),
		Nome:         strings.TrimSpace(r.FormValue("nome")),
		CPF:          limpaDigitos(r.FormValue("cpf")),
		Matricula:    strings.TrimSpace(r.FormValue("matricula")),
		Cargo:        strings.TrimSpace(r.FormValue("cargo")),
		CBO:          limpaDigitos(r.FormValue("cbo")),
		DataAdmissao: strings.TrimSpace(r.FormValue("data_admissao")),
		Setor:        strings.TrimSpace(r.FormValue("setor")),
		Status:       "ativo",
		CriadoEm:     time.Now(),
	}

	if len(c.CPF) != 11 {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "CPF inválido. Deve conter exatamente 11 dígitos numéricos.",
			FlashErro:     true,
		})
		return
	}

	if err := s.db.SalvarColaborador(c); err != nil {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Erro ao cadastrar colaborador: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	lista, _ := s.db.ListarColaboradores()
	s.render(w, r, "colaboradores", DadosViewColaboradores{
		Titulo:        "Colaboradores",
		MenuAtivo:     "colaboradores",
		Config:        cfg,
		Colaboradores: lista,
		MensagemFlash: fmt.Sprintf("Colaborador '%s' cadastrado com sucesso!", c.Nome),
	})
}

func (s *Servidor) handleExcluirColaborador(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	id := r.PathValue("id")

	if _, err := s.db.ObterColaborador(id); err != nil {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Colaborador não encontrado para exclusão.",
			FlashErro:     true,
		})
		return
	}

	// Achado B-04: erro de exclusão é reportado ao usuário.
	if err := s.db.ExcluirColaborador(id); err != nil {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Falha ao excluir o colaborador: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	lista, _ := s.db.ListarColaboradores()
	s.render(w, r, "colaboradores", DadosViewColaboradores{
		Titulo:        "Colaboradores",
		MenuAtivo:     "colaboradores",
		Config:        cfg,
		Colaboradores: lista,
		MensagemFlash: "Colaborador excluído com sucesso!",
	})
}

func (s *Servidor) handleDownloadModeloCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"modelo_colaboradores_esocial.csv\"")

	modelo := `nome,cpf,matricula,cargo,cbo,data_admissao,setor
"Carlos Eduardo da Silva","12345678901","MAT-001","Eletricista de Manutenção","951105","2023-01-15","Manutenção Industrial"
"Mariana Souza Santos","98765432100","MAT-002","Operadora de Máquinas","919205","2022-08-10","Produção"
"Roberto Almeida Lima","45678912300","MAT-003","Supervisor de Manutenção","950105","2021-03-01","Engenharia"
`
	_, _ = w.Write([]byte(modelo))
}

func (s *Servidor) handleImportarCSV(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	// Limite de corpo/arquivo para o CSV importado (achado M-02).
	r.Body = http.MaxBytesReader(w, r.Body, limiteUploadCSV)
	arquivo, headerCSV, err := r.FormFile("arquivo_csv")
	if err == nil && headerCSV != nil && headerCSV.Size > limiteUploadCSV {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Arquivo CSV excede o limite de 10 MB permitido.",
			FlashErro:     true,
		})
		return
	}
	if err != nil {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Falha ao ler arquivo CSV enviado.",
			FlashErro:     true,
		})
		return
	}
	defer arquivo.Close()

	leitor := csv.NewReader(io.LimitReader(arquivo, limiteUploadCSV))
	leitor.FieldsPerRecord = -1
	leitor.LazyQuotes = true

	linhas, err := leitor.ReadAll()
	if err != nil {
		lista, _ := s.db.ListarColaboradores()
		s.render(w, r, "colaboradores", DadosViewColaboradores{
			Titulo:        "Colaboradores",
			MenuAtivo:     "colaboradores",
			Config:        cfg,
			Colaboradores: lista,
			MensagemFlash: "Erro ao processar conteúdo CSV: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	importados := 0
	for i, lin := range linhas {
		if i == 0 && (strings.Contains(strings.ToLower(lin[0]), "nome") || strings.Contains(strings.ToLower(lin[1]), "cpf")) {
			continue // Pula linha de cabeçalho
		}
		if len(lin) < 2 {
			continue
		}

		nome := strings.TrimSpace(lin[0])
		cpf := limpaDigitos(lin[1])
		if nome == "" || len(cpf) != 11 {
			continue
		}

		matricula := ""
		if len(lin) > 2 {
			matricula = strings.TrimSpace(lin[2])
		}
		cargo := ""
		if len(lin) > 3 {
			cargo = strings.TrimSpace(lin[3])
		}
		cbo := ""
		if len(lin) > 4 {
			cbo = limpaDigitos(lin[4])
		}
		admissao := ""
		if len(lin) > 5 {
			admissao = strings.TrimSpace(lin[5])
		}
		setor := ""
		if len(lin) > 6 {
			setor = strings.TrimSpace(lin[6])
		}

		c := &storage.Colaborador{
			ID:           fmt.Sprintf("colab-%s", cpf),
			Nome:         nome,
			CPF:          cpf,
			Matricula:    matricula,
			Cargo:        cargo,
			CBO:          cbo,
			DataAdmissao: admissao,
			Setor:        setor,
			Status:       "ativo",
			CriadoEm:     time.Now(),
		}
		_ = s.db.SalvarColaborador(c)
		importados++
	}

	lista, _ := s.db.ListarColaboradores()
	s.render(w, r, "colaboradores", DadosViewColaboradores{
		Titulo:        "Colaboradores",
		MenuAtivo:     "colaboradores",
		Config:        cfg,
		Colaboradores: lista,
		MensagemFlash: fmt.Sprintf("Importação concluída: %d trabalhadores cadastrados/atualizados!", importados),
	})
}

// -------------------------------------------------------------
// HANDLERS: EVENTO S-2240
// -------------------------------------------------------------

type DadosViewEditorS2240 struct {
	Titulo                   string
	MenuAtivo                string
	Config                   *storage.Configuracao
	Colaboradores            []storage.Colaborador
	ColaboradorSelecionadoID string
	Hoje                     string
	MensagemFlash            string
	FlashErro                bool
}

func (s *Servidor) handleEditorS2240(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabs, _ := s.db.ListarColaboradores()
	colabID := r.URL.Query().Get("colaborador_id")

	s.render(w, r, "evento_s2240", DadosViewEditorS2240{
		Titulo:                   "S-2240 • Condições Ambientais",
		MenuAtivo:                "s2240",
		Config:                   cfg,
		Colaboradores:            colabs,
		ColaboradorSelecionadoID: colabID,
		Hoje:                     time.Now().Format("2006-01-02"),
	})
}

func (s *Servidor) handleSalvarS2240(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabID := r.FormValue("colaborador_id")
	colab, err := s.db.ObterColaborador(colabID)
	if err != nil || colab == nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2240", DadosViewEditorS2240{
			Titulo:        "S-2240 • Condições Ambientais",
			MenuAtivo:     "s2240",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			MensagemFlash: "Selecione um colaborador válido para vincular o evento.",
			FlashErro:     true,
		})
		return
	}

	// Sem o CNPJ do empregador o documento é inválido no leiaute S-1.3 (nrInsc).
	if limpaDigitos(cfg.CNPJ) == "" {
		s.handleFilaComFlash(w, r, "Configure o CNPJ da empresa em Certificado & Empresa antes de gerar eventos do eSocial.", true)
		return
	}

	params := ParametrosS2240{
		ID:                 GerarIDEvento(cfg.CNPJ),
		Ambiente:           cfg.Ambiente,
		CNPJ:               cfg.CNPJ,
		CPFTrabalhador:     colab.CPF,
		Matricula:          colab.Matricula,
		DataInicio:         r.FormValue("dt_inicio"),
		DescAtividade:      r.FormValue("desc_atividade"),
		LocalAmbiente:      r.FormValue("local_ambiente"),
		CodigoRisco:        r.FormValue("codigo_risco"),
		NomeRisco:          r.FormValue("nome_risco"),
		TipoAvaliacao:      r.FormValue("tipo_avaliacao"),
		Intensidade:        r.FormValue("intensidade"),
		UtilizaEPC:         r.FormValue("utiliza_epc"),
		EfficazEPC:         r.FormValue("eficaz_epc"),
		UtilizaEPI:         r.FormValue("utiliza_epi"),
		CAEPI:              r.FormValue("ca_epi"),
		NomeResp:           r.FormValue("nome_resp"),
		CPFResp:            r.FormValue("cpf_resp"),
		OrgaoClasse:        r.FormValue("orgao_classe"),
		NumRegistro:        r.FormValue("num_registro"),
		UFRegistro:         r.FormValue("uf_registro"),
		DscSetor:           r.FormValue("dsc_setor"),
		UnidadeMedida:      r.FormValue("unidade_medida"),
		TecnicaMedicao:     r.FormValue("tecnica_medicao"),
		EficazEPI:          r.FormValue("eficaz_epi"),
		MedicaoProtecao:    r.FormValue("med_protecao"),
		CondicaoFunc:       r.FormValue("cond_func"),
		UsoIninterrupto:    r.FormValue("uso_inint"),
		PrazoValidade:      r.FormValue("prazo_validade"),
		PeriodicidadeTroca: r.FormValue("periodic_troca"),
		Higienizacao:       r.FormValue("higienizacao"),
	}
	xmlGerado := GerarXMLS2240(params)

	evento := &storage.Evento{
		ID:            params.ID,
		Tipo:          "S-2240",
		ColaboradorID: colab.ID,
		Ambiente:      cfg.Ambiente,
		Status:        "pronto",
		XMLGerado:     xmlGerado,
		CriadoEm:      time.Now(),
		AtualizadoEm:  time.Now(),
	}

	if err := s.db.SalvarEvento(evento); err != nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2240", DadosViewEditorS2240{
			Titulo:        "S-2240 • Condições Ambientais",
			MenuAtivo:     "s2240",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			MensagemFlash: "Erro ao gravar evento: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	// Redireciona para a central de fila
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento S-2240 (%s) gerado com sucesso para %s e inserido na fila!", evento.ID, colab.Nome), false)
}

// -------------------------------------------------------------
// HANDLERS: EVENTO S-2210 (CAT)
// -------------------------------------------------------------

type DadosViewEditorS2210 struct {
	Titulo                   string
	MenuAtivo                string
	Config                   *storage.Configuracao
	Colaboradores            []storage.Colaborador
	ColaboradorSelecionadoID string
	Hoje                     string
	MensagemFlash            string
	FlashErro                bool

	// Tabelas oficiais usadas pelo formulário da CAT.
	SituacoesGeradoras []data.ItemTabela
	PartesCorpo        []data.ItemTabela
	AgentesCausadores  []data.ItemTabela
	NaturezasLesao     []data.ItemTabela
}

func (s *Servidor) handleEditorS2210(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabs, _ := s.db.ListarColaboradores()
	colabID := r.URL.Query().Get("colaborador_id")

	s.render(w, r, "evento_s2210", DadosViewEditorS2210{
		Titulo:                   "S-2210 • Comunicação de Acidente (CAT)",
		MenuAtivo:                "s2210",
		Config:                   cfg,
		Colaboradores:            colabs,
		ColaboradorSelecionadoID: colabID,
		Hoje:                     time.Now().Format("2006-01-02"),
		SituacoesGeradoras:       s.situacoesGeradora,
		PartesCorpo:              s.partesCorpo,
		AgentesCausadores:        s.agentesCausadores,
		NaturezasLesao:           s.naturezasLesao,
	})
}

func (s *Servidor) handleSalvarS2210(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabID := r.FormValue("colaborador_id")
	colab, err := s.db.ObterColaborador(colabID)
	if err != nil || colab == nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2210", DadosViewEditorS2210{
			Titulo:             "S-2210 • Comunicação de Acidente (CAT)",
			MenuAtivo:          "s2210",
			Config:             cfg,
			Colaboradores:      colabs,
			Hoje:               time.Now().Format("2006-01-02"),
			SituacoesGeradoras: s.situacoesGeradora,
			PartesCorpo:        s.partesCorpo,
			AgentesCausadores:  s.agentesCausadores,
			NaturezasLesao:     s.naturezasLesao,
			MensagemFlash:      "Selecione o trabalhador acidentado.",
			FlashErro:          true,
		})
		return
	}

	// Sem o CNPJ do empregador o documento é inválido no leiaute S-1.3 (nrInsc).
	if limpaDigitos(cfg.CNPJ) == "" {
		s.handleFilaComFlash(w, r, "Configure o CNPJ da empresa em Certificado & Empresa antes de gerar eventos do eSocial.", true)
		return
	}

	params := ParametrosS2210{
		ID:             GerarIDEvento(cfg.CNPJ),
		Ambiente:       cfg.Ambiente,
		CNPJ:           cfg.CNPJ,
		CPFTrabalhador: colab.CPF,
		Matricula:      colab.Matricula,
		DtAcidente:     r.FormValue("dt_acidente"),
		HrAcidente:     r.FormValue("hr_acidente"),
		TpAcidente:     r.FormValue("tp_acidente"),
		HouveAfast:     r.FormValue("houve_afast"),
		HouveObito:     r.FormValue("houve_obito"),
		ComunPolicia:   r.FormValue("comun_policia"),
		DscLocal:       r.FormValue("desc_local"),
		DtAtendimento:  r.FormValue("dt_atendimento"),
		CID10:          r.FormValue("cid10"),
		DescLesao:      r.FormValue("desc_lesao"),
		NomeMedico:     r.FormValue("nome_medico"),
		CRMMedico:      r.FormValue("crm_medico"),
		UFMedico:       r.FormValue("uf_medico"),
		HrsTrabAntes:   r.FormValue("hrs_trab_antes"),
		TpCat:          r.FormValue("tp_cat"),
		CodSitGeradora: r.FormValue("cod_sit_geradora"),
		IniciatCAT:     r.FormValue("iniciat_cat"),
		ObsCAT:         r.FormValue("obs_cat"),
		UltDiaTrab:     r.FormValue("ult_dia_trab"),
		TpLocal:        r.FormValue("tp_local"),
		DscLograd:      r.FormValue("dsc_lograd"),
		NrLograd:       r.FormValue("nr_lograd"),
		Bairro:         r.FormValue("bairro"),
		CEP:            r.FormValue("cep"),
		CodMunic:       r.FormValue("cod_munic"),
		UF:             r.FormValue("uf_local"),
		ParteCorpo:     r.FormValue("parte_corpo"),
		Lateralidade:   r.FormValue("lateralidade"),
		AgenteCausador: r.FormValue("agente_causador"),
		HrAtendimento:  r.FormValue("hr_atendimento"),
		IndInternacao:  r.FormValue("ind_internacao"),
		DurTrat:        r.FormValue("dur_trat"),
		IndAfast:       r.FormValue("ind_afast"),
		DscCompLesao:   r.FormValue("dsc_comp_lesao"),
		DiagProvavel:   r.FormValue("diag_provavel"),
		Observacao:     r.FormValue("observacao"),
		OrgaoClasseMed: r.FormValue("orgao_classe_med"),
	}
	xmlGerado := GerarXMLS2210(params)

	evento := &storage.Evento{
		ID:            params.ID,
		Tipo:          "S-2210",
		ColaboradorID: colab.ID,
		Ambiente:      cfg.Ambiente,
		Status:        "pronto",
		XMLGerado:     xmlGerado,
		CriadoEm:      time.Now(),
		AtualizadoEm:  time.Now(),
	}

	if err := s.db.SalvarEvento(evento); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao gravar o evento S-2210: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("CAT S-2210 (%s) gerada para %s!", evento.ID, colab.Nome), false)
}

// -------------------------------------------------------------
// HANDLERS: EVENTO S-2220 (ASO)
// -------------------------------------------------------------

type DadosViewEditorS2220 struct {
	Titulo                   string
	MenuAtivo                string
	Config                   *storage.Configuracao
	Colaboradores            []storage.Colaborador
	ColaboradorSelecionadoID string
	Hoje                     string
	MensagemFlash            string
	FlashErro                bool

	// Tabela 27 - procedimentos diagnósticos.
	Procedimentos []data.ItemTabela
}

func (s *Servidor) handleEditorS2220(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabs, _ := s.db.ListarColaboradores()
	colabID := r.URL.Query().Get("colaborador_id")

	s.render(w, r, "evento_s2220", DadosViewEditorS2220{
		Titulo:                   "S-2220 • Monitoramento da Saúde (ASO)",
		MenuAtivo:                "s2220",
		Config:                   cfg,
		Colaboradores:            colabs,
		ColaboradorSelecionadoID: colabID,
		Hoje:                     time.Now().Format("2006-01-02"),
		Procedimentos:            s.procedimentos,
	})
}

func (s *Servidor) handleSalvarS2220(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	colabID := r.FormValue("colaborador_id")
	colab, err := s.db.ObterColaborador(colabID)
	if err != nil || colab == nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2220", DadosViewEditorS2220{
			Titulo:        "S-2220 • Monitoramento da Saúde (ASO)",
			MenuAtivo:     "s2220",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			Procedimentos: s.procedimentos,
			MensagemFlash: "Selecione o trabalhador avaliado no ASO.",
			FlashErro:     true,
		})
		return
	}

	// Sem o CNPJ do empregador o documento é inválido no leiaute S-1.3 (nrInsc).
	if limpaDigitos(cfg.CNPJ) == "" {
		s.handleFilaComFlash(w, r, "Configure o CNPJ da empresa em Certificado & Empresa antes de gerar eventos do eSocial.", true)
		return
	}

	params := ParametrosS2220{
		ID:             GerarIDEvento(cfg.CNPJ),
		Ambiente:       cfg.Ambiente,
		CNPJ:           cfg.CNPJ,
		CPFTrabalhador: colab.CPF,
		Matricula:      colab.Matricula,
		TipoExame:      r.FormValue("tipo_exame"),
		DataASO:        r.FormValue("dt_aso"),
		ResultadoASO:   r.FormValue("resultado_aso"),
		NomeMedico:     r.FormValue("nome_medico"),
		CRMMedico:      r.FormValue("crm_medico"),
		UFMedico:       r.FormValue("uf_medico"),
		ProcRealizado:  r.FormValue("proc_realizado"),
		ObsProc:        r.FormValue("obs_proc"),
		OrdExame:       r.FormValue("ord_exame"),
		IndResult:      r.FormValue("ind_result"),
		CPFCoord:       r.FormValue("cpf_coord"),
		UFCoord:        r.FormValue("uf_coord"),
		ObsASO:         r.FormValue("obs_aso"),
	}
	xmlGerado := GerarXMLS2220(params)

	evento := &storage.Evento{
		ID:            params.ID,
		Tipo:          "S-2220",
		ColaboradorID: colab.ID,
		Ambiente:      cfg.Ambiente,
		Status:        "pronto",
		XMLGerado:     xmlGerado,
		CriadoEm:      time.Now(),
		AtualizadoEm:  time.Now(),
	}

	if err := s.db.SalvarEvento(evento); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao gravar o evento S-2220: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento de ASO S-2220 (%s) gerado para %s!", evento.ID, colab.Nome), false)
}

func (s *Servidor) handleImportarXMLASO(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	// Limite de corpo/arquivo para o XML importado (achado M-02).
	r.Body = http.MaxBytesReader(w, r.Body, limiteUploadXML)
	arquivo, headerXML, err := r.FormFile("arquivo_xml_aso")
	if err == nil && headerXML != nil && headerXML.Size > limiteUploadXML {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2220", DadosViewEditorS2220{
			Titulo:        "S-2220 • Monitoramento da Saúde (ASO)",
			MenuAtivo:     "s2220",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			Procedimentos: s.procedimentos,
			MensagemFlash: "Arquivo XML excede o limite de 5 MB permitido.",
			FlashErro:     true,
		})
		return
	}
	if err != nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2220", DadosViewEditorS2220{
			Titulo:        "S-2220 • Monitoramento da Saúde (ASO)",
			MenuAtivo:     "s2220",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			Procedimentos: s.procedimentos,
			MensagemFlash: "Falha ao receber arquivo XML.",
			FlashErro:     true,
		})
		return
	}
	defer arquivo.Close()

	xmlBytes, err := io.ReadAll(io.LimitReader(arquivo, limiteUploadXML))
	if err != nil {
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2220", DadosViewEditorS2220{
			Titulo:        "S-2220 • Monitoramento da Saúde (ASO)",
			MenuAtivo:     "s2220",
			Config:        cfg,
			Colaboradores: colabs,
			Hoje:          time.Now().Format("2006-01-02"),
			Procedimentos: s.procedimentos,
			MensagemFlash: "Erro ao ler arquivo XML enviado: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	// Extração inteligente de campos do XML do ASO
	xmlStr := string(xmlBytes)
	cpf := extrairTagXML(xmlStr, "cpfTrab")
	if cpf == "" {
		cpf = extrairTagXML(xmlStr, "cpf")
	}

	// Localiza o colaborador pelo CPF extraído
	colabs, _ := s.db.ListarColaboradores()
	var colabID string
	colabNome := "Trabalhador Identificado"
	for _, c := range colabs {
		if c.CPF == limpaDigitos(cpf) {
			colabID = c.ID
			colabNome = c.Nome
			break
		}
	}

	idEvento := GerarIDEvento(cfg.CNPJ)
	evento := &storage.Evento{
		ID:            idEvento,
		Tipo:          "S-2220",
		ColaboradorID: colabID,
		Ambiente:      cfg.Ambiente,
		Status:        "pronto",
		XMLGerado:     xmlStr,
		CriadoEm:      time.Now(),
		AtualizadoEm:  time.Now(),
	}

	if err := s.db.SalvarEvento(evento); err != nil {
		colabsErro, _ := s.db.ListarColaboradores()
		s.render(w, r, "evento_s2220", DadosViewEditorS2220{
			Titulo:        "S-2220 • Monitoramento da Saúde (ASO)",
			MenuAtivo:     "s2220",
			Config:        cfg,
			Colaboradores: colabsErro,
			Hoje:          time.Now().Format("2006-01-02"),
			Procedimentos: s.procedimentos,
			MensagemFlash: "Falha ao gravar o evento importado: " + err.Error(),
			FlashErro:     true,
		})
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("XML de ASO importado com sucesso para %s (ID: %s)!", colabNome, idEvento), false)
}

// -------------------------------------------------------------
// HANDLERS: CATÁLOGO DE EVENTOS (36 EVENTOS) & EDITOR GENÉRICO
// -------------------------------------------------------------

type DadosViewMenuEventos struct {
	Titulo        string
	MenuAtivo     string
	Config        *storage.Configuracao
	Grupos        []GrupoCatalogoView
	MensagemFlash string
	FlashErro     bool
}

func (s *Servidor) handleMenuEventos(w http.ResponseWriter, r *http.Request) {
	s.handleMenuEventosComFlash(w, r, "", false)
}

func (s *Servidor) handleMenuEventosComFlash(w http.ResponseWriter, r *http.Request, msg string, errFlash bool) {
	cfg, _ := s.db.ObterConfiguracao()
	grupos := s.obterGruposCatalogo("todos", "")

	s.render(w, r, "menu_eventos", DadosViewMenuEventos{
		Titulo:        "Catálogo de Eventos (36)",
		MenuAtivo:     "catalogo",
		Config:        cfg,
		Grupos:        grupos,
		MensagemFlash: msg,
		FlashErro:     errFlash,
	})
}

func (s *Servidor) handleCatalogoFiltro(w http.ResponseWriter, r *http.Request) {
	grupo := r.URL.Query().Get("grupo")
	q := r.URL.Query().Get("q")
	grupos := s.obterGruposCatalogo(grupo, q)

	tmpl := s.templates["menu_eventos"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "catalogo_eventos", struct {
		Grupos []GrupoCatalogoView
	}{
		Grupos: grupos,
	})
}

func (s *Servidor) handleVerEventoCatalogo(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	codNorm := strings.ToUpper(codigo)
	switch codNorm {
	case "S-2240", "S2240":
		s.handleEditorS2240(w, r)
		return
	case "S-2210", "S2210":
		s.handleEditorS2210(w, r)
		return
	case "S-2220", "S2220":
		s.handleEditorS2220(w, r)
		return
	default:
		s.handleNovoEventoGenerico(w, r)
		return
	}
}

type DadosViewEditorGenerico struct {
	Titulo                string
	MenuAtivo             string
	Config                *storage.Configuracao
	EventoInfo            *esocial.EventoCatalogo
	Colaboradores         []storage.Colaborador
	XMLTemplatePreenchido string
	MensagemFlash         string
	FlashErro             bool
}

func (s *Servidor) handleNovoEventoGenerico(w http.ResponseWriter, r *http.Request) {
	codigo := r.PathValue("codigo")
	evt := esocial.ObterEventoCatalogo(codigo)
	if evt == nil {
		s.handleMenuEventosComFlash(w, r, "Evento não localizado no catálogo: "+codigo, true)
		return
	}

	cfg, _ := s.db.ObterConfiguracao()
	colabs, _ := s.db.ListarColaboradores()

	idEvt := GerarIDEvento(cfg.CNPJ)
	hoje := time.Now().Format("2006-01-02")
	hojeMes := time.Now().Format("2006-01")

	xmlPreenchido := evt.TemplateXML
	// Substituições formato maiúsculo
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{ID}}", idEvt)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{IND_RETIF}}", "1")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{TP_AMB}}", strconv.Itoa(cfg.Ambiente))
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{PROC_EMI}}", "1")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{VER_PROC}}", "1.0.0")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{TP_INSC}}", "1")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{NR_INSC}}", cfg.CNPJ)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{CPF_TRAB}}", "00000000000")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{CPF_BENEF}}", "00000000000")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{MATRICULA}}", "MAT-001")
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_INICIO}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_ADM}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_ACID}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_ASO}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_ALTERACAO}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_DESLIG}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{DT_TERMINO}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{INI_VALID}}", hojeMes)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{PER_APUR}}", hojeMes)

	// Substituições formato dot-notation
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.ID}}", idEvt)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.CNPJ}}", cfg.CNPJ)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.Ambiente}}", strconv.Itoa(cfg.Ambiente))
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.Hoje}}", hoje)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.HojeMes}}", hojeMes)
	xmlPreenchido = strings.ReplaceAll(xmlPreenchido, "{{.CPF}}", "00000000000")

	s.render(w, r, "editor_generico", DadosViewEditorGenerico{
		Titulo:                fmt.Sprintf("Emitir %s • %s", evt.Codigo, evt.Nome),
		MenuAtivo:             "catalogo",
		Config:                cfg,
		EventoInfo:            evt,
		Colaboradores:         colabs,
		XMLTemplatePreenchido: xmlPreenchido,
	})
}

func (s *Servidor) handleSalvarEventoGenerico(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	codigo := r.FormValue("codigo_evento")
	xmlConteudo := strings.TrimSpace(r.FormValue("xml_conteudo"))
	colabID := r.FormValue("colaborador_id")

	if limpaDigitos(cfg.CNPJ) == "" {
		s.handleFilaComFlash(w, r, "Configure o CNPJ da empresa em Certificado & Empresa antes de gerar eventos do eSocial.", true)
		return
	}

	if xmlConteudo == "" {
		s.handleNovoEventoGenerico(w, r)
		return
	}

	var parsed struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(xmlConteudo), &parsed); err != nil {
		evt := esocial.ObterEventoCatalogo(codigo)
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "editor_generico", DadosViewEditorGenerico{
			Titulo:                fmt.Sprintf("Emitir %s", codigo),
			MenuAtivo:             "catalogo",
			Config:                cfg,
			EventoInfo:            evt,
			Colaboradores:         colabs,
			XMLTemplatePreenchido: xmlConteudo,
			MensagemFlash:         "Erro de sintaxe XML: " + err.Error(),
			FlashErro:             true,
		})
		return
	}

	// Achado B-02: o ID só é aceito no padrão oficial do eSocial
	// (ID + tpInsc + nrInsc(14) + timestamp(14) + seq(5) = 36 caracteres).
	idEvento := GerarIDEvento(cfg.CNPJ)
	avisoID := ""
	if idInformado := extrairIDEvento(xmlConteudo); idInformado != "" {
		if idEventoValido(idInformado) {
			idEvento = idInformado
		} else {
			avisoID = fmt.Sprintf(" O ID informado (%s) não segue o padrão oficial ID+14+14+5 dígitos e foi substituído por %s.", idInformado, idEvento)
		}
	}

	evento := &storage.Evento{
		ID:            idEvento,
		Tipo:          codigo,
		ColaboradorID: colabID,
		Ambiente:      cfg.Ambiente,
		Status:        "pronto",
		XMLGerado:     xmlConteudo,
		CriadoEm:      time.Now(),
		AtualizadoEm:  time.Now(),
	}

	// Achado B-04: erro de persistência não é mais descartado.
	if err := s.db.SalvarEvento(evento); err != nil {
		evt := esocial.ObterEventoCatalogo(codigo)
		colabs, _ := s.db.ListarColaboradores()
		s.render(w, r, "editor_generico", DadosViewEditorGenerico{
			Titulo:                fmt.Sprintf("Emitir %s", codigo),
			MenuAtivo:             "catalogo",
			Config:                cfg,
			EventoInfo:            evt,
			Colaboradores:         colabs,
			XMLTemplatePreenchido: xmlConteudo,
			MensagemFlash:         "Falha ao gravar o evento: " + err.Error(),
			FlashErro:             true,
		})
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s (%s) enfileirado para validação e assinatura.%s", codigo, idEvento, avisoID), false)
}

// -------------------------------------------------------------
// HANDLERS: CENTRAL DE TRANSMISSÃO / FILA
// -------------------------------------------------------------

type DadosViewFila struct {
	Titulo          string
	MenuAtivo       string
	Config          *storage.Configuracao
	Eventos         []storage.Evento
	FiltroStatus    string
	TotalGeral      int
	TotalProntos    int
	TotalAssinados  int
	TotalSimulados  int
	TotalAceitos    int
	TotalRejeitados int
	MensagemFlash   string
	FlashErro       bool
}

func (s *Servidor) handleFila(w http.ResponseWriter, r *http.Request) {
	s.handleFilaComFlash(w, r, "", false)
}

func (s *Servidor) handleFilaComFlash(w http.ResponseWriter, r *http.Request, msg string, errFlash bool) {
	cfg, _ := s.db.ObterConfiguracao()
	todos, _ := s.db.ListarEventos()

	st := r.URL.Query().Get("status")
	if st == "" {
		st = "todos"
	}

	var filtrados []storage.Evento
	var nPronto, nAssinado, nSimulado, nAceito, nRejeitado int
	for _, e := range todos {
		switch e.Status {
		case "pronto":
			nPronto++
		case "assinado":
			nAssinado++
		case "simulado":
			nSimulado++
		case "aceito":
			nAceito++
		case "rejeitado":
			nRejeitado++
		}

		if st == "todos" || e.Status == st {
			filtrados = append(filtrados, e)
		}
	}

	s.render(w, r, "fila", DadosViewFila{
		Titulo:          "Central de Transmissão eSocial",
		MenuAtivo:       "fila",
		Config:          cfg,
		Eventos:         filtrados,
		FiltroStatus:    st,
		TotalGeral:      len(todos),
		TotalProntos:    nPronto,
		TotalAssinados:  nAssinado,
		TotalSimulados:  nSimulado,
		TotalAceitos:    nAceito,
		TotalRejeitados: nRejeitado,
		MensagemFlash:   msg,
		FlashErro:       errFlash,
	})
}

func (s *Servidor) handleTabelaFila(w http.ResponseWriter, r *http.Request) {
	st := r.URL.Query().Get("status")
	if st == "" {
		st = "todos"
	}

	todos, _ := s.db.ListarEventos()
	var filtrados []storage.Evento
	for _, e := range todos {
		if st == "todos" || e.Status == st {
			filtrados = append(filtrados, e)
		}
	}

	tmpl := s.templates["fila"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "tabela_fila", DadosViewFila{
		Eventos:      filtrados,
		FiltroStatus: st,
	})
}

func (s *Servidor) handleDownloadXML(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		http.Error(w, "Evento não encontrado", http.StatusNotFound)
		return
	}

	conteudo := evento.XMLAssinado
	if conteudo == "" {
		conteudo = evento.XMLGerado
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xml\"", evento.ID))
	_, _ = w.Write([]byte(conteudo))
}

func (s *Servidor) handleDetalhesEvento(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		http.Error(w, "Evento não encontrado", http.StatusNotFound)
		return
	}

	tmpl := s.templates["fila"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "detalhes_evento", struct {
		Evento *storage.Evento
	}{
		Evento: evento,
	})
}

func (s *Servidor) handleValidarEvento(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		s.handleFilaComFlash(w, r, "Evento não localizado.", true)
		return
	}

	// Achado M-03: usa a validação real por schema (xmllint + XSD oficiais) quando o
	// tipo de evento possui schema embutido; sem schema, aplica (e informa) apenas as
	// checagens estruturais básicas.
	_, temSchema := esocial.ObterNomeXSD(evento.Tipo)
	errosSchema := esocial.ValidarXSD([]byte(evento.XMLGerado), evento.Tipo)

	if temSchema == nil {
		// Existe XSD oficial para este tipo: o resultado do validador é definitivo.
		if errosSchema != nil {
			msg := "Reprovado na validação por schema oficial: " + errosSchema.Error()
			if err := s.db.AtualizarStatusEvento(id, "rejeitado", "", "", msg); err != nil {
				s.handleFilaComFlash(w, r, "Falha ao registrar a rejeição: "+err.Error(), true)
				return
			}
			s.handleFilaComFlash(w, r, msg, true)
			return
		}
		modo := "validação por schema oficial (xmllint)"
		if !esocial.XmllintDisponivel() {
			modo = "validação estrutural nativa (xmllint ausente no sistema - instale-o para conferência por schema)"
		}
		msg := "XML aprovado na " + modo + "."
		if err := s.db.AtualizarStatusEvento(id, evento.Status, evento.Recibo, evento.Protocolo, msg); err != nil {
			s.handleFilaComFlash(w, r, "Falha ao registrar a validação: "+err.Error(), true)
			return
		}
		s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s validado com sucesso (%s).", id, modo), false)
		return
	}

	// Tipos sem schema embutido: checagens estruturais básicas, com modo informado.
	if valido, erros := ValidarEventoXSD(evento.Tipo, evento.XMLGerado); !valido {
		msg := "Erros de validação: " + strings.Join(erros, "; ")
		if err := s.db.AtualizarStatusEvento(id, "rejeitado", "", "", msg); err != nil {
			s.handleFilaComFlash(w, r, "Falha ao registrar a rejeição: "+err.Error(), true)
			return
		}
		s.handleFilaComFlash(w, r, msg, true)
		return
	}

	msg := "Checagens estruturais básicas aprovadas (schema XSD oficial não disponível para o evento " + evento.Tipo + ")."
	if err := s.db.AtualizarStatusEvento(id, evento.Status, evento.Recibo, evento.Protocolo, msg); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao registrar a validação: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s aprovado nas checagens estruturais básicas (sem schema XSD para este tipo).", id), false)
}

func (s *Servidor) handleAssinarEvento(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		s.handleFilaComFlash(w, r, "Evento não localizado.", true)
		return
	}

	// Achado A-02: o envelope gerado é explicitamente identificado como SIMULADO.
	// Nenhuma assinatura com valor jurídico é produzida enquanto o certificado não
	// for aplicado pelo fluxo real de assinatura.
	assinado := SimularAssinatura(evento.XMLGerado, cfg.RazaoSocial)
	evento.XMLAssinado = assinado
	evento.Status = "assinado"
	evento.MensagemRetorno = "ASSINATURA SIMULADA: envelope XMLDSig demonstrativo gerado localmente, SEM certificado digital aplicado e SEM validade jurídica."
	evento.AtualizadoEm = time.Now()

	if err := s.db.SalvarEvento(evento); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao gravar a assinatura simulada: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s recebeu assinatura SIMULADA (demonstração, sem validade jurídica).", id), false)
}

func (s *Servidor) handleTransmitirEvento(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		s.handleFilaComFlash(w, r, "Evento não localizado.", true)
		return
	}

	// Achado A-02: nenhum recibo/protocolo oficial é inventado. O evento é marcado
	// como "simulado" até que exista integração real com o webservice do eSocial.
	mensagem := MensagemTransmissaoSimulada(cfg.Ambiente, evento.Tipo)
	if err := s.db.AtualizarStatusEvento(id, "simulado", "", "", mensagem); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao registrar a simulação: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s marcado como SIMULADO. %s", id, mensagem), true)
}

func (s *Servidor) handleConsultarRecibo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	evento, err := s.db.ObterEvento(id)
	if err != nil || evento == nil {
		s.handleFilaComFlash(w, r, "Evento não localizado.", true)
		return
	}

	if evento.Recibo != "" {
		s.handleFilaComFlash(w, r, fmt.Sprintf("Recibo registrado para o evento: %s", evento.Recibo), false)
		return
	}

	// Achado A-02: sem integração real não existe recibo. Nada é inventado.
	msg := "Nenhum recibo disponível: este evento não foi transmitido ao eSocial. " +
		"A consulta de recibo real depende da integração com o webservice oficial (não conectada nesta versão)."
	if err := s.db.AtualizarStatusEvento(id, "simulado", "", "", msg); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao registrar a consulta: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, msg, true)
}

func (s *Servidor) handleExcluirEvento(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.ObterEvento(id); err != nil {
		s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s não encontrado na fila.", id), true)
		return
	}
	// Achado B-04: erro de exclusão é reportado ao usuário.
	if err := s.db.ExcluirEvento(id); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao excluir o evento: "+err.Error(), true)
		return
	}
	s.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s removido da fila.", id), false)
}

func (s *Servidor) handleAssinarLote(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	todos, _ := s.db.ListarEventos()

	qtd := 0
	falhas := 0
	for _, e := range todos {
		if e.Status == "pronto" || e.Status == "rejeitado" {
			assinado := SimularAssinatura(e.XMLGerado, cfg.RazaoSocial)
			e.XMLAssinado = assinado
			e.Status = "assinado"
			e.MensagemRetorno = "ASSINATURA SIMULADA em lote: envelope demonstrativo, sem certificado aplicado e sem validade jurídica."
			e.AtualizadoEm = time.Now()
			if err := s.db.SalvarEvento(&e); err != nil {
				falhas++
				continue
			}
			qtd++
		}
	}

	msg := fmt.Sprintf("Lote processado: %d eventos com assinatura SIMULADA (sem validade jurídica).", qtd)
	if falhas > 0 {
		msg += fmt.Sprintf(" %d evento(s) falharam ao gravar.", falhas)
	}
	s.handleFilaComFlash(w, r, msg, falhas > 0)
}

func (s *Servidor) handleTransmitirLote(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	todos, _ := s.db.ListarEventos()

	qtd := 0
	falhas := 0
	for _, e := range todos {
		if e.Status == "assinado" {
			mensagem := MensagemTransmissaoSimulada(cfg.Ambiente, e.Tipo)
			if err := s.db.AtualizarStatusEvento(e.ID, "simulado", "", "", mensagem); err != nil {
				falhas++
				continue
			}
			qtd++
		}
	}

	msg := fmt.Sprintf("Lote processado: %d eventos marcados como SIMULADOS (nenhum dado foi enviado ao eSocial).", qtd)
	if falhas > 0 {
		msg += fmt.Sprintf(" %d evento(s) falharam ao atualizar.", falhas)
	}
	s.handleFilaComFlash(w, r, msg, true)
}

// -------------------------------------------------------------
// HANDLERS: APIS DE AUTOCOMPLETE / BUSCA
// -------------------------------------------------------------

func normalizarTexto(s string) string {
	s = strings.ToLower(s)
	r := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c",
	)
	return r.Replace(s)
}

func (s *Servidor) handleBuscaRiscos(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	qNorm := normalizarTexto(q)

	var resultados []data.Risco
	if qNorm != "" {
		for _, rk := range s.riscos {
			if strings.Contains(normalizarTexto(rk.Codigo), qNorm) ||
				strings.Contains(normalizarTexto(rk.Nome), qNorm) ||
				strings.Contains(normalizarTexto(rk.Categoria), qNorm) {
				resultados = append(resultados, rk)
				if len(resultados) >= 15 {
					break
				}
			}
		}
	}

	tmpl := s.templates["evento_s2240"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "resultado_riscos", struct {
		Riscos []data.Risco
	}{
		Riscos: resultados,
	})
}

func (s *Servidor) handleBuscaCBOs(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	qNorm := normalizarTexto(q)

	var resultados []data.CBO
	if qNorm != "" {
		for _, cb := range s.cbos {
			if strings.Contains(normalizarTexto(cb.Codigo), qNorm) ||
				strings.Contains(normalizarTexto(cb.Titulo), qNorm) {
				resultados = append(resultados, cb)
				if len(resultados) >= 15 {
					break
				}
			}
		}
	}

	tmpl := s.templates["colaboradores"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "resultado_cbos", struct {
		CBOs []data.CBO
	}{
		CBOs: resultados,
	})
}

// Utilitário simples para extração de tag XML
func extrairTagXML(xmlStr, tag string) string {
	abertura := "<" + tag + ">"
	fechamento := "</" + tag + ">"
	i := strings.Index(xmlStr, abertura)
	if i == -1 {
		return ""
	}
	f := strings.Index(xmlStr[i:], fechamento)
	if f == -1 {
		return ""
	}
	return strings.TrimSpace(xmlStr[i+len(abertura) : i+f])
}

// Assegura imports usados
var _ = bytes.Buffer{}
var _ = xml.Header

// idEventoValido verifica o padrão oficial do identificador de evento do eSocial:
// "ID" + tpInsc (1) + nrInsc (14) + YYYYMMDDHHMMSS (14) + sequencial (5) = 36 caracteres.
func idEventoValido(id string) bool {
	if len(id) != 36 || !strings.HasPrefix(id, "ID") {
		return false
	}
	for _, c := range id[2:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// extrairIDEvento lê o atributo Id do elemento raiz do evento sem confiar em
// posições arbitrárias do documento (achado B-02).
func extrairIDEvento(xmlConteudo string) string {
	indice := strings.Index(xmlConteudo, "Id=\"")
	if indice < 0 {
		return ""
	}
	inicio := indice + 4
	fim := strings.Index(xmlConteudo[inicio:], "\"")
	if fim <= 0 {
		return ""
	}
	return strings.TrimSpace(xmlConteudo[inicio : inicio+fim])
}
