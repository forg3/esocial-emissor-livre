package importer

import (
	"strings"
	"testing"
)

// Amostras de retorno do eSocial (S-5001 por trabalhador e S-5011 consolidado).
const amostraS5001 = `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtBasesTrab/v_S_01_03_00">
  <evtBasesTrab Id="ID1000000000000002026020112000000001">
    <ideEvento><perApur>2026-01</perApur></ideEvento>
    <ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678</nrInsc></ideEmpregador>
    <ideTrabalhador><cpfTrab>111.222.333-44</cpfTrab><matricula>MAT-001</matricula></ideTrabalhador>
    <infoBases>
      <vrBcCpMensal>1500.00</vrBcCpMensal>
      <vrBcCp13>250.00</vrBcCp13>
      <vrCpDescPR>165.00</vrCpDescPR>
    </infoBases>
  </evtBasesTrab>
</eSocial>`

const amostraS5011 = `<?xml version="1.0" encoding="UTF-8"?>
<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtCS/v_S_01_03_00">
  <evtCS Id="ID1000000000000002026020112000000002">
    <ideEvento><perApur>2026-01</perApur></ideEvento>
    <ideEmpregador><tpInsc>1</tpInsc><nrInsc>12345678</nrInsc></ideEmpregador>
    <infoCS>
      <nrRecArqBase>1.2.202602.0000000000000001</nrRecArqBase>
      <infoCRContrib><tpCR>113901</tpCR><vrCR>1.234,56</vrCR></infoCRContrib>
      <infoCRContrib><tpCR>113902</tpCR><vrCR>100,00</vrCR></infoCRContrib>
    </infoCS>
  </evtCS>
</eSocial>`

func TestImportarRetornoS5001(t *testing.T) {
	registros, err := ImportarRetorno(strings.NewReader(amostraS5001))
	if err != nil {
		t.Fatalf("falha ao importar S-5001: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("esperado 1 registro, obtido %d", len(registros))
	}
	r := registros[0]
	if r.Tipo != "evtBasesTrab" {
		t.Errorf("tipo esperado evtBasesTrab, obtido %q", r.Tipo)
	}
	if r.PerApur != "2026-01" {
		t.Errorf("período esperado 2026-01, obtido %q", r.PerApur)
	}
	if r.CPFTrab != "11122233344" {
		t.Errorf("CPF deveria ser normalizado para dígitos, obtido %q", r.CPFTrab)
	}
	if r.Matricula != "MAT-001" {
		t.Errorf("matrícula esperada MAT-001, obtida %q", r.Matricula)
	}
	// 1500,00 + 250,00 + 165,00 = 1915,00
	if r.TotalCentavos != 191500 {
		t.Errorf("total esperado 191500 centavos, obtido %d", r.TotalCentavos)
	}
	if len(r.Valores) != 3 {
		t.Errorf("esperados 3 campos monetários, obtidos %d (%v)", len(r.Valores), r.Valores)
	}
}

func TestImportarRetornoS5011ValoresBrasileiros(t *testing.T) {
	registros, err := ImportarRetorno(strings.NewReader(amostraS5011))
	if err != nil {
		t.Fatalf("falha ao importar S-5011: %v", err)
	}
	if len(registros) != 1 {
		t.Fatalf("esperado 1 registro, obtido %d", len(registros))
	}
	r := registros[0]
	if r.Tipo != "evtCS" {
		t.Errorf("tipo esperado evtCS, obtido %q", r.Tipo)
	}
	if r.NRRecibo != "1.2.202602.0000000000000001" {
		t.Errorf("número do recibo base não capturado: %q", r.NRRecibo)
	}
	// 1234,56 + 100,00 = 1334,56 -> 133456 centavos
	if r.TotalCentavos != 133456 {
		t.Errorf("total esperado 133456 centavos, obtido %d (%v)", r.TotalCentavos, r.Valores)
	}
}

func TestImportarRetornoRejeitaXMLSemTotalizador(t *testing.T) {
	_, err := ImportarRetorno(strings.NewReader(`<eSocial><evtInfoEmpregador/></eSocial>`))
	if err == nil {
		t.Fatalf("XML sem evento de totalização deveria ser rejeitado")
	}
}

func TestImportarRetornoXMLInvalido(t *testing.T) {
	_, err := ImportarRetorno(strings.NewReader(`<eSocial><evtCS>`))
	if err == nil {
		t.Fatalf("XML malformado deveria retornar erro")
	}
}

func TestValorParaCentavos(t *testing.T) {
	casos := map[string]int64{
		"1500.00":  150000,
		"1.500,00": 150000,
		"1234,56":  123456,
		"0,01":     1,
		"100":      10000,
	}
	for entrada, esperado := range casos {
		if obtido, ok := valorParaCentavos(entrada); !ok || obtido != esperado {
			t.Errorf("valorParaCentavos(%q) = %d (%v), esperado %d", entrada, obtido, ok, esperado)
		}
	}
	if _, ok := valorParaCentavos("abc"); ok {
		t.Errorf("valor não numérico deveria ser ignorado")
	}
}
