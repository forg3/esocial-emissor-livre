package importer

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
)

// ColaboradorImportado representa os dados de um trabalhador mapeados a partir do arquivo CSV.
type ColaboradorImportado struct {
	Linha        int    `json:"linha"`
	Nome         string `json:"nome"`
	CPF          string `json:"cpf"`           // Apenas dígitos
	Matricula    string `json:"matricula"`
	Cargo        string `json:"cargo"`
	CBO          string `json:"cbo"`
	DataAdmissao string `json:"data_admissao"` // No formato ISO YYYY-MM-DD
	Setor        string `json:"setor"`
}

// ErroLinha detalha um problema de validação específico encontrado em uma linha do CSV.
type ErroLinha struct {
	Linha    int    `json:"linha"`
	Coluna   string `json:"coluna,omitempty"`
	Mensagem string `json:"mensagem"`
}

func (e ErroLinha) Error() string {
	if e.Coluna != "" {
		return fmt.Sprintf("linha %d, coluna '%s': %s", e.Linha, e.Coluna, e.Mensagem)
	}
	return fmt.Sprintf("linha %d: %s", e.Linha, e.Mensagem)
}

// ValidarCPF verifica a integridade de um CPF pelo algoritmo de dígitos verificadores oficiais.
func ValidarCPF(cpf string) bool {
	// Remove pontuações e caracteres não numéricos
	var digitos []int
	for _, r := range cpf {
		if unicode.IsDigit(r) {
			digitos = append(digitos, int(r-'0'))
		}
	}

	if len(digitos) != 11 {
		return false
	}

	// Rejeita sequências com todos os dígitos repetidos (ex: 111.111.111-11)
	todosIguais := true
	for i := 1; i < 11; i++ {
		if digitos[i] != digitos[0] {
			todosIguais = false
			break
		}
	}
	if todosIguais {
		return false
	}

	// 1º Dígito Verificador
	soma := 0
	for i := 0; i < 9; i++ {
		soma += digitos[i] * (10 - i)
	}
	resto := soma % 11
	d1 := 11 - resto
	if d1 >= 10 {
		d1 = 0
	}
	if digitos[9] != d1 {
		return false
	}

	// 2º Dígito Verificador
	soma = 0
	for i := 0; i < 10; i++ {
		soma += digitos[i] * (11 - i)
	}
	resto = soma % 11
	d2 := 11 - resto
	if d2 >= 10 {
		d2 = 0
	}
	return digitos[10] == d2
}

// LimparDocumento extrai somente dígitos numéricos.
func LimparDocumento(doc string) string {
	var sb strings.Builder
	for _, r := range doc {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// NormalizarData tenta converter diversos formatos populares de data para AAAA-MM-DD.
func NormalizarData(dataStr string) (string, error) {
	dataStr = strings.TrimSpace(dataStr)
	if dataStr == "" {
		return "", errors.New("data vazia")
	}

	formatos := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"2/1/2006",
		"02/01/06",
		"2006/01/02",
	}

	for _, f := range formatos {
		if t, err := time.Parse(f, dataStr); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("formato de data não reconhecido: %s (utilize DD/MM/AAAA ou AAAA-MM-DD)", dataStr)
}

func normalizarCabecalho(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Remove acentos comuns
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u", "ü", "u",
		"ç", "c",
		"_", " ",
		"-", " ",
	)
	return replacer.Replace(s)
}

// detectarSeparador analisa o conteúdo inicial para definir se usa vírgula ou ponto-e-vírgula.
func detectarSeparador(conteudo []byte) rune {
	linhas := bytes.Split(conteudo, []byte("\n"))
	if len(linhas) == 0 {
		return ','
	}
	primeiraLinha := string(linhas[0])
	pontosEVirgulas := strings.Count(primeiraLinha, ";")
	virgulas := strings.Count(primeiraLinha, ",")
	if pontosEVirgulas > virgulas {
		return ';'
	}
	return ','
}

// ImportarColaboradoresCSV lê e processa um arquivo CSV, mapeando os colaboradores e validando campos.
func ImportarColaboradoresCSV(r io.Reader) ([]ColaboradorImportado, []ErroLinha, error) {
	conteudo, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}

	// Remove BOM UTF-8 se presente
	conteudo = bytes.TrimPrefix(conteudo, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(conteudo)) == 0 {
		return nil, nil, errors.New("o arquivo CSV está vazio")
	}

	separador := detectarSeparador(conteudo)
	reader := csv.NewReader(bytes.NewReader(conteudo))
	reader.Comma = separador
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	registros, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao interpretar CSV: %w", err)
	}

	if len(registros) < 2 {
		return nil, nil, errors.New("o arquivo CSV deve conter o cabeçalho e pelo menos uma linha de dados")
	}

	// Identifica as colunas pelos cabeçalhos
	cabecalho := registros[0]
	idxNome := -1
	idxCPF := -1
	idxMatricula := -1
	idxCargo := -1
	idxCBO := -1
	idxDataAdmissao := -1
	idxSetor := -1

	for i, col := range cabecalho {
		norm := normalizarCabecalho(col)
		switch {
		case strings.Contains(norm, "nome") || strings.Contains(norm, "colaborador") || strings.Contains(norm, "trabalhador") || strings.Contains(norm, "empregado"):
			if idxNome == -1 {
				idxNome = i
			}
		case strings.Contains(norm, "cpf") || strings.Contains(norm, "documento"):
			if idxCPF == -1 {
				idxCPF = i
			}
		case strings.Contains(norm, "matricula") || strings.Contains(norm, "registro") || norm == "cod" || norm == "codigo":
			if idxMatricula == -1 {
				idxMatricula = i
			}
		case strings.Contains(norm, "cargo") || strings.Contains(norm, "funcao") || strings.Contains(norm, "ocupacao"):
			if idxCargo == -1 {
				idxCargo = i
			}
		case strings.Contains(norm, "cbo"):
			if idxCBO == -1 {
				idxCBO = i
			}
		case strings.Contains(norm, "admissao") || strings.Contains(norm, "dtadm") || strings.Contains(norm, "data adm"):
			if idxDataAdmissao == -1 {
				idxDataAdmissao = i
			}
		case strings.Contains(norm, "setor") || strings.Contains(norm, "departamento") || strings.Contains(norm, "depto") || strings.Contains(norm, "area") || strings.Contains(norm, "lotacao"):
			if idxSetor == -1 {
				idxSetor = i
			}
		}
	}

	if idxNome == -1 || idxCPF == -1 {
		return nil, nil, errors.New("cabeçalho inválido: as colunas 'Nome' e 'CPF' são estritamente obrigatórias no CSV")
	}

	var colaboradores []ColaboradorImportado
	var erros []ErroLinha

	for i := 1; i < len(registros); i++ {
		linhaNum := i + 1 // Linha 1 é o cabeçalho
		linha := registros[i]

		// Ignora linhas totalmente em branco
		vazia := true
		for _, v := range linha {
			if strings.TrimSpace(v) != "" {
				vazia = false
				break
			}
		}
		if vazia {
			continue
		}

		getCol := func(idx int) string {
			if idx >= 0 && idx < len(linha) {
				return strings.TrimSpace(linha[idx])
			}
			return ""
		}

		nome := getCol(idxNome)
		cpfBruto := getCol(idxCPF)
		cpfLimpo := LimparDocumento(cpfBruto)
		matricula := getCol(idxMatricula)
		cargo := getCol(idxCargo)
		cbo := getCol(idxCBO)
		dataAdmBruta := getCol(idxDataAdmissao)
		setor := getCol(idxSetor)

		var linhaComErro bool

		// 1. Validação do Nome
		if nome == "" {
			erros = append(erros, ErroLinha{
				Linha:    linhaNum,
				Coluna:   "Nome",
				Mensagem: "o nome do colaborador não pode ser vazio",
			})
			linhaComErro = true
		}

		// 2. Validação do CPF
		if !ValidarCPF(cpfLimpo) {
			erros = append(erros, ErroLinha{
				Linha:    linhaNum,
				Coluna:   "CPF",
				Mensagem: fmt.Sprintf("CPF '%s' inválido pelo algoritmo oficial da Receita Federal", cpfBruto),
			})
			linhaComErro = true
		}

		// 3. Validação de Data de Admissão
		dataAdmFormatada := ""
		if dataAdmBruta != "" {
			var errData error
			dataAdmFormatada, errData = NormalizarData(dataAdmBruta)
			if errData != nil {
				erros = append(erros, ErroLinha{
					Linha:    linhaNum,
					Coluna:   "DataAdmissao",
					Mensagem: errData.Error(),
				})
				linhaComErro = true
			}
		}

		if !linhaComErro {
			colaboradores = append(colaboradores, ColaboradorImportado{
				Linha:        linhaNum,
				Nome:         nome,
				CPF:          cpfLimpo,
				Matricula:    matricula,
				Cargo:        cargo,
				CBO:          cbo,
				DataAdmissao: dataAdmFormatada,
				Setor:        setor,
			})
		}
	}

	return colaboradores, erros, nil
}

// GerarModeloCSV gera um arquivo CSV pronto para download contendo o layout oficial
// e exemplos válidos para importação no sistema.
func GerarModeloCSV() []byte {
	var sb strings.Builder
	// Cabeçalho compatível com Excel e sistemas de RH
	sb.WriteString("Nome,CPF,Matricula,Cargo,CBO,DataAdmissao,Setor\n")
	sb.WriteString("Maria da Silva,12345678909,MAT-001,Assistente Administrativo,4110-10,2023-01-10,Administrativo\n")
	sb.WriteString("Carlos Eduardo Souza,98765432100,MAT-002,Eletricista de Manutenção,7156-15,2022-05-15,Manutenção\n")
	sb.WriteString("Ana Paula Oliveira,45678912364,MAT-003,Engenheira de Segurança,2149-15,2021-08-01,SESMT\n")
	return []byte(sb.String())
}
