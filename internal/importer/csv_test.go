package importer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/importer"
)

func TestValidarCPF(t *testing.T) {
	testes := []struct {
		cpf      string
		esperado bool
	}{
		{"12345678909", true},
		{"123.456.789-09", true},
		{"98765432100", true},
		{"987.654.321-00", true},
		{"45678912364", true},
		{"52998224725", true},
		{"11144477735", true},

		// Casos inválidos
		{"11111111111", false},
		{"00000000000", false},
		{"99999999999", false},
		{"12345678900", false}, // Dígito verificador errado
		{"123456789", false},   // Curto
		{"123456789012", false}, // Longo
		{"abc12345678", false},
		{"", false},
	}

	for _, tt := range testes {
		obtido := importer.ValidarCPF(tt.cpf)
		if obtido != tt.esperado {
			t.Errorf("ValidarCPF(%q) = %v; esperado %v", tt.cpf, obtido, tt.esperado)
		}
	}
}

func TestNormalizarData(t *testing.T) {
	testes := []struct {
		entrada  string
		esperado string
		temErro  bool
	}{
		{"2026-09-22", "2026-09-22", false},
		{"22/09/2026", "2026-09-22", false},
		{"22-09-2026", "2026-09-22", false},
		{"01/01/2023", "2023-01-01", false},
		{"1/1/2023", "2023-01-01", false},
		{"invalida", "", true},
		{"", "", true},
	}

	for _, tt := range testes {
		obtido, err := importer.NormalizarData(tt.entrada)
		if (err != nil) != tt.temErro {
			t.Errorf("NormalizarData(%q) erro = %v, esperado erro? %v", tt.entrada, err, tt.temErro)
		}
		if obtido != tt.esperado {
			t.Errorf("NormalizarData(%q) = %q, esperado %q", tt.entrada, obtido, tt.esperado)
		}
	}
}

func TestImportarColaboradoresCSV_Delimitadores(t *testing.T) {
	// Teste 1: Vírgula
	csvVirgula := `Nome,CPF,Matricula,Cargo,CBO,DataAdmissao,Setor
Maria Santos,12345678909,MAT-101,Analista de RH,2524-05,10/01/2023,Recursos Humanos
Joao Paulo,98765432100,MAT-102,Operador de Caldeira,8611-05,2022-04-15,Operacional`

	colabs, erros, err := importer.ImportarColaboradoresCSV(strings.NewReader(csvVirgula))
	if err != nil {
		t.Fatalf("erro inesperado ao importar CSV com vírgula: %v", err)
	}
	if len(erros) > 0 {
		t.Fatalf("erros de validação inesperados: %v", erros)
	}
	if len(colabs) != 2 {
		t.Fatalf("esperava 2 colaboradores, obteve %d", len(colabs))
	}
	if colabs[0].DataAdmissao != "2023-01-10" {
		t.Errorf("data de admissão de Maria Santos não convertida para ISO: %s", colabs[0].DataAdmissao)
	}

	// Teste 2: Ponto-e-vírgula (padrão Excel Brasil)
	csvPontoEVirgula := `Nome;CPF;Matrícula;Cargo;CBO;Data de Admissão;Setor
Fernanda Lima;45678912364;MAT-201;Técnica de Segurança;3516-05;01/08/2021;Segurança do Trabalho`

	colabsPonto, errosPonto, err := importer.ImportarColaboradoresCSV(strings.NewReader(csvPontoEVirgula))
	if err != nil {
		t.Fatalf("erro inesperado ao importar CSV com ponto-e-vírgula: %v", err)
	}
	if len(errosPonto) > 0 {
		t.Fatalf("erros inesperados: %v", errosPonto)
	}
	if len(colabsPonto) != 1 {
		t.Fatalf("esperava 1 colaborador, obteve %d", len(colabsPonto))
	}
	if colabsPonto[0].Matricula != "MAT-201" || colabsPonto[0].Setor != "Segurança do Trabalho" {
		t.Errorf("dados mapeados incorretamente: %+v", colabsPonto[0])
	}
}

func TestImportarColaboradoresCSV_ErrosLinha(t *testing.T) {
	csvComErros := `Nome,CPF,Matricula,Cargo,CBO,DataAdmissao,Setor
,12345678909,MAT-01,Assistente,4110-10,2023-01-01,Adm
Jose Invalido,11111111111,MAT-02,Operador,7156-15,2023-01-01,Fabrica
Data Ruim,98765432100,MAT-03,Vendedor,5211-10,32/13/20999,Comercial
Valido Sobrevivente,45678912364,MAT-04,Gerente,1423-05,2020-05-10,Diretoria`

	colabs, erros, err := importer.ImportarColaboradoresCSV(strings.NewReader(csvComErros))
	if err != nil {
		t.Fatalf("erro ao processar CSV: %v", err)
	}

	if len(colabs) != 1 {
		t.Errorf("esperava 1 colaborador sobrevivente válido, obteve %d", len(colabs))
	}
	if colabs[0].Nome != "Valido Sobrevivente" {
		t.Errorf("colaborador válido esperado não encontrado: %s", colabs[0].Nome)
	}

	if len(erros) != 3 {
		t.Errorf("esperava exatamente 3 erros de linha, obteve %d: %+v", len(erros), erros)
	}
}

func TestGerarModeloCSV_e_Reimportacao(t *testing.T) {
	modelo := importer.GerarModeloCSV()
	if len(modelo) == 0 {
		t.Fatal("o modelo gerado está vazio")
	}

	// O próprio modelo gerado deve ser 100% importável sem nenhum erro
	colabs, erros, err := importer.ImportarColaboradoresCSV(bytes.NewReader(modelo))
	if err != nil {
		t.Fatalf("falha ao importar o modelo gerado: %v", err)
	}
	if len(erros) > 0 {
		t.Fatalf("o modelo gerado possui erros de validação: %v", erros)
	}
	if len(colabs) != 3 {
		t.Fatalf("esperava 3 colaboradores de exemplo no modelo, obteve %d", len(colabs))
	}

	// Verifica se os CPFs foram limpos e validados
	for _, c := range colabs {
		if !importer.ValidarCPF(c.CPF) {
			t.Errorf("CPF do modelo de exemplo inválido: %s (%s)", c.CPF, c.Nome)
		}
		if c.DataAdmissao == "" {
			t.Errorf("Data de admissão não formatada para colaborador %s", c.Nome)
		}
	}
}
