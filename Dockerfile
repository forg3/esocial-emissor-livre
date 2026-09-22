# Imagem base leve com glibc para compatibilidade total com SQLite CGO
FROM debian:bookworm-slim

# Instala certificados SSL/TLS da ICP-Brasil e timezone
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copia o binário compilado com assets embutidos
COPY dist/validador-esocial /app/validador-esocial

# Diretório de persistência para o banco de dados SQLite
RUN mkdir -p /app/dados && chmod 777 /app/dados

EXPOSE 8000

ENV PORTA=8000
ENV DADOS_DIR=/app/dados

# Metadados OCI para vinculação automática com o GitHub Packages do repositório
LABEL org.opencontainers.image.title="Validador eSocial"
LABEL org.opencontainers.image.description="Aplicativo autônomo para elaboração, validação e transmissão dos 36 eventos do eSocial"
LABEL org.opencontainers.image.url="https://github.com/forg3/validador-esocial"
LABEL org.opencontainers.image.source="https://github.com/forg3/validador-esocial"
LABEL org.opencontainers.image.version="v1.0-alpha"
LABEL org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/app/validador-esocial", "-porta", "8000", "-dados", "/app/dados", "-sem-navegador"]
