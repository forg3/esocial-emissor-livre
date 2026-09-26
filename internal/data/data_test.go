package data

import "testing"

// As tabelas eram amostras (40/50/30/26/20 itens): CAT ou ASO fora da amostra não
// podia ser preenchido. Contagens do leiaute S-1.3 (NT 07/2026).
func TestTabelasDaCATEDoASOSaoAsCompletas(t *testing.T) {
	casos := []struct {
		nome  string
		fn    func() ([]ItemTabela, error)
		total int
	}{
		{"Tabela 13", CarregarPartesCorpo, 45},
		{"Tabela 14", CarregarAgentesCausadores, 248},
		{"Tabela 15", CarregarSituacoesGeradoras, 60},
		{"Tabela 17", CarregarNaturezasLesao, 29},
		{"Tabela 27", CarregarProcedimentosDiagnosticos, 1450},
	}
	for _, c := range casos {
		itens, err := c.fn()
		if err != nil {
			t.Fatalf("%s: %v", c.nome, err)
		}
		if len(itens) != c.total {
			t.Errorf("%s: esperados %d itens, obtidos %d", c.nome, c.total, len(itens))
		}
	}
}
