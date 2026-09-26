package soap

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
)

// Ambientes suportados pelo eSocial
const (
	AmbienteProducao        = 1
	AmbienteProducaoRestrita = 2 // Homologação
)

// Endpoints oficiais do eSocial
const (
	URLProducaoEnvio    = "https://webservices.envio.esocial.gov.br/servicos/empregador/enviarloteeventos/WsEnviarLoteEventos.svc"
	URLProducaoConsulta = "https://webservices.consulta.esocial.gov.br/servicos/empregador/consultarloteeventos/WsConsultarLoteEventos.svc"
	URLRestritaEnvio    = "https://webservices.producaorestrita.esocial.gov.br/servicos/empregador/enviarloteeventos/WsEnviarLoteEventos.svc"
	URLRestritaConsulta = "https://webservices.producaorestrita.esocial.gov.br/servicos/empregador/consultarloteeventos/WsConsultarLoteEventos.svc"
)

// SOAP Actions oficiais do eSocial
const (
	SOAPActionEnvioLote    = "http://www.esocial.gov.br/servicos/empregador/lote/eventos/envio/v1_1_0/ServicoEnviarLoteEventos/EnviarLoteEventos"
	SOAPActionConsultaLote = "http://www.esocial.gov.br/servicos/empregador/lote/eventos/consulta/v1_1_0/ServicoConsultarLoteEventos/ConsultarLoteEventos"
)

// Ocorrencia representa advertências ou inconsistências retornadas pelo webservice do eSocial.
type Ocorrencia struct {
	Tipo        int    `json:"tipo"`        // 1 = Advertência, 2 = Erro impeditivo
	Codigo      int    `json:"codigo"`      // Código numérico da ocorrência do governo
	Descricao   string `json:"descricao"`   // Mensagem detalhada
	Localizacao string `json:"localizacao"` // Campo ou nó XML onde o erro foi identificado
}

// ReciboEvento contém os dados de processamento individual de cada evento transmitido.
type ReciboEvento struct {
	IdEvento          string       `json:"id_evento"`
	NumeroRecibo      string       `json:"numero_recibo"`
	CodigoResposta    int          `json:"codigo_resposta"`
	DescricaoResposta string       `json:"descricao_resposta"`
	Ocorrencias       []Ocorrencia `json:"ocorrencias"`
}

// RespostaEnvioLote encapsula os dados extraídos do retorno da chamada de envio de lote.
type RespostaEnvioLote struct {
	Sucesso           bool         `json:"sucesso"`
	CodigoResposta    int          `json:"codigo_resposta"`
	DescricaoResposta string       `json:"descricao_resposta"`
	ProtocoloEnvio    string       `json:"protocolo_envio"`
	DataRecepcao      time.Time    `json:"data_recepcao"`
	Ocorrencias       []Ocorrencia `json:"ocorrencias"`
	XMLBruto          string       `json:"xml_bruto"`
}

// RespostaConsultaLote encapsula os recibos e dados da consulta de processamento do lote.
type RespostaConsultaLote struct {
	Sucesso           bool           `json:"sucesso"`
	CodigoResposta    int            `json:"codigo_resposta"`
	DescricaoResposta string         `json:"descricao_resposta"`
	ProtocoloEnvio    string         `json:"protocolo_envio"`
	Eventos           []ReciboEvento `json:"eventos"`
	Ocorrencias       []Ocorrencia   `json:"ocorrencias"`
	XMLBruto          string         `json:"xml_bruto"`
}

// ClienteSOAP gerencia as requisições HTTPS mTLS com o eSocial.
type ClienteSOAP struct {
	cert                    crypto.Certificado
	httpClient              *http.Client
	urlEnvioPersonalizada   string
	urlConsultaPersonalizada string
}

// OpcaoClienteSOAP permite configurar parâmetros adicionais no ClienteSOAP.
type OpcaoClienteSOAP func(*ClienteSOAP)

// ComURLPersonalizada permite sobrescrever os endpoints padrão (muito útil em testes unitários/mock).
func ComURLPersonalizada(urlEnvio, urlConsulta string) OpcaoClienteSOAP {
	return func(c *ClienteSOAP) {
		c.urlEnvioPersonalizada = urlEnvio
		c.urlConsultaPersonalizada = urlConsulta
	}
}

// ComHTTPClient permite injetar um cliente HTTP customizado.
func ComHTTPClient(client *http.Client) OpcaoClienteSOAP {
	return func(c *ClienteSOAP) {
		c.httpClient = client
	}
}

// NovoClienteSOAP cria uma nova instância de ClienteSOAP configurando mTLS com o certificado fornecido.
func NovoClienteSOAP(cert crypto.Certificado, opts ...OpcaoClienteSOAP) (*ClienteSOAP, error) {
	if cert == nil {
		return nil, errors.New("certificado digital não pode ser nulo para cliente SOAP mTLS")
	}

	tlsCert, err := cert.TLSCertificate()
	if err != nil {
		return nil, fmt.Errorf("falha ao preparar certificado para TLS mútuo (mTLS): %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: false, // WebServices governamentais WCF/IIS operam com maior estabilidade em HTTP/1.1
		MaxIdleConns:      10,
		IdleConnTimeout:   30 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	c := &ClienteSOAP{
		cert:       cert,
		httpClient: client,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// ObterURLEnvio retorna o endpoint de envio de acordo com o ambiente especificado.
func (c *ClienteSOAP) ObterURLEnvio(ambiente int) string {
	if c.urlEnvioPersonalizada != "" {
		return c.urlEnvioPersonalizada
	}
	if ambiente == AmbienteProducao {
		return URLProducaoEnvio
	}
	return URLRestritaEnvio
}

// ObterURLConsulta retorna o endpoint de consulta de acordo com o ambiente especificado.
func (c *ClienteSOAP) ObterURLConsulta(ambiente int) string {
	if c.urlConsultaPersonalizada != "" {
		return c.urlConsultaPersonalizada
	}
	if ambiente == AmbienteProducao {
		return URLProducaoConsulta
	}
	return URLRestritaConsulta
}

// EnviarLote envia um ou mais eventos assinados ao eSocial via SOAP com mTLS.
// Se o XML informado for um evento individual (<eSocial ...><evt...>), ele é automaticamente
// encapsulado dentro da estrutura de lote padrão do eSocial.
func (c *ClienteSOAP) EnviarLote(ctx context.Context, xmlAssinado []byte, ambiente int) (*RespostaEnvioLote, error) {
	xmlLote := c.prepararXMLLote(xmlAssinado)

	envelope := fmt.Sprintf(
		`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:v1="http://www.esocial.gov.br/servicos/empregador/lote/eventos/envio/v1_1_0">`+
			`<soapenv:Header/>`+
			`<soapenv:Body>`+
			`<v1:EnviarLoteEventos>`+
			`<v1:loteEventos>%s</v1:loteEventos>`+
			`</v1:EnviarLoteEventos>`+
			`</soapenv:Body>`+
			`</soapenv:Envelope>`,
		string(xmlLote),
	)

	endpoint := c.ObterURLEnvio(ambiente)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição HTTP SOAP: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", SOAPActionEnvioLote)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na comunicação HTTP mTLS com eSocial: %w", err)
	}
	defer resp.Body.Close()

	corpoResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta do eSocial: %w", err)
	}

	return c.interpretarRespostaEnvio(corpoResp)
}

// ConsultarLote consulta o protocolo de envio no webservice do eSocial para obter recibos ou erros de validação.
func (c *ClienteSOAP) ConsultarLote(ctx context.Context, protocolo string, ambiente int) (*RespostaConsultaLote, error) {
	protocoloLimpo := strings.TrimSpace(protocolo)
	if protocoloLimpo == "" {
		return nil, errors.New("número de protocolo não informado para consulta")
	}

	envelope := fmt.Sprintf(
		`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:v1="http://www.esocial.gov.br/servicos/empregador/lote/eventos/consulta/v1_1_0">`+
			`<soapenv:Header/>`+
			`<soapenv:Body>`+
			`<v1:ConsultarLoteEventos>`+
			`<v1:consulta>`+
			`<eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/consulta/v1_0_0">`+
			`<consultaLoteEventos>`+
			`<protocoloEnvio>%s</protocoloEnvio>`+
			`</consultaLoteEventos>`+
			`</eSocial>`+
			`</v1:consulta>`+
			`</v1:ConsultarLoteEventos>`+
			`</soapenv:Body>`+
			`</soapenv:Envelope>`,
		protocoloLimpo,
	)

	endpoint := c.ObterURLConsulta(ambiente)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(envelope))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição de consulta SOAP: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", SOAPActionConsultaLote)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na comunicação HTTP mTLS com eSocial: %w", err)
	}
	defer resp.Body.Close()

	corpoResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta da consulta: %w", err)
	}

	return c.interpretarRespostaConsulta(corpoResp)
}

// prepararXMLLote assegura que o evento esteja encapsulado no elemento de lote de envio do eSocial.
func (c *ClienteSOAP) prepararXMLLote(xmlAssinado []byte) []byte {
	conteudo := strings.TrimSpace(string(xmlAssinado))

	// Remove declaração <?xml ...?> para evitar conflito de embedding
	if strings.HasPrefix(conteudo, "<?xml") {
		if idx := strings.Index(conteudo, "?>"); idx != -1 {
			conteudo = strings.TrimSpace(conteudo[idx+2:])
		}
	}

	// Se já for um lote completo de eventos, retorna direto
	if strings.Contains(conteudo, "envioLoteEventos") {
		return []byte(conteudo)
	}

	// Extrai CNPJ do certificado para usar no ideEmpregador / ideTransmissor
	cnpj := c.cert.CNPJ()
	if cnpj == "" {
		cnpj = "00000000000000"
	}
	// Se tiver 14 dígitos, o empregador geralmente usa os 8 primeiros dígitos (raiz do CNPJ) ou 14 dígitos
	nrInscEmpregador := cnpj
	if len(cnpj) == 14 {
		nrInscEmpregador = cnpj[:8]
	}

	// Extrai Id do evento para a tag <evento Id="...">
	idEvento := "ID1"
	reID := regexp.MustCompile(`Id="([^"]+)"`)
	if match := reID.FindStringSubmatch(conteudo); len(match) > 1 {
		idEvento = match[1]
	}

	lote := fmt.Sprintf(
		`<eSocial xmlns="http://www.esocial.gov.br/schema/lote/eventos/envio/v1_1_1">`+
			`<envioLoteEventos grupo="1">`+
			`<ideEmpregador>`+
			`<tpInsc>1</tpInsc>`+
			`<nrInsc>%s</nrInsc>`+
			`</ideEmpregador>`+
			`<ideTransmissor>`+
			`<tpInsc>1</tpInsc>`+
			`<nrInsc>%s</nrInsc>`+
			`</ideTransmissor>`+
			`<eventos>`+
			`<evento Id="%s">%s</evento>`+
			`</eventos>`+
			`</envioLoteEventos>`+
			`</eSocial>`,
		nrInscEmpregador,
		cnpj,
		idEvento,
		conteudo,
	)

	return []byte(lote)
}

func (c *ClienteSOAP) interpretarRespostaEnvio(corpo []byte) (*RespostaEnvioLote, error) {
	xmlStr := string(corpo)
	resultado := &RespostaEnvioLote{
		XMLBruto: xmlStr,
	}

	// Verifica se há SOAP Fault
	if strings.Contains(xmlStr, "<soap:Fault>") || strings.Contains(xmlStr, "<Fault>") {
		faultString := extrairTagXML(xmlStr, "faultstring")
		return resultado, fmt.Errorf("falha SOAP retornada pelo webservice: %s", faultString)
	}

	// Extrai código e descrição de resposta
	cdResp := extrairTagXML(xmlStr, "cdResposta")
	if cdResp != "" {
		resultado.CodigoResposta, _ = strconv.Atoi(cdResp)
	}
	resultado.DescricaoResposta = extrairTagXML(xmlStr, "descResposta")

	// No eSocial: 201 = Lote recebido com sucesso (com advertências); 202 = Lote recebido com sucesso
	if resultado.CodigoResposta == 201 || resultado.CodigoResposta == 202 {
		resultado.Sucesso = true
	}

	// Protocolo de Envio
	resultado.ProtocoloEnvio = extrairTagXML(xmlStr, "protocoloEnvio")

	// Data/hora de recepção
	dhRecepcao := extrairTagXML(xmlStr, "dhRecepcao")
	if dhRecepcao != "" {
		if t, err := time.Parse(time.RFC3339, dhRecepcao); err == nil {
			resultado.DataRecepcao = t
		} else if t, err := time.Parse("2006-01-02T15:04:05", dhRecepcao); err == nil {
			resultado.DataRecepcao = t
		}
	}

	// Extrai eventuais ocorrências de erro/aviso
	resultado.Ocorrencias = extrairOcorrenciasXML(xmlStr)

	return resultado, nil
}

func (c *ClienteSOAP) interpretarRespostaConsulta(corpo []byte) (*RespostaConsultaLote, error) {
	xmlStr := string(corpo)
	resultado := &RespostaConsultaLote{
		XMLBruto: xmlStr,
	}

	// Verifica se há SOAP Fault
	if strings.Contains(xmlStr, "<soap:Fault>") || strings.Contains(xmlStr, "<Fault>") {
		faultString := extrairTagXML(xmlStr, "faultstring")
		return resultado, fmt.Errorf("falha SOAP retornada pelo webservice: %s", faultString)
	}

	// Código e descrição geral do lote
	cdResp := extrairTagXML(xmlStr, "cdResposta")
	if cdResp != "" {
		resultado.CodigoResposta, _ = strconv.Atoi(cdResp)
	}
	resultado.DescricaoResposta = extrairTagXML(xmlStr, "descResposta")

	// 201 = Lote processado com sucesso; 202 = Lote processado com advertências
	if resultado.CodigoResposta == 201 || resultado.CodigoResposta == 202 {
		resultado.Sucesso = true
	}

	resultado.ProtocoloEnvio = extrairTagXML(xmlStr, "protocoloEnvio")
	resultado.Ocorrencias = extrairOcorrenciasXML(xmlStr)

	// Extrai recibos individuais dos eventos
	resultado.Eventos = extrairRecibosEventosXML(xmlStr)

	return resultado, nil
}

// extrairTagXML localiza o conteúdo da primeira ocorrência de uma tag XML de forma tolerante a namespaces.
func extrairTagXML(xmlStr, tagName string) string {
	padrao := fmt.Sprintf(`(?i)<(?:\w+:)?%s\b[^>]*>([^<]+)</(?:\w+:)?%s>`, tagName, tagName)
	re := regexp.MustCompile(padrao)
	match := re.FindStringSubmatch(xmlStr)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// extrairOcorrenciasXML localiza todos os blocos <ocorrencia> no XML e extrai seus atributos.
func extrairOcorrenciasXML(xmlStr string) []Ocorrencia {
	var ocorrencias []Ocorrencia

	reBloco := regexp.MustCompile(`(?s)<(?:\w+:)?ocorrencia\b[^>]*>(.*?)</(?:\w+:)?ocorrencia>`)
	matches := reBloco.FindAllStringSubmatch(xmlStr, -1)

	for _, m := range matches {
		bloco := m[1]
		tipoStr := extrairTagXML(bloco, "tipo")
		codStr := extrairTagXML(bloco, "codigo")
		desc := extrairTagXML(bloco, "descricao")
		loc := extrairTagXML(bloco, "localizacao")

		tipo, _ := strconv.Atoi(tipoStr)
		codigo, _ := strconv.Atoi(codStr)

		if desc != "" || codigo != 0 {
			ocorrencias = append(ocorrencias, Ocorrencia{
				Tipo:        tipo,
				Codigo:      codigo,
				Descricao:   desc,
				Localizacao: loc,
			})
		}
	}

	return ocorrencias
}

// extrairRecibosEventosXML processa os blocos de recibos de entrega dos eventos retornados na consulta.
func extrairRecibosEventosXML(xmlStr string) []ReciboEvento {
	var recibos []ReciboEvento

	// Localiza os blocos <evento Id="..."> dentro de <retornoEventos>
	reEvento := regexp.MustCompile(`(?s)<(?:\w+:)?evento\b[^>]*\bId="([^"]+)"[^>]*>(.*?)</(?:\w+:)?evento>`)
	matches := reEvento.FindAllStringSubmatch(xmlStr, -1)

	for _, m := range matches {
		idEvento := m[1]
		blocoEvento := m[2]

		nrRecibo := extrairTagXML(blocoEvento, "nrRecibo")
		cdRespStr := extrairTagXML(blocoEvento, "cdResposta")
		descResp := extrairTagXML(blocoEvento, "descResposta")

		cdResp, _ := strconv.Atoi(cdRespStr)
		ocorr := extrairOcorrenciasXML(blocoEvento)

		recibos = append(recibos, ReciboEvento{
			IdEvento:          idEvento,
			NumeroRecibo:      nrRecibo,
			CodigoResposta:    cdResp,
			DescricaoResposta: descResp,
			Ocorrencias:       ocorr,
		})
	}

	// Caso não encontre pelo bloco agrupado, tenta busca direta por <nrRecibo>
	if len(recibos) == 0 {
		nrRecibo := extrairTagXML(xmlStr, "nrRecibo")
		if nrRecibo != "" {
			recibos = append(recibos, ReciboEvento{
				NumeroRecibo:   nrRecibo,
				CodigoResposta: 201,
			})
		}
	}

	return recibos
}

