# Pente fino e pentest — 26/09/2026

Escopo: todo o código (local = GitHub, commit `202de28`), testes, análise estática
(`go vet`, `staticcheck`, `gosec`, `govulncheck`) e teste dinâmico da instância rodando
(binário compilado, `127.0.0.1`, pasta de dados temporária).

## O que resistiu (teste dinâmico)

| Ataque | Resultado |
|---|---|
| Acesso sem login | 303 para `/login` |
| DNS rebinding (cabeçalho `Host` forjado) | 421 |
| Força bruta no login (com token CSRF válido) | 8 tentativas → 429, inclusive para a senha certa até a janela expirar |
| Escrita sem token CSRF / com `Origin` de outro site | 403 / 403 |
| XXE (`<!ENTITY x SYSTEM "file:///etc/passwd">`) na importação de ASO e na validação | Nada vazou; o parser recusa a entidade |
| "Billion laughs" (bomba de entidades) | Recusado em milissegundos; memória estável (~25 MB) |
| Cabeçalhos | CSP restritiva, `X-Frame-Options: DENY`, `nosniff`, `no-referrer` |
| Dependências (`govulncheck`) | Nenhuma vulnerabilidade alcançável pelo código (1 em `x/crypto/openpgp`, pacote que o projeto não usa) |

## Achados e correções

| # | Achado | Gravidade | Correção |
|---|---|---|---|
| 1 | **A interface nunca fazia assinatura A1 real**: assinar (avulso e em lote) sempre gerava o envelope simulado, e a transmissão real exige assinatura real — o envio oficial anunciado na v1.1.1 não tinha como acontecer pela tela | Alta (funcional) | `assinarEvento`: com certificado A1 configurado e a **senha informada**, assina com `crypto.AssinarXML`; sem senha, continua simulado e dito como tal. Campo de senha aparece sempre que há certificado e vai junto nos botões de assinar |
| 2 | XML de ASO importado entrava na fila como **"pronto" sem validação** | Média | Importação valida (XSD oficial) e grava como **rejeitado** o que não passa, com o motivo |
| 3 | Assinar aceitava evento **rejeitado** ou nunca validado (avulso e em lote) | Média | Evento rejeitado não é assinado; toda assinatura revalida o XML antes |
| 4 | Tabelas 13/14/15/17/27 eram amostras (40/50/30/26/20 itens) | Média (funcional) | Tabelas completas do leiaute S-1.3 (NT 07/2026): 45/248/60/29/1.450 |
| 5 | Certificado enviado gravado com o **nome escolhido por quem envia** | Baixa (o Go já reduz a `filepath.Base`) | Nome fixo (`certificado-empresa.pfx` / `certificado.pfx`) e só `.pfx`/`.p12` |
| 6 | `xmllint` sem `--nonet` | Baixa | `--nonet` |
| 7 | Código morto (`oidCPF`, `soapFault`) e API obsoleta em teste (`pkcs12.Encode`) | Informativa | Removidos / `pkcs12.Modern.Encode` |

## Falsos positivos do `gosec` (conferidos à mão)

- G709 (`xml.Unmarshal`): o `encoding/xml` do Go não resolve entidades externas — confirmado no teste dinâmico.
- G703/G304 (caminho): componentes do caminho agora são fixos ou gerados (`perfil-<hash do CNPJ>`).
- G120: o formulário já é limitado (`ParseMultipartForm(4 MB)` + `MaxBytesReader`).
- G124 (cookie sem `Secure`): o app roda em `http://127.0.0.1`; `Secure` quebraria o login local.

## Pendente

- `gofmt`: 9 arquivos antigos fora do padrão de formatação (sem efeito de segurança).
- Testar a assinatura e a transmissão com **certificado A1 de verdade** em produção restrita
  quando o e-CNPJ for comprado.

## Testes novos

`internal/data/data_test.go` (tabelas completas) e `internal/web/fluxo_assinatura_test.go`
(ASO malformado não entra como pronto; evento rejeitado não é assinado; assinatura A1 real pela
interface validada com `crypto.ValidarAssinaturaXML`). Escritos antes da correção — os quatro
falhavam.
