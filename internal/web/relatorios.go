package web

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/importer"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

// Limite de tamanho do XML de retorno importado (achado M-02).
const limiteUploadRetorno = 10 << 20 // 10 MB

// DadosViewRelatorios alimenta a página de relatórios e auditoria de retorno.
type DadosViewRelatorios struct {
	Titulo             string
	MenuAtivo          string
	Config             *storage.Configuracao
	Resumo             []storage.ResumoRetornoTotalizacao
	Retornos           []storage.RetornoTotalizacao
	Periodos           []string
	FiltroPeriodo      string
	TotalEventosLocais int
	MensagemFlash      string
	FlashErro          bool
}

// handleRelatorios renderiza a consolidação dos totalizadores importados.
func (s *Servidor) handleRelatorios(w http.ResponseWriter, r *http.Request) {
	s.handleRelatoriosComFlash(w, r, "", false)
}

func (s *Servidor) handleRelatoriosComFlash(w http.ResponseWriter, r *http.Request, msg string, errFlash bool) {
	cfg, _ := s.db.ObterConfiguracao()
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))

	todos, err := s.db.ListarRetornosTotalizacao("")
	if err != nil {
		s.render(w, r, "relatorios", DadosViewRelatorios{
			Titulo: "Relatórios", MenuAtivo: "relatorios", Config: cfg,
			MensagemFlash: "Falha ao consultar os totalizadores: " + err.Error(), FlashErro: true,
		})
		return
	}

	periodos := map[string]bool{}
	for _, t := range todos {
		if t.PerApur != "" {
			periodos[t.PerApur] = true
		}
	}
	var listaPeriodos []string
	for p := range periodos {
		listaPeriodos = append(listaPeriodos, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(listaPeriodos)))

	filtrados := todos
	if periodo != "" {
		filtrados = nil
		for _, t := range todos {
			if t.PerApur == periodo {
				filtrados = append(filtrados, t)
			}
		}
	}

	resumo, _ := s.db.ObterResumoRetornos()
	if periodo != "" {
		var resumoFiltrado []storage.ResumoRetornoTotalizacao
		for _, res := range resumo {
			if res.PerApur == periodo {
				resumoFiltrado = append(resumoFiltrado, res)
			}
		}
		resumo = resumoFiltrado
	}

	eventos, _ := s.db.ListarEventos()

	s.render(w, r, "relatorios", DadosViewRelatorios{
		Titulo:             "Relatórios e Auditoria de Retorno",
		MenuAtivo:          "relatorios",
		Config:             cfg,
		Resumo:             resumo,
		Retornos:           filtrados,
		Periodos:           listaPeriodos,
		FiltroPeriodo:      periodo,
		TotalEventosLocais: len(eventos),
		MensagemFlash:      msg,
		FlashErro:          errFlash,
	})
}

// handleImportarRetorno importa um XML de retorno do eSocial (S-5001/S-5011 e correlatos).
func (s *Servidor) handleImportarRetorno(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, limiteUploadRetorno)

	arquivo, header, err := r.FormFile("arquivo_retorno")
	if err != nil {
		msg := "Falha ao receber o arquivo de retorno."
		if _, excedeu := err.(*http.MaxBytesError); excedeu {
			msg = "Arquivo de retorno excede o limite de 10 MB permitido."
		}
		s.handleRelatoriosComFlash(w, r, msg, true)
		return
	}
	defer arquivo.Close()

	if header != nil && header.Size > limiteUploadRetorno {
		s.handleRelatoriosComFlash(w, r, "Arquivo de retorno excede o limite de 10 MB permitido.", true)
		return
	}

	registros, err := importer.ImportarRetorno(arquivo)
	if err != nil {
		s.handleRelatoriosComFlash(w, r, "Não foi possível ler o retorno: "+err.Error(), true)
		return
	}

	importados := 0
	for i := range registros {
		reg := registros[i]
		idBase := reg.Tipo + "-" + reg.PerApur + "-" + reg.CPFTrab + "-" + reg.NRRecibo
		if reg.CPFTrab == "" && reg.NRRecibo == "" {
			idBase += fmt.Sprintf("-%d", i)
		}
		item := &storage.RetornoTotalizacao{
			ID:            fmt.Sprintf("ret-%x", hashTexto(idBase)),
			Tipo:          reg.Tipo,
			DescricaoTipo: descricaoTipoRetorno(reg.Tipo),
			PerApur:       reg.PerApur,
			CPFTrab:       reg.CPFTrab,
			Matricula:     reg.Matricula,
			NRRecibo:      reg.NRRecibo,
			Valores:       reg.Valores,
			TotalCentavos: reg.TotalCentavos,
			ImportadoEm:   time.Now(),
		}
		if err := s.db.SalvarRetornoTotalizacao(item); err != nil {
			s.handleRelatoriosComFlash(w, r, "Falha ao gravar o totalizador: "+err.Error(), true)
			return
		}
		importados++
	}

	s.handleRelatoriosComFlash(w, r,
		fmt.Sprintf("Retorno importado: %d totalizador(es) registrado(s) para conferência.", importados), false)
}

// handleExportarRelatorios gera o CSV de conferência dos totalizadores.
func (s *Servidor) handleExportarRelatorios(w http.ResponseWriter, r *http.Request) {
	periodo := strings.TrimSpace(r.URL.Query().Get("periodo"))
	lista, err := s.db.ListarRetornosTotalizacao(periodo)
	if err != nil {
		http.Error(w, "Falha ao consultar os totalizadores", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="relatorio-totalizadores-esocial.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM para Excel

	escritor := csv.NewWriter(w)
	_ = escritor.Write([]string{"periodo_apuracao", "evento", "descricao", "cpf_trabalhador", "matricula",
		"numero_recibo_base", "campos_memorias", "total_centavos", "total_reais", "importado_em"})

	for _, item := range lista {
		chaves := make([]string, 0, len(item.Valores))
		for k := range item.Valores {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		var memorias []string
		for _, k := range chaves {
			memorias = append(memorias, k+"="+item.Valores[k])
		}
		_ = escritor.Write([]string{
			item.PerApur,
			item.Tipo,
			item.DescricaoTipo,
			item.CPFTrab,
			item.Matricula,
			item.NRRecibo,
			strings.Join(memorias, "; "),
			fmt.Sprintf("%d", item.TotalCentavos),
			fmt.Sprintf("%.2f", float64(item.TotalCentavos)/100.0),
			item.ImportadoEm.Format("02/01/2006 15:04"),
		})
	}
	escritor.Flush()
}

// handleExcluirRetorno remove um totalizador importado.
func (s *Servidor) handleExcluirRetorno(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.ExcluirRetornoTotalizacao(id); err != nil {
		s.handleRelatoriosComFlash(w, r, "Falha ao excluir o totalizador: "+err.Error(), true)
		return
	}
	s.handleRelatoriosComFlash(w, r, "Totalizador removido do relatório.", false)
}

func descricaoTipoRetorno(tipo string) string {
	switch tipo {
	case "evtBasesTrab":
		return "S-5001 Contribuições sociais por trabalhador"
	case "evtIrrfBenef":
		return "S-5002 IRRF por trabalhador"
	case "evtFGTS":
		return "S-5003 FGTS por trabalhador"
	case "evtCS":
		return "S-5011 Contribuições sociais consolidadas"
	case "evtIrrf":
		return "S-5012 IRRF consolidado"
	case "evtFGTSCons":
		return "S-5013 FGTS consolidado"
	default:
		return tipo
	}
}

// hashTexto produz um identificador estável (FNV-1a) para o registro importado.
func hashTexto(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

// formatarCentavosBR formata valores monetários no padrão brasileiro (R$ 1.234,56).
func formatarCentavosBR(centavos int64) string {
	sinal := ""
	if centavos < 0 {
		sinal = "-"
		centavos = -centavos
	}
	inteiro := centavos / 100
	resto := centavos % 100

	texto := strconv.FormatInt(inteiro, 10)
	var grupos []string
	for len(texto) > 3 {
		grupos = append([]string{texto[len(texto)-3:]}, grupos...)
		texto = texto[:len(texto)-3]
	}
	grupos = append([]string{texto}, grupos...)
	return fmt.Sprintf("%sR$ %s,%02d", sinal, strings.Join(grupos, "."), resto)
}
