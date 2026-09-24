# Validador eSocial

<p align="center">
  <img src="internal/web/static/img/validador-logo-oficial-cropped.png" alt="Validador eSocial" width="380">
</p>

[![Licença MIT](https://img.shields.io/badge/licen%C3%A7a-MIT-blue.svg)](LICENSE)
[![FOSS](https://img.shields.io/badge/FOSS-Free%20%26%20Open%20Source-green.svg)](https://en.wikipedia.org/wiki/Free_and_open-source_software)
[![Open Source](https://img.shields.io/badge/Open%20Source-%E2%99%A5-orange.svg)](https://opensource.org/)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org/)
[![HTMX](https://img.shields.io/badge/HTMX-2.0+-336699.svg)](https://htmx.org/)
[![eSocial](https://img.shields.io/badge/eSocial-Leiaute%20S--1.3-005CA9.svg)](https://www.gov.br/esocial/pt-br/documentacao-tecnica)

Software web livre, leve e direto para elaboração, validação estrutural (XSD) e assinatura digital (XMLDSig A1) dos eventos do **eSocial** (versão S-1.3), com execução local e autenticação obrigatória.

> **Estado atual (v1.1.1-alpha):** a assinatura com certificado A1 é real; o **envio ao webservice oficial ainda é simulado** — nenhum dado é transmitido ao governo e **nenhum recibo oficial é gerado**. Eventos processados nesse fluxo ficam com o status `simulado` e a interface exibe o aviso "MODO SIMULAÇÃO". Não utilize os comprovantes para cumprimento de obrigação acessória até a integração real ser habilitada.

---

## O que é

O **Validador eSocial** é uma aplicação web autônoma e portátil desenvolvida em Go e HTMX, concebida para que empresas, contadores, clínicas de SST e engenheiros de segurança possam manipular, validar e transmitir eventos do eSocial diretamente ao governo federal (gov.br), sem a necessidade de softwares proprietários caros, plataformas engessadas ou assinaturas corporativas complexas.

## Como usar

### Modo 1: Execução Nativa Portátil (Sem Instalar Nada)

Você **não precisa ter o Go instalado** e não precisa de banco de dados ou dependências externas. O aplicativo é distribuído como um binário nativo único e auto-contido. Ao ser iniciado, **abre automaticamente o seu navegador** no painel local:

1. Acesse a página de [Releases](https://github.com/forg3/validador-esocial/releases/tag/v1.1.1-alpha).
2. Baixe o pacote correspondente ao seu sistema operacional:
   - **Windows (x86_64):** Extraia o arquivo `.zip` e execute `validador-esocial.exe`.
   - **macOS Apple Silicon (M1 / M2 / M3 / M4 / M5 / M6+):** Baixe `validador-esocial-v1.1.1-alpha-darwin-arm64.tar.gz`, extraia e execute `./validador-esocial-darwin-arm64`.
   - **macOS Intel (x86_64):** Baixe `validador-esocial-v1.1.1-alpha-darwin-amd64.tar.gz`, extraia e execute `./validador-esocial-darwin-amd64`.
   - **Linux (x86_64):** Baixe `validador-esocial-v1.1.1-alpha-linux-amd64.tar.gz`, extraia e execute `./validador-esocial`.
3. O navegador será iniciado automaticamente em `http://localhost:8000`.

### Modo 2: A Partir do Código-Fonte (Para Desenvolvedores)

Se desejar compilar diretamente:

```bash
# Clone o repositório
git clone https://github.com/forg3/validador-esocial.git
cd validador-esocial

# Inicie o serviço diretamente (porta padrão 8000)
go run cmd/server/main.go
```

Acesse a interface no navegador em `http://localhost:8000` e informe a **senha local** exibida no terminal na
primeira execução (ela fica salva com hash em `dados/auth.json`, permissão `0600`).

### Instalação por pacote

```bash
# Debian/Ubuntu (.deb)
sudo dpkg -i validador-esocial_1.1.1-alpha_linux_amd64.deb

# Fedora/RHEL/openSUSE (.rpm)
sudo rpm -i validador-esocial-1.1.1-alpha.x86_64.rpm

# Windows: execute o instalador .msi (cria atalhos no Menu Iniciar e na Área de Trabalho)
#   validador-esocial-1.1.1-alpha-windows-amd64.msi
```

Após instalar, execute `validador-esocial` (o atalho no Windows). O binário é auto-contido: cria o banco SQLite em
`./dados`, sobe o servidor em `127.0.0.1:8000` e abre o navegador. A senha local de acesso é exibida no terminal na
primeira execução (ou defina com `-senha` / `ESOCIAL_SENHA`).

### Transmissão real ao eSocial (opcional)

Por padrão o sistema opera em **modo simulado** (nada é enviado ao governo). Para habilitar o envio oficial:

1. Carregue o certificado A1 em **Empresas e Certificados** (perfil ativo) ou em **Certificado & Empresa**.
2. Em **Certificado & Empresa → Modo de transmissão ao eSocial**, selecione **Real** e aplique.
3. Assine os eventos com o certificado e use **Transmitir** na fila, informando a senha do certificado no campo exibido.
4. Use **Recibo** para consultar o processamento do lote e gravar o número de recibo oficial.

Para apontar para um proxy/homologação, defina `ESOCIAL_WS_ENVIO` e `ESOCIAL_WS_CONSULTA`.

### Senha de acesso e exposição de rede

```bash
# Define a senha de acesso manualmente
go run cmd/server/main.go -senha "MINHA_SENHA_FORTE"
# ou
ESOCIAL_SENHA="MINHA_SENHA_FORTE" go run cmd/server/main.go

# Expõe na rede (use apenas com TLS/proxy reverso e senha forte)
go run cmd/server/main.go -host 0.0.0.0 -senha "MINHA_SENHA_FORTE"
```

A aplicação escuta **somente em `127.0.0.1` por padrão**. Todas as rotas exigem sessão autenticada
(cookie `HttpOnly` + `SameSite=Strict`), requisições de escrita exigem token anti-CSRF, o cabeçalho `Host`
é validado contra a allowlist local (anti DNS rebinding), há limite de tamanho para uploads e rate limit
nas tentativas de login e de validação da senha do certificado.

---

## Segurança (v1.1-alpha)

Esta release incorpora uma auditoria de segurança completa (5 categorias: isolamento/controle de acesso, permissões,
IDOR, segredos expostos e XSS) com **todos os achados corrigidos** — o relatório está em
[`docs/security-audit/relatorio-auditoria-seguranca.pdf`](docs/security-audit/relatorio-auditoria-seguranca.pdf).

**Correções aplicadas**

- **Autenticação obrigatória**: senha local (hash com salt, 100k iterações), sessão em cookie `HttpOnly` +
  `SameSite=Strict`, tela de login e logout. Escuta padrão apenas em `127.0.0.1` (flag `-host` para expor).
- **Anti-CSRF**: token em todos os formulários e no cabeçalho `X-CSRF-Token` das requisições HTMX, com validação de
  `Origin`/`Referer` em todo método de escrita.
- **Anti DNS rebinding**: validação do cabeçalho `Host` contra allowlist local.
- **Fim da assinatura/recibo falsos**: o fluxo simulado é rotulado (`MODO SIMULAÇÃO` na interface, `ASSINATURA SIMULADA`
  no XML, status `simulado`) e **nenhum protocolo ou recibo oficial é gerado** sem transmissão real.
- **XSS/HTML injection**: saída do teste de certificado migrada para `html/template` (escaping contextual).
- **XML injection**: escape e validação de domínio (datas ISO, hora HHMM, UF oficial, códigos controlados) em todos os
  campos dos geradores de eventos.
- **Limites de entrada**: 2 MB (certificado), 5 MB (XML), 10 MB (CSV) e 12 MB por requisição de escrita.
- **Validação por schema real**: `xmllint` + XSD oficiais embutidos, com modo degradado informado ao usuário.
- **Rate limit**: 8 tentativas/minuto no login e 6/minuto na validação da senha do certificado A1.
- **Hardening**: banco e certificados com permissão `0600` (diretórios `0700`), container sem privilégios (UID 10001),
  cabeçalhos CSP/`X-Content-Type-Options`/`X-Frame-Options`/`Referrer-Policy`, erros de persistência reportados ao usuário.

**Melhorias**

- **Sprint 6 (certificados A3 / PKCS#11)**: driver PKCS#11 sob a tag de build `pkcs11`, com seleção de módulo/slot/PIN na
  interface, leitura do certificado do token e assinatura delegada ao hardware (a chave privada nunca sai do dispositivo).
- **Sprint 1 (conformidade de leiaute)**: os geradores dos eventos S-2210 (CAT), S-2220 (ASO) e S-2240 (condições ambientais) foram reescritos para o leiaute oficial S-1.3 (`ideVinculo`, `agNoc`, `epcEpi`/`epiCompl`, `respReg`, CAT completa com `codSitGeradora`/`iniciatCAT`/`localAcidente`/`parteAtingida`/`agenteCausador`/`atestado` e ASO com `exMedOcup`/`aso`/`exame`/`medico`).
- Filtro e contador de eventos `simulado` na fila de transmissão.
- Mensagens de validação passam a informar exatamente o modo executado e o que divergiu.
- Senha de acesso definível por `-senha` ou `ESOCIAL_SENHA`, com geração automática na primeira execução.
- Suíte de testes ampliada com 13 testes de regressão de segurança (`internal/web/seguranca_test.go`).

---

## Versão e Releases

**Versão Atual:** `v1.1.1-alpha` (Release com **correções de segurança**, **leiautes S-1.3 aderentes ao XSD oficial**, **relatórios de auditoria de retorno**, **múltiplos certificados/procurações** e **transmissão real opcional** ao webservice oficial, para testes de conformidade com Leiaute S-1.3 NT 07/2026).  
Download dos binários pré-compilados portáteis para Linux, Windows e macOS na aba [Releases](https://github.com/forg3/validador-esocial/releases/tag/v1.1.1-alpha).

---

## O que faz

- **Cobertura Integral dos 36 Eventos Oficiais**: Catálogo visual estruturado por departamento (SESMT, Medicina Ocupacional, RH/DP, Folha, Jurídico e RPPS) e editor guiado com sincronização em tempo real com o XML S-1.3.
- **Validação Estrutural Rígida**: Confere os arquivos XML diretamente contra os esquemas XSD oficiais do leiaute S-1.3 (via `xmllint`, com modo degradado explicitamente informado quando a ferramenta não está instalada). Os documentos gerados pelos editores de **S-2210, S-2220 e S-2240 são aprovados pelo schema oficial** — verificado por teste automatizado que executa o `xmllint` contra os XSD embutidos.
- **Importação e Conferência de XML/CSV**: Recebe arquivos XML (como ASO) e realiza importação de colaboradores em lote via CSV com modelo pronto para download.
- **Tabelas oficiais do eSocial embutidas**: Tabela 13 (parte do corpo atingida), Tabela 14 (agente causador), Tabela 15 (situação geradora), Tabela 17 (natureza da lesão), Tabela 24 (agentes nocivos) e Tabela 27 (procedimentos diagnósticos), disponíveis como listas nos formulários — sem digitação de códigos.
- **Assinatura Digital Local (A1)**: Assina eventos utilizando certificado digital ICP-Brasil **A1** (`.pfx` / `.p12`) diretamente na máquina do usuário (`internal/crypto`, XMLDSig + C14N). O fluxo de demonstração gera um envelope **explicitamente marcado como simulado** (`ASSINATURA SIMULADA`), sem validade jurídica.
- **Transmissão ao eSocial (em integração)**: o cliente mTLS dos web services oficiais (`internal/soap`, TLS 1.2+, certificado A1) está implementado, mas **ainda não conectado à interface**. Nesta versão o envio é apenas simulado, sem protocolo e sem recibo.
- **Auditoria de Eventos**: Rastreamento do ciclo de vida (`pronto` → `assinado` → `simulado` → `transmitido` → `aceito`/`rejeitado`), com registro de recibo e mensagens oficiais.
- **Relatórios e Auditoria de Retorno (S-5001/S-5011)**: importação dos totalizadores devolvidos pelo eSocial, consolidação por período (previdenciário, FGTS e IRRF), memórias de cálculo por trabalhador e exportação em CSV.
- **Múltiplas Empresas e Procurações**: perfis por CNPJ com certificado A1 próprio, dados do procurador eletrônico e ativação com sincronização automática do empregador emissor.
- **Transmissão Real Opcional**: modo **Real** envia os lotes assinados ao webservice oficial do eSocial via mTLS (`EnviarLoteEventos`) e consulta os recibos (`ConsultarLoteEventos`), com senha do certificado solicitada a cada envio e nunca armazenada.

---

- [x] **Leiautes S-1.3 aderentes ao XSD oficial (Sprint 1)**: geradores de S-2210, S-2220 e S-2240 reescritos conforme o leiaute oficial, com tabelas 13/14/15/17/27 embutidas e validação por schema aprovada nos testes de regressão.
- [x] **Auditoria de Segurança e Hardening (v1.1-alpha)**: autenticação obrigatória, anti-CSRF, validação de Host, limites de entrada, rate limit, cabeçalhos de segurança e relatório de auditoria publicado em `docs/security-audit/`.
- [x] **Publicação de Packages (GitHub Packages)**: Imagem de container publicada no GitHub Container Registry (`ghcr.io/forg3/validador-esocial:v1.1.1-alpha`).
- [x] **Pacotes de Distribuição (.deb, .rpm e MSI)**: instaladores nativos gerados pelo pipeline de release (GoReleaser para `.deb`/`.rpm`/`.tar.gz`/`.zip` e WiX Toolset para o `.msi` do Windows).
- [x] **Pipeline de CI/CD para Releases**: workflow `release.yml` executa build, `go vet`, testes, compilação cruzada (linux/darwin/windows em amd64 e arm64), pacotes `.deb`/`.rpm`, instalador `.msi`, checksums SHA-256 e publicação automática na release — além da imagem de container no GHCR.
- [~] **Suporte a Certificados A3 (Tokens USB e Smartcards via PKCS#11)**: implementado e habilitável por build
  (`go build -tags pkcs11 ./cmd/server`) — a chave privada permanece no token e a assinatura é delegada ao hardware.
  A validação final depende de um token físico (não executável em ambiente de CI).
- [x] **Módulo de Relatórios e Auditoria de Retorno**: importação dos totalizadores S-5001/S-5011, consolidação por período (previdenciário/FGTS/IRRF), memórias de cálculo e exportação CSV.
- [x] **Múltiplos Certificados e Procurações Eletrônicas**: perfis multiempresa com certificado A1 por CNPJ, dados do procurador eletrônico e ativação sincronizada.

---

## O que não faz

- **Transmissão real exige certificado A1 válido e homologação**: o envio oficial está implementado (mTLS + `EnviarLoteEventos`/`ConsultarLoteEventos`) e validado com webservice simulado; a conferência final depende de certificado real em Produção Restrita. O modo padrão continua **simulado**, com aviso na interface.
- **Certificados A3 exigem build dedicada**: o suporte a token/smartcard via PKCS#11 está implementado, mas não é
  incluído na build padrão (que aceita certificados **A1** em `.pfx`/`.p12`). Para habilitar, compile com
  `go build -tags pkcs11 ./cmd/server` e informe o módulo PKCS#11 do fabricante na tela **Certificado & Empresa**.
- **Não é um ERP de Folha de Pagamento**: Não calcula holerites, encargos sindicais complexos, horas extras ou rescisões trabalhistas. O software opera sobre os dados brutos necessários para a geração do evento.
- **Não armazena chaves privadas em nuvem**: Em implantações corporativas multiusuário, o sistema não transfere certificados A1 para repositórios desprotegidos.
- **Não substitui a responsabilidade técnica legal**: O preenchimento e a validação de laudos (LTCAT, PGR, PCMSO) e CATs continuam sendo atribuições formais dos profissionais habilitados (Médicos do Trabalho, Engenheiros e Técnicos de Segurança, e Contadores).

---

## Matriz Completa de Eventos do eSocial (Leiaute S-1.3)

O eSocial é composto por **36 eventos** organizados por competência técnica e departamento responsável:

### 🛡️ SESMT & Engenharia de Segurança do Trabalho
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-1005** | Tabela de Estabelecimentos, Obras ou Unidades | Tabelas | Até dia 15 do mês seguinte ao início ou alteração |
| **S-2210** | Comunicação de Acidente de Trabalho (CAT) | Não Periódico | 1º dia útil seguinte; imediato em caso de morte |
| **S-2240** | Condições Ambientais do Trabalho - Agentes Nocivos | Não Periódico | Até dia 15 do mês subsequente à admissão/alteração |

### 🩺 Clínicas de Medicina Ocupacional (SST)
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-2220** | Monitoramento da Saúde do Trabalhador (ASO) | Não Periódico | Até dia 15 do mês subsequente à emissão do exame |

### 👥 Recursos Humanos & Departamento Pessoal (RH / DP)
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-2190** | Registro Preliminar de Trabalhador | Não Periódico | Até o final do dia anterior ao início da prestação |
| **S-2200** | Cadastramento Inicial do Vínculo e Admissão | Não Periódico | Até dia 15 do mês seguinte (ou dia anterior ao início) |
| **S-2205** | Alteração de Dados Cadastrais do Trabalhador | Não Periódico | Até dia 15 do mês subsequente à alteração |
| **S-2206** | Alteração de Contrato de Trabalho | Não Periódico | Até dia 15 do mês subsequente à alteração |
| **S-2230** | Afastamento Temporário | Não Periódico | Conforme motivo (dia 15 do mês subsequente ou até 16º dia) |
| **S-2231** | Cessão / Exercício em Outro Órgão | Não Periódico | Até dia 15 do mês subsequente |
| **S-2298** | Reintegração / Outros Provimentos | Não Periódico | Até dia 15 do mês subsequente |
| **S-2299** | Desligamento | Não Periódico | Até 10 dias após o término ou dia 15 |
| **S-2300** | Trabalhador Sem Vínculo de Emprego - Início | Não Periódico | Até dia 15 do mês subsequente ao início |
| **S-2306** | Trabalhador Sem Vínculo de Emprego - Alteração | Não Periódico | Até dia 15 do mês subsequente à alteração |
| **S-2399** | Trabalhador Sem Vínculo de Emprego - Término | Não Periódico | Até dia 15 do mês subsequente |
| **S-3000** | Exclusão de Eventos | Não Periódico | Sempre que houver necessidade de cancelamento |

### 💰 Contabilidade, Fiscal & Folha de Pagamento
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-1000** | Informações do Empregador / Contribuinte | Tabelas | Antes de qualquer outro evento |
| **S-1010** | Tabela de Rubricas da Folha de Pagamento | Tabelas | Antes do envio dos eventos de remuneração |
| **S-1020** | Tabela de Lotações Tributárias | Tabelas | Antes dos eventos de remuneração |
| **S-1200** | Remuneração de Trabalhador vinculado ao RGPS | Periódico | Até dia 15 do mês subsequente à competência |
| **S-1210** | Pagamentos de Rendimentos do Trabalho | Periódico | Até dia 15 do mês subsequente ao pagamento |
| **S-1260** | Comercialização da Produção Rural Pessoa Física | Periódico | Até dia 15 do mês subsequente |
| **S-1270** | Contratação de Avulsos Não Portuários | Periódico | Até dia 15 do mês subsequente |
| **S-1280** | Informações Complementares aos Periódicos | Periódico | Até dia 15 do mês subsequente |
| **S-1298** | Reabertura dos Eventos Periódicos | Periódico | Quando houver retificação necessária |
| **S-1299** | Fechamento dos Eventos Periódicos | Periódico | Até dia 15 do mês subsequente |

### ⚖️ Jurídico & Contencioso
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-1070** | Tabela de Processos Administrativos e Judiciais | Tabelas | Antes dos eventos que utilizem o processo |
| **S-8200** | Anotação Judicial do Vínculo | Não Periódico | Conforme determinação judicial |

### 🏛️ Setor Público & Regime Próprio (RPPS)
| Evento | Nome do Evento | Categoria | Prazo Legal (MOS S-1.3) |
| :--- | :--- | :--- | :--- |
| **S-1202** | Remuneração de Servidor vinculado ao RPPS | Periódico | Até dia 15 do mês subsequente |
| **S-1207** | Benefícios - Entes Públicos | Periódico | Até dia 15 do mês subsequente |
| **S-2400** | Cadastro de Beneficiário - Entes Públicos | Não Periódico | Até dia 15 do mês subsequente |
| **S-2405** | Alteração de Dados Cadastrais do Beneficiário | Não Periódico | Até dia 15 do mês subsequente |
| **S-2410** | Cadastro de Benefício - RPPS | Não Periódico | Até dia 15 do mês subsequente |
| **S-2416** | Alteração do Benefício - RPPS | Não Periódico | Até dia 15 do mês subsequente |
| **S-2418** | Reativação de Benefício - RPPS | Não Periódico | Até dia 15 do mês subsequente |
| **S-2420** | Término do Benefício - RPPS | Não Periódico | Até dia 15 do mês subsequente |

---

## Contribuições

Este é um projeto **FOSS** (Free and Open-Source Software) regido pela **Licença MIT**. Contribuições de desenvolvedores, contadores, médicos do trabalho e engenheiros de segurança são calorosamente bem-vindas.

Para contribuir:
1. Abra uma *issue* no GitHub discutindo o problema ou a nova funcionalidade.
2. Crie uma *branch* a partir da `main` (`feature/meu-recurso` ou `fix/ajuste-xsd`).
3. Garanta que qualquer modificação estrutural de XML seja validada contra os esquemas XSD oficiais da versão vigente.
4. Abra um *Pull Request* detalhado.

---

## Autor

**André Santo**  
Contato e Repositório Oficial: [github.com/forg3](https://github.com/forg3)
