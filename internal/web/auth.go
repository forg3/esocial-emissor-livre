package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// AUTENTICACAO LOCAL + SESSAO + CSRF + RATE LIMIT
//
// Correcao dos achados:
//   C-01 (aplicacao sem autenticacao / bind em 0.0.0.0)
//   A-01 (CSRF em endpoints de escrita)
//   M-04 (forca bruta da senha do certificado A1)
//   B-03 (cabecalhos de seguranca)
// -----------------------------------------------------------------------------

const (
	nomeCookieSessao = "esocial_sessao"
	cabecalhoCSRF    = "X-CSRF-Token"
	arquivoAuth      = "auth.json"
)

// OpcoesAuth parametriza a autenticacao local.
type OpcoesAuth struct {
	// Senha em texto claro informada pelo operador (flag -senha ou env ESOCIAL_SENHA).
	// Quando vazia, uma senha aleatoria e gerada e devolvida por SenhaInicial().
	Senha string
	// Diretorio onde o hash da senha e persistido (padrao ./dados).
	Diretorio string
	// Hosts adicionais aceitos no cabecalho Host (allowlist anti DNS rebinding).
	HostsExtras []string
	// Duracao da sessao em horas (padrao 12).
	HorasSessao int
}

type registroAuth struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type sessao struct {
	ExpiraEm time.Time
}

// limitador implementa rate limit simples por chave (IP) com janela deslizante.
type limitador struct {
	mu         sync.Mutex
	tentativas map[string][]time.Time
	limite     int
	janela     time.Duration
}

func novoLimitador(limite int, janela time.Duration) *limitador {
	return &limitador{tentativas: make(map[string][]time.Time), limite: limite, janela: janela}
}

func (l *limitador) permitir(chave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	agora := time.Now()
	validas := l.tentativas[chave][:0]
	for _, t := range l.tentativas[chave] {
		if agora.Sub(t) < l.janela {
			validas = append(validas, t)
		}
	}
	if len(validas) >= l.limite {
		l.tentativas[chave] = validas
		return false
	}
	l.tentativas[chave] = append(validas, agora)
	return true
}

// autenticacao concentra o estado de autenticacao do processo.
type autenticacao struct {
	mu           sync.RWMutex
	sessoes      map[string]sessao
	salt         []byte
	hashSenha    []byte
	senhaInicial string
	csrf         string
	cspNonce     string
	hostsOK      map[string]bool
	hostLivre    bool
	horas        int
	limiteLogin  *limitador
	limiteCert   *limitador
	arquivo      string
}

func novoServicoAuth(opts OpcoesAuth) (*autenticacao, error) {
	a := &autenticacao{
		sessoes:     make(map[string]sessao),
		hostsOK:     map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true, "[::1]": true},
		horas:       opts.HorasSessao,
		limiteLogin: novoLimitador(8, time.Minute),
		limiteCert:  novoLimitador(6, time.Minute),
	}
	if a.horas <= 0 {
		a.horas = 12
	}

	for _, h := range opts.HostsExtras {
		h = strings.TrimSpace(strings.ToLower(h))
		if h == "" {
			continue
		}
		if h == "0.0.0.0" || h == "::" || h == "*" {
			a.hostLivre = true
			continue
		}
		a.hostsOK[h] = true
	}

	dir := opts.Diretorio
	if dir == "" {
		dir = "./dados"
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("falha ao criar diretorio de autenticacao: %w", err)
	}
	a.arquivo = filepath.Join(dir, arquivoAuth)

	if err := a.carregarOuCriarCredencial(opts.Senha); err != nil {
		return nil, err
	}

	csrf, err := aleatorioHex(32)
	if err != nil {
		return nil, err
	}
	a.csrf = csrf

	nonce, err := aleatorioHex(16)
	if err != nil {
		return nil, err
	}
	a.cspNonce = nonce

	return a, nil
}

// carregarOuCriarCredencial define o hash da senha: se o operador informou uma
// senha, ela prevalece (e substitui o hash persistido); caso contrario reutiliza
// o hash salvo em disco ou gera uma senha aleatoria na primeira execucao.
func (a *autenticacao) carregarOuCriarCredencial(senhaInformada string) error {
	if senhaInformada != "" {
		salt, err := aleatorioBytes(16)
		if err != nil {
			return err
		}
		a.salt = salt
		a.hashSenha = derivarHash(salt, senhaInformada)
		return a.persistir()
	}

	if dados, err := os.ReadFile(a.arquivo); err == nil {
		var reg registroAuth
		if err := json.Unmarshal(dados, &reg); err == nil && reg.Salt != "" && reg.Hash != "" {
			salt, errS := hex.DecodeString(reg.Salt)
			hash, errH := hex.DecodeString(reg.Hash)
			if errS == nil && errH == nil && len(salt) > 0 && len(hash) > 0 {
				a.salt = salt
				a.hashSenha = hash
				return nil
			}
		}
	}

	senha, err := aleatorioHex(12)
	if err != nil {
		return err
	}
	salt, err := aleatorioBytes(16)
	if err != nil {
		return err
	}
	a.salt = salt
	a.hashSenha = derivarHash(salt, senha)
	a.senhaInicial = senha
	return a.persistir()
}

func (a *autenticacao) persistir() error {
	reg := registroAuth{Salt: hex.EncodeToString(a.salt), Hash: hex.EncodeToString(a.hashSenha)}
	dados, err := json.Marshal(reg)
	if err != nil {
		return err
	}
	return os.WriteFile(a.arquivo, dados, 0600)
}

// SenhaInicial devolve a senha gerada automaticamente (vazia quando o operador
// informou a propria senha ou quando o hash foi reutilizado de disco).
func (a *autenticacao) SenhaInicial() string { return a.senhaInicial }

// CSRF devolve o token anti-CSRF do processo (usado nos formularios e no header HTMX).
func (a *autenticacao) CSRF() string { return a.csrf }

// CSPNonce devolve o nonce da politica de conteudo.
func (a *autenticacao) CSPNonce() string { return a.cspNonce }

func derivarHash(salt []byte, senha string) []byte {
	// Derivacao com SHA-256 iterado (100k) - suficiente para um segredo local
	// de alta entropia, sem dependencia externa.
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(senha))
	digest := h.Sum(nil)
	for i := 0; i < 100000; i++ {
		h.Reset()
		h.Write(salt)
		h.Write(digest)
		digest = h.Sum(nil)
	}
	return digest
}

func (a *autenticacao) verificarSenha(senha string) bool {
	calc := derivarHash(a.salt, senha)
	return subtle.ConstantTimeCompare(calc, a.hashSenha) == 1
}

func (a *autenticacao) criarSessao() (string, error) {
	token, err := aleatorioHex(32)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	a.sessoes[token] = sessao{ExpiraEm: time.Now().Add(time.Duration(a.horas) * time.Hour)}
	a.mu.Unlock()
	return token, nil
}

func (a *autenticacao) sessaoValida(token string) bool {
	if token == "" {
		return false
	}
	a.mu.RLock()
	s, ok := a.sessoes[token]
	a.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(s.ExpiraEm) {
		a.mu.Lock()
		delete(a.sessoes, token)
		a.mu.Unlock()
		return false
	}
	return true
}

func (a *autenticacao) encerrarSessao(token string) {
	a.mu.Lock()
	delete(a.sessoes, token)
	a.mu.Unlock()
}

func aleatorioBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("falha ao gerar aleatoriedade criptografica: %w", err)
	}
	return b, nil
}

func aleatorioHex(n int) (string, error) {
	b, err := aleatorioBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ipDoCliente(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func constantesIguais(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// -----------------------------------------------------------------------------
// MIDDLEWARES
// -----------------------------------------------------------------------------

// middlewareCabecalhos adiciona cabecalhos de seguranca (achado B-03).
func (s *Servidor) middlewareCabecalhos(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; "+
				"form-action 'self'; img-src 'self' data:; font-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self' 'unsafe-inline'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// middlewareHost valida o cabecalho Host (mitiga DNS rebinding).
func (s *Servidor) middlewareHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.hostLivre {
			host := strings.ToLower(r.Host)
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			host = strings.Trim(host, "[]")
			if !s.auth.hostsOK[host] {
				http.Error(w, "Host nao autorizado", http.StatusMisdirectedRequest)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func rotaPublica(p string) bool {
	return p == "/login" || p == "/logout" || strings.HasPrefix(p, "/static/") ||
		p == "/favicon.ico" || p == "/robots.txt"
}

// middlewareAuth exige sessao valida em todas as rotas nao publicas (achado C-01).
func (s *Servidor) middlewareAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rotaPublica(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(nomeCookieSessao)
		if err == nil && s.auth.sessaoValida(cookie.Value) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.Error(w, "Sessao expirada ou inexistente. Autentique-se em /login.", http.StatusUnauthorized)
	})
}

// middlewareCSRF valida Origin/Referer e token anti-CSRF em metodos de escrita (achado A-01).
func (s *Servidor) middlewareCSRF(next http.Handler) http.Handler {
	metodos := map[string]bool{
		http.MethodPost: true, http.MethodPut: true,
		http.MethodPatch: true, http.MethodDelete: true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !metodos[r.Method] {
			next.ServeHTTP(w, r)
			return
		}

		// Limite global de corpo para qualquer escrita (achado M-02).
		r.Body = http.MaxBytesReader(w, r.Body, 12<<20)

		if origem := r.Header.Get("Origin"); origem != "" {
			if !mesmoHost(origem, r.Host) {
				http.Error(w, "Origem nao autorizada", http.StatusForbidden)
				return
			}
		}
		if ref := r.Header.Get("Referer"); ref != "" {
			if !mesmoHost(ref, r.Host) {
				http.Error(w, "Referer nao autorizado", http.StatusForbidden)
				return
			}
		}

		if err := r.ParseMultipartForm(4 << 20); err != nil {
			if _, ok := err.(*http.MaxBytesError); ok {
				http.Error(w, "Corpo da requisicao excede o limite permitido.", http.StatusRequestEntityTooLarge)
				return
			}
		}

		token := r.Header.Get(cabecalhoCSRF)
		if token == "" {
			token = r.FormValue("csrf_token")
		}
		if !constantesIguais(token, s.auth.csrf) {
			http.Error(w, "Token anti-CSRF ausente ou invalido. Recarregue a pagina.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func mesmoHost(origem, host string) bool {
	u := strings.ToLower(origem)
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "https://")
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return strings.EqualFold(u, strings.ToLower(host))
}

// -----------------------------------------------------------------------------
// HANDLERS DE LOGIN
// -----------------------------------------------------------------------------

type dadosLogin struct {
	Erro        string
	CSRF        string
	RazaoSocial string
}

func (s *Servidor) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.templates["login"]
	if !ok {
		http.Error(w, "Template de login indisponivel", http.StatusInternalServerError)
		return
	}
	cfg, _ := s.db.ObterConfiguracao()
	dados := dadosLogin{CSRF: s.auth.csrf}
	if cfg != nil {
		dados.RazaoSocial = cfg.RazaoSocial
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "login.html", dados)
}

func (s *Servidor) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := ipDoCliente(r)
	if !s.auth.limiteLogin.permitir(ip) {
		log.Printf("[seguranca] rate limit de login atingido para %s", ip)
		w.Header().Set("Retry-After", "60")
		http.Error(w, "Muitas tentativas de login. Aguarde um minuto.", http.StatusTooManyRequests)
		return
	}

	senha := r.FormValue("senha")
	if !s.auth.verificarSenha(senha) {
		log.Printf("[seguranca] tentativa de login invalida de %s", ip)
		tmpl := s.templates["login"]
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_ = tmpl.ExecuteTemplate(w, "login.html", dadosLogin{
			Erro: "Senha incorreta. Verifique a senha exibida no terminal ou em dados/auth.json.",
			CSRF: s.auth.csrf,
		})
		return
	}

	token, err := s.auth.criarSessao()
	if err != nil {
		http.Error(w, "Falha ao iniciar sessao", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     nomeCookieSessao,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   s.auth.horas * 3600,
	})
	log.Printf("[seguranca] login efetuado com sucesso de %s", ip)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Servidor) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(nomeCookieSessao); err == nil {
		s.auth.encerrarSessao(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     nomeCookieSessao,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// permitirTesteCertificado aplica rate limit na validacao de senha do certificado (achado M-04).
func (s *Servidor) permitirTesteCertificado(r *http.Request) bool {
	return s.auth.limiteCert.permitir(ipDoCliente(r))
}
