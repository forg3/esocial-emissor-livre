package esocial_test

import (
	"strings"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/esocial"
)

func TestGerarIDEvento(t *testing.T) {
	// 1. Testa geração com timestamp explícito determinístico
	dataFixa := time.Date(2026, 9, 22, 12, 30, 45, 0, time.UTC)
	id := esocial.GerarIDEventoComTempo(esocial.TpInscCNPJ, "12.345.678/0001-95", dataFixa, 1)

	if len(id) != 36 {
		t.Fatalf("esperava ID com 36 caracteres, obteve %d: %s", len(id), id)
	}

	esperado := "ID1123456780001952026092212304500001"
	if id != esperado {
		t.Errorf("ID gerado incompatível.\nEsperado: %s\nObtido:   %s", esperado, id)
	}

	// Valida pelo validador oficial de formato
	if err := esocial.ValidarIDEvento(id); err != nil {
		t.Errorf("ValidarIDEvento falhou para ID válido: %v", err)
	}

	// 2. Testa CPF com padding de zeros à esquerda
	idCPF := esocial.GerarIDEventoComTempo(esocial.TpInscCPF, "123.456.789-09", dataFixa, 42)
	esperadoCPF := "ID2000123456789092026092212304500042"
	if idCPF != esperadoCPF {
		t.Errorf("ID gerado para CPF incompatível.\nEsperado: %s\nObtido:   %s", esperadoCPF, idCPF)
	}

	// 3. Testa função padrão com time.Now()
	idNow := esocial.GerarIDEvento(esocial.TpInscCNPJ, "12345678000195", 1)
	if err := esocial.ValidarIDEvento(idNow); err != nil {
		t.Errorf("ValidarIDEvento falhou para idNow: %v", err)
	}

	// 4. Testa rejeição de IDs inválidos
	if err := esocial.ValidarIDEvento("ID123"); err == nil {
		t.Error("esperava erro para ID curto")
	}
	if err := esocial.ValidarIDEvento("XX1123456780001952026092212304500001"); err == nil {
		t.Error("esperava erro para ID sem prefixo 'ID'")
	}
}

func TestS1000XML_ValidacaoXSD(t *testing.T) {
	dataFixa := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	id := esocial.GerarIDEventoComTempo(esocial.TpInscCNPJ, "12345678000195", dataFixa, 1)

	evt := esocial.NovoEventoS1000Inclusao(
		id,
		esocial.AmbienteProducaoRestrita,
		esocial.TpInscCNPJ,
		"12345678000195",
		"2026-01",
		"01", // Classificação tributária 01 (Empresa enquadrada no Simples)
		0,    // Não desonerada
		1,    // Optou pelo registro eletrônico
	)

	xmlBytes, err := esocial.GerarXMLS1000(evt)
	if err != nil {
		t.Fatalf("falha ao gerar XML S-1000: %v", err)
	}

	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, `<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtInfoEmpregador/v_S_01_03_00">`) {
		t.Errorf("namespace S-1000 não encontrado no XML gerado")
	}
	if !strings.Contains(xmlStr, `Id="`+id+`"`) {
		t.Errorf("atributo Id não encontrado no XML gerado")
	}
	if !strings.Contains(xmlStr, `<classTrib>01</classTrib>`) {
		t.Errorf("classTrib incorreto no XML gerado")
	}

	// Validação formal contra o esquema oficial XSD
	if err := esocial.ValidarXSD(xmlBytes, "S-1000"); err != nil {
		t.Fatalf("validação XSD falhou para S-1000: %v", err)
	}
}

func TestS2240XML_ValidacaoXSD(t *testing.T) {
	dataFixa := time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC)
	id := esocial.GerarIDEventoComTempo(esocial.TpInscCNPJ, "12345678000195", dataFixa, 10)

	tpAvalQuant := 1
	unMedDBA := 4
	ideOCCrea := 4

	evt := &esocial.EventoS2240{
		EvtExpRisco: esocial.EvtExpRisco{
			Id: id,
			IdeEvento: esocial.IdeEventoTrab{
				IndRetif: 1,
				TpAmb:    esocial.AmbienteProducaoRestrita,
				ProcEmi:  esocial.ProcEmiAppEmpregador,
				VerProc:  esocial.VersaoAplicativo,
			},
			IdeEmpregador: esocial.IdeEmpregador{
				TpInsc: esocial.TpInscCNPJ,
				NrInsc: "12345678000195",
			},
			IdeVinculo: esocial.IdeVinculoSST{
				CpfTrab:   "12345678909",
				Matricula: "MAT-2026-001",
			},
			InfoExpRisco: esocial.InfoExpRisco{
				DtIniCondicao: "2026-01-01",
				InfoAmb: []esocial.InfoAmb{
					{
						LocalAmb: 1,
						DscSetor: "Usinagem Mecânica",
						TpInsc:   esocial.TpInscCNPJ,
						NrInsc:   "12345678000195",
					},
				},
				InfoAtiv: esocial.InfoAtiv{
					DscAtivDes: "Operar torno mecânico para fabricação de engrenagens industriais",
				},
				AgNoc: []esocial.AgNoc{
					{
						CodAgNoc:   "01.18.001", // Ruído
						DscAgNoc:   "Ruído contínuo ou intermitente",
						TpAval:     &tpAvalQuant,
						IntConc:    "86.5000",
						LimTol:     "85.0000",
						UnMed:      &unMedDBA,
						TecMedicao: "Dosimetria NHO-01 Fundacentro",
						EpcEpi: &esocial.EpcEpi{
							UtilizEPC: 1,
							UtilizEPI: 2,
							EficEpi:   "S",
							Epi: []esocial.EpiItem{
								{DocAval: "12345"},
							},
							EpiCompl: &esocial.EpiCompl{
								MedProtecao:   "S",
								CondFuncto:    "S",
								UsoInint:      "S",
								PrzValid:      "S",
								PeriodicTroca: "S",
								Higienizacao:  "S",
							},
						},
					},
				},
				RespReg: []esocial.RespReg{
					{
						CpfResp: "12345678909",
						IdeOC:   &ideOCCrea,
						NrOC:    "SP123456",
						UfOC:    "SP",
					},
				},
				Obs: &esocial.ObsS2240{
					ObsCompl: "Laudo Técnico das Condições Ambientais do Trabalho (LTCAT) atualizado",
				},
			},
		},
	}

	xmlBytes, err := esocial.GerarXMLS2240(evt)
	if err != nil {
		t.Fatalf("falha ao gerar XML S-2240: %v", err)
	}

	// Validação formal contra o esquema oficial XSD evtExpRisco.xsd
	if err := esocial.ValidarXSD(xmlBytes, "S-2240"); err != nil {
		t.Fatalf("validação XSD falhou para S-2240: %v\nXML gerado:\n%s", err, string(xmlBytes))
	}
}

func TestS2210XML_ValidacaoXSD(t *testing.T) {
	dataFixa := time.Date(2026, 9, 22, 11, 30, 0, 0, time.UTC)
	id := esocial.GerarIDEventoComTempo(esocial.TpInscCNPJ, "12345678000195", dataFixa, 20)

	evt := &esocial.EventoS2210{
		EvtCAT: esocial.EvtCAT{
			Id: id,
			IdeEvento: esocial.IdeEventoTrab{
				IndRetif: 1,
				TpAmb:    esocial.AmbienteProducaoRestrita,
				ProcEmi:  esocial.ProcEmiAppEmpregador,
				VerProc:  esocial.VersaoAplicativo,
			},
			IdeEmpregador: esocial.IdeEmpregador{
				TpInsc: esocial.TpInscCNPJ,
				NrInsc: "12345678000195",
			},
			IdeVinculo: esocial.IdeVinculoSST{
				CpfTrab:   "12345678909",
				Matricula: "MAT-2026-001",
			},
			CAT: esocial.DadosCAT{
				DtAcid:           "2026-09-01",
				TpAcid:           1, // Típico
				HrAcid:           "0930",
				HrsTrabAntesAcid: "0130",
				TpCat:            1, // Inicial
				IndCatObito:      "N",
				IndComunPolicia:  "N",
				CodSitGeradora:   "303000000",
				IniciatCAT:       1, // Empregador
				ObsCAT:           "Queda de mesmo nível no corredor de acesso",
				HouveAfast:       "S",
				LocalAcidente: esocial.LocalAcidente{
					TpLocal:   1,
					DscLocal:  "Pátio interno de estocagem",
					DscLograd: "Avenida das Indústrias",
					NrLograd:  "500",
					Bairro:    "Distrito Industrial",
					Cep:       "12345678",
					CodMunic:  "3550308", // São Paulo
					Uf:        "SP",
				},
				ParteAtingida: esocial.ParteAtingida{
					CodParteAting: "752000000", // Tornozelo
					Lateralidade:  2,           // Direita
				},
				AgenteCausador: esocial.AgenteCausador{
					CodAgntCausador: "302000000", // Piso
				},
				Atestado: esocial.AtestadoMedico{
					DtAtendimento: "2026-09-01",
					HrAtendimento: "1015",
					IndInternacao: "N",
					DurTrat:       7,
					IndAfast:      "S",
					DscLesao:      "702000000", // Entorse
					CodCID:        "S934",      // Entorse de tornozelo
					Emitente: esocial.EmitenteMedico{
						NmEmit: "Dr. Carlos Eduardo Medeiros",
						IdeOC:  1, // CRM
						NrOC:   "123456",
						UfOC:   "SP",
					},
				},
			},
		},
	}

	xmlBytes, err := esocial.GerarXMLS2210(evt)
	if err != nil {
		t.Fatalf("falha ao gerar XML S-2210: %v", err)
	}

	// Validação formal contra o esquema oficial XSD evtCAT.xsd
	if err := esocial.ValidarXSD(xmlBytes, "S-2210"); err != nil {
		t.Fatalf("validação XSD falhou para S-2210: %v\nXML gerado:\n%s", err, string(xmlBytes))
	}
}

func TestS2220XML_ValidacaoXSD_e_Parse(t *testing.T) {
	dataFixa := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	id := esocial.GerarIDEventoComTempo(esocial.TpInscCNPJ, "12345678000195", dataFixa, 30)

	resApto := esocial.ResultadoApto
	ordInicial := 1
	indNormal := 1

	evt := &esocial.EventoS2220{
		EvtMonit: esocial.EvtMonit{
			Id: id,
			IdeEvento: esocial.IdeEventoTrab{
				IndRetif: 1,
				TpAmb:    esocial.AmbienteProducaoRestrita,
				ProcEmi:  esocial.ProcEmiAppEmpregador,
				VerProc:  esocial.VersaoAplicativo,
			},
			IdeEmpregador: esocial.IdeEmpregador{
				TpInsc: esocial.TpInscCNPJ,
				NrInsc: "12345678000195",
			},
			IdeVinculo: esocial.IdeVinculoSST{
				CpfTrab:   "12345678909",
				Matricula: "MAT-2026-001",
			},
			ExMedOcup: esocial.ExMedOcup{
				TpExameOcup: esocial.ExamePeriodico,
				ASO: esocial.DadosASO{
					DtAso:  "2026-09-01",
					ResAso: &resApto,
					Exame: []esocial.ExameItem{
						{
							DtExm:         "2026-09-01",
							ProcRealizado: "0281", // Audiometria tonal
							OrdExame:      &ordInicial,
							IndResult:     &indNormal,
						},
					},
					Medico: esocial.MedicoASO{
						NmMed: "Dra. Luciana Mendes Silva",
						NrCRM: "987654",
						UfCRM: "SP",
					},
				},
				RespMonit: &esocial.RespMonit{
					NmResp: "Dr. Antonio Marcos Ramos",
					NrCRM:  "456123",
					UfCRM:  "SP",
				},
			},
		},
	}

	xmlBytes, err := esocial.GerarXMLS2220(evt)
	if err != nil {
		t.Fatalf("falha ao gerar XML S-2220: %v", err)
	}

	// Validação formal contra o esquema oficial XSD evtMonit.xsd
	if err := esocial.ValidarXSD(xmlBytes, "S-2220"); err != nil {
		t.Fatalf("validação XSD falhou para S-2220: %v\nXML gerado:\n%s", err, string(xmlBytes))
	}

	// Teste de parser/importação de XML recebido de clínica externa
	parsed, err := esocial.ParseS2220(xmlBytes)
	if err != nil {
		t.Fatalf("falha ao executar ParseS2220: %v", err)
	}

	if parsed.EvtMonit.IdeVinculo.CpfTrab != "12345678909" {
		t.Errorf("CPF divergente no ParseS2220: esperado 12345678909, obtido %s", parsed.EvtMonit.IdeVinculo.CpfTrab)
	}
	if parsed.EvtMonit.ExMedOcup.ASO.DtAso != "2026-09-01" {
		t.Errorf("DtAso divergente no ParseS2220: esperado 2026-09-01, obtido %s", parsed.EvtMonit.ExMedOcup.ASO.DtAso)
	}
	if parsed.EvtMonit.ExMedOcup.ASO.Medico.NmMed != "Dra. Luciana Mendes Silva" {
		t.Errorf("Médico divergente no ParseS2220: esperado Dra. Luciana Mendes Silva, obtido %s", parsed.EvtMonit.ExMedOcup.ASO.Medico.NmMed)
	}
	if len(parsed.EvtMonit.ExMedOcup.ASO.Exame) != 1 || parsed.EvtMonit.ExMedOcup.ASO.Exame[0].ProcRealizado != "0281" {
		t.Errorf("Exame complementar 0281 não recuperado corretamente")
	}
}

func TestTabelasOficiais(t *testing.T) {
	// 1. Riscos da Tabela 24
	riscos, err := esocial.ObterRiscos()
	if err != nil {
		t.Fatalf("falha ao obter riscos: %v", err)
	}
	if len(riscos) == 0 {
		t.Fatal("lista de riscos está vazia")
	}

	r09, ok := esocial.BuscarRiscoPorCodigo("09.01.001")
	if !ok {
		t.Fatal("código 09.01.001 (ausência de agente nocivo) não encontrado na Tabela 24")
	}
	if !strings.Contains(strings.ToLower(r09.Nome), "ausência") {
		t.Errorf("nome do risco 09.01.001 inesperado: %s", r09.Nome)
	}

	filtrados := esocial.FiltrarRiscos("Ruído")
	if len(filtrados) == 0 {
		t.Error("filtro por 'Ruído' não retornou nenhum item")
	}

	// 2. CBOs
	cbos, err := esocial.ObterCBOs()
	if err != nil {
		t.Fatalf("falha ao obter CBOs: %v", err)
	}
	if len(cbos) == 0 {
		t.Fatal("lista de CBOs está vazia")
	}

	cbo, ok := esocial.BuscarCBOPorCodigo("4110-10")
	if !ok {
		// Tenta sem hífen
		cbo, ok = esocial.BuscarCBOPorCodigo("411010")
	}
	if !ok {
		t.Fatal("CBO 4110-10 (Assistente administrativo) não encontrado")
	}
	if cbo.Codigo == "" {
		t.Error("CBO retornado sem código")
	}

	cbosFilt := esocial.FiltrarCBOs("Engenheiro")
	if len(cbosFilt) == 0 {
		t.Error("filtro por 'Engenheiro' não retornou nenhum item")
	}
}
