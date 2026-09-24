### Validador eSocial — v1.1-alpha

Release de **segurança e qualidade** sobre a `v1.0-alpha`: uma auditoria completa de segurança foi executada e
**todos os achados foram corrigidos**, acompanhados de melhorias de usabilidade, endurecimento da configuração e
ampliação da suíte de testes. Continua sendo a versão de testes de conformidade com o Leiaute S-1.3 (NT 07/2026).

> **Atenção (importante):** nesta versão a **transmissão ao eSocial continua simulada** — nenhum dado é enviado ao
> governo e **nenhum recibo oficial é gerado**. Os eventos processados nesse fluxo recebem o status `simulado` e a
> interface exibe o aviso permanente "MODO SIMULAÇÃO". Não utilize os comprovantes para cumprimento de obrigação
> acessória até a integração real com o webservice oficial ser habilitada.

---

### Guia de Download por Plataforma

| Arquivo do Pacote | Sistema Operacional Alvo | Arquitetura / Processador |
| :--- | :--- | :--- |
| **`ghcr.io/forg3/validador-esocial:v1.1-alpha`** | Container OCI (Docker / Podman / Kubernetes) | Linux amd64 (GitHub Packages) |
| **`validador-esocial-v1.1-alpha-linux-amd64.tar.gz`** | Linux (Ubuntu, Debian, Fedora, RHEL, Arch) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.1-alpha-windows-amd64.zip`** | Windows 10, Windows 11, Windows Server | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.1-alpha-darwin-arm64.tar.gz`** | macOS (Sonoma, Sequoia ou superior) | **Apple Silicon (chips M1, M2, M3, M4, M5, M6+)** |
| **`validador-esocial-v1.1-alpha-darwin-amd64.tar.gz`** | macOS (High Sierra a Monterey ou superior) | Processadores **Intel 64-bit** |

---

### Correções de Segurança

Auditoria cobrindo isolamento/controle de acesso, permissões definidas no cliente, IDOR, exposição de segredos e
entradas sem tratamento (XSS/XML injection). Relatório completo com evidências em
[`docs/security-audit/relatorio-auditoria-seguranca.pdf`](docs/security-audit/relatorio-auditoria-seguranca.pdf).

- **Autenticação obrigatória em todas as rotas** — senha local com *salt* e 100.000 iterações de SHA-256, sessão em
  cookie `HttpOnly` + `SameSite=Strict`, tela de login, logout e expiração automática. *(achado crítico)*
- **Escuta restrita a `127.0.0.1` por padrão** — exposição na rede passa a ser explícita (`-host 0.0.0.0`), com aviso
  no log e validação do cabeçalho `Host` contra allowlist local (anti DNS rebinding). *(achado crítico)*
- **Proteção anti-CSRF** — token em todos os formulários e no cabeçalho `X-CSRF-Token` das requisições HTMX, com
  validação de `Origin`/`Referer` em todo método de escrita.
- **Fim da assinatura e dos recibos fictícios** — o envelope de demonstração é marcado como `ASSINATURA SIMULADA`, o
  status passa a ser `simulado` e nenhum protocolo/recibo é gravado sem transmissão real.
- **Correção de HTML injection/XSS** no teste de certificado (saída migrada para `html/template` com escaping
  contextual; função `safeHTML` removida).
- **Correção de XML injection** nos geradores de eventos (escape de todos os campos + validação de domínio: datas ISO,
  hora HHMM, UF oficial, códigos controlados).
- **Limites de entrada** — 2 MB (certificado `.pfx`), 5 MB (XML), 10 MB (CSV) e 12 MB por requisição de escrita.
- **Validação por schema real** — `xmllint` + XSD oficiais embutidos, com indicação explícita do modo executado e
  motivo da reprovação.
- **Rate limit** — 8 tentativas/minuto no login e 6/minuto na validação da senha do certificado A1, com registro em log.
- **Hardening de arquivos e container** — banco SQLite e certificados com `0600` (diretórios `0700`), imagem Docker
  executando como usuário sem privilégios (UID 10001) com `chmod 700` no diretório de dados.
- **Cabeçalhos de segurança** — CSP (`default-src 'self'`, `frame-ancestors 'none'`, `object-src 'none'`,
  `base-uri 'none'`, `form-action 'self'`), `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` e COOP.
- **Erros de persistência não são mais silenciados** — falhas de gravação/exclusão passam a ser reportadas ao usuário.

---

### Melhorias

- Filtro e contador de eventos **`simulado`** na fila de transmissão, com descrição clara de status.
- Senha de acesso configurável por `-senha` ou `ESOCIAL_SENHA`; na primeira execução é gerada e exibida no terminal
  (hash persistido em `dados/auth.json`, permissão `0600`).
- Mensagens de validação informam exatamente o modo executado e o que divergiu no documento.
- Suíte de testes ampliada: **13 novos testes de regressão de segurança** (`internal/web/seguranca_test.go`).
- Pipeline de CI agora instala o `xmllint`, executa `go build`, `go vet` e `go test ./...` antes de publicar a imagem,
  e compila o binário de forma estática (`CGO_ENABLED=0`).

---

### Como usar (container)

```bash
# Defina uma senha forte e publique a porta localmente
podman run -d --name validador-esocial \
  -p 127.0.0.1:8000:8000 \
  -e ESOCIAL_SENHA="SUA_SENHA_FORTE" \
  -v validador-dados:/app/dados \
  ghcr.io/forg3/validador-esocial:v1.1-alpha

# Acesse http://localhost:8000 e informe a senha definida acima
```

### Como usar (binário nativo)

```bash
./validador-esocial            # abre o navegador em http://localhost:8000
./validador-esocial -senha "SUA_SENHA_FORTE"
```

---

### Limitações conhecidas

- **Transmissão ao eSocial ainda simulada** (sem protocolo/recibo oficial) — o cliente mTLS já está implementado em
  `internal/soap`, porém não conectado à interface.
- **Leiautes do editor ainda não 100% aderentes ao XSD oficial S-1.3** — com a validação por schema agora ativa,
  divergências como `ideVinculo`, `agNoc` e códigos de órgão de classe são reportadas explicitamente; a adequação
  completa dos geradores está no roadmap.
- **Certificados A3 (token/smartcard)** ainda não suportados.
