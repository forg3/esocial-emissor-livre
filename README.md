# eSocial Emissor Livre

[![Licença MIT](https://img.shields.io/badge/licen%C3%A7a-MIT-blue.svg)](LICENSE)
[![FOSS](https://img.shields.io/badge/FOSS-Free%20%26%20Open%20Source-green.svg)](https://en.wikipedia.org/wiki/Free_and_open-source_software)
[![Open Source](https://img.shields.io/badge/Open%20Source-%E2%99%A5-orange.svg)](https://opensource.org/)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go&logoColor=white)](https://golang.org/)
[![HTMX](https://img.shields.io/badge/HTMX-2.0+-336699.svg)](https://htmx.org/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4+-38B2AC.svg?logo=tailwind-css&logoColor=white)](https://tailwindcss.com/)
[![eSocial](https://img.shields.io/badge/eSocial-Leiaute%20S--1.3-005CA9.svg)](https://www.gov.br/esocial/pt-br/documentacao-tecnica)

Software web livre, leve e direto para elaboração, validação estrutural (XSD), assinatura digital (XMLDSig A1), transmissão via webservice oficial e auditoria de recibos dos eventos do **eSocial** (versão S-1.3).

---

## O que é

O **eSocial Emissor Livre** é uma aplicação web autônoma desenvolvida em Go, Tailwind CSS e HTMX, concebida para que empresas, contadores, clínicas de SST e engenheiros de segurança possam manipular e transmitir eventos do eSocial diretamente ao governo federal (gov.br), sem a necessidade de softwares proprietários caros, plataformas engessadas ou assinaturas corporativas complexas.

## Como usar

### Pré-requisitos
- Go 1.22 ou superior instalado.
- Certificado digital ICP-Brasil modelo A1 (`.pfx` ou `.p12`).
- PostgreSQL (opcional para persistência avançada de histórico e fila; suporta modo embarcado/local).

### Execução Local

```bash
# Clone o repositório
git clone https://github.com/forg3/esocial-emissor-livre.git
cd esocial-emissor-livre

# Baixe as dependências e inicie o serviço
go mod tidy
go run cmd/server/main.go
```

Acesse a interface no navegador em `http://localhost:8080`.

---

## O que faz

- **Edição e Montagem Guiada**: Formulários rápidos para elaboração de eventos de Segurança e Saúde no Trabalho (S-2210, S-2220, S-2240) e tabelas iniciais.
- **Validação Estrutural Rígida**: Confere os arquivos XML diretamente contra os esquemas XSD oficiais do leiaute S-1.3 do eSocial antes de qualquer tentativa de envio.
- **Importação e Conferência de XML**: Recebe arquivos XML gerados por terceiros (ex.: ASO de clínicas médicas) e realiza cruzamento de dados cadastrais (CPF, vínculo, datas) prevenindo autuações por erro material.
- **Assinatura Digital Local**: Assina eventos utilizando certificado digital ICP-Brasil A1 diretamente na camada da aplicação, mantendo a chave privada segura.
- **Transmissão Direta com WebServices Oficiais**: Envia lotes e consulta recibos nos ambientes de **Produção** e **Produção Restrita** (homologação) do eSocial.
- **Auditoria de Eventos e Recibos**: Rastreamento do ciclo de vida dos eventos (`pronto` → `assinado` → `transmitido` → `aceito` ou `rejeitado`), registrando números de recibo e mensagens de erro governamentais.

---

## O que ainda pode fazer (Roadmap)

- [ ] Cobertura completa de eventos da Folha de Pagamento (S-1200 a S-1299).
- [ ] Módulo de importação em massa via planilhas (CSV/XLSX) para clínicas ocupacionais e contabilidades.
- [ ] Exportação de relatórios de conformidade e conferência de débitos previdenciários (S-5001/S-5011).
- [ ] Pacote executável único multiplataforma (desktop/CLI local) sem dependência externa de banco de dados.
- [ ] Integração com múltiplos certificados digitais para escritórios contábeis que gerenciam diversas procurações eletrônicas.

---

## O que não faz

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
