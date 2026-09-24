package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/soap"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

// -----------------------------------------------------------------------------
// TRANSMISSÃO REAL AO ESOCIAL (webservice oficial, mTLS com certificado A1)
//
// O modo de transmissão é definido na configuração ("simulado" por padrão).
// No modo real o sistema:
//   1. exige o evento assinado e o certificado A1 carregado;
//   2. envia o lote ao webservice (EnviarLoteEventos) via mTLS;
//   3. persiste o protocolo e a mensagem de retorno oficiais;
//   4. consulta o processamento (ConsultarLoteEventos) para obter os recibos.
//
// Os endpoints podem ser sobrescritos pelas variáveis ESOCIAL_WS_ENVIO e
// ESOCIAL_WS_CONSULTA (útil para homologação/proxy corporativo e para testes).
// -----------------------------------------------------------------------------

// novoClienteESocial monta o cliente SOAP mTLS a partir do certificado A1 informado.
func novoClienteESocial(caminhoCertificado, senha string) (*soap.ClienteSOAP, error) {
	if strings.TrimSpace(caminhoCertificado) == "" {
		return nil, fmt.Errorf("nenhum certificado A1 configurado para o perfil ativo")
	}
	if _, err := os.Stat(caminhoCertificado); err != nil {
		return nil, fmt.Errorf("certificado não encontrado no disco: %w", err)
	}
	if senha == "" {
		return nil, fmt.Errorf("informe a senha do certificado para transmitir em modo real")
	}

	cert, err := crypto.CarregarA1Arquivo(caminhoCertificado, senha)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir o certificado A1: %w", err)
	}

	opcoes := []soap.OpcaoClienteSOAP{}
	if urlEnvio, urlConsulta := os.Getenv("ESOCIAL_WS_ENVIO"), os.Getenv("ESOCIAL_WS_CONSULTA"); urlEnvio != "" || urlConsulta != "" {
		if urlEnvio == "" {
			urlEnvio = urlConsulta
		}
		if urlConsulta == "" {
			urlConsulta = urlEnvio
		}
		opcoes = append(opcoes, soap.ComURLPersonalizada(urlEnvio, urlConsulta))
	}

	return soap.NovoClienteSOAP(cert, opcoes...)
}

// transmitirEventoReal envia o lote assinado ao webservice oficial e persiste o protocolo.
func (s *Servidor) transmitirEventoReal(w http.ResponseWriter, r *http.Request, cfg *storage.Configuracao, evento *storage.Evento) {
	senha := strings.TrimSpace(r.FormValue("senha_a1"))
	cliente, err := novoClienteESocial(cfg.CertificadoPath, senha)
	if err != nil {
		s.handleFilaComFlash(w, r, "Transmissão real não realizada: "+err.Error(), true)
		return
	}

	assinado := evento.XMLAssinado
	if strings.TrimSpace(assinado) == "" {
		s.handleFilaComFlash(w, r, "Assine o evento antes de transmitir em modo real (a assinatura é obrigatória no envio oficial).", true)
		return
	}
	// O envelope simulado carrega o comentário "ASSINATURA SIMULADA" e o valor
	// base64 do marcador; ambos são recusados no envio oficial.
	if !strings.Contains(assinado, "<Signature") {
		s.handleFilaComFlash(w, r, "O evento não possui assinatura XMLDSig no XML. Assine com o certificado A1 antes do envio oficial.", true)
		return
	}
	if strings.Contains(assinado, "ASSINATURA SIMULADA") ||
		strings.Contains(assinado, "U0lNVUxBQ0FPLU5BTy1WQUxJREEtSlVSSURJQ0FNRU5URQ==") {
		s.handleFilaComFlash(w, r,
			"O XML está com assinatura SIMULADA e seria rejeitado pelo eSocial. Assine novamente com o certificado A1 antes de transmitir em modo real.", true)
		return
	}

	ctx, cancelar := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancelar()

	resposta, err := cliente.EnviarLote(ctx, []byte(assinado), cfg.Ambiente)
	if err != nil {
		msg := "Falha de comunicação com o webservice do eSocial: " + err.Error()
		_ = s.db.AtualizarStatusEvento(evento.ID, evento.Status, "", "", msg)
		s.handleFilaComFlash(w, r, msg, true)
		return
	}

	status := "transmitido"
	mensagem := fmt.Sprintf("Lote enviado ao eSocial (código %d): %s", resposta.CodigoResposta, resposta.DescricaoResposta)
	if !resposta.Sucesso {
		status = "rejeitado"
		mensagem = fmt.Sprintf("Lote rejeitado pelo eSocial (código %d): %s", resposta.CodigoResposta, resposta.DescricaoResposta)
		if len(resposta.Ocorrencias) > 0 {
			var detalhes []string
			for _, oc := range resposta.Ocorrencias {
				detalhes = append(detalhes, fmt.Sprintf("[%d] %s (%s)", oc.Codigo, oc.Descricao, oc.Localizacao))
			}
			mensagem += " Ocorrências: " + strings.Join(detalhes, "; ")
		}
	}

	if err := s.db.AtualizarStatusEvento(evento.ID, status, "", resposta.ProtocoloEnvio, mensagem); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao registrar o retorno do eSocial: "+err.Error(), true)
		return
	}

	s.handleFilaComFlash(w, r, mensagem, !resposta.Sucesso)
}

// consultarReciboReal consulta o processamento do lote e persiste os recibos oficiais.
func (s *Servidor) consultarReciboReal(w http.ResponseWriter, r *http.Request, cfg *storage.Configuracao, evento *storage.Evento) {
	senha := strings.TrimSpace(r.FormValue("senha_a1"))
	cliente, err := novoClienteESocial(cfg.CertificadoPath, senha)
	if err != nil {
		s.handleFilaComFlash(w, r, "Consulta real não realizada: "+err.Error(), true)
		return
	}
	if strings.TrimSpace(evento.Protocolo) == "" {
		s.handleFilaComFlash(w, r, "O evento ainda não possui protocolo de envio. Transmita o lote antes de consultar o recibo.", true)
		return
	}

	ctx, cancelar := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancelar()

	resposta, err := cliente.ConsultarLote(ctx, evento.Protocolo, cfg.Ambiente)
	if err != nil {
		msg := "Falha na consulta ao webservice do eSocial: " + err.Error()
		_ = s.db.AtualizarStatusEvento(evento.ID, evento.Status, "", "", msg)
		s.handleFilaComFlash(w, r, msg, true)
		return
	}

	status := evento.Status
	recibo := ""
	mensagem := fmt.Sprintf("Consulta ao eSocial (código %d): %s", resposta.CodigoResposta, resposta.DescricaoResposta)

	for _, ev := range resposta.Eventos {
		if ev.IdEvento != "" && ev.IdEvento != evento.ID {
			continue
		}
		if ev.NumeroRecibo != "" {
			recibo = ev.NumeroRecibo
		}
		switch ev.CodigoResposta {
		case 201, 202:
			status = "aceito"
		case 0:
			// sem código de resposta específico: mantém o status atual
		default:
			status = "rejeitado"
		}
		if ev.DescricaoResposta != "" {
			mensagem += " | " + ev.DescricaoResposta
		}
		for _, oc := range ev.Ocorrencias {
			mensagem += fmt.Sprintf(" | [%d] %s (%s)", oc.Codigo, oc.Descricao, oc.Localizacao)
		}
		break
	}

	if err := s.db.AtualizarStatusEvento(evento.ID, status, recibo, "", mensagem); err != nil {
		s.handleFilaComFlash(w, r, "Falha ao registrar o retorno da consulta: "+err.Error(), true)
		return
	}

	if recibo != "" {
		mensagem += " Recibo: " + recibo
	}
	s.handleFilaComFlash(w, r, mensagem, status == "rejeitado")
}

// handleDefinirModoTransmissao alterna entre envio simulado e real.
func (s *Servidor) handleDefinirModoTransmissao(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	modo := strings.TrimSpace(r.FormValue("modo_transmissao"))
	if modo != "real" {
		modo = "simulado"
	}

	if modo == "real" {
		if strings.TrimSpace(cfg.CertificadoPath) == "" {
			s.render(w, r, "certificado", DadosViewConfiguracao{
				Titulo: "Certificado & Empresa", MenuAtivo: "configuracao", Config: cfg,
				MensagemFlash: "Carregue um certificado A1 antes de habilitar a transmissão real.",
				FlashErro:     true,
			})
			return
		}
		_, err := os.Stat(cfg.CertificadoPath)
		if err != nil {
			s.render(w, r, "certificado", DadosViewConfiguracao{
				Titulo: "Certificado & Empresa", MenuAtivo: "configuracao", Config: cfg,
				MensagemFlash: "O certificado configurado não foi encontrado no disco. Carregue-o novamente.",
				FlashErro:     true,
			})
			return
		}
	}

	cfg.ModoTransmissao = modo
	cfg.AtualizadoEm = time.Now()
	if err := s.db.SalvarConfiguracao(cfg); err != nil {
		s.render(w, r, "certificado", DadosViewConfiguracao{
			Titulo: "Certificado & Empresa", MenuAtivo: "configuracao", Config: cfg,
			MensagemFlash: "Falha ao salvar o modo de transmissão: " + err.Error(),
			FlashErro:     true,
		})
		return
	}

	msg := "Modo de transmissão definido como SIMULADO: nenhum dado é enviado ao governo."
	if modo == "real" {
		msg = "Modo de transmissão REAL habilitado: os eventos assinados serão enviados ao webservice oficial do eSocial."
	}
	s.render(w, r, "certificado", DadosViewConfiguracao{
		Titulo: "Certificado & Empresa", MenuAtivo: "configuracao", Config: cfg,
		MensagemFlash: msg,
	})
}
