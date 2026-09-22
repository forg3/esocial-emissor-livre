package esocial

import (
	"fmt"
	"strings"
	"sync"

	"github.com/forg3/esocial-emissor-livre/internal/data"
)

var (
	tabelasOnce sync.Once
	mapaRiscos  map[string]data.Risco
	listaRiscos []data.Risco

	mapaCBOs  map[string]data.CBO
	listaCBOs []data.CBO

	erroCarga error
)

func inicializarTabelas() {
	riscos, err := data.CarregarRiscosEsocial()
	if err != nil {
		erroCarga = fmt.Errorf("erro ao carregar riscos da Tabela 24: %w", err)
		return
	}
	listaRiscos = riscos
	mapaRiscos = make(map[string]data.Risco, len(riscos))
	for _, r := range riscos {
		mapaRiscos[r.Codigo] = r
	}

	cbos, err := data.CarregarCBOs()
	if err != nil {
		erroCarga = fmt.Errorf("erro ao carregar tabela CBO: %w", err)
		return
	}
	listaCBOs = cbos
	mapaCBOs = make(map[string]data.CBO, len(cbos))
	for _, c := range cbos {
		// Indexa por código limpo sem hífen e com hífen
		mapaCBOs[c.Codigo] = c
		codLimpo := strings.ReplaceAll(c.Codigo, "-", "")
		mapaCBOs[codLimpo] = c
	}
}

// ObterRiscos retorna a lista completa da Tabela 24 do eSocial.
func ObterRiscos() ([]data.Risco, error) {
	tabelasOnce.Do(inicializarTabelas)
	if erroCarga != nil {
		return nil, erroCarga
	}
	return listaRiscos, nil
}

// BuscarRiscoPorCodigo busca um agente nocivo específico pelo código oficial (ex: "01.01.001").
func BuscarRiscoPorCodigo(codigo string) (data.Risco, bool) {
	tabelasOnce.Do(inicializarTabelas)
	if mapaRiscos == nil {
		return data.Risco{}, false
	}
	r, ok := mapaRiscos[codigo]
	return r, ok
}

// FiltrarRiscos busca agentes nocivos que contenham o termo no nome ou código.
func FiltrarRiscos(termo string) []data.Risco {
	tabelasOnce.Do(inicializarTabelas)
	termoNorm := strings.ToLower(strings.TrimSpace(termo))
	var res []data.Risco
	for _, r := range listaRiscos {
		if strings.Contains(strings.ToLower(r.Nome), termoNorm) || strings.Contains(r.Codigo, termoNorm) {
			res = append(res, r)
		}
	}
	return res
}

// ObterCBOs retorna a lista completa de ocupações CBO oficiais.
func ObterCBOs() ([]data.CBO, error) {
	tabelasOnce.Do(inicializarTabelas)
	if erroCarga != nil {
		return nil, erroCarga
	}
	return listaCBOs, nil
}

// BuscarCBOPorCodigo busca uma ocupação CBO por código (ex: "4110-10" ou "411010").
func BuscarCBOPorCodigo(codigo string) (data.CBO, bool) {
	tabelasOnce.Do(inicializarTabelas)
	if mapaCBOs == nil {
		return data.CBO{}, false
	}
	c, ok := mapaCBOs[strings.TrimSpace(codigo)]
	return c, ok
}

// FiltrarCBOs filtra ocupações CBO por título ou código.
func FiltrarCBOs(termo string) []data.CBO {
	tabelasOnce.Do(inicializarTabelas)
	termoNorm := strings.ToLower(strings.TrimSpace(termo))
	var res []data.CBO
	for _, c := range listaCBOs {
		if strings.Contains(strings.ToLower(c.Titulo), termoNorm) || strings.Contains(c.Codigo, termoNorm) {
			res = append(res, c)
		}
	}
	return res
}
