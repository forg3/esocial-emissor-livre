### Validador eSocial — v1.0-alpha

Versão pública oficial do **Validador eSocial**, aplicativo web livre, autônomo e de código aberto para elaboração, validação estrutural contra esquemas XSD, certificação digital ICP-Brasil (A1) e transmissão oficial dos 36 eventos do eSocial (versão S-1.3 NT 07/2026).

---

### Guia de Download por Plataforma

| Arquivo do Pacote | Sistema Operacional Alvo | Arquitetura / Processador |
| :--- | :--- | :--- |
| **`validador-esocial-v1.0-alpha-windows-amd64.zip`** | Windows 10, Windows 11, Windows Server | 64-bit (x86_64 / amd64) |
| **`validador-esocial-v1.0-alpha-darwin-arm64.tar.gz`** | macOS (Sonoma, Sequoia ou superior) | **Apple Silicon (chips M1, M2, M3, M4, M5, M6+)** |
| **`validador-esocial-v1.0-alpha-darwin-amd64.tar.gz`** | macOS (High Sierra a Monterey ou superior) | Processadores **Intel 64-bit** |
| **`validador-esocial-v1.0-alpha-linux-amd64.tar.gz`** | Linux (Ubuntu, Debian, Fedora, RHEL, Arch) | 64-bit (x86_64 / amd64) |
| **`ghcr.io/forg3/validador-esocial:v1.0-alpha`** | Container OCI (Docker / Podman / Kubernetes) | Linux amd64 (GitHub Packages) |

---

### Como usar

Para executar o sistema em qualquer máquina externa sem Go, bibliotecas ou ambientes de desenvolvimento instalados, acesse a página de Releases https://github.com/forg3/validador-esocial/releases/tag/v1.0-alpha e efetue o download do arquivo compactado correspondente ao sistema operacional da máquina:

- **Windows (x86_64):** `validador-esocial-v1.0-alpha-windows-amd64.zip`
- **macOS com Apple Silicon (M1 a M6):** `validador-esocial-v1.0-alpha-darwin-arm64.tar.gz`
- **macOS Intel:** `validador-esocial-v1.0-alpha-darwin-amd64.tar.gz`
- **Linux:** `validador-esocial-v1.0-alpha-linux-amd64.tar.gz`

Descompacte o pacote em qualquer pasta do computador (como a Área de Trabalho). No Windows, basta um duplo clique no arquivo `validador-esocial.exe`. No macOS ou Linux, certifique-se de que a permissão de execução está concedida via terminal com `chmod +x validador-esocial*` e inicialize o executável.

O programa é auto-contido: ele instancia o banco de dados SQLite local, inicia o serviço HTTP e dispara automaticamente a abertura da interface no navegador padrão em `http://localhost:8000`. Para encerrar, basta fechar o prompt ou pressionar `Ctrl+C`.

---

### Principais Funcionalidades

- **Zero Dependências:** Binário 100% estático com banco de dados SQLite embutido e arquivos web (CSS, HTMX, imagens) servidos de memória interna.
- **Abertura Automática do Navegador:** Abre o painel automaticamente em `http://localhost:8000` logo após a inicialização.
- **Cobertura Integral dos 36 Eventos Oficiais:** Organizados em 6 grupos funcionais (SESMT, Medicina Ocupacional, RH & DP, Folha & Contabilidade, Jurídico e RPPS).
- **Validação de Leiaute S-1.3:** Motor de validação contra schemas XSD oficiais do governo federal.
- **Assinatura Digital ICP-Brasil:** Suporte nativo para certificados A1 (.pfx / .p12) e preparação para A3 (token PKCS#11).
- **Importação em Lote:** Importador de colaboradores via planilha CSV com modelo oficial para download.
