# ═══════════════════════════════════════════════════════════════
# Dockerfile — Entorno aislado para testing de ZyroAgentCLI v2
# ═══════════════════════════════════════════════════════════════
# Build:   docker build -t zyrocli-test -f test/Dockerfile .
# Run:     docker run --rm -it zyrocli-test
#
# Nota: Crea un .dockerignore con:
#   .git/
#   .venv/
#   node_modules/
#   __pycache__/
#   *.pyc
#   zyrocli
#   .helix/
# ═══════════════════════════════════════════════════════════════

FROM ubuntu:24.04

LABEL description="ZyroAgentCLI v2 — E2E test environment"
LABEL maintainer="ZyroCLI Team"

# ── Evitar prompts interactivos de apt ─────────────────────────
ENV DEBIAN_FRONTEND=noninteractive

# ── Instalar dependencias base del sistema ─────────────────────
# NOTA: NO instalamos HelixDB, OpenCode, ni Ollama aquí.
#       Deben instalarse vía `zyro setup` (test de instalación).
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    git \
    ca-certificates \
    python3 \
    python3-pip \
    python3-venv \
    nodejs \
    npm \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# ── Instalar Go 1.22 (Ubuntu 22.04 ships Go 1.18, muy antiguo) ─
# El proyecto requiere Go 1.21+ según go.mod.
RUN curl -sL https://go.dev/dl/go1.23.0.linux-amd64.tar.gz \
    | tar -C /usr/local -xz && \
    /usr/local/go/bin/go version

ENV PATH=$PATH:/usr/local/go/bin:/root/.local/bin:/root/.cargo/bin

# ── Instalar Docker CLI cliente (binario estático) ─────────────
# Solo el cliente, no el daemon. Suficiente para `zyro doctor`.
RUN curl -sL https://download.docker.com/linux/static/stable/x86_64/docker-24.0.7.tgz \
    | tar -xz && \
    mv docker/docker /usr/local/bin/docker && \
    rm -rf docker && \
    docker --version

# ── Verificar Node + npx (necesario para install.sh) ─────────
RUN node --version && npm --version && npx --version

# ── Copiar repositorio al contenedor ───────────────────────────
COPY . /repo
WORKDIR /repo

# ── Compilar zyrocli por adelantado ────────────────────────────
# El binario se copia a /usr/local/bin/zyrocli para que esté en PATH.
RUN go build -o /usr/local/bin/zyrocli ./cmd/zyrocli && \
    chmod +x /usr/local/bin/zyrocli && \
    zyrocli --help > /dev/null 2>&1 && \
    echo "[BUILD] zyrocli compilado correctamente"

# ── Entrypoint: ejecuta la batería de tests ────────────────────
CMD ["bash", "test/e2e_test.sh"]
