package web_test

import (
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/data"
	"github.com/forg3/esocial-emissor-livre/internal/esocial"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

// TestConformidadeXSDGeraEventosValidos valida os documentos produzidos pelos três
// geradores contra os XSD oficiais do leiaute S-1.3 embutidos no binário
// (internal/data/xsd/*.xsd), usando o mesmo validador da aplicação (xmllint).
//
// Este é o teste de aceite do achado M-05: os leiautes gerados pelo editor devem
// ser aderentes ao schema oficial.
func TestConformidadeXSDGeraEventosValidos(t *testing.T) {
	if !esocial.XmllintDisponivel() {
		t.Skip("xmllint ausente: validação por schema não executável neste ambiente")
	}

	casos := []struct {
		nome string
		tipo string
		xml  string
	}{
		{
			nome: "S-2240 condições ambientais de trabalho",
			tipo: "S-2240",
			xml: web.GerarXMLS2240(web.ParametrosS2240{
				ID:                 "ID1000000000000002026010112000000001",
				Ambiente:           2,
				CNPJ:               "12345678000190",
				CPFTrabalhador:     "11122233344",
				Matricula:          "MAT-001",
				DataInicio:         "2026-01-01",
				DescAtividade:      "Operação de prensa hidráulica no setor de estamparia",
				LocalAmbiente:      "1",
				DscSetor:           "Estamparia",
				CodigoRisco:        "01.01.001",
				NomeRisco:          "Ruído contínuo ou intermitente",
				TipoAvaliacao:      "1",
				Intensidade:        "87.5",
				UnidadeMedida:      "4",
				TecnicaMedicao:     "Medição conforme NHO-01 da Fundacentro",
				UtilizaEPC:         "2",
				EfficazEPC:         "S",
				UtilizaEPI:         "2",
				EficazEPI:          "S",
				CAEPI:              "12345",
				MedicaoProtecao:    "S",
				CondicaoFunc:       "S",
				UsoIninterrupto:    "S",
				PrazoValidade:      "S",
				PeriodicidadeTroca: "S",
				Higienizacao:       "S",
				CPFResp:            "11122233344",
				OrgaoClasse:        "4",
				NumRegistro:        "123456",
				UFRegistro:         "SP",
			}),
		},
		{
			nome: "S-2210 comunicação de acidente de trabalho",
			tipo: "S-2210",
			xml: web.GerarXMLS2210(web.ParametrosS2210{
				ID:             "ID1000000000000002026010112000000002",
				Ambiente:       2,
				CNPJ:           "12345678000190",
				CPFTrabalhador: "11122233344",
				Matricula:      "MAT-001",
				DtAcidente:     "2026-01-05",
				HrAcidente:     "1430",
				TpAcidente:     "1",
				HrsTrabAntes:   "0630",
				TpCat:          "1",
				HouveAfast:     "S",
				HouveObito:     "N",
				ComunPolicia:   "N",
				CodSitGeradora: "303010040",
				IniciatCAT:     "1",
				ObsCAT:         "Acidente típico durante manuseio de ferramenta manual.",
				UltDiaTrab:     "2026-01-05",
				TpLocal:        "1",
				DscLocal:       "Oficina de manutenção mecânica, bancada 03.",
				DscLograd:      "Avenida Industrial",
				NrLograd:       "1500",
				Bairro:         "Distrito Industrial",
				CEP:            "01310100",
				CodMunic:       "3550308",
				UF:             "SP",
				ParteCorpo:     "755070000",
				Lateralidade:   "1",
				AgenteCausador: "303010040",
				DtAtendimento:  "2026-01-05",
				HrAtendimento:  "1500",
				IndInternacao:  "N",
				DurTrat:        "7",
				IndAfast:       "S",
				DescLesao:      "702010000",
				DscCompLesao:   "Ferimento incisivo no dedo indicador direito",
				DiagProvavel:   "Ferida cortocontusa",
				CID10:          "S61.0",
				Observacao:     "Atendimento realizado no ambulatório da empresa.",
				NomeMedico:     "Dr. Lucas Moreira",
				CRMMedico:      "154821",
				UFMedico:       "SP",
				OrgaoClasseMed: "1",
			}),
		},
		{
			nome: "S-2220 monitoramento da saúde (ASO)",
			tipo: "S-2220",
			xml: web.GerarXMLS2220(web.ParametrosS2220{
				ID:             "ID1000000000000002026010112000000003",
				Ambiente:       2,
				CNPJ:           "12345678000190",
				CPFTrabalhador: "11122233344",
				Matricula:      "MAT-001",
				TipoExame:      "1",
				DataASO:        "2026-01-10",
				ResultadoASO:   "1",
				ProcRealizado:  "0295",
				OrdExame:       "1",
				IndResult:      "1",
				NomeMedico:     "Dr. Lucas Moreira",
				CRMMedico:      "154821",
				UFMedico:       "SP",
				NomeCoord:      "Dra. Marina Alves",
				CPFCoord:       "11122233344",
				CRMCoord:       "654321",
				UFCoord:        "SP",
			}),
		},
		{
			nome: "S-2220 sem médico coordenador (campo opcional)",
			tipo: "S-2220",
			xml: web.GerarXMLS2220(web.ParametrosS2220{
				ID:             "ID1000000000000002026010112000000004",
				Ambiente:       2,
				CNPJ:           "12345678000190",
				CPFTrabalhador: "11122233344",
				TipoExame:      "9", // valor legado do formulário: deve virar 4 (demissional)
				DataASO:        "2026-02-01",
				ResultadoASO:   "2",
				ProcRealizado:  "0281",
				ObsProc:        "Audiometria ocupacional conforme NR-07.",
				OrdExame:       "1",
				NomeMedico:     "Dr. Lucas Moreira",
				CRMMedico:      "154821",
				UFMedico:       "SP",
			}),
		},
		{
			nome: "S-2240 avaliação qualitativa (sem campos quantitativos)",
			tipo: "S-2240",
			xml: web.GerarXMLS2240(web.ParametrosS2240{
				ID:             "ID1000000000000002026010112000000005",
				Ambiente:       2,
				CNPJ:           "12345678000190",
				CPFTrabalhador: "11122233344",
				DataInicio:     "2026-01-01",
				CodigoRisco:    "09.01.001",
				TipoAvaliacao:  "2",
				UtilizaEPC:     "0",
				UtilizaEPI:     "1",
				CPFResp:        "11122233344",
				OrgaoClasse:    "1",
				UFRegistro:     "SP",
			}),
		},
	}

	for _, caso := range casos {
		if err := esocial.ValidarXSD([]byte(caso.xml), caso.tipo); err != nil {
			t.Errorf("%s: XML gerado não é aderente ao XSD oficial: %v\n%s", caso.nome, err, caso.xml)
		}
	}
}

// TestGeradoresUsamEstruturaOficial garante que a estrutura mínima esperada pelo
// leiaute S-1.3 está presente (regressão do achado M-05).
func TestGeradoresUsamEstruturaOficial(t *testing.T) {
	xml2240 := web.GerarXMLS2240(web.ParametrosS2240{
		ID: "ID1000000000000002026010112000000001", Ambiente: 2, CNPJ: "12345678000190",
		CPFTrabalhador: "11122233344", Matricula: "MAT-1", CodigoRisco: "09.01.001", CPFResp: "11122233344",
	})
	for _, obrigatorio := range []string{"<ideVinculo>", "<cpfTrab>", "<matricula>", "<infoExpRisco>", "<agNoc>", "<codAgNoc>", "<respReg>"} {
		if !strings.Contains(xml2240, obrigatorio) {
			t.Errorf("S-2240 deveria conter %s", obrigatorio)
		}
	}
	for _, proibido := range []string{"<ideTrabalhador>", "<agenteNoc>"} {
		if strings.Contains(xml2240, proibido) {
			t.Errorf("S-2240 não pode mais emitir %s (estrutura inválida no S-1.3)", proibido)
		}
	}

	xml2210 := web.GerarXMLS2210(web.ParametrosS2210{
		ID: "ID1000000000000002026010112000000002", Ambiente: 2, CNPJ: "12345678000190",
		CPFTrabalhador: "11122233344", DtAcidente: "2026-01-05", TpAcidente: "1",
	})
	for _, obrigatorio := range []string{"<ideVinculo>", "<cat>", "<codSitGeradora>", "<iniciatCAT>", "<localAcidente>", "<dscLograd>", "<nrLograd>", "<parteAtingida>", "<lateralidade>", "<agenteCausador>", "<atestado>", "<emitente>", "<nmEmit>", "<durTrat>", "<indInternacao>"} {
		if !strings.Contains(xml2210, obrigatorio) {
			t.Errorf("S-2210 deveria conter %s", obrigatorio)
		}
	}

	xml2220 := web.GerarXMLS2220(web.ParametrosS2220{
		ID: "ID1000000000000002026010112000000003", Ambiente: 2, CNPJ: "12345678000190",
		CPFTrabalhador: "11122233344", TipoExame: "1", DataASO: "2026-01-10",
	})
	for _, obrigatorio := range []string{"<ideVinculo>", "<exMedOcup>", "<tpExameOcup>", "<aso>", "<dtAso>", "<exame>", "<dtExm>", "<procRealizado>", "<medico>", "<nmMed>"} {
		if !strings.Contains(xml2220, obrigatorio) {
			t.Errorf("S-2220 deveria conter %s", obrigatorio)
		}
	}
	if strings.Contains(xml2220, "<tpExameOcup>9</tpExameOcup>") {
		t.Errorf("código 9 (legado) não existe no XSD; deve ser convertido para 4 (demissional)")
	}
}

// TestTabelasOficiaisEmbutidas confere que as tabelas do eSocial usadas pelos
// formulários estão disponíveis e com códigos no formato oficial.
func TestTabelasOficiaisEmbutidas(t *testing.T) {
	partes, err := data.CarregarPartesCorpo()
	if err != nil || len(partes) == 0 {
		t.Fatalf("Tabela 13 indisponível: %v", err)
	}
	agentes, err := data.CarregarAgentesCausadores()
	if err != nil || len(agentes) == 0 {
		t.Fatalf("Tabela 14 indisponível: %v", err)
	}
	situacoes, err := data.CarregarSituacoesGeradoras()
	if err != nil || len(situacoes) == 0 {
		t.Fatalf("Tabela 15 indisponível: %v", err)
	}
	lesoes, err := data.CarregarNaturezasLesao()
	if err != nil || len(lesoes) == 0 {
		t.Fatalf("Tabela 17 indisponível: %v", err)
	}
	procedimentos, err := data.CarregarProcedimentosDiagnosticos()
	if err != nil || len(procedimentos) == 0 {
		t.Fatalf("Tabela 27 indisponível: %v", err)
	}

	for nome, itens := range map[string][]data.ItemTabela{
		"Tabela 13": partes, "Tabela 14": agentes, "Tabela 15": situacoes,
		"Tabela 17": lesoes, "Tabela 27": procedimentos,
	} {
		tamanho := 9
		if nome == "Tabela 27" {
			tamanho = 4
		}
		for _, item := range itens {
			if len(item.Codigo) != tamanho {
				t.Errorf("%s: código %q deveria ter %d dígitos", nome, item.Codigo, tamanho)
			}
			if strings.TrimSpace(item.Descricao) == "" {
				t.Errorf("%s: código %q sem descrição", nome, item.Codigo)
			}
		}
	}
}
