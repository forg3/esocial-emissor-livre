package esocial_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/esocial"
)

func TestCatalogoCompleto_36Eventos(t *testing.T) {
	cat := esocial.ObterCatalogo()
	if len(cat) != 36 {
		t.Fatalf("esperava exatamente 36 eventos no catálogo oficial do eSocial S-1.3, encontrou %d", len(cat))
	}

	gruposValidos := map[string]bool{
		esocial.GrupoSESMT:              true,
		esocial.GrupoClinicaOcupacional: true,
		esocial.GrupoRHDP:               true,
		esocial.GrupoContabilidadeFolha: true,
		esocial.GrupoJuridico:           true,
		esocial.GrupoRPPSSetorPublico:   true,
	}

	categoriasValidas := map[string]bool{
		esocial.CategoriaTabelas:       true,
		esocial.CategoriaNaoPeriodicos: true,
		esocial.CategoriaPeriodicos:    true,
	}

	codigosVistos := make(map[string]bool)

	for _, evt := range cat {
		// Código único e bem formatado
		if codigosVistos[evt.Codigo] {
			t.Errorf("evento duplicado no catálogo: %s", evt.Codigo)
		}
		codigosVistos[evt.Codigo] = true

		if !strings.HasPrefix(evt.Codigo, "S-") {
			t.Errorf("código de evento inválido: %s", evt.Codigo)
		}

		if evt.Nome == "" {
			t.Errorf("evento %s está sem Nome", evt.Codigo)
		}

		if !categoriasValidas[evt.Categoria] {
			t.Errorf("evento %s possui Categoria inválida: %s", evt.Codigo, evt.Categoria)
		}

		if !gruposValidos[evt.Grupo] {
			t.Errorf("evento %s possui Grupo de responsabilidade inválido: %s", evt.Codigo, evt.Grupo)
		}

		if evt.Responsavel == "" {
			t.Errorf("evento %s está sem Responsável definido", evt.Codigo)
		}

		if evt.Descricao == "" {
			t.Errorf("evento %s está sem Descrição", evt.Codigo)
		}

		if evt.Prazo == "" {
			t.Errorf("evento %s está sem Prazo oficial do MOS", evt.Codigo)
		}

		if !strings.HasPrefix(evt.TagRaiz, "evt") {
			t.Errorf("evento %s possui TagRaiz inválida: %s (deve iniciar com 'evt')", evt.Codigo, evt.TagRaiz)
		}

		prefixoEsperado := "http://www.esocial.gov.br/schema/evt/" + evt.TagRaiz + "/v_S_01_03_00"
		if evt.Namespace != prefixoEsperado {
			t.Errorf("evento %s com namespace divergente:\nEsperado: %s\nObtido:   %s", evt.Codigo, prefixoEsperado, evt.Namespace)
		}

		if !strings.Contains(evt.TemplateXML, "<eSocial") || !strings.Contains(evt.TemplateXML, "</eSocial>") {
			t.Errorf("evento %s com TemplateXML malformado (faltam tags <eSocial>)", evt.Codigo)
		}
		if !strings.Contains(evt.TemplateXML, "{{ID}}") {
			t.Errorf("evento %s com TemplateXML sem o marcador {{ID}}", evt.Codigo)
		}
	}
}

func TestListarGrupos(t *testing.T) {
	grupos := esocial.ListarGrupos()
	if len(grupos) != 6 {
		t.Fatalf("esperava 6 grupos de responsabilidade técnica, encontrou %d", len(grupos))
	}

	esperados := []string{
		esocial.GrupoSESMT,
		esocial.GrupoClinicaOcupacional,
		esocial.GrupoRHDP,
		esocial.GrupoContabilidadeFolha,
		esocial.GrupoJuridico,
		esocial.GrupoRPPSSetorPublico,
	}

	for i, g := range esperados {
		if grupos[i] != g {
			t.Errorf("grupo[%d] = %s; esperado %s", i, grupos[i], g)
		}
	}
}

func TestListarPorGrupo(t *testing.T) {
	contagensEsperadas := map[string]int{
		esocial.GrupoSESMT:              1,  // S-2240
		esocial.GrupoClinicaOcupacional: 2,  // S-2210, S-2220
		esocial.GrupoRHDP:               12, // S-2190, S-2200, S-2205, S-2206, S-2230, S-2231, S-2298, S-2299, S-2300, S-2306, S-2399, S-3000
		esocial.GrupoContabilidadeFolha: 11, // S-1000, S-1005, S-1010, S-1020, S-1200, S-1210, S-1260, S-1270, S-1280, S-1298, S-1299
		esocial.GrupoJuridico:           4,  // S-1070, S-2500, S-2501, S-2555
		esocial.GrupoRPPSSetorPublico:   6,  // S-1202, S-1207, S-2400, S-2405, S-2410, S-2416
	}

	total := 0
	for grupo, esperada := range contagensEsperadas {
		evts := esocial.ListarPorGrupo(grupo)
		if len(evts) != esperada {
			t.Errorf("grupo '%s': esperava %d eventos, obteve %d", grupo, esperada, len(evts))
		}
		total += len(evts)
	}

	if total != 36 {
		t.Errorf("soma total dos eventos filtrados por grupo = %d; esperava 36", total)
	}
}

func TestObterEventoPorCodigo(t *testing.T) {
	testes := []struct {
		busca       string
		esperadoCod string
		deveExistir bool
	}{
		{"S-1000", "S-1000", true},
		{"s-1000", "S-1000", true},
		{"S1000", "S-1000", true},
		{"1000", "S-1000", true},
		{"S-2240", "S-2240", true},
		{"2240", "S-2240", true},
		{"S-2200", "S-2200", true},
		{"2200", "S-2200", true},
		{"S-2500", "S-2500", true},
		{"2500", "S-2500", true},
		{"S-9999", "", false},
		{"inexistente", "", false},
	}

	for _, tt := range testes {
		evt := esocial.ObterEventoPorCodigo(tt.busca)
		if tt.deveExistir {
			if evt == nil {
				t.Errorf("esperava encontrar evento para busca '%s', retornou nil", tt.busca)
			} else if evt.Codigo != tt.esperadoCod {
				t.Errorf("busca '%s' retornou código %s; esperado %s", tt.busca, evt.Codigo, tt.esperadoCod)
			}
		} else {
			if evt != nil {
				t.Errorf("esperava nil para busca inválida '%s', obteve %+v", tt.busca, evt)
			}
		}
	}
}

func TestGerarXMLEventoGenerico_TodosOs36Eventos(t *testing.T) {
	catalogo := esocial.ObterCatalogo()

	dadosCustom := map[string]string{
		"tpInsc":    "1",
		"nrInsc":    "12345678000195",
		"tpAmb":     "2",
		"cpfTrab":   "12345678909",
		"matricula": "MAT-2026-X",
		"nmTrab":    "Carlos Silva de Souza",
	}

	for _, evt := range catalogo {
		xmlBytes, err := esocial.GerarXMLEventoGenerico(evt.Codigo, dadosCustom)
		if err != nil {
			t.Fatalf("falha ao gerar XML genérico para %s: %v", evt.Codigo, err)
		}

		xmlStr := string(xmlBytes)

		// 1. Cabeçalho XML e envelope raiz
		if !strings.HasPrefix(xmlStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
			t.Errorf("evento %s não inicia com declaração XML UTF-8", evt.Codigo)
		}
		if !strings.Contains(xmlStr, `<eSocial xmlns="`+evt.Namespace+`">`) {
			t.Errorf("evento %s sem namespace correto no elemento raiz", evt.Codigo)
		}

		// 2. Tag Raiz do evento
		if !strings.Contains(xmlStr, "<"+evt.TagRaiz) {
			t.Errorf("evento %s não contém a tag raiz esperada <%s", evt.Codigo, evt.TagRaiz)
		}

		// 3. Identificador de 36 caracteres
		if !strings.Contains(xmlStr, `Id="ID1`) {
			t.Errorf("evento %s sem ID iniciando por 'ID1'", evt.Codigo)
		}

		// 4. Garante que não sobrou nenhum placeholder no formato {{...}}
		if strings.Contains(xmlStr, "{{") {
			t.Errorf("evento %s contém placeholders não substituídos no XML final:\n%s", evt.Codigo, xmlStr)
		}

		// 5. Validação de sintaxe com decoder XML
		var envelope struct {
			XMLName xml.Name
		}
		if err := xml.Unmarshal(xmlBytes, &envelope); err != nil {
			t.Errorf("XML gerado para %s não pôde ser decodificado: %v\nXML:\n%s", evt.Codigo, err, xmlStr)
		}
	}
}
