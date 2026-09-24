### Validador eSocial — v1.1.1-alpha

Release de **evolução funcional** sobre a v1.1-alpha: conformidade total dos leiautes com o XSD oficial, auditoria de
retorno do eSocial, gestão multiempresa com múltiplos certificados e **transmissão real opcional** ao webservice oficial.
Continua sendo a versão de testes de conformidade com o Leiaute S-1.3 (NT 07/2026).

---

### Guia de Download por Plataforma

| Arquivo do Pacote | Sistema Operacional Alvo | Arquitetura / Processador |
| :--- | :--- | :--- |
| **`ghcr.io/forg3/validador-esocial:v1.1.1-alpha`** | Container OCI (Docker / Podman / Kubernetes) | Linux amd64 (GitHub Packages) |
| **`validador-esocial_1.1.1-alpha_linux_amd64.deb`** | Debian, Ubuntu, Mint e derivados | Linux amd64 (arm64 também disponível) |
| **`validador-esocial-1.1.1-alpha.x86_64.rpm`** | Fedora, RHEL, CentOS, openSUSE | Linux amd64 (arm64 também disponível) |
| **`validador-esocial-1.1.1-alpha-windows-amd64.msi`** | Windows 10/11 e Windows Server (instalador) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.1.1-alpha-linux-amd64.tar.gz`** | Linux (Ubuntu, Debian, Fedora, RHEL, Arch) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.1.1-alpha-windows-amd64.zip`** | Windows 10, Windows 11, Windows Server | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.1.1-alpha-darwin-arm64.tar.gz`** | macOS (Sonoma, Sequoia ou superior) | **Apple Silicon (M1 a M6+)** |
| **`validador-esocial-v1.1.1-alpha-darwin-amd64.tar.gz`** | macOS (High Sierra a Monterey ou superior) | Processadores **Intel 64-bit** |

Todas as releases incluem `SHA256SUMS.txt` para verificação de integridade.

---

### O que mudou nesta versão

#### 1. Leiautes S-1.3 aderentes ao XSD oficial (Sprint 1)

Os geradores de **S-2210 (CAT)**, **S-2220 (ASO)** e **S-2240 (condições ambientais)** foram reescritos para o leiaute
oficial. Antes, os documentos eram reprovados pelo schema (`ideTrabalhador` em vez de `ideVinculo`, `agenteNoc` em vez de
`agNoc`, blocos obrigatórios ausentes).

- `ideVinculo` (CPF + matrícula) em todos os eventos
- **S-2240**: `agNoc`, `epcEpi` completo (`utilizEPC`/`eficEpc`/`utilizEPI`/`eficEpi`/`epi`/`epiCompl`) e `respReg` com código de órgão de classe; campos quantitativos (`intConc`/`unMed`/`tecMedicao`) quando a avaliação é quantitativa
- **S-2210**: `codSitGeradora`, `iniciatCAT`, `obsCAT`, `ultDiaTrab`, `houveAfast`, `localAcidente` completo, `parteAtingida` + `lateralidade`, `agenteCausador` e `atestado` completo com `emitente`
- **S-2220**: `exMedOcup`/`aso`/`exame`/`medico`/`respMonit`, com `tpExameOcup` restrito aos códigos oficiais (0 a 4)
- **Tabelas oficiais embutidas** e selecionáveis nos formulários: 13 (parte do corpo), 14 (agente causador), 15 (situação geradora), 17 (natureza da lesão) e 27 (procedimentos diagnósticos)
- Validação por `xmllint` contra os XSD oficiais aprovando os três eventos gerados pela interface

#### 2. Relatórios e auditoria de retorno — S-5001/S-5011 (Sprint 3)

Nova área **Relatórios** para conferência dos totalizadores devolvidos pelo eSocial:

- Importação do XML de retorno (S-5001/S-5002/S-5003 por trabalhador e S-5011/S-5012/S-5013 consolidados), com limite de 10 MB
- Consolidação por período de apuração: previdenciário, FGTS, IRRF e total conferido
- Memórias de cálculo por campo (`vr*`), identificação do trabalhador (CPF/matrícula) e número do recibo base
- Exportação **CSV** (com BOM para Excel) para conferência externa

#### 3. Múltiplos certificados e procurações eletrônicas (Sprint 4)

Nova área **Empresas e Certificados**:

- Cadastro de múltiplos perfis (CNPJ, razão social, ambiente e tipo de certificado)
- Certificado A1 por perfil, gravado com permissão `0600` em `dados/certificados/<perfil>/`
- Dados do **procurador eletrônico** (nome e CPF/CNPJ) por perfil
- Ativação de perfil com sincronização automática do CNPJ/ambiente/certificado usados na emissão
- Proteção contra exclusão do perfil ativo

#### 4. Transmissão real ao eSocial (Sprint 5)

- Seletor de **modo de transmissão** na tela de certificado: **Simulado** (padrão) ou **Real**
- No modo real o envio usa o webservice oficial com **mTLS + certificado A1** (`EnviarLoteEventos`), persistindo protocolo e mensagem oficiais, e a consulta de recibo usa `ConsultarLoteEventos`
- A senha do certificado é solicitada a cada envio e **não é armazenada**
- Guardas de segurança: recusa envio sem assinatura XMLDSig, recusa assinatura simulada e exige certificado válido
- Endpoints sobrescrevíveis por `ESOCIAL_WS_ENVIO` / `ESOCIAL_WS_CONSULTA` (homologação, proxy corporativo ou testes)
- Fluxo completo validado por teste automatizado com webservice simulado (mTLS → protocolo → recibo)

#### 5. Melhorias e correções

- Versão embutida no binário (`-X main.versao`) e exibida no banner de inicialização
- Pipeline de release com GoReleaser (binários, `.deb`, `.rpm`, checksums) e instalador `.msi` via WiX
- Documentação atualizada (README) e relatório de auditoria de segurança revisado
- 40 testes automatizados no total, incluindo conformidade XSD, relatórios, perfis e transmissão real

---

### Como usar (container)

```bash
podman run -d --name validador-esocial \
  -p 127.0.0.1:8000:8000 \
  -e ESOCIAL_SENHA="SUA_SENHA_FORTE" \
  -v validador-dados:/app/dados \
  ghcr.io/forg3/validador-esocial:v1.1.1-alpha
```

### Como usar (pacote nativo)

```bash
sudo dpkg -i validador-esocial_1.1.1-alpha_linux_amd64.deb   # Debian/Ubuntu
sudo rpm -i validador-esocial-1.1.1-alpha.x86_64.rpm          # Fedora/RHEL/openSUSE
# Windows: execute validador-esocial-1.1.1-alpha-windows-amd64.msi
validador-esocial -senha "SUA_SENHA_FORTE"
```

---

### Limitações conhecidas

- **Transmissão real requer certificado A1 válido e homologação prévia**: o envio oficial está implementado e testado
  com webservice simulado, mas a validação final depende de um certificado real e do ambiente de Produção Restrita.
- **Certificados A3 (token/smartcard)** ainda não suportados (roadmap).
- O modo de transmissão padrão permanece **simulado**, com aviso visível na interface.
