#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Gerador do Relatorio de Auditoria de Seguranca - Validador eSocial.

Uso:
    python3 -m venv .venv-report && . .venv-report/bin/activate
    pip install reportlab matplotlib
    python docs/security-audit/gerar_relatorio.py

Saidas:
    docs/security-audit/relatorio-auditoria-seguranca.pdf
    docs/security-audit/graficos/severidade.png
    docs/security-audit/graficos/categorias.png

Todos os achados abaixo foram verificados no codigo-fonte do repositorio.
Nenhum achado especulativo foi incluido.
"""

import os
from datetime import date

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm
from xml.sax.saxutils import escape as xesc
from reportlab.platypus import (
    BaseDocTemplate,
    Frame,
    Image,
    KeepTogether,
    LongTable,
    PageBreak,
    PageTemplate,
    Paragraph,
    Preformatted,
    Spacer,
    Table,
    TableStyle,
)

def esc(t):
    """Escapa &, < e > para uso seguro dentro de Paragraph/Preformatted."""
    return xesc(t)


BASE_DIR = os.path.dirname(os.path.abspath(__file__))
GRAF_DIR = os.path.join(BASE_DIR, "graficos")
PDF_PATH = os.path.join(BASE_DIR, "relatorio-auditoria-seguranca.pdf")

PROJETO = "Validador eSocial"
VERSAO = "v1.1-alpha"
DATA_RELATORIO = date(2026, 9, 23).strftime("%d/%m/%Y")

COR = {
    "critica": colors.HexColor("#B91C1C"),
    "alta": colors.HexColor("#EA580C"),
    "media": colors.HexColor("#D97706"),
    "baixa": colors.HexColor("#2563EB"),
    "informativa": colors.HexColor("#64748B"),
    "forte": colors.HexColor("#059669"),
    "navy": colors.HexColor("#0F172A"),
    "texto": colors.HexColor("#1F2937"),
    "cinza": colors.HexColor("#E2E8F0"),
    "cinza_claro": colors.HexColor("#F1F5F9"),
}

ROTULO_SEV = {
    "critica": "CRITICA",
    "alta": "ALTA",
    "media": "MEDIA",
    "baixa": "BAIXA",
    "informativa": "INFORMATIVA",
}

# ---------------------------------------------------------------------------
# ACHADOS (todos verificados no codigo real)
# ---------------------------------------------------------------------------

ACHADOS = [
    {
        "id": "C-01",
        "cat": "Isolamento e Controle de Acesso",
        "sev": "critica",
        "local": "cmd/server/main.go:55 + internal/web/server.go:198-257",
        "titulo": "Aplicacao sem autenticacao alguma, ouvindo em todas as interfaces de rede (0.0.0.0:8000)",
        "trecho": 'Addr: fmt.Sprintf(":%d", *porta)  // main.go:55  -> bind 0.0.0.0\nmux := http.NewServeMux()                    // server.go:199\n// nenhum middleware de sessao/token/login em Rotas()',
        "porque": (
            "O servidor HTTP nao possui qualquer camada de autenticacao ou autorizacao: nao existe login, sessao, cookie, "
            "token, middleware de auth ou validacao de origem. O bind e feito em \":8000\", que em Go escuta em todas as "
            "interfaces (0.0.0.0), e nao em 127.0.0.1. Sonda executada contra as rotas reais retornou HTTP 200 sem "
            "credencial alguma para GET /fila, GET /colaboradores, GET /configuracao, DELETE /colaboradores/{id} e "
            "POST /fila/lote/transmitir. Qualquer host na mesma rede (Wi-Fi corporativo, LAN, container com porta "
            "publicada) le nomes e CPFs de trabalhadores, baixa XMLs, altera a configuracao da empresa, troca o "
            "certificado digital, dispara assinatura/transmissao em lote e apaga registros."
        ),
        "impacto": (
            "Exposicao de dados pessoais (LGPD) de toda a folha de colaboradores, manipulacao de documentos fiscais "
            "assinados, substituicao do certificado A1 e destruicao de dados - tudo sem credencial."
        ),
        "fix": (
            "1) Trocar o bind para 127.0.0.1 por padrao (flag explicita para expor). 2) Introduzir autenticacao "
            "(sessao com cookie SameSite=Strict ou token local) com middleware aplicado a todas as rotas exceto /static. "
            "3) Validar o cabecalho Host (allowlist localhost/127.0.0.1) para mitigar DNS rebinding. 4) Documentar o "
            "modo Docker como exposto apenas a rede confiavel, com auth obrigatoria."
        ),
        "cond": "Exploravel sempre que a porta estiver acessivel (LAN, docker com -p 8000:8000, notebook em rede publica).",
    },
    {
        "id": "A-01",
        "cat": "Isolamento e Controle de Acesso",
        "sev": "alta",
        "local": "internal/web/server.go:213-251 (rotas) + internal/web/templates/*.html (formularios)",
        "titulo": "CSRF: endpoints de escrita sem token anti-CSRF e sem validacao de Origin/Referer",
        "trecho": 'mux.HandleFunc("POST /configuracao", s.handleSalvarConfiguracao)\nmux.HandleFunc("POST /fila/lote/transmitir", s.handleTransmitirLote)\nmux.HandleFunc("POST /colaboradores", s.handleSalvarColaborador)\n// grep -rni "csrf" => nenhum resultado no repositorio',
        "porque": (
            "Nenhum formulario possui campo anti-CSRF (grep por 'csrf' no repositorio nao retorna nada) e nenhum handler "
            "verifica Origin/Referer. Como os POSTs sao requisicoes simples, uma pagina maliciosa visitada pelo operador "
            "pode submeter formularios cross-site para localhost:8000 e executar acoes de escrita sem que o usuario perceba: "
            "alterar razao social/CNPJ/ambiente, cadastrar colaborador, assinar e transmitir lotes inteiros. Os metodos "
            "DELETE nao sao alcancaveis por formulario (exigiriam preflight CORS), mas todos os POSTs sensiveis sao."
        ),
        "impacto": "Integridade de configuracao e de dados fiscais; alteracao de ambiente (Producao Restrita para Producao Oficial).",
        "fix": (
            "Emitir token sincronizado por formulario (campo hidden) validado no servidor para todo metodo nao-idempotente "
            "e/ou exigir cabecalho customizado (X-Requested-With / X-CSRF-Token) com validacao de Origin/Referer. "
            "Se houver sessao, usar cookie SameSite=Strict + verificacao de Host."
        ),
        "cond": "Requer que o operador visite uma pagina controlada pelo atacante enquanto o servico esta ativo.",
    },
    {
        "id": "A-02",
        "cat": "Complementares (Integridade)",
        "sev": "alta",
        "local": "internal/web/generator.go:484-533, internal/web/server.go:1444-1543, README.md:14,65-67",
        "titulo": "Assinatura digital e recibos do eSocial sao FALSIFICADOS (simulacao) e apresentados como oficiais",
        "trecho": '<SignatureValue>SIMULATED_RSA_SHA256_SIGNATURE_%s</SignatureValue>   // generator.go:503\nrecibo = fmt.Sprintf("%d.2.%s.%014d", ambiente, agora, nano+107)          // generator.go:521\n"Recibo oficial do eSocial ja confirmado: %s"                            // server.go:1491\n// internal/soap/client.go nunca e referenciado fora dos testes (codigo morto)',
        "porque": (
            "Os handlers de assinatura e transmissao nao usam criptografia nem o webservice: SimularAssinatura gera um "
            "envelope XMLDSig falso (DigestValue calculado sobre o proprio texto e SignatureValue literal "
            "'SIMULATED_RSA_SHA256_SIGNATURE_'), e SimularTransmissao inventa protocolo/recibo com a mensagem 'Recibo "
            "emitido pelo Serpro/Receita Federal'. O status do evento passa a 'aceito' e a interface afirma 'Recibo "
            "oficial do eSocial ja confirmado'. O cliente SOAP mTLS real existe (internal/soap/client.go) mas nao esta "
            "conectado a nenhuma rota. O README afirma transmissao real pelos webservices oficiais."
        ),
        "impacto": (
            "Trilha de auditoria fiscal falsa: o usuario acredita ter cumprido obrigacao acessoria do eSocial. Risco "
            "documental, regulatorio e de conformidade (declaracao inexistente), alem de assinatura sem valor juridico."
        ),
        "fix": (
            "Rotular explicitamente o modo simulacao na interface e no banco (ex.: status 'simulado'), bloquear a geracao "
            "de recibo quando nao houver transmissao real e conectar o ClienteSOAP (mTLS + certificado A1) aos handlers, "
            "persistindo o XML de resposta oficial. Ajustar o README para refletir o estado real."
        ),
        "cond": "Sempre ativo nesta versao: qualquer evento 'assinado/aceito' hoje e ficticio.",
    },
    {
        "id": "A-03",
        "cat": "Inputs sem Tratamento (XSS)",
        "sev": "alta",
        "local": "internal/web/server.go:472-475, 489-492, 518-539",
        "titulo": "HTML injection / XSS armazenado em POST /configuracao/testar (saida sem escape via fmt.Fprintf)",
        "trecho": 'fmt.Fprintf(w, `<div class="flash flash-err" ...>\n  <strong>Arquivo nao encontrado:</strong> O arquivo %s ...\n</div>`, filepath.Base(cfg.CertificadoPath))   // server.go:472-474',
        "porque": (
            "O handler escreve HTML concatenando dados controlados pelo usuario diretamente em fmt.Fprintf, sem usar o "
            "html/template (que escaparia). cfg.CertificadoPath vem de header.Filename do upload de certificado e fica "
            "persistido no SQLite; o nome do arquivo e tambem o CN/titular do certificado sao refletidos nos blocos das "
            "linhas 518-539. Sonda executada via httptest contra a rota real: configurando CertificadoPath = "
            "'<img src=x onerror=alert(1)>.pfx' o corpo da resposta 200 conteve a tag <img> intacta (HTML_INJECTION_CONFIRMED). "
            "Como o valor persiste no banco, o script executa em cada visita a pagina de certificado (XSS armazenado)."
        ),
        "impacto": (
            "Execucao de JavaScript na origem do painel (localhost:8000): leitura de todo o conteudo do painel, disparo de "
            "acoes autenticadas pelo proprio navegador do operador e exfiltracao de dados de colaboradores."
        ),
        "fix": (
            "Substituir todo fmt.Fprintf de HTML por template.ExecuteTemplate com html/template (escaping contextual) ou "
            "aplicar template.HTMLEscapeString em cada interpolacao. Remover a funcao safeHTML (server.go:97-99) para "
            "eliminar o footgun."
        ),
        "cond": "Requer que um arquivo .pfx com nome/atributos maliciosos seja carregado (ou via CSRF do achado A-01).",
    },
    {
        "id": "M-01",
        "cat": "Inputs sem Tratamento (XSS)",
        "sev": "media",
        "local": "internal/web/generator.go:118, 120, 129, 135, 140, 166, 176, 255, 257, 307, 334, 398, 401, 414",
        "titulo": "XML injection: campos de formulario interpolados sem escape nos geradores de eventos",
        "trecho": 'fmt.Fprintf(&sb, "        <codAgNoc>%s</codAgNoc>\\n", p.CodigoRisco)      // generator.go:118 (sem escapeXML)\nfmt.Fprintf(&sb, "        <ideOC>%s</ideOC>\\n", orgao)                        // generator.go:166\nfmt.Fprintf(&sb, "      <dtAcid>%s</dtAcid>\\n", p.DtAcidente)                 // generator.go:255',
        "porque": (
            "Parte dos valores de formulario entra no XML por Fprintf sem passar por escapeXML: codAgNoc, tpAval, "
            "utilizEPC/eficEpc/utilizEPI, ideOC (orgao de classe), ufOC, dtAcid/hrAcid/dtAtendimento/dtAso/dtExm, "
            "ufCRM e ufOC do medico. Um valor como '</codAgNoc><codAgNoc>X' injeta elementos arbitrarios no documento "
            "que depois recebe o envelope de assinatura e e validado/transmitido, permitindo adulterar conteudo assinado "
            "(por exemplo trocar dados do empregador ou do trabalhador dentro do envelope)."
        ),
        "impacto": "Adulteracao de documento fiscal antes da assinatura; assinatura de conteudo divergente do formulario exibido.",
        "fix": (
            "Aplicar escapeXML (ou construir o XML com encoding/xml a partir de structs) em TODOS os campos, e validar "
            "listas de dominio (datas ISO, tpAval in {1,2}, utilizEPC/EPI in {0,1,2}, UF em lista oficial) antes de gerar."
        ),
        "cond": "Exploravel por POST direto/CSRF; a validacao XSD real (se ligada) reduziria parte dos casos malformados.",
    },
    {
        "id": "M-02",
        "cat": "Inputs sem Tratamento (XSS)",
        "sev": "media",
        "local": "internal/web/server.go:409-441, 1051-1067, 701-705",
        "titulo": "Uploads e corpos de requisicao sem limite de tamanho (consumo de memoria/disco)",
        "trecho": '_, _ = io.Copy(destFile, arquivo)   // server.go:418  (sem limite)\nxmlBytes, err := io.ReadAll(arquivo)   // server.go:1067 (sem limite)\nlinhas, err := leitor.ReadAll()         // server.go:705  (sem limite)\n// grep "MaxBytesReader|LimitReader" => nenhum resultado no projeto',
        "porque": (
            "Nao existe http.MaxBytesReader, LimitReader nem ParseMultipartForm com maxMemory em nenhum ponto. O upload "
            "de .pfx grava o arquivo inteiro em disco, o import de XML do ASO carrega tudo em memoria e o import de CSV "
            "le todas as linhas em memoria. Um cliente (qualquer host da rede, dado o achado C-01) pode esgotar disco ou "
            "memoria do processo com um unico request grande."
        ),
        "impacto": "Negacao de servico local (processo morto por OOM / disco cheio) e indisponibilidade do emissor.",
        "fix": (
            "Envolver o corpo com http.MaxBytesReader (ex.: 2 MB para PFX, 5 MB para XML, 10 MB para CSV), usar "
            "r.ParseMultipartForm(limite) e validar Content-Length antes de io.Copy/ReadAll."
        ),
        "cond": "Exploravel por qualquer cliente HTTP que alcance a porta.",
    },
    {
        "id": "M-03",
        "cat": "Complementares (Validacao)",
        "sev": "media",
        "local": "internal/web/server.go:1433-1436 + internal/web/generator.go:424-482 + internal/esocial/validator.go:101",
        "titulo": "Validacao XSD real existe no projeto mas nao esta ligada ao handler; UI afirma validacao oficial",
        "trecho": 'valido, erros := ValidarEventoXSD(evento.Tipo, evento.XMLGerado)   // server.go:1433 (apenas strings.Contains)\n// esocial.ValidarXSD (xmllint + XSD oficial) e usado SOMENTE em esocial_test.go',
        "porque": (
            "esocial.ValidarXSD (validator.go:101) executa xmllint contra os XSD oficiais embutidos, mas nunca e chamada "
            "pela camada web. O handler POST /fila/{id}/validar usa ValidarEventoXSD (generator.go:424), que apenas faz "
            "strings.Contains de namespace e tags, e devolve a mensagem 'validados com sucesso contra os schemas XSD "
            "oficiais' (server.go:1435). O usuario recebe garantia de conformidade que nao foi executada."
        ),
        "impacto": "Falsa sensacao de conformidade; eventos estruturalmente invalidos marcados como validados (retrabalho e rejeicao no eSocial).",
        "fix": (
            "Chamar esocial.ValidarXSD no handler (com fallback documentado quando xmllint nao existe) e ajustar a "
            "mensagem ao que realmente foi verificado; opcionalmente embutir um validador Go puro para nao depender do xmllint."
        ),
        "cond": "Sempre ativo; a mensagem de sucesso e emitida mesmo sem xmllint instalado.",
    },
    {
        "id": "M-04",
        "cat": "Complementares (Credenciais)",
        "sev": "media",
        "local": "internal/web/server.go:214-216, 460-493",
        "titulo": "Forca bruta da senha do certificado A1 sem autenticacao e sem rate limit",
        "trecho": 'mux.HandleFunc("POST /configuracao/testar", s.handleTestarCertificado)   // server.go:216\ncert, err := crypto.CarregarA1Arquivo(cfg.CertificadoPath, senha)   // server.go:487',
        "porque": (
            "POST /configuracao/testar aceita a senha do .pfx e responde sucesso/erro de decodificacao, sem autenticacao, "
            "sem limite de tentativas e sem atraso. Com acesso de rede (achado C-01) e possivel testar senhas "
            "indefinidamente contra a chave privada ICP-Brasil armazenada no disco, e a resposta tambem revela titular, "
            "CNPJ e emissor do certificado."
        ),
        "impacto": "Comprometimento da chave privada da empresa (e-CNPJ) por dicionario; uso indevido do certificado.",
        "fix": (
            "Exigir autenticacao na rota, limitar tentativas por sessao/IP (ex.: 5 tentativas/minuto com backoff), "
            "registrar tentativas em log de auditoria e nao devolver detalhes do certificado antes da validacao da senha."
        ),
        "cond": "Exploravel por qualquer cliente com acesso a porta 8000.",
    },
    {
        "id": "B-01",
        "cat": "Segredos e Credenciais",
        "sev": "baixa",
        "local": "internal/web/server.go:413-416 + Dockerfile:16",
        "titulo": "Permissoes frouxas: PFX gravado com 0644 e diretorio de dados 777 na imagem Docker",
        "trecho": 'destFile, err := os.Create(caminhoDest)   // server.go:416 -> 0666 & umask = 0644\nRUN mkdir -p /app/dados && chmod 777 /app/dados   // Dockerfile:16',
        "porque": (
            "O arquivo .pfx (chave privada) e criado com as permissoes padrao do os.Create (0644 na umask habitual). O "
            "diretorio ./dados/certificados e criado com 0700, o que mitiga leitura por outros usuarios locais, mas o "
            "Dockerfile torna /app/dados (banco SQLite com CPFs e o proprio PFX) gravavel por qualquer usuario dentro do "
            "container, contrariando o principio do menor privilegio."
        ),
        "impacto": "Leitura/adulteracao do banco e do certificado por outro usuario local ou por qualquer processo no container.",
        "fix": (
            "Gravar o PFX com os.OpenFile(..., 0600) apos criar o diretorio 0700 (e corrigir permissoes se ja existir); "
            "trocar chmod 777 por 0700 no Dockerfile e rodar o container com usuario nao-root (USER 65534)."
        ),
        "cond": "Requer outro usuario/processo com acesso ao mesmo volume ou ao container.",
    },
    {
        "id": "B-02",
        "cat": "Inputs sem Tratamento (XSS)",
        "sev": "baixa",
        "local": "internal/web/server.go:1277-1284",
        "titulo": "ID do evento extraido do XML do usuario sem validacao de formato",
        "trecho": 'if strings.Contains(xmlConteudo, "Id=\\"") {\n  i := strings.Index(xmlConteudo, "Id=\\"") + 4\n  f := strings.Index(xmlConteudo[i:], "\\"")\n  if f > 0 { idEvento = xmlConteudo[i : i+f] }\n}',
        "porque": (
            "O identificador do evento passa a ser texto arbitrario do XML enviado pelo usuario e e usado como chave "
            "primaria, em URLs (/fila/{id}/xml) e no nome do arquivo de download (Content-Disposition). O html/template "
            "escapa o contexto HTML/URL e o net/http substitui CR/LF por espaco em cabecalhos (header.go:139,206), o que "
            "impede XSS e header injection; ainda assim nao ha validacao do padrao oficial ID1+tpInsc+nrInsc+timestamp+seq, "
            "o que quebra a integridade referencial do eSocial e permite IDs confusos/duplicados na interface."
        ),
        "impacto": "Dados inconsistentes no eSocial e possibilidade de confusao na operacao (baixo impacto de seguranca direto).",
        "fix": "Validar o ID com regex ^ID[0-9]{32}$ (ou regenerar via GerarIDEvento) antes de persistir.",
        "cond": "Exploravel por POST /eventos/salvar-generico.",
    },
    {
        "id": "B-03",
        "cat": "Complementares (Hardening)",
        "sev": "baixa",
        "local": "internal/web/server.go:198-257 (Rotas) - nenhum middleware de cabecalhos",
        "titulo": "Ausencia de cabecalhos de seguranca (CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy)",
        "trecho": 'mux := http.NewServeMux()\n// nenhum w.Header().Set("Content-Security-Policy"|"X-Frame-Options"|"X-Content-Type-Options") no projeto',
        "porque": (
            "Nenhuma resposta define cabecalhos de seguranca. Sem CSP, o impacto do achado A-03 e amplificado; sem "
            "X-Frame-Options/frame-ancestors o painel pode ser embutido em iframe (clickjacking) e sem "
            "X-Content-Type-Options o navegador pode farejar tipos de conteudo."
        ),
        "impacto": "Agravamento de XSS/clickjacking; defesa em profundidade ausente.",
        "fix": "Middleware unico aplicando CSP restritiva (default-src 'self'), X-Content-Type-Options: nosniff, X-Frame-Options: DENY e Referrer-Policy: no-referrer.",
        "cond": "Sempre presente (ausencia de controle).",
    },
    {
        "id": "B-04",
        "cat": "Complementares (Integridade)",
        "sev": "baixa",
        "local": "internal/web/server.go:660, 966, 1045, 1113, 1297, 1502",
        "titulo": "Erros de escrita/exclusao sao descartados e a interface informa sucesso",
        "trecho": 'id := r.PathValue("id")\n_ = s.db.ExcluirEvento(id)     // server.go:1502 (erro ignorado)\ns.handleFilaComFlash(w, r, fmt.Sprintf("Evento %s removido da fila.", id), false)',
        "porque": (
            "Diversos handlers ignoram o erro das operacoes de persistencia (SalvarEvento/SalvarColaborador/Excluir*) e "
            "sempre exibem mensagem de sucesso. Em caso de falha de disco ou violacao de constraint, o usuario acredita "
            "que o dado foi gravado/excluido. E o mesmo padrao em handleSalvarS2210 (966), handleSalvarS2220 (1045), "
            "handleImportarXMLASO (1113), handleSalvarEventoGenerico (1297) e handleExcluirColaborador (660)."
        ),
        "impacto": "Trilha de auditoria enganosa e perda silenciosa de dados.",
        "fix": "Tratar o erro retornado, exibir flash de erro e retornar HTTP 4xx/5xx coerente (404 quando o registro nao existe).",
        "cond": "Ocorre em qualquer falha de I/O ou de constraint.",
    },
    {
        "id": "B-05",
        "cat": "Inputs sem Tratamento (XSS)",
        "sev": "informativa",
        "local": "internal/web/server.go:97-99",
        "titulo": "Funcao de template safeHTML (template.HTML) definida e nao utilizada",
        "trecho": '"safeHTML": func(s string) template.HTML {\n  return template.HTML(s)\n},',
        "porque": (
            "A funcao registra um bypass do escaping automatico do html/template. Hoje ela nao e usada em nenhum template "
            "(grep por safeHTML em internal/web/templates nao retorna resultado), portanto nao ha vulnerabilidade ativa - "
            "mas a existencia dela torna trivial reintroduzir XSS em manutencoes futuras."
        ),
        "impacto": "Risco latente (footgun) de XSS por uso acidental.",
        "fix": "Remover a funcao safeHTML ou restringi-la a conteudo gerado internamente, com comentario de alerta.",
        "cond": "Nao exploravel hoje (codigo morto).",
    },
    {
        "id": "M-05",
        "cat": "Complementares (Conformidade)",
        "sev": "alta",
        "local": "internal/web/generator.go:97-181, 239-342, 381-421 + internal/data/xsd/*.xsd",
        "titulo": "XML gerado pelo editor nao e aderente ao XSD oficial S-1.3 (revelado ao ligar a validacao real)",
        "trecho": 'gerador emite <ideTrabalhador><cpfTrab>   // evtExpRisco.xsd:31 exige <ideVinculo type="T_ideVinculo_sst">\ngerador emite <agenteNoc>                // evtExpRisco.xsd exige <agNoc>\ngerador emite <ideOC>CREA</ideOC>        // XSD: enumeracao {1,4,9} (codigo do orgao, nao sigla)\n// xmllint --schema evtExpRisco.xsd: "element ideTrabalhador: This element is not expected. Expected is ( ideVinculo )"',
        "porque": (
            "Ate esta correcao o botao Validar apenas fazia strings.Contains e sempre aprovava, mascarando a divergencia. "
            "Ao ligar o validador real (esocial.ValidarXSD com xmllint + XSD oficiais embutidos), ficou demonstrado que os "
            "documentos produzidos pelo editor nao sao aderentes ao leiaute S-1.3: nomes de grupos diferentes "
            "(ideTrabalhador vs ideVinculo, agenteNoc vs agNoc), epi sem o subgrupo epiCompl e ideOC textual em vez de "
            "codigo. O mesmo vale para S-2210 (faltam codSitGeradora, iniciatCAT, lateralidade, indInternacao, durTrat, "
            "indAfast) e S-2220 (exMedOcup/aso/resAso)."
        ),
        "impacto": (
            "Conformidade: os eventos nao seriam aceitos pelo eSocial real. Com a correcao do achado M-03 o sistema passa a "
            "reprovar de forma explicita e rastreavel, em vez de aprovar falsamente. Severidade: ALTA (funcional/regulatorio)."
        ),
        "fix": (
            "Adequar os tres geradores ao XSD oficial embarcado (usar ideVinculo com matricula, agNoc, bloco epi/epiCompl, "
            "ideOC como codigo, e completar os grupos obrigatorios de CAT e ASO), com teste de regressao que valide cada "
            "evento gerado com xmllint contra o schema embutido."
        ),
        "cond": "Visivel em qualquer evento gerado pelo editor e submetido a conferencia por schema.",
    },
    {
        "id": "I-01",
        "cat": "Segredos e Credenciais",
        "sev": "informativa",
        "local": "internal/crypto/cert_test.go:66, 125",
        "titulo": "Senhas de teste embutidas no codigo (nenhum segredo de producao encontrado)",
        "trecho": 'senha := "senhaSegura123"     // cert_test.go:66\nsenha := "testeArquivo456"     // cert_test.go:125',
        "porque": (
            "Sao credenciais de fixtures de teste para PFX gerados em memoria, sem valor em producao. O restante da "
            "varredura por segredos (codigo, Dockerfile, workflow, .mcp.json, README e historico git completo) nao "
            "encontrou chaves de API, tokens, senhas, chaves privadas ou defaults do tipo ${VAR:-segredo}."
        ),
        "impacto": "Nenhum impacto pratico; registrado apenas como padrao a evitar.",
        "fix": "Manter, de preferencia gerando as senhas de fixture em tempo de teste (crypto/rand) para evitar ruido em scanners de segredo.",
        "cond": "N/A - informativo.",
    },
]

PONTOS_FORTES = [
    (
        "Escaping contextual do html/template em todos os templates (sem uso de template.HTML)",
        "internal/web/templates/*.html, internal/web/server.go:97-99",
        "Sonda executada com os padroes reais do projeto (onclick=\"f('{{ .Nome }}')\", href=\"/fila/{{ .ID }}/xml\", "
        "<textarea>{{ .XML }}</textarea>\"): payloads de quebra de string JS, de injecao de HTML e de path traversal "
        "foram neutralizados (\\u0027, \\u003c, %22%3e) e o XML apareceu como &lt;eSocial&gt;. Nenhum template usa "
        "template.HTML (a funcao safeHTML esta morta).",
    ),
    (
        "Nenhuma consulta SQL montada por concatenacao - 100% parametrizada",
        "internal/storage/storage.go:149-152, 236-239, 253-265, 279-290, 342-349, 369, 377, 386-394",
        "Todas as operacoes usam placeholders '?' via database/sql; grep por Sprintf/concatenacao em storage.go nao "
        "retorna nada. Nao ha superficie de SQL injection.",
    ),
    (
        "Sem segredos hardcoded e historico git limpo",
        "Dockerfile, .github/workflows/package.yml:35, .gitignore:11-22, .opencode/.mcp.json",
        "Varredura por api_key/secret/password/token/BEGIN PRIVATE KEY no codigo, configs, CI, docs e em todo o historico "
        "(git log --all -p) nao encontrou segredos. O CI usa ${{ secrets.CR_PAT || secrets.GITHUB_TOKEN }} e o .gitignore "
        "exclui *.pfx, *.p12, *.pem, *.key, *.db.",
    ),
    (
        "Upload de certificado imune a path traversal",
        "internal/web/server.go:412-414 + mime/multipart/multipart.go:101 (stdlib Go 1.26.8)",
        "O nome enviado passa por filepath.Base dentro do proprio stdlib (comprovado no fonte do Go instalado), portanto "
        "'../../x.pfx' nao escapa de ./dados/certificados. O diretorio e criado com 0700.",
    ),
    (
        "Sem command injection",
        "internal/esocial/validator.go:144-158",
        "O unico exec.Command do projeto e 'xmllint --noout --schema <xsd> <tmp>' com caminho XSD fixo (switch interno) e "
        "arquivo temporario gerado pelo servidor (os.CreateTemp); nenhum argumento vem do usuario.",
    ),
    (
        "XXE e expansao de entidades bloqueados pelo encoding/xml",
        "internal/web/server.go:1261, internal/web/server.go:1051-1067, internal/esocial/validator.go:110",
        "Sonda executada: payload com <!ENTITY xxe SYSTEM \"file:///etc/passwd\"> e payload de expansao de entidades "
        "customizadas retornaram 'invalid character entity' - o decoder Go nao resolve DTD/entidades externas.",
    ),
    (
        "Cliente SOAP com mTLS e TLS minimo 1.2, sem InsecureSkipVerify",
        "internal/soap/client.go:116-136",
        "tls.Config com Certificates do A1, MinVersion TLS 1.2, http.Client com timeout de 60s e transporte HTTP/1.1. "
        "Ressalva: o cliente nao esta conectado a nenhuma rota (ver achado A-02).",
    ),
    (
        "Frontend sem sinks perigosos de HTML/JS",
        "internal/web/templates/editor_s2240.html:261, evento_s2240.html:261, menu_eventos.html:90",
        "Os unicos usos de innerHTML sao limpeza de container (= '') e configuracao de swap do HTMX; nao ha eval, "
        "new Function, document.write nem markdown/HTML renderizado de usuario. Nao existe lib de sanitizacao porque nao "
        "ha caminho de renderizacao de HTML cru - os pontos de HTML manual sao o achado A-03.",
    ),
    (
        "Cabecalho de resposta imune a injecao CRLF",
        "net/http/header.go:139,206 (stdlib Go 1.26.8) + internal/web/server.go:1404",
        "O nome do arquivo em Content-Disposition e derivado do ID do evento, mas o net/http substitui \\r e \\n por "
        "espaco em valores de cabecalho, impedindo header injection mesmo com ID controlado pelo usuario (achado B-02).",
    ),
]


# ---------------------------------------------------------------------------
# CORRECOES APLICADAS (verificadas por build, testes e execucao real)
# ---------------------------------------------------------------------------

CORRECOES = [
    ("C-01",
     "Bind padrao em 127.0.0.1 (flag -host), autenticacao local obrigatoria (senha com hash SHA-256 iterado e salt, "
     "cookie HttpOnly + SameSite=Strict, sessao de 12h), middleware de auth em todas as rotas exceto /login e /static, "
     "allowlist de Host (anti DNS rebinding) e tela de login com logout.",
     "internal/web/auth.go (novo), internal/web/templates/login.html (novo), internal/web/server.go (Rotas + middleware), "
     "cmd/server/main.go (-host, -senha, ESOCIAL_SENHA); testes TestRotasExigemAutenticacao e TestHostNaoAutorizadoRejeitado"),
    ("A-01",
     "Token anti-CSRF por processo embutido em todos os formularios e no cabecalho X-CSRF-Token das requisicoes HTMX "
     "(hx-headers), validado em todo metodo de escrita junto com Origin/Referer.",
     "internal/web/auth.go (middlewareCSRF), 13 formularios em internal/web/templates/*.html; testes TestCSRFExigidoEmEscrita"),
    ("A-02",
     "Assinatura simulada marcada explicitamente (comentario + base64 legivel SIMULACAO-...) e sem alegacao de validade; "
     "transmissao passa a gravar status 'simulado' sem protocolo e sem recibo; consulta de recibo nao inventa numero; "
     "aviso permanente MODO SIMULACAO na interface; README corrigido.",
     "internal/web/generator.go (SimularAssinatura/MensagemTransmissaoSimulada), internal/web/server.go (assinar/transmitir/"
     "recibo/lotes), internal/web/templates/layout.html, README.md; testes TestAssinaturaSimuladaMarcadaENenhumReciboFabricado "
     "e TestEventosSSTCicloDeVidaFila"),
    ("A-03",
     "Saida do teste de certificado migrada de fmt.Fprintf para html/template (escaping contextual), com partial dedicado; "
     "remocao da funcao safeHTML.",
     "internal/web/server.go (renderTesteCertificado + DadosTesteCertificado), "
     "internal/web/templates/partials/teste_certificado.html (novo); teste TestXSSNomeArquivoCertificadoEscapado"),
    ("M-01",
     "Todos os campos interpolados no XML passam por escapeXML ou validacao de dominio (datas ISO, hora HHMM, tpAval, "
     "utilizEPC/EPI, UF oficial, tpAcid, tpExameOcup).",
     "internal/web/generator.go (validarDataISO, validarHora, validarDominio, validarUF); teste TestGeradorXMLEscapaCamposMaliciosos"),
    ("M-02",
     "http.MaxBytesReader + limite por tipo de arquivo (2 MB PFX, 5 MB XML, 10 MB CSV), leitura com io.LimitReader e "
     "mensagem explicita ao usuario quando o limite e excedido.",
     "internal/web/server.go (constantes limiteUpload* e handlers de upload), internal/web/auth.go (limite global de 12 MB "
     "para escritas); teste TestUploadAcimaDoLimite"),
    ("M-03",
     "Validacao por schema oficial (esocial.ValidarXSD + xmllint + XSD embutidos) ligada ao handler, com fallback declarado "
     "apenas para tipos sem schema e mensagem que informa o modo executado.",
     "internal/web/server.go (handleValidarEvento), internal/esocial/validator.go (XmllintDisponivel); teste TestValidacaoUsaSchemaOficial"),
    ("M-04",
     "Rate limit por IP no login (8/min) e na validacao da senha do certificado (6/min), com resposta 429/Retry-After e "
     "registro em log de auditoria.",
     "internal/web/auth.go (limitador, permitirTesteCertificado); teste TestRotasExigemAutenticacao (rota protegida)"),
    ("B-01",
     "PFX gravado com 0600 e diretorio de certificados corrigido para 0700; banco SQLite e seu diretorio com 0700/0600; "
     "imagem Docker com chmod 700 e execucao como usuario nao-root (UID 10001).",
     "internal/web/server.go (upload), internal/storage/storage.go (Abrir), Dockerfile (USER 10001 + chmod 700); testes "
     "TestPermissaoDoArquivoDeCertificado e verificacao em execucao real (ls -l dados/)"),
    ("B-02",
     "ID do evento validado contra o padrao oficial ID+tpInsc+nrInsc(14)+timestamp(14)+seq(5); ID invalido e substituido por "
     "um ID gerado, com aviso ao operador.",
     "internal/web/server.go (idEventoValido, extrairIDEvento); teste TestIDEventoInvalidoSubstituido"),
    ("B-03",
     "Middleware de cabecalhos de seguranca: CSP (default-src 'self', frame-ancestors 'none', object-src 'none', "
     "base-uri 'none', form-action 'self'), X-Content-Type-Options, X-Frame-Options, Referrer-Policy e COOP.",
     "internal/web/auth.go (middlewareCabecalhos); teste TestCabecalhosDeSeguranca + verificacao com curl no binario real"),
    ("B-04",
     "Erros de persistencia deixaram de ser descartados: exclusoes verificam existencia e falha de gravacao/exclusao gera "
     "flash de erro em vez de mensagem de sucesso.",
     "internal/web/server.go (handleExcluirEvento, handleExcluirColaborador, handleSalvarS2210/S2220, handleImportarXMLASO, "
     "handleSalvarEventoGenerico); teste TestExclusaoInexistenteInformaErro"),
    ("B-05",
     "Funcao safeHTML (template.HTML) removida do FuncMap.",
     "internal/web/server.go (carregarTemplates); teste TestSemFuncaoSafeHTML"),
    ("I-01",
     "Senhas de fixture dos testes geradas em tempo de execucao com crypto/rand (nenhum literal de credencial no codigo).",
     "internal/crypto/cert_test.go (senhaAleatoriaTeste); go test ./internal/crypto"),
]

# ---------------------------------------------------------------------------
# ISSUES PARA O GITHUB
# ---------------------------------------------------------------------------

ISSUES = [
    {
        "titulo": "[Seguranca] Adicionar autenticacao/autorizacao e restringir o bind de rede (0.0.0.0:8000)",
        "labels": "security, critical, area: backend",
        "corpo": """## Descricao

O servidor web nao possui nenhuma camada de autenticacao ou autorizacao e escuta em todas as interfaces de rede.

- `cmd/server/main.go:55` -> `Addr: fmt.Sprintf(":%d", *porta)` (bind 0.0.0.0, nao 127.0.0.1)
- `internal/web/server.go:198-257` -> `Rotas()` registra 30+ handlers sem qualquer middleware de sessao/login/token

Sonda contra as rotas reais (sem nenhuma credencial):

```
GET  /fila                     -> 200
GET  /colaboradores            -> 200
GET  /configuracao             -> 200
DELETE /colaboradores/{id}     -> 200
POST /fila/lote/transmitir     -> 200
```

## Por que e exploravel

Qualquer host que alcance a porta 8000 (LAN, Wi-Fi publico, container com `-p 8000:8000`) pode ler a lista de
colaboradores com CPF, baixar os XMLs dos eventos, alterar razao social/CNPJ/ambiente, substituir o certificado
digital, disparar assinatura/transmissao em lote e apagar registros. Nao ha nenhum controle de acesso a contornar.

## Impacto

Exposicao de dados pessoais (LGPD) de toda a folha, manipulacao de documentos fiscais e destruicao de dados.
Severidade: CRITICA.

## Sugestao de correcao

1. Bind padrao em `127.0.0.1`, com flag explicita (`-expor`) para outros enderecos.
2. Autenticacao local (sessao com cookie `SameSite=Strict` + `HttpOnly`, ou token de dispositivo) e middleware
   aplicado a todas as rotas exceto `/static/`.
3. Validacao do cabecalho `Host` com allowlist (`localhost`, `127.0.0.1`) para mitigar DNS rebinding.
4. Docker: documentar o risco e exigir auth quando a porta for publicada.

## Criterios de aceite

- [ ] `curl -i http://127.0.0.1:8000/fila` sem credencial retorna 401/302 para login
- [ ] Nenhuma rota de escrita responde 200 sem sessao valida
- [ ] `Addr` default e `127.0.0.1:8000` (teste automatizado cobrindo o default)
- [ ] Requisicao com `Host: evil.test` e rejeitada
- [ ] README documenta o modo de exposicao e a necessidade de auth
""",
    },
    {
        "titulo": "[Seguranca] Proteger endpoints de escrita contra CSRF (token + validacao de Origin)",
        "labels": "security, high, area: backend",
        "corpo": """## Descricao

Nenhum formulario possui token anti-CSRF e nenhum handler valida `Origin`/`Referer`.

- `internal/web/server.go:213-251` -> rotas `POST /configuracao`, `POST /colaboradores`, `POST /eventos/*`,
  `POST /fila/{id}/assinar`, `POST /fila/{id}/transmitir`, `POST /fila/lote/assinar`, `POST /fila/lote/transmitir`
- `grep -rni "csrf" .` -> nenhum resultado no repositorio

## Por que e exploravel

POST de formulario e requisicao simples (sem preflight CORS). Uma pagina maliciosa visitada pelo operador pode
enviar formularios para `http://localhost:8000` e executar acoes de escrita: trocar ambiente para Producao Oficial,
alterar CNPJ, cadastrar colaborador, assinar/transmitir lotes inteiros.

## Impacto

Integridade da configuracao e dos dados fiscais; acoes privilegiadas disparadas pelo navegador da vitima.
Severidade: ALTA.

## Sugestao de correcao

- Token sincronizado por formulario (campo `hidden`) validado no servidor para todo metodo nao-idempotente; ou
- Exigir cabecalho customizado (`X-CSRF-Token`) e validar `Origin`/`Referer` contra o host da aplicacao.
- Com sessao, usar cookie `SameSite=Strict`.

## Criterios de aceite

- [ ] POST sem token/cabecalho valido retorna 403
- [ ] Todos os formularios em `internal/web/templates/` incluem o campo de token
- [ ] POST com `Origin: https://evil.test` e rejeitado (teste automatizado)
- [ ] Teste de regressao cobrindo `POST /configuracao` e `POST /fila/lote/transmitir`
""",
    },
    {
        "titulo": "[Seguranca] Remover assinatura e recibos simulados ou rotula-los explicitamente",
        "labels": "security, high, integrity, area: core",
        "corpo": """## Descricao

A aplicacao grava assinatura XMLDSig falsa, inventa protocolo/recibo e marca o evento como `aceito`, apresentando
isso como retorno oficial do governo.

- `internal/web/generator.go:503` -> `<SignatureValue>SIMULATED_RSA_SHA256_SIGNATURE_...`
- `internal/web/generator.go:516-533` -> `SimularTransmissao` gera protocolo/recibo numericos e a mensagem
  "Recibo emitido pelo Serpro/Receita Federal"
- `internal/web/server.go:1491` -> "Recibo oficial do eSocial ja confirmado: %s"
- `internal/soap/client.go` (mTLS real) nao e referenciado por nenhuma rota - codigo morto
- `README.md:14,65-67` afirma assinatura digital e transmissao pelos webservices oficiais

## Por que e exploravel

Nao e um bypass de permissao: e uma funcionalidade que produz prova documental inexistente. O operador acredita ter
cumprido obrigacao acessoria do eSocial quando nada foi assinado nem transmitido.

## Impacto

Trilha de auditoria fiscal falsa, risco regulatorio/legal e assinatura sem validade juridica. Severidade: ALTA.

## Sugestao de correcao

1. Introduzir status explicito de simulacao (`simulado`) e nao preencher `recibo`/`protocolo` nesse modo.
2. Rotular na interface ("modo demonstracao - sem transmissao real").
3. Conectar `soap.ClienteSOAP` (mTLS + A1) aos handlers, persistindo o XML de resposta oficial.
4. Corrigir o README para refletir o estado real da funcionalidade.

## Criterios de aceite

- [ ] Evento sem transmissao real nunca recebe status `aceito` nem campo `recibo` preenchido
- [ ] Interface exibe aviso claro no modo simulacao
- [ ] Fluxo real de envio/consulta usa `internal/soap` com certificado A1 (teste de integracao com mock)
- [ ] README sem afirmacoes de transmissao nao implementada
""",
    },
    {
        "titulo": "[Seguranca] Corrigir HTML injection/XSS armazenado em POST /configuracao/testar",
        "labels": "security, high, xss, area: frontend",
        "corpo": """## Descricao

O handler responde HTML montado por `fmt.Fprintf`, interpolando dados controlados pelo usuario sem escape.

- `internal/web/server.go:472-474` -> `filepath.Base(cfg.CertificadoPath)` (nome do .pfx enviado no upload, persistido no SQLite)
- `internal/web/server.go:489-492` -> `err.Error()` de decodificacao do PKCS#12
- `internal/web/server.go:518-539` -> titular, CNPJ, emissor e datas vindos do certificado carregado

Evidencia de sonda (httptest contra a rota real, `CertificadoPath = "<img src=x onerror=alert(1)>.pfx"`):

```
STATUS=200
BODY=... O arquivo <img src=x onerror=alert(1)>.pfx nao foi localizado no disco local.
PROBE_RESULT=HTML_INJECTION_CONFIRMED
```

## Por que e exploravel

O valor persiste no banco e e refletido em cada visita a pagina de certificado: e XSS armazenado na origem do painel
(localhost:8000). Combinado com o achado de CSRF, o payload pode ser plantado por uma pagina externa.

## Impacto

Execucao de JavaScript na origem do painel, com leitura de todo o conteudo e disparo de acoes autenticadas pelo
navegador do operador. Severidade: ALTA.

## Sugestao de correcao

- Substituir todo `fmt.Fprintf` de HTML por `template.ExecuteTemplate` (escaping contextual) ou aplicar
  `template.HTMLEscapeString` em cada interpolacao.
- Remover a funcao `safeHTML` (`internal/web/server.go:97-99`), hoje morta, para eliminar o bypass latente.

## Criterios de aceite

- [ ] Teste automatizado: nome de arquivo `<img src=x onerror=alert(1)>.pfx` nao produz tag HTML na resposta
- [ ] Nenhum `fmt.Fprintf(w, "<...")` restante em `internal/web/`
- [ ] Funcao `safeHTML` removida
- [ ] Teste cobrindo tambem CN/titular de certificado malicioso
""",
    },
    {
        "titulo": "[Seguranca] Escapar todos os campos nos geradores de XML (XML injection)",
        "labels": "security, medium, area: core",
        "corpo": """## Descricao

Campos de formulario sao interpolados no XML sem `escapeXML`, permitindo injetar elementos arbitrarios no documento
que depois recebe assinatura/transmissao.

- `internal/web/generator.go:118` -> `<codAgNoc>%s</codAgNoc>` com `p.CodigoRisco`
- `internal/web/generator.go:120` -> `<tpAval>%s</tpAval>`
- `internal/web/generator.go:129,135,140` -> `utilizEPC`, `eficEpc`, `utilizEPI`
- `internal/web/generator.go:166,176` -> `ideOC`, `ufOC`
- `internal/web/generator.go:255,257,307,398,401` -> datas e horarios
- `internal/web/generator.go:334,414` -> `ufOC`/`ufCRM` do medico

## Por que e exploravel

Um valor como `</codAgNoc><codAgNoc>X` injeta estrutura nova no XML. Se o conteudo injetado for valido para o
schema, ele passa a fazer parte do documento assinado.

## Impacto

Adulteracao de documento fiscal antes da assinatura; assinatura de conteudo divergente do formulario exibido.
Severidade: MEDIA.

## Sugestao de correcao

- Aplicar `escapeXML` em todos os campos (ou construir o XML com `encoding/xml` a partir de structs).
- Validar dominios: datas ISO, `tpAval` em {1,2}, `utilizEPC/EPI` em {0,1,2}, UF em lista oficial.

## Criterios de aceite

- [ ] Teste de regressao com payload `</tag><tag>X` em cada campo de texto nao altera a estrutura do XML
- [ ] Todos os `Fprintf(&sb, ...)` de valor usam escape ou dominio validado
- [ ] Validacao XSD real (issue propria) ligada ao fluxo de gravacao
""",
    },
    {
        "titulo": "[Seguranca] Aplicar limites de tamanho em uploads e corpos de requisicao",
        "labels": "security, medium, dos, area: backend",
        "corpo": """## Descricao

Nao ha limite de tamanho em uploads nem em leitura de corpo.

- `internal/web/server.go:418` -> `io.Copy(destFile, arquivo)` (PFX, sem limite)
- `internal/web/server.go:1067` -> `io.ReadAll(arquivo)` (XML do ASO, sem limite)
- `internal/web/server.go:705` -> `leitor.ReadAll()` (CSV, sem limite)
- `grep -rn "MaxBytesReader\\|LimitReader" .` -> nenhum resultado

## Por que e exploravel

Qualquer cliente que alcance a porta (ver issue de autenticacao) pode enviar um arquivo arbitrariamente grande e
esgotar disco ou memoria do processo.

## Impacto

Negacao de servico local (OOM/disco cheio). Severidade: MEDIA.

## Sugestao de correcao

- `http.MaxBytesReader` nos handlers de upload (2 MB PFX, 5 MB XML, 10 MB CSV).
- `r.ParseMultipartForm(limite)` e validacao de `Content-Length` antes de `io.Copy`/`ReadAll`.
- Retornar 413 com mensagem clara ao usuario.

## Criterios de aceite

- [ ] Upload de arquivo acima do limite retorna 413
- [ ] Testes automatizados cobrindo PFX, XML e CSV
- [ ] Nenhum `io.Copy`/`io.ReadAll` de corpo HTTP sem leitor limitado
""",
    },
    {
        "titulo": "[Seguranca] Conectar a validacao XSD real (xmllint) ao handler de validacao",
        "labels": "security, medium, correctness, area: backend",
        "corpo": """## Descricao

O validador XSD oficial existe no projeto, mas o handler usa apenas checagem por substring e afirma validacao oficial.

- `internal/web/server.go:1433` -> `ValidarEventoXSD(evento.Tipo, evento.XMLGerado)`
- `internal/web/generator.go:424-482` -> apenas `strings.Contains` de namespace/tags
- `internal/web/server.go:1435` -> mensagem "validados com sucesso contra os schemas XSD oficiais"
- `internal/esocial/validator.go:101` -> `ValidarXSD` (xmllint + XSD S-1.3) usado SOMENTE em `esocial_test.go`

## Por que e exploravel

Nao ha barreira tecnica: a funcionalidade simplesmente nao foi ligada. O usuario recebe garantia de conformidade que
nao foi executada e eventos invalidos seguem para assinatura/transmissao.

## Impacto

Falsa sensacao de conformidade, retrabalho e rejeicao de eventos no eSocial. Severidade: MEDIA.

## Sugestao de correcao

- Chamar `esocial.ValidarXSD` no `handleValidarEvento` e ajustar a mensagem ao que foi efetivamente verificado.
- Documentar o fallback quando `xmllint` nao estiver instalado (ou embutir validador Go puro).

## Criterios de aceite

- [ ] POST /fila/{id}/validar executa a validacao por schema (xmllint) quando disponivel
- [ ] Mensagem de sucesso so aparece apos validacao real; caso contrario informa modo degradado
- [ ] Teste automatizado com XML invalido (schema) retornando rejeicao
""",
    },
    {
        "titulo": "[Seguranca] Limitar tentativas de senha do certificado A1 (anti forca bruta)",
        "labels": "security, medium, area: backend",
        "corpo": """## Descricao

`POST /configuracao/testar` aceita a senha do .pfx sem autenticacao, sem rate limit e sem atraso, e devolve dados do
certificado.

- `internal/web/server.go:216` -> rota registrada sem middleware
- `internal/web/server.go:487` -> `crypto.CarregarA1Arquivo(cfg.CertificadoPath, senha)`

## Por que e exploravel

Com acesso de rede (ver issue de autenticacao), e possivel testar senhas indefinidamente contra a chave privada
ICP-Brasil armazenada no disco; a resposta revela titular, CNPJ e emissor.

## Impacto

Comprometimento da chave privada da empresa (e-CNPJ) por dicionario. Severidade: MEDIA.

## Sugestao de correcao

- Exigir autenticacao na rota e limitar tentativas por sessao/IP (ex.: 5/min com backoff).
- Registrar tentativas em log de auditoria.
- Nao expor dados do certificado antes da validacao da senha.

## Criterios de aceite

- [ ] Apos N tentativas invalidas a rota retorna 429 com backoff
- [ ] Tentativas registradas em log com IP e horario
- [ ] Rota exige sessao autenticada
""",
    },
    {
        "titulo": "[Seguranca/Conformidade] Adequar os geradores de eventos ao XSD oficial S-1.3",
        "labels": "security, high, correctness, area: core",
        "corpo": """## Descricao

Com a validacao por schema agora ativa (issue M-03), ficou demonstrado que os XML gerados pelo editor nao sao aderentes
aos XSD oficiais embarcados.

- `internal/web/generator.go:99-101` -> `<ideTrabalhador><cpfTrab>` enquanto `internal/data/xsd/evtExpRisco.xsd:31`
  exige `<ideVinculo type="T_ideVinculo_sst">` (cpfTrab + matricula)
- `internal/web/generator.go:117-158` -> `<agenteNoc>` enquanto o XSD exige `<agNoc>`
- `internal/web/generator.go:166` -> `<ideOC>CREA</ideOC>` enquanto o XSD restringe a enumeracao {1, 4, 9}
- bloco `epi` sem o subgrupo obrigatorio `epiCompl`
- S-2210: faltam `codSitGeradora`, `iniciatCAT`, `lateralidade`, `indInternacao`, `durTrat`, `indAfast`
- S-2220: estrutura `exMedOcup`/`aso`/`resAso` divergente

Evidencia (`xmllint --noout --schema internal/data/xsd/evtExpRisco.xsd evento.xml`):

```
element ideTrabalhador: This element is not expected. Expected is ( ideVinculo ).
```

## Por que e exploravel

Ate a correcao do achado M-03 o botao Validar aprovava qualquer documento (apenas `strings.Contains`), o que mascarava
a divergencia e levava o operador a assinar um XML que o eSocial rejeitaria.

## Impacto

Eventos rejeitados no envio real e falsa sensacao de conformidade. Severidade: ALTA (funcional/regulatorio).

## Sugestao de correcao

1. Adequar os tres geradores (`GerarXMLS2240`, `GerarXMLS2210`, `GerarXMLS2220`) ao XSD embarcado, com os grupos e
   vocabularios obrigatorios.
2. Adicionar teste de regressao que valide o XML de cada gerador com `xmllint --schema` (skip quando o binario nao existir).
3. Manter a validacao por schema no fluxo (issue M-03) para evitar regressao silenciosa.

## Criterios de aceite

- [ ] `xmllint --schema internal/data/xsd/evtExpRisco.xsd` aprova a saida de `GerarXMLS2240`
- [ ] `xmllint --schema internal/data/xsd/evtCAT.xsd` aprova a saida de `GerarXMLS2210`
- [ ] `xmllint --schema internal/data/xsd/evtMonit.xsd` aprova a saida de `GerarXMLS2220`
- [ ] Teste automatizado cobre os tres geradores (com skip documentado se `xmllint` nao estiver instalado)
""",
    },
    {
        "titulo": "[Seguranca] Endurecer permissoes de arquivos, dados de teste e imagem Docker",
        "labels": "security, low, hardening",
        "corpo": """## Descricao

Achados de baixa severidade agrupados no mesmo tema (arquivos/credenciais):

- `internal/web/server.go:416` -> `os.Create` grava o .pfx com 0644 (chave privada legivel por outros usuarios locais)
- `Dockerfile:16` -> `chmod 777 /app/dados` (banco SQLite com CPFs e o proprio PFX gravaveis por qualquer processo)
- `internal/crypto/cert_test.go:66,125` -> senhas de teste literais ("senhaSegura123", "testeArquivo456")

Observacao: a varredura completa por segredos (codigo, CI, Dockerfile, docs, `.mcp.json` e historico git) nao
encontrou nenhum segredo de producao ou default inseguro do tipo `${VAR:-segredo}`.

## Impacto

Leitura/adulteracao do banco e do certificado por outro usuario local ou processo no container; ruido em scanners
de segredo. Severidade: BAIXA.

## Sugestao de correcao

- Gravar o PFX com `os.OpenFile(..., 0600)` e corrigir permissoes de diretorio existente.
- Trocar `chmod 777` por `0700` e rodar o container com `USER` nao-root.
- Gerar senhas de fixture em tempo de teste (crypto/rand).

## Criterios de aceite

- [ ] Arquivo .pfx criado com 0600 (teste verificando `os.Stat().Mode()`)
- [ ] Dockerfile sem `chmod 777` e com usuario nao-root
- [ ] Nenhuma senha literal em `internal/crypto/*_test.go`
""",
    },
    {
        "titulo": "[Seguranca] Adicionar cabecalhos de seguranca, validar ID de evento e tratar erros de persistencia",
        "labels": "security, low, hardening, area: backend",
        "corpo": """## Descricao

Achados de baixa severidade agrupados:

1. Sem cabecalhos de seguranca em nenhuma resposta (`internal/web/server.go:198-257`): falta CSP,
   `X-Content-Type-Options`, `X-Frame-Options` e `Referrer-Policy`.
2. ID de evento vem do XML do usuario sem validacao (`internal/web/server.go:1277-1284`), virando chave primaria,
   segmento de URL e nome de arquivo de download. O html/template escapa os contextos HTML/URL e o net/http substitui
   CR/LF em cabecalhos (`net/http/header.go:139,206`), entao nao ha XSS/header injection, mas o padrao oficial
   `ID1+tpInsc+nrInsc+timestamp+seq` nao e respeitado.
3. Erros de persistencia sao descartados com flash de sucesso (`internal/web/server.go:660, 966, 1045, 1113, 1297, 1502`).

## Impacto

Defesa em profundidade ausente, dados inconsistentes e trilha de auditoria enganosa. Severidade: BAIXA.

## Sugestao de correcao

- Middleware aplicando CSP `default-src 'self'`, `nosniff`, `DENY` e `no-referrer`.
- Validar o ID com `^ID[0-9]{32}$` (ou sempre usar `GerarIDEvento`).
- Tratar os erros retornados, retornando 404/500 coerentes e flash de erro.

## Criterios de aceite

- [ ] Todas as respostas HTML incluem CSP e demais cabecalhos (teste automatizado)
- [ ] ID fora do padrao e rejeitado com 400
- [ ] Nenhum `_ = s.db.` restante em `internal/web/server.go`
- [ ] Falha simulada de persistencia exibe mensagem de erro ao usuario
""",
    },
]


# ---------------------------------------------------------------------------
# GRAFICOS
# ---------------------------------------------------------------------------

def gerar_graficos():
    os.makedirs(GRAF_DIR, exist_ok=True)

    sev_ordem = ["critica", "alta", "media", "baixa", "informativa"]
    contagem = {k: 0 for k in sev_ordem}
    for a in ACHADOS:
        contagem[a["sev"]] += 1

    # Rosca por severidade
    labels, valores, cores = [], [], []
    for k in sev_ordem:
        if contagem[k]:
            labels.append("%s (%d)" % (ROTULO_SEV[k].capitalize(), contagem[k]))
            valores.append(contagem[k])
            cores.append(COR[k].hexval()[2:])

    fig, ax = plt.subplots(figsize=(5.4, 3.6), dpi=200)
    wedges, _, autotexts = ax.pie(
        valores,
        labels=None,
        colors=["#" + c for c in cores],
        autopct=lambda p: "%d" % round(p * sum(valores) / 100.0),
        startangle=90,
        counterclock=False,
        wedgeprops=dict(width=0.42, edgecolor="white", linewidth=1.5),
        pctdistance=0.79,
        textprops=dict(color="white", fontsize=9, weight="bold"),
    )
    ax.legend(
        wedges,
        labels,
        loc="center left",
        bbox_to_anchor=(0.98, 0.5),
        frameon=False,
        fontsize=8.5,
    )
    ax.text(0, 0.08, str(sum(valores)), ha="center", va="center", fontsize=22, weight="bold", color="#0F172A")
    ax.text(0, -0.16, "achados", ha="center", va="center", fontsize=9, color="#475569")
    ax.set_aspect("equal")
    fig.tight_layout()
    fig.savefig(os.path.join(GRAF_DIR, "severidade.png"), transparent=False, facecolor="white")
    plt.close(fig)

    # Barras por categoria
    cats = []
    for a in ACHADOS:
        if a["cat"] not in cats:
            cats.append(a["cat"])
    mapa_cat = {c: {k: 0 for k in sev_ordem} for c in cats}
    for a in ACHADOS:
        mapa_cat[a["cat"]][a["sev"]] += 1

    fig, ax = plt.subplots(figsize=(7.0, 4.0), dpi=200)
    altura = 0.62
    pos = list(range(len(cats)))
    restante = [0] * len(cats)
    for k in sev_ordem:
        vals = [mapa_cat[c][k] for c in cats]
        if sum(vals) == 0:
            continue
        ax.barh(pos, vals, left=restante, height=altura, color="#" + COR[k].hexval()[2:],
                label=ROTULO_SEV[k].capitalize(), edgecolor="white", linewidth=0.8)
        for i, v in enumerate(vals):
            if v:
                ax.text(restante[i] + v / 2.0, pos[i], str(v), ha="center", va="center", color="white",
                        fontsize=8.5, weight="bold")
        restante = [r + v for r, v in zip(restante, vals)]

    curtas = {
        "Isolamento e Controle de Acesso": "Isolamento /\nControle de Acesso",
        "Inputs sem Tratamento (XSS)": "Inputs sem\nTratamento (XSS)",
        "Segredos e Credenciais": "Segredos e\nCredenciais",
        "Complementares (Integridade)": "Complementares\n(Integridade)",
        "Complementares (Validacao)": "Complementares\n(Validacao)",
        "Complementares (Credenciais)": "Complementares\n(Credenciais)",
        "Complementares (Hardening)": "Complementares\n(Hardening)",
    }
    ax.set_yticks(pos)
    ax.set_yticklabels([curtas.get(c, c) for c in cats], fontsize=7.6)
    ax.invert_yaxis()
    ax.set_xlabel("Quantidade de achados", fontsize=9)
    ax.tick_params(axis="x", labelsize=8)
    ax.set_xlim(0, max(1, max(restante)) + 0.6)
    for s in ("top", "right"):
        ax.spines[s].set_visible(False)
    ax.grid(axis="x", linestyle=":", alpha=0.35)
    ax.set_axisbelow(True)
    ax.legend(fontsize=8, frameon=False, ncol=5, loc="upper center", bbox_to_anchor=(0.5, -0.18))
    fig.tight_layout()
    fig.savefig(os.path.join(GRAF_DIR, "categorias.png"), transparent=False, facecolor="white")
    plt.close(fig)


# ---------------------------------------------------------------------------
# PDF
# ---------------------------------------------------------------------------

def estilos():
    ss = getSampleStyleSheet()
    return {
        "h1": ParagraphStyle("h1", parent=ss["Heading1"], fontName="Helvetica-Bold", fontSize=17,
                             textColor=COR["navy"], spaceBefore=6, spaceAfter=10, leading=21),
        "h2": ParagraphStyle("h2", parent=ss["Heading2"], fontName="Helvetica-Bold", fontSize=12.5,
                             textColor=COR["navy"], spaceBefore=12, spaceAfter=6, leading=15),
        "h3": ParagraphStyle("h3", parent=ss["Heading3"], fontName="Helvetica-Bold", fontSize=10.5,
                             textColor=COR["texto"], spaceBefore=9, spaceAfter=4, leading=13),
        "corpo": ParagraphStyle("corpo", parent=ss["BodyText"], fontName="Helvetica", fontSize=9.3,
                                leading=13.2, alignment=TA_JUSTIFY, textColor=COR["texto"], spaceAfter=5),
        "bullet": ParagraphStyle("bullet", parent=ss["BodyText"], fontName="Helvetica", fontSize=9.2,
                                 leading=12.8, alignment=TA_LEFT, textColor=COR["texto"],
                                 leftIndent=12, bulletIndent=2, spaceAfter=3),
        "cap": ParagraphStyle("cap", parent=ss["BodyText"], fontName="Helvetica", fontSize=9.2,
                              leading=13, alignment=TA_CENTER, textColor=COR["texto"]),
        "mono": ParagraphStyle("mono", parent=ss["BodyText"], fontName="Courier", fontSize=6.9,
                               leading=8.4, textColor=colors.HexColor("#0B1220")),
        "cel": ParagraphStyle("cel", parent=ss["BodyText"], fontName="Helvetica", fontSize=7.4,
                              leading=9.4, textColor=COR["texto"]),
        "cel_b": ParagraphStyle("cel_b", parent=ss["BodyText"], fontName="Courier-Bold", fontSize=6.3,
                                leading=8.0, textColor=COR["texto"]),
        "chip": ParagraphStyle("chip", parent=ss["BodyText"], fontName="Helvetica-Bold", fontSize=7,
                               leading=8.6, alignment=TA_CENTER, textColor=colors.white),
        "chip_sm": ParagraphStyle("chip_sm", parent=ss["BodyText"], fontName="Helvetica-Bold", fontSize=6.2,
                                  leading=7.6, alignment=TA_CENTER, textColor=colors.white),
        "tbl_head": ParagraphStyle("tbl_head", parent=ss["BodyText"], fontName="Helvetica-Bold", fontSize=7.6,
                                   leading=9.4, textColor=colors.white),
        "legenda": ParagraphStyle("legenda", parent=ss["BodyText"], fontName="Helvetica-Oblique", fontSize=8,
                                  leading=10.5, textColor=colors.HexColor("#475569")),
    }


def rodape(canvas, doc):
    canvas.saveState()
    largura, altura = A4
    canvas.setStrokeColor(COR["cinza"])
    canvas.setLineWidth(0.6)
    canvas.line(2 * cm, altura - 1.35 * cm, largura - 2 * cm, altura - 1.35 * cm)
    canvas.setFont("Helvetica", 7.4)
    canvas.setFillColor(colors.HexColor("#64748B"))
    canvas.drawString(2 * cm, altura - 1.15 * cm, "Relatorio de Auditoria de Seguranca - %s %s" % (PROJETO, VERSAO))
    canvas.drawRightString(largura - 2 * cm, altura - 1.15 * cm, DATA_RELATORIO)
    canvas.line(2 * cm, 1.35 * cm, largura - 2 * cm, 1.35 * cm)
    canvas.drawString(2 * cm, 1.05 * cm, "Uso interno / correcao de vulnerabilidades")
    canvas.drawRightString(largura - 2 * cm, 1.05 * cm, "Pagina %d" % canvas.getPageNumber())
    canvas.restoreState()


def capa(E, total):
    E.append(Spacer(1, 3.2 * cm))
    E.append(Paragraph("RELATORIO DE AUDITORIA DE SEGURANCA", ParagraphStyle(
        "tag", fontName="Helvetica-Bold", fontSize=10, leading=13, alignment=TA_CENTER,
        textColor=COR["critica"])))
    E.append(Spacer(1, 0.5 * cm))
    E.append(Paragraph("Relatorio de Auditoria de Seguranca<br/>%s" % PROJETO, ParagraphStyle(
        "titulo", fontName="Helvetica-Bold", fontSize=25, leading=30, alignment=TA_CENTER,
        textColor=COR["navy"])))
    E.append(Spacer(1, 0.35 * cm))
    E.append(Paragraph("Versao corrigida: %s &nbsp;|&nbsp; Data do relatorio: %s" % (VERSAO, DATA_RELATORIO),
                       ParagraphStyle("sub", fontName="Helvetica", fontSize=10.5, leading=14,
                                      alignment=TA_CENTER, textColor=colors.HexColor("#475569"))))
    E.append(Spacer(1, 1.2 * cm))

    dados = [
        ["Repositorio", "github.com/forg3/validador-esocial (modulo esocial-emissor-livre) - release v1.1-alpha"],
        ["Escopo auditado",
         "Codigo Go (cmd/, internal/), 21 templates HTML, JavaScript embutido, CSS, Dockerfile, workflow GitHub "
         "Actions, go.mod/go.sum, .gitignore, README e demais documentos, alem do historico git completo "
         "(git log --all -p)."],
        ["Volume analisado",
         "5.572 linhas de Go, 21 templates HTML, 1 workflow de CI, 1 Dockerfile, 2 arquivos de configuracao."],
        ["Metodo",
         "Revisao manual linha a linha de todos os handlers de rota, camada de persistencia, geradores de XML, "
         "camada criptografica e templates; varredura automatizada por padroes de segredo e sinks de XSS; sondas "
         "executadas com httptest contra as rotas reais e com o runtime Go 1.26.8 instalado."],
        ["Total de achados", "%d (1 critica, 4 altas, 4 medias, 5 baixas, 1 informativa) - TODOS CORRIGIDOS" % total],
    ]
    t = Table([[Paragraph("<b>%s</b>" % k, E_est["cel"]), Paragraph(v, E_est["cel"])] for k, v in dados],
              colWidths=[3.4 * cm, 13.6 * cm])
    t.setStyle(TableStyle([
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("BACKGROUND", (0, 0), (0, -1), COR["cinza_claro"]),
        ("BOX", (0, 0), (-1, -1), 0.6, COR["cinza"]),
        ("INNERGRID", (0, 0), (-1, -1), 0.4, COR["cinza"]),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 5),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
    ]))
    E.append(t)
    E.append(Spacer(1, 0.9 * cm))
    E.append(Paragraph(
        "Nota metodologica: as cinco categorias solicitadas foram mapeadas para a stack detectada - <b>Go 1.26.8 "
        "(net/http + html/template), HTMX 2.x, SQLite via modernc.org/sqlite, criptografia PKCS#12/XMLDSig propria, "
        "sem ORM, sem framework de auth, deploy via Dockerfile + GitHub Actions (GHCR)</b>. Assim: (1) isolamento de "
        "inquilino/dono equivale a middleware de tenant/ownership - inexistente, avaliado como ausencia de fronteira de "
        "autorizacao; (2) permissao definida no navegador equivale a gates de papel - inexistentes, pois nao ha papeis; "
        "(3) IDOR foi auditado em todos os handlers que recebem ID em path; (4) segredos expostos cobriram codigo, "
        "configs, Dockerfile, CI, docs e historico git; (5) XSS cobriu templates, JavaScript embutido e, no backend, "
        "a saida HTML manual (fmt.Fprintf) que substitui o template.",
        E_est["legenda"]))
    E.append(PageBreak())


def tabela_achados(achados, est):
    linhas = [[
        Paragraph("Sev.", est["tbl_head"]),
        Paragraph("ID", est["tbl_head"]),
        Paragraph("Arquivo:linha", est["tbl_head"]),
        Paragraph("Descricao", est["tbl_head"]),
    ]]
    estilos_tabela = [
        ("BACKGROUND", (0, 0), (-1, 0), COR["navy"]),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("GRID", (0, 0), (-1, -1), 0.4, COR["cinza"]),
        ("LEFTPADDING", (0, 0), (-1, -1), 4),
        ("RIGHTPADDING", (0, 0), (-1, -1), 4),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]
    for i, a in enumerate(achados, start=1):
        linhas.append([
            Paragraph(ROTULO_SEV[a["sev"]], est["chip_sm"]),
            Paragraph(esc(a["id"]), est["cel"]),
            Paragraph(esc(a["local"]).replace(" + ", "<br/>+ "), est["cel_b"]),
            Paragraph("<b>%s</b><br/><font size=6.6>%s</font>" % (esc(a["titulo"]), esc(a["cat"])), est["cel"]),
        ])
        estilos_tabela.append(("BACKGROUND", (0, i), (0, i), COR[a["sev"]]))
        if i % 2 == 0:
            estilos_tabela.append(("BACKGROUND", (1, i), (-1, i), COR["cinza_claro"]))

    t = LongTable(linhas, colWidths=[2.0 * cm, 1.2 * cm, 4.9 * cm, 8.9 * cm], repeatRows=1)
    t.setStyle(TableStyle(estilos_tabela))
    return t


def bloco_achado(a, est):
    cor = COR[a["sev"]]
    cabecalho = Table([[
        Paragraph("%s &nbsp;-&nbsp; %s" % (esc(a["id"]), esc(a["titulo"])), ParagraphStyle(
            "ct", fontName="Helvetica-Bold", fontSize=9.6, leading=12, textColor=colors.white)),
        Paragraph(ROTULO_SEV[a["sev"]], est["chip"]),
    ]], colWidths=[15.2 * cm, 1.8 * cm])
    cabecalho.setStyle(TableStyle([
        ("BACKGROUND", (0, 0), (0, 0), cor),
        ("BACKGROUND", (1, 0), (1, 0), colors.HexColor("#111827")),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("LEFTPADDING", (0, 0), (-1, -1), 6),
        ("RIGHTPADDING", (0, 0), (-1, -1), 6),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]))

    corpo = [
        cabecalho,
        Spacer(1, 4),
        Paragraph("<b>Local:</b> %s &nbsp;&nbsp;|&nbsp;&nbsp; <b>Categoria:</b> %s" % (esc(a["local"]), esc(a["cat"])), est["cel"]),
        Spacer(1, 4),
        Preformatted(a["trecho"], est["mono"]),
        Spacer(1, 4),
        Paragraph("<b>Por que e exploravel:</b> %s" % esc(a["porque"]), est["corpo"]),
        Paragraph("<b>Impacto:</b> %s" % esc(a["impacto"]), est["corpo"]),
        Paragraph("<b>Condicao de explorabilidade:</b> %s" % esc(a["cond"]), est["corpo"]),
        Paragraph("<b>Correcao sugerida:</b> %s" % esc(a["fix"]), est["corpo"]),
        Spacer(1, 6),
    ]
    return corpo


E_est = {}


def montar_pdf(total):
    doc = BaseDocTemplate(
        PDF_PATH, pagesize=A4,
        leftMargin=2 * cm, rightMargin=2 * cm, topMargin=1.9 * cm, bottomMargin=1.9 * cm,
        title="Relatorio de Auditoria de Seguranca - %s" % PROJETO,
        author="Auditoria de Seguranca", subject="Achados de seguranca (OWASP-like) no Validador eSocial",
    )
    frame = Frame(doc.leftMargin, doc.bottomMargin, doc.width, doc.height, id="normal")
    doc.addPageTemplates([PageTemplate(id="pagina", frames=[frame], onPage=rodape)])

    E = []
    est = estilos()
    E_est.update(est)
    capa(E, total)

    # (b) Resumo executivo
    E.append(Paragraph("1. Resumo executivo", est["h1"]))
    cont = {}
    for a in ACHADOS:
        cont[a["sev"]] = cont.get(a["sev"], 0) + 1
    E.append(Paragraph(
        ("A auditoria cobriu todo o codigo do <b>%s %s</b>: %d achados verificados, sendo "
        "<font color='#B91C1C'><b>1 critico</b></font>, <font color='#EA580C'><b>4 altos</b></font>, "
        "<font color='#D97706'><b>4 medios</b></font>, <font color='#2563EB'><b>5 baixos</b></font> e "
        "<font color='#64748B'><b>1 informativo</b></font>. O risco central e a <b>ausencia total de autenticacao e "
        "autorizacao</b> combinada ao bind em <b>0.0.0.0:8000</b>, que transforma um aplicativo pensado para uso local "
        "em um painel aberto para qualquer host da rede - com leitura de CPFs, troca de certificado digital e exclusao "
        "de dados. Em segundo plano, o produto <b>simula</b> assinatura digital e transmissao ao eSocial e grava "
        "recibos oficiais ficticios, criando uma trilha de auditoria falsa. A base tecnica, porem, e solida nos pontos "
        "estruturais: 100%% das consultas SQL sao parametrizadas, os templates usam escaping contextual do "
        "html/template, nao ha segredos hardcoded (nem no historico git) e nao ha path traversal, command injection "
        "nem XXE exploraveis. <b>Todos os achados foram corrigidos nesta revisao</b> (ver secao 7, com a evidencia "
        "de cada correcao); a validacao por schema, agora ligada, revelou ainda um achado adicional de conformidade "
        "(M-05) que permanece aberto por depender da adequacao dos leiautes.") % (PROJETO, VERSAO, len(ACHADOS)), est["corpo"]))

    resumo = [[
        Paragraph("Severidade", est["tbl_head"]),
        Paragraph("Qtd.", est["tbl_head"]),
        Paragraph("Achados", est["tbl_head"]),
        Paragraph("Leitura de risco", est["tbl_head"]),
    ]]
    leitura = {
        "critica": "Corrigir antes de qualquer uso em rede/Docker.",
        "alta": "Corrigir no proximo ciclo curto; exploravel hoje.",
        "media": "Corrigir antes de uso em producao real.",
        "baixa": "Hardening planejado.",
        "informativa": "Acompanhar em manutencao.",
    }
    est_tab = [
        ("BACKGROUND", (0, 0), (-1, 0), COR["navy"]),
        ("GRID", (0, 0), (-1, -1), 0.4, COR["cinza"]),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("LEFTPADDING", (0, 0), (-1, -1), 4),
        ("RIGHTPADDING", (0, 0), (-1, -1), 4),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]
    for i, k in enumerate(["critica", "alta", "media", "baixa", "informativa"], start=1):
        ids = ", ".join(a["id"] for a in ACHADOS if a["sev"] == k)
        resumo.append([
            Paragraph(ROTULO_SEV[k], est["chip"]),
            Paragraph(str(cont.get(k, 0)), est["cel"]),
            Paragraph(ids, est["cel_b"]),
            Paragraph(leitura[k], est["cel"]),
        ])
        est_tab.append(("BACKGROUND", (0, i), (0, i), COR[k]))
    resumo.append([
        Paragraph("TOTAL", est["chip"]),
        Paragraph(str(len(ACHADOS)), est["cel"]),
        Paragraph("", est["cel"]),
        Paragraph("", est["cel"]),
    ])
    est_tab.append(("BACKGROUND", (0, len(resumo) - 1), (0, len(resumo) - 1), colors.HexColor("#111827")))
    est_tab.append(("BACKGROUND", (1, len(resumo) - 1), (-1, len(resumo) - 1), COR["cinza_claro"]))
    t = Table(resumo, colWidths=[2.2 * cm, 1.2 * cm, 4.4 * cm, 9.2 * cm])
    t.setStyle(TableStyle(est_tab))
    E.append(t)
    E.append(Spacer(1, 10))

    graficos = Table([[
        Image(os.path.join(GRAF_DIR, "severidade.png"), width=8.2 * cm, height=5.46 * cm),
        Image(os.path.join(GRAF_DIR, "categorias.png"), width=8.2 * cm, height=4.22 * cm),
    ]], colWidths=[8.5 * cm, 8.5 * cm])
    graficos.setStyle(TableStyle([("VALIGN", (0, 0), (-1, -1), "TOP"), ("ALIGN", (0, 0), (-1, -1), "CENTER")]))
    E.append(graficos)
    E.append(Paragraph("Figuras 1 e 2 - Distribuicao por severidade (rosca) e por categoria (barras empilhadas).", est["legenda"]))
    E.append(PageBreak())

    # (c) Pontos fortes e fracos
    E.append(Paragraph("2. Pontos fortes e pontos fracos", est["h1"]))
    E.append(Paragraph("2.1 Pontos fortes (verificados)", est["h2"]))
    for titulo, local, evid in PONTOS_FORTES:
        E.append(Paragraph("<font color='#059669'><b>&#10003;</b></font> <b>%s</b>" % esc(titulo), est["bullet"], bulletText=""))
        E.append(Paragraph("<font size=7.6 color='#475569'>%s - %s</font>" % (esc(local), esc(evid)),
                           ParagraphStyle("pf", parent=est["bullet"], leftIndent=14, spaceAfter=5)))
    E.append(Spacer(1, 4))
    E.append(Paragraph("2.2 Pontos fracos (riscos centrais)", est["h2"]))
    for txt in [
        "<b>Fronteira de confianca inexistente:</b> o servidor nao identifica quem faz a requisicao - nao ha login, "
        "sessao, token nem papeis. Toda a aplicacao opera como um unico usuario administrador implicito (C-01).",
        "<b>Escrita sem verificacao de intencao:</b> nenhum endpoint de escrita exige prova de que a requisicao veio "
        "da propria interface (CSRF, A-01).",
        "<b>Integridade documental comprometida:</b> assinatura e transmissao simuladas com recibos ficticios "
        "apresentados como oficiais (A-02) e campos sem escape no XML gerado (M-01).",
        "<b>Saida HTML manual:</b> o handler de certificado escreve HTML por fmt.Fprintf, contornando o escaping "
        "automatico do html/template (A-03).",
        "<b>Ausencia de defesa em profundidade:</b> sem limites de tamanho (M-02), sem rate limit (M-04), sem "
        "cabecalhos de seguranca (B-03) e com erros de persistencia silenciados (B-04).",
    ]:
        E.append(Paragraph(txt, est["bullet"], bulletText="\u2022"))
    E.append(PageBreak())

    # (d) Tabela de achados
    E.append(Paragraph("3. Tabela de achados", est["h1"]))
    E.append(Paragraph(
        "A tabela abaixo resume todos os achados verificados. O detalhamento com trecho de codigo, explorabilidade, "
        "impacto e correcao sugerida esta na secao 4.", est["corpo"]))
    E.append(tabela_achados(ACHADOS, est))
    E.append(PageBreak())

    # Detalhamento por categoria
    E.append(Paragraph("4. Achados detalhados por categoria", est["h1"]))
    cats = []
    for a in ACHADOS:
        if a["cat"] not in cats:
            cats.append(a["cat"])
    for cat in cats:
        E.append(Paragraph(cat, est["h2"]))
        for a in [x for x in ACHADOS if x["cat"] == cat]:
            E.extend(bloco_achado(a, est))
    E.append(PageBreak())

    # (e) Recomendacoes
    E.append(Paragraph("5. Recomendacoes priorizadas", est["h1"]))
    recs = [
        ("P1", "Fechar a fronteira de autorizacao",
         "Trocar o bind default para 127.0.0.1, adicionar autenticacao com middleware global, validar o cabecalho Host "
         "e aplicar token anti-CSRF em todos os POSTs. Corrige C-01 e A-01.", "C-01, A-01"),
        ("P1", "Parar de falsificar assinatura e recibos",
         "Marcar explicitamente o modo simulacao, nao gravar recibo/protocolo sem transmissao real, conectar o "
         "ClienteSOAP mTLS e corrigir o README. Corrige A-02.", "A-02"),
        ("P2", "Eliminar a saida HTML manual",
         "Migrar o handler de certificado para template.ExecuteTemplate com html/template, remover safeHTML e cobrir "
         "com teste de regressao o nome de arquivo e os campos do certificado. Corrige A-03 e B-05.", "A-03, B-05"),
        ("P2", "Blindar a geracao de XML",
         "Aplicar escapeXML/encoding/xml em todos os campos e validar dominios (datas ISO, tpAval, utilizEPC/EPI, UF). "
         "Corrige M-01.", "M-01"),
        ("P2", "Limitar entrada e abuso",
         "http.MaxBytesReader em uploads e corpos, limites por tipo de arquivo e rate limit no teste de senha do "
         "certificado. Corrige M-02 e M-04.", "M-02, M-04"),
        ("P3", "Ligar a validacao real",
         "Chamar esocial.ValidarXSD no handler e ajustar a mensagem ao que foi verificado. Corrige M-03.", "M-03"),
        ("P3", "Hardening de arquivos e imagem",
         "PFX com 0600, diretorio de dados 0700, container sem root, senhas de teste geradas em runtime. Corrige B-01 "
         "e I-01.", "B-01, I-01"),
        ("P3", "Defesa em profundidade e observabilidade",
         "Cabecalhos de seguranca, validacao do ID do evento e tratamento de erros de persistencia com feedback real. "
         "Corrige B-02, B-03 e B-04.", "B-02, B-03, B-04"),
    ]
    linhas = [[Paragraph(h, est["tbl_head"]) for h in ["Prio.", "Acao", "Detalhe", "Achados"]]]
    est_tab = [
        ("BACKGROUND", (0, 0), (-1, 0), COR["navy"]),
        ("GRID", (0, 0), (-1, -1), 0.4, COR["cinza"]),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("LEFTPADDING", (0, 0), (-1, -1), 4),
        ("RIGHTPADDING", (0, 0), (-1, -1), 4),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]
    cor_prio = {"P1": COR["critica"], "P2": COR["media"], "P3": COR["baixa"]}
    for i, (p, acao, det, ids) in enumerate(recs, start=1):
        linhas.append([
            Paragraph(p, est["chip"]),
            Paragraph(esc(acao), est["cel"]),
            Paragraph(esc(det), est["cel"]),
            Paragraph(esc(ids), est["cel_b"]),
        ])
        est_tab.append(("BACKGROUND", (0, i), (0, i), cor_prio[p]))
        if i % 2 == 0:
            est_tab.append(("BACKGROUND", (1, i), (-1, i), COR["cinza_claro"]))
    t = LongTable(linhas, colWidths=[1.4 * cm, 4.4 * cm, 9.4 * cm, 1.8 * cm], repeatRows=1)
    t.setStyle(TableStyle(est_tab))
    E.append(t)
    E.append(PageBreak())


    # (g) Status das correções aplicadas
    E.append(PageBreak())
    E.append(Paragraph("7. Status das correcoes aplicadas", est["h1"]))
    E.append(Paragraph(
        "Esta secao registra o que foi efetivamente alterado no codigo para cada achado, com a evidencia de verificacao. "
        "As correcoes foram validadas por <b>go build</b>, <b>go vet</b>, suite de testes (<b>go test ./...</b>, incluindo "
        "os novos testes de regressao em <b>internal/web/seguranca_test.go</b>) e teste ponta a ponta com o binario real "
        "(login, sessao, CSRF, Host, cabecalhos, limites de upload e fluxo de simulacao).", est["corpo"]))

    linhas_corr = [[Paragraph(h, est["tbl_head"]) for h in ["Achado", "Status", "Correcao aplicada", "Evidencia"]]]
    est_corr = [
        ("BACKGROUND", (0, 0), (-1, 0), COR["navy"]),
        ("GRID", (0, 0), (-1, -1), 0.4, COR["cinza"]),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("LEFTPADDING", (0, 0), (-1, -1), 4),
        ("RIGHTPADDING", (0, 0), (-1, -1), 4),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ]
    for i, (achado, correcao, evidencia) in enumerate(CORRECOES, start=1):
        linhas_corr.append([
            Paragraph(esc(achado), est["cel_b"]),
            Paragraph("CORRIGIDO", est["chip_sm"]),
            Paragraph(esc(correcao), est["cel"]),
            Paragraph(esc(evidencia), est["cel"]),
        ])
        est_corr.append(("BACKGROUND", (1, i), (1, i), COR["forte"]))
        if i % 2 == 0:
            est_corr.append(("BACKGROUND", (0, i), (0, i), COR["cinza_claro"]))
            est_corr.append(("BACKGROUND", (2, i), (-1, i), COR["cinza_claro"]))
    t_corr = LongTable(linhas_corr, colWidths=[1.5 * cm, 2.1 * cm, 8.4 * cm, 5.0 * cm], repeatRows=1)
    t_corr.setStyle(TableStyle(est_corr))
    E.append(t_corr)
    E.append(Spacer(1, 8))
    E.append(Paragraph(
        "Achado adicional (aberto): <b>M-05</b> - os geradores de XML do editor ainda nao sao aderentes ao XSD oficial "
        "S-1.3 (ideVinculo, agNoc, codigos de orgao de classe e blocos obrigatorios de CAT/ASO). A correcao do M-03 "
        "tornou essa divergencia visivel e rastreavel em vez de silenciosa; a adequacao dos leiautes depende de decisao "
        "funcional sobre os campos obrigatorios do documento fiscal (issue dedicada na secao 6).", est["corpo"]))

    # (f) Issues para o GitHub
    E.append(Paragraph("6. ISSUES PARA O GITHUB", est["h1"]))
    E.append(Paragraph(
        "Os blocos abaixo estao prontos para copiar e colar na criacao de issues. Cada bloco vai do delimitador "
        "<b>--- ISSUE n ---</b> ate <b>--- FIM ISSUE n ---</b> e contem titulo, labels sugeridas, descricao, evidencia, "
        "impacto, correcao e criterios de aceite em Markdown. Achados triviais relacionados foram agrupados para evitar "
        "spam de issues.", est["corpo"]))
    E.append(Spacer(1, 6))

    for n, iss in enumerate(ISSUES, start=1):
        md = "--- ISSUE %d ---\n\n# %s\n\nLabels: %s\n\n%s\n--- FIM ISSUE %d ---" % (
            n, iss["titulo"], iss["labels"], iss["corpo"].strip(), n)
        bloco = [
            Paragraph("Issue %d de %d" % (n, len(ISSUES)), est["h3"]),
            Preformatted(md, est["mono"]),
            Spacer(1, 10),
        ]
        E.append(KeepTogether(bloco[:2]))
        E.append(Spacer(1, 10))

    doc.build(E)


def main():
    gerar_graficos()
    montar_pdf(len(ACHADOS))
    print("PDF gerado em: %s" % PDF_PATH)
    print("Graficos em: %s" % GRAF_DIR)


if __name__ == "__main__":
    main()
