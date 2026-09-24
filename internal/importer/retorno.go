package importer

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// RetornoTotalizacao representa um evento de totalização recebido do eSocial
// (S-5001/S-5002/S-5003 por trabalhador e S-5011/S-5012/S-5013 consolidados).
type RetornoTotalizacao struct {
	// Tipo é o nome do evento no XML (ex.: evtBasesTrab, evtCS, evtFGTS).
	Tipo string
	// PerApur é o período de apuração informado (ex.: 2026-01).
	PerApur string
	// CPFTrab e Matricula identificam o trabalhador (vazios nos consolidados).
	CPFTrab   string
	Matricula string
	// NRRecibo é o número do recibo/arquivo base informado pelo governo.
	NRRecibo string
	// Valores mapeia cada campo monetário (vr*) encontrado para o seu valor textual.
	Valores map[string]string
	// TotalCentavos é a soma dos campos monetários do registro, em centavos.
	TotalCentavos int64
}

// tiposTotalizacao lista os eventos de retorno reconhecidos pelo importador.
var tiposTotalizacao = map[string]string{
	"evtBasesTrab":   "S-5001 Contribuições sociais por trabalhador",
	"evtIrrfBenef":   "S-5002 IRRF por trabalhador",
	"evtFGTS":        "S-5003 FGTS por trabalhador",
	"evtCS":          "S-5011 Contribuições sociais consolidadas",
	"evtIrrf":        "S-5012 IRRF consolidado",
	"evtFGTSCons":    "S-5013 FGTS consolidado",
	"evtTotal":       "Totalizador",
	"evtBasesFGTS":   "Bases FGTS",
	"evtInfoContrib": "Informações de contribuições",
}

// ImportarRetorno lê um XML de retorno do eSocial e extrai os totalizadores de
// forma tolerante: identifica o tipo do evento pelo elemento raiz, captura a
// identificação do trabalhador/período/recibo e soma todos os campos monetários
// (elementos cujo nome inicia com "vr").
//
// A leitura é intencionalmente genérica porque os leiautes de retorno podem
// variar entre versões de NT; a conferência de valores é feita por campo.
func ImportarRetorno(r io.Reader) ([]RetornoTotalizacao, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charsetReaderTolerante

	var (
		pilha      []string
		atual      *RetornoTotalizacao
		registros  []RetornoTotalizacao
		textoAtual strings.Builder
	)

	fechar := func() {
		if atual != nil && (atual.Tipo != "" || len(atual.Valores) > 0) {
			registros = append(registros, *atual)
		}
		atual = nil
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("XML de retorno inválido: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			nome := t.Name.Local
			pilha = append(pilha, nome)

			if _, ok := tiposTotalizacao[nome]; ok {
				fechar()
				atual = &RetornoTotalizacao{Tipo: nome, Valores: map[string]string{}}
				continue
			}
			if atual == nil {
				continue
			}
			// ignora o envelope eSocial ao capturar identificação
			textoAtual.Reset()

		case xml.CharData:
			if atual == nil {
				continue
			}
			textoAtual.Write(t)

		case xml.EndElement:
			nome := t.Name.Local
			if len(pilha) > 0 {
				pilha = pilha[:len(pilha)-1]
			}
			valor := strings.TrimSpace(textoAtual.String())
			textoAtual.Reset()

			if atual == nil {
				continue
			}
			if _, ok := tiposTotalizacao[nome]; ok {
				fechar()
				continue
			}
			if valor == "" {
				continue
			}

			switch {
			case nome == "perApur" || nome == "perRef":
				if atual.PerApur == "" {
					atual.PerApur = valor
				}
			case nome == "cpfTrab":
				if atual.CPFTrab == "" {
					atual.CPFTrab = apenasDigitos(valor)
				}
			case nome == "matricula":
				if atual.Matricula == "" {
					atual.Matricula = valor
				}
			case nome == "nrRecArqBase" || nome == "nrRecibo" || nome == "nrReciboServ":
				if atual.NRRecibo == "" {
					atual.NRRecibo = valor
				}
			}

			if strings.HasPrefix(strings.ToLower(nome), "vr") {
				if centavos, ok := valorParaCentavos(valor); ok {
					chave := nome
					if _, existe := atual.Valores[chave]; existe {
						chave = nome + "#" + strconv.Itoa(len(atual.Valores))
					}
					atual.Valores[chave] = valor
					atual.TotalCentavos += centavos
				}
			}
		}
	}
	fechar()

	if len(registros) == 0 {
		return nil, fmt.Errorf("nenhum evento de totalização (S-5001/S-5011) reconhecido no XML enviado")
	}
	return registros, nil
}

// valorParaCentavos converte valores monetários do eSocial ("1500.00",
// "1.500,00", "1500") para centavos.
func valorParaCentavos(v string) (int64, bool) {
	limpo := strings.TrimSpace(v)
	if limpo == "" {
		return 0, false
	}
	limpo = strings.ReplaceAll(limpo, " ", "")
	// formato brasileiro: 1.500,00
	if strings.Contains(limpo, ",") {
		limpo = strings.ReplaceAll(limpo, ".", "")
		limpo = strings.ReplaceAll(limpo, ",", ".")
	}
	f, err := strconv.ParseFloat(limpo, 64)
	if err != nil {
		return 0, false
	}
	if f < 0 {
		f = -f
	}
	return int64(f*100 + 0.5), true
}

func apenasDigitos(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// charsetReaderTolerante aceita documentos declarados como ISO-8859-1.
func charsetReaderTolerante(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "utf8", "":
		return input, nil
	case "iso-8859-1", "latin1", "windows-1252":
		return input, nil
	default:
		return input, nil
	}
}
