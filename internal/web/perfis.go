package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

// Limite do certificado enviado por perfil (achado M-02).
const limiteUploadPerfilPFX = 2 << 20 // 2 MB

// DadosViewPerfis alimenta a página de gestão de perfis multi-empresa.
type DadosViewPerfis struct {
	Titulo        string
	MenuAtivo     string
	Config        *storage.Configuracao
	Perfis        []storage.PerfilEmpresa
	PerfilAtivoID string
	MensagemFlash string
	FlashErro     bool
}

// handlePerfis lista os perfis cadastrados (múltiplos certificados/procurações).
func (s *Servidor) handlePerfis(w http.ResponseWriter, r *http.Request) {
	s.handlePerfisComFlash(w, r, "", false)
}

func (s *Servidor) handlePerfisComFlash(w http.ResponseWriter, r *http.Request, msg string, errFlash bool) {
	cfg, _ := s.db.ObterConfiguracao()
	perfis, err := s.db.ListarPerfis()
	if err != nil {
		s.render(w, r, "perfis", DadosViewPerfis{
			Titulo: "Empresas e Certificados", MenuAtivo: "perfis", Config: cfg,
			MensagemFlash: "Falha ao consultar os perfis: " + err.Error(), FlashErro: true,
		})
		return
	}

	ativoID := ""
	for _, p := range perfis {
		if p.Ativo {
			ativoID = p.ID
		}
	}

	s.render(w, r, "perfis", DadosViewPerfis{
		Titulo:        "Empresas e Certificados",
		MenuAtivo:     "perfis",
		Config:        cfg,
		Perfis:        perfis,
		PerfilAtivoID: ativoID,
		MensagemFlash: msg,
		FlashErro:     errFlash,
	})
}

// handleSalvarPerfil cria ou atualiza um perfil de empresa (multi-empresa/procuração).
func (s *Servidor) handleSalvarPerfil(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.FormValue("perfil_id"))
	cnpj := limpaDigitos(r.FormValue("cnpj"))
	razao := strings.TrimSpace(r.FormValue("razao_social"))

	if len(cnpj) != 14 {
		s.handlePerfisComFlash(w, r, "CNPJ inválido: informe os 14 dígitos do contribuinte.", true)
		return
	}
	if razao == "" {
		s.handlePerfisComFlash(w, r, "Informe a razão social do perfil.", true)
		return
	}

	perfil := &storage.PerfilEmpresa{}
	if id != "" {
		existente, err := s.db.ObterPerfil(id)
		if err != nil {
			s.handlePerfisComFlash(w, r, "Perfil não encontrado para edição.", true)
			return
		}
		perfil = existente
	} else {
		perfil.ID = fmt.Sprintf("perfil-%x", hashTexto(cnpj))
		perfil.CriadoEm = time.Now()
	}

	perfil.RazaoSocial = razao
	perfil.CNPJ = cnpj
	perfil.Ambiente = 2
	if amb, err := parseAmbiente(r.FormValue("ambiente")); err == nil {
		perfil.Ambiente = amb
	}
	if tipo := strings.TrimSpace(r.FormValue("tipo_certificado")); tipo != "" {
		perfil.TipoCertificado = tipo
	} else if perfil.TipoCertificado == "" {
		perfil.TipoCertificado = "A1"
	}
	perfil.ProcuradorNome = strings.TrimSpace(r.FormValue("procurador_nome"))
	perfil.ProcuradorDoc = limpaDigitos(r.FormValue("procurador_doc"))
	perfil.AtualizadoEm = time.Now()

	if err := s.db.SalvarPerfil(perfil); err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao gravar o perfil: "+err.Error(), true)
		return
	}

	// O primeiro perfil cadastrado passa a ser o ativo automaticamente.
	perfis, _ := s.db.ListarPerfis()
	if len(perfis) == 1 {
		if err := s.db.AtivarPerfil(perfil.ID); err != nil {
			s.handlePerfisComFlash(w, r, "Perfil gravado, porém falhou a ativação: "+err.Error(), true)
			return
		}
	}

	s.handlePerfisComFlash(w, r, fmt.Sprintf("Perfil de %s gravado com sucesso.", razao), false)
}

// handleAtivarPerfil define o perfil ativo e sincroniza a configuração global.
func (s *Servidor) handleAtivarPerfil(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.AtivarPerfil(id); err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao ativar o perfil: "+err.Error(), true)
		return
	}
	perfil, _ := s.db.ObterPerfil(id)
	nome := id
	if perfil != nil {
		nome = perfil.RazaoSocial
	}
	s.handlePerfisComFlash(w, r, fmt.Sprintf("Perfil ativo alterado para %s. Os próximos eventos usarão este CNPJ e certificado.", nome), false)
}

// handleExcluirPerfil remove um perfil (o perfil ativo não pode ser removido diretamente).
func (s *Servidor) handleExcluirPerfil(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	perfil, err := s.db.ObterPerfil(id)
	if err != nil {
		s.handlePerfisComFlash(w, r, "Perfil não encontrado.", true)
		return
	}
	if perfil.Ativo {
		s.handlePerfisComFlash(w, r, "Não é possível excluir o perfil ativo. Ative outro perfil antes de excluir este.", true)
		return
	}
	if err := s.db.ExcluirPerfil(id); err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao excluir o perfil: "+err.Error(), true)
		return
	}
	s.handlePerfisComFlash(w, r, fmt.Sprintf("Perfil de %s removido.", perfil.RazaoSocial), false)
}

// handleUploadCertificadoPerfil recebe o certificado A1 de um perfil específico.
func (s *Servidor) handleUploadCertificadoPerfil(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	perfil, err := s.db.ObterPerfil(id)
	if err != nil {
		s.handlePerfisComFlash(w, r, "Perfil não encontrado.", true)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, limiteUploadPerfilPFX)
	arquivo, header, err := r.FormFile("arquivo_pfx")
	if err != nil {
		msg := "Falha ao receber o certificado do perfil."
		if _, excedeu := err.(*http.MaxBytesError); excedeu {
			msg = "Arquivo de certificado excede o limite de 2 MB permitido."
		}
		s.handlePerfisComFlash(w, r, msg, true)
		return
	}
	defer arquivo.Close()

	if header != nil && header.Size > limiteUploadPerfilPFX {
		s.handlePerfisComFlash(w, r, "Arquivo de certificado excede o limite de 2 MB permitido.", true)
		return
	}

	destinoDir := filepath.Join("./dados/certificados", perfil.ID)
	if err := os.MkdirAll(destinoDir, 0700); err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao preparar o diretório do certificado: "+err.Error(), true)
		return
	}
	_ = os.Chmod(destinoDir, 0700)

	// Nome fixo: o nome escolhido por quem envia não decide onde o arquivo vai parar.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pfx" && ext != ".p12" {
		s.handlePerfisComFlash(w, r, "Envie o certificado A1 em arquivo .pfx ou .p12.", true)
		return
	}
	caminho := filepath.Join(destinoDir, "certificado"+ext)
	destino, err := os.OpenFile(caminho, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao gravar o certificado: "+err.Error(), true)
		return
	}
	if _, err := copiarLimitado(destino, arquivo, limiteUploadPerfilPFX); err != nil {
		destino.Close()
		s.handlePerfisComFlash(w, r, "Falha ao gravar o certificado: "+err.Error(), true)
		return
	}
	destino.Close()
	_ = os.Chmod(caminho, 0600)

	perfil.CertificadoPath = caminho
	senha := r.FormValue("senha_a1")
	if senha != "" {
		cert, errCert := crypto.CarregarA1Arquivo(caminho, senha)
		if errCert != nil {
			// mantém o arquivo, mas informa que a senha não conferiu
			_ = s.db.SalvarPerfil(perfil)
			s.handlePerfisComFlash(w, r, "Certificado gravado, porém a senha informada não conferiu: "+errCert.Error(), true)
			return
		}
		perfil.CertificadoValidoAte = cert.ValidoAte()
		if perfil.RazaoSocial == "" {
			perfil.RazaoSocial = cert.RazaoSocial()
		}
		if perfil.CNPJ == "" {
			perfil.CNPJ = cert.CNPJ()
		}
	}
	perfil.AtualizadoEm = time.Now()
	if err := s.db.SalvarPerfil(perfil); err != nil {
		s.handlePerfisComFlash(w, r, "Falha ao atualizar o perfil com o certificado: "+err.Error(), true)
		return
	}

	// Se este é o perfil ativo, sincroniza o certificado na configuração global.
	if perfil.Ativo {
		_ = s.db.AtivarPerfil(perfil.ID)
	}

	msg := fmt.Sprintf("Certificado do perfil %s gravado com sucesso.", perfil.RazaoSocial)
	if !perfil.CertificadoValidoAte.IsZero() {
		msg += fmt.Sprintf(" Válido até %s.", perfil.CertificadoValidoAte.Format("02/01/2006"))
	}
	s.handlePerfisComFlash(w, r, msg, false)
}

func parseAmbiente(v string) (int, error) {
	switch strings.TrimSpace(v) {
	case "1":
		return 1, nil
	case "2", "":
		return 2, nil
	default:
		return 0, fmt.Errorf("ambiente inválido")
	}
}

// copiarLimitado copia até o limite informado, evitando consumo ilimitado de disco.
func copiarLimitado(destino *os.File, origem io.Reader, limite int64) (int64, error) {
	total, err := io.Copy(destino, io.LimitReader(origem, limite+1))
	if err != nil {
		return total, err
	}
	if total > limite {
		return total, fmt.Errorf("arquivo excede o limite de %d MB", limite/(1<<20))
	}
	return total, nil
}
