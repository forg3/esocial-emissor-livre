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

## Matriz de Responsabilidades dos Eventos (Leiaute S-1.3)

| Categoria | Evento | Descrição | Responsável Principal | Fonte Primária |
| :--- | :--- | :--- | :--- | :--- |
| **Tabelas** | S-1000 | Informações do Empregador/Contribuinte | Contador / RH | Contrato Social / Cartão CNPJ |
| **Tabelas** | S-1005 | Estabelecimentos e Obras de Construção Civil | Engenharia / Contador | CNO / Alvará / Endereço da Obra |
| **Tabelas** | S-1010 | Tabela de Rubricas da Folha | Contador / DP | Plano de Cargos e Salários |
| **Tabelas** | S-1020 | Lotações Tributárias | Contador / Fiscal | Enquadramento FPAS / CNAE |
| **Tabelas** | S-1070 | Processos Administrativos e Judiciais | Jurídico / Contador | Peças e Decisões Judiciais |
| **Não Periódico** | S-2190 | Registro Preliminar de Trabalhador | Contador / RH / DP | Ficha de Contratação |
| **Não Periódico** | S-2200 | Cadastramento Inicial e Admissão | Contador / RH / DP | Documentos Admissionais e CTPS |
| **Não Periódico** | S-2205 | Alteração de Dados Cadastrais | Contador / RH / DP | Documentos do Trabalhador |
| **Não Periódico** | S-2206 | Alteração de Contrato de Trabalho | Contador / RH / DP | Aditivos Contratuais |
| **Não Periódico** | **S-2210** | **Comunicação de Acidente de Trabalho (CAT)** | **SST / Engenharia / RH** | **Atestado Médico e Análise de Acidente** |
| **Não Periódico** | **S-2220** | **Monitoramento da Saúde do Trabalhador (ASO)** | **Clínica de Medicina Ocupacional** | **Prontuário Médico / PCMSO** |
| **Não Periódico** | **S-2240** | **Condições Ambientais - Agentes Nocivos** | **Engenharia de Segurança / SST** | **LTCAT / PGR / Avaliações Ambientais** |
| **Não Periódico** | S-2299 | Desligamento | Contador / RH / DP | Termo de Rescisão de Contrato |
| **Periódico** | S-1200 | Remuneração de Trabalhador | Contador / Folha / DP | Folha de Pagamento Mensal |
| **Periódico** | S-1210 | Pagamentos de Rendimentos do Trabalho | Contador / Financeiro | Comprovantes de Quitação Bancária |
| **Periódico** | S-1299 | Fechamento dos Eventos Periódicos | Contador / DP | Apuração Mensal Consolidada |

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
