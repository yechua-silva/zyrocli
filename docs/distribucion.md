# Distribución de ZyroAgentCLI

> **Fecha:** 2026-06-15
> **Propósito:** Investigar y definir estrategias para publicar ZyroAgentCLI como binario instalable desde npm, GitHub Releases, Homebrew, go install y otros canales.
> **Contexto:** ZyroAgentCLI es un binario Go (`module github.com/yechua-silva/zyrocli`). Se compila con `go build -o zyrocli ./cmd/zyrocli`. Actualmente solo se distribuye via `git clone + ./scripts/install.sh`.

---

## Tabla de Contenidos

1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Opción 1: GitHub Releases + Go专用 CI/CD](#opción-1-github-releases--go-ci-cd)
3. [Opción 2: npm package (wrapper con binarios pre-compilados)](#opción-2-npm-package-wrapper-con-binarios-pre-compilados)
4. [Opción 3: go install (nativo Go)](#opción-3-go-install-nativo-go)
5. [Opción 4: Homebrew tap](#opción-4-homebrew-tap)
6. [Opción 5: curl | bash (script existente)](#opción-5-curl--bash-script-existente)
7. [Comparativa y Veredicto](#comparativa-y-veredicto)
8. [Plan de Implementación Recomendado](#plan-de-implementación-recomendado)
9. [Apéndice: Ejemplos Reales](#apéndice-ejemplos-reales)
10. [Apéndice: Publicación en npm con cuenta personal](#apéndice-publicación-en-npm-con-cuenta-personal)

---

## Resumen Ejecutivo

ZyroAgentCLI necesita **múltiples canales de distribución** porque distintos usuarios tienen distintas preferencias:

| Canal | Perfil de usuario |
|-------|-------------------|
| `npx zyrocli` | Usuarios Node.js, sin Go, quieren probar rápido |
| `brew install zyrocli` | Usuarios macOS (Homebrew es el estándar de facto) |
| `go install github.com/yechua-silva/zyrocli@latest` | Usuarios Go, ya tienen toolchain |
| `curl -sSL https://zyro.dev/install.sh \| bash` | Usuarios Linux, script tradicional |
| GitHub Releases (descarga directa) | Cualquier usuario, CI/CD pipelines |

**Recomendación principal:** Automatizar GitHub Releases con Actions como fuente de verdad, y luego construir los demás canales como wrappers que descargan desde allí.

---

## Opción 1: GitHub Releases + Go CI/CD

### Descripción

El flujo más estándar para proyectos Go: un workflow de GitHub Actions compila el binario para múltiples plataformas y lo sube como asset de una Release.

### Plataformas objetivo

| Plataforma | GOOS | GOARCH | Binario |
|-----------|------|--------|---------|
| Linux x86_64 | `linux` | `amd64` | `zyrocli-linux-amd64` |
| Linux ARM64 | `linux` | `arm64` | `zyrocli-linux-arm64` |
| macOS Intel | `darwin` | `amd64` | `zyrocli-darwin-amd64` |
| macOS Apple Silicon | `darwin` | `arm64` | `zyrocli-darwin-arm64` |
| Windows x86_64 | `windows` | `amd64` | `zyrocli-windows-amd64.exe` |

### Workflow propuesto (`.github/workflows/release.yml`)

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'  # v0.1.0, v1.0.0, etc.

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### GoReleaser config (`.goreleaser.yaml`)

```yaml
# .goreleaser.yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: zyrocli
    main: ./cmd/zyrocli
    binary: zyrocli
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: zyrocli-bin
    format: binary
    name_template: "{{ .Binary }}-{{ .Os }}-{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"

checksum:
  name_template: 'checksums.txt'

release:
  draft: false
  prerelease: auto
  header: |
    ## ZyroCLI {{ .Version }}
    > {{ .Date }}
  footer: |
    ### Instalación
    ```bash
    # Con Go (si tenés Go instalado)
    go install github.com/yechua-silva/zyrocli@{{ .Tag }}

    # Con npm
    npx zyrocli@{{ .Version }}

    # macOS (Homebrew)
    brew install secko/tap/zyrocli
    ```

changelog:
  use: github
  groups:
    - title: "🚀 Features"
      regexp: '^.*?feat(\([^)]+\))?!?:.*$'
      order: 0
    - title: "🐛 Bug fixes"
      regexp: '^.*?fix(\([^)]+\))?!?:.*$'
      order: 1
    - title: "🧰 Maintenance"
      order: 2
```

### Commands manuales (sin GoReleaser)

Si se prefiere evitar GoReleaser, se puede usar `gh release create`:

```bash
# Compilar para todas las plataformas
GOOS=linux GOARCH=amd64 go build -o dist/zyrocli-linux-amd64 ./cmd/zyrocli
GOOS=darwin GOARCH=amd64 go build -o dist/zyrocli-darwin-amd64 ./cmd/zyrocli
GOOS=darwin GOARCH=arm64 go build -o dist/zyrocli-darwin-arm64 ./cmd/zyrocli
GOOS=windows GOARCH=amd64 go build -o dist/zyrocli-windows-amd64.exe ./cmd/zyrocli

# Generar checksums
cd dist && sha256sum * > checksums.txt

# Crear release y subir assets
gh release create v0.1.0 \
  --title "v0.1.0" \
  --notes "Release notes..." \
  dist/*
```

### Pros

- ✅ **Estándar de la industria Go.** Todos los proyectos Go importantes usan este mecanismo.
- ✅ **Sin dependencias externas.** GitHub Actions + GitHub Releases es todo lo que se necesita.
- ✅ **Verificable.** Checksums SHA256 para cada binario.
- ✅ **Fuente de verdad.** Otros canales (npm, Homebrew) pueden descargar desde aquí.
- ✅ **Automático.** Se dispara con `git tag v0.1.0 && git push --tags`.

### Contras

- ❌ No es instalable directamente con `npx` (los usuarios Node.js esperan `npx zyrocli`).
- ❌ No hay un comando de instalación estándar (cada usuario debe descargar manualmente).

---

## Opción 2: npm package (wrapper con binarios pre-compilados)

### Descripción

Publicar un paquete npm que actúa como wrapper: el `bin` apunta a un script JS que detecta la plataforma y ejecuta el binario Go correcto. El binario se descarga en `postinstall` desde GitHub Releases.

Este es el patrón usado por **esbuild** (Go), **sass** (Dart), **quicktype** (Haskell) y otros.

### Dos variantes

#### Variante A: Monopaquete con descarga en postinstall (más simple)

Un solo paquete `zyrocli` en npm. El `postinstall.js` descarga el binario correcto.

```
zyrocli/
├── package.json
├── bin/
│   └── zyrocli.js          ← entry point (shebang node)
├── scripts/
│   └── postinstall.js      ← descarga binario según plataforma
├── lib/
│   └── platform.js         ← mapeo plataforma → nombre binario
└── README.md
```

**package.json:**
```json
{
  "name": "zyrocli",
  "version": "0.1.0",
  "description": "ZyroAgentCLI — Gentle AI orchestration for structured SDD workflows",
  "bin": {
    "zyrocli": "bin/zyrocli.js"
  },
  "scripts": {
    "postinstall": "node scripts/postinstall.js"
  },
  "repository": {
    "type": "git",
    "url": "git+https://github.com/yechua-silva/zyrocli.git"
  },
  "engines": {
    "node": ">=18"
  },
  "license": "MIT"
}
```

**bin/zyrocli.js:**
```js
#!/usr/bin/env node
const { execFileSync } = require('child_process');
const path = require('path');

const binPath = path.join(__dirname, '..', 'bin', 'zyrocli');

try {
  execFileSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
} catch (err) {
  process.exit(err.status || 1);
}
```

**scripts/postinstall.js:**
```js
#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const https = require('https');
const { createHash } = require('crypto');

const PLATFORM_MAP = {
  'linux-x64':      'zyrocli-linux-amd64',
  'linux-arm64':    'zyrocli-linux-arm64',
  'darwin-x64':     'zyrocli-darwin-amd64',
  'darwin-arm64':   'zyrocli-darwin-arm64',
  'win32-x64':      'zyrocli-windows-amd64.exe',
};

function getPlatformKey() {
  const os = process.platform;   // 'darwin', 'linux', 'win32'
  const arch = process.arch;     // 'x64', 'arm64'
  if (os === 'win32' && arch === 'x64') return 'win32-x64';
  if (arch === 'x64') return `${os}-x64`;
  if (arch === 'arm64') return `${os}-arm64`;
  throw new Error(`Unsupported platform: ${os} ${arch}`);
}

const pkg = require('../package.json');
const version = pkg.version;
const platformKey = getPlatformKey();
const binaryName = PLATFORM_MAP[platformKey];
const url = `https://github.com/yechua-silva/zyrocli/releases/download/v${version}/${binaryName}`;
const dest = path.join(__dirname, '..', 'bin', 'zyrocli');

if (!fs.existsSync(path.dirname(dest))) {
  fs.mkdirSync(path.dirname(dest), { recursive: true });
}

console.log(`[zyrocli] Downloading ${binaryName}...`);

https.get(url, (res) => {
  if (res.statusCode === 302 || res.statusCode === 301) {
    // Follow redirect
    https.get(res.headers.location, writeStream);
    return;
  }
  if (res.statusCode !== 200) {
    console.error(`[zyrocli] Download failed: HTTP ${res.statusCode}`);
    process.exit(1);
  }
  writeStream(res);
}).on('error', (err) => {
  console.error(`[zyrocli] Download error: ${err.message}`);
  process.exit(1);
});

function writeStream(stream) {
  const file = fs.createWriteStream(dest, { mode: 0o755 });
  stream.pipe(file);
  file.on('finish', () => {
    console.log(`[zyrocli] Installed to ${dest}`);
  });
  file.on('error', (err) => {
    console.error(`[zyrocli] Write error: ${err.message}`);
    process.exit(1);
  });
}
```

**Uso:**
```bash
# Instalación global
npm install -g zyrocli
zyrocli --help

# Ejecución directa sin instalar
npx zyrocli setup
```

#### Variante B: Arquitectura con optionalDependencies (estilo esbuild, más robusta)

Usar **optionalDependencies** para que npm maneje la descarga del binario correcto. Cada plataforma tiene su propio sub-paquete.

```
zyrocli/                          ← paquete principal
├── package.json                  ← depende de @zyrocli/linux-x64 etc.
├── bin/esbuild                   ← JS shim que detecta y ejecuta
└── install.js                    ← fallback si optionalDeps fallan

@zyrocli/linux-x64/              ← paquete de plataforma específica
├── package.json
└── bin/zyrocli                  ← binario compilado

@zyrocli/darwin-arm64/
├── package.json
└── bin/zyrocli

@zyrocli/darwin-amd64/
├── package.json
└── bin/zyrocli

@zyrocli/win32-x64/
├── package.json
└── zyrocli.exe
```

**package.json (principal):**
```json
{
  "name": "@secko/zyrocli",
  "version": "0.1.0",
  "bin": {
    "zyrocli": "bin/zyrocli.js"
  },
  "optionalDependencies": {
    "@zyrocli/linux-x64": "0.1.0",
    "@zyrocli/linux-arm64": "0.1.0",
    "@zyrocli/darwin-x64": "0.1.0",
    "@zyrocli/darwin-arm64": "0.1.0",
    "@zyrocli/win32-x64": "0.1.0"
  }
}
```

**package.json (por plataforma, ej: `@zyrocli/linux-x64/package.json`):**
```json
{
  "name": "@zyrocli/linux-x64",
  "version": "0.1.0",
  "os": ["linux"],
  "cpu": ["x64"],
  "bin": {
    "zyrocli": "bin/zyrocli"
  }
}
```

**Ventaja:** npm/yarn/pnpm resuelven las optionalDependencies automáticamente. Si la plataforma coincide, npm descarga el paquete; si no, lo omite silenciosamente.

**Desventaja:** Requiere publicar **5+ paquetes** en npm (uno por plataforma + el principal). Más mantenimiento.

### Referencia: esbuild como caso de estudio

| Aspecto | esbuild |
|---------|---------|
| Lenguaje original | Go |
| Paquete npm principal | `esbuild` (10M+ semanales) |
| Sub-paquetes | 24 paquetes `@esbuild/*` (una por plataforma) |
| Mecanismo | `optionalDependencies` + `install.js` |
| Entry point | `bin/esbuild` (JS shim) |
| Hash verification | SHA256 checksums en `package.json` |

esbuild define `esbuild.binaryHashes` en su `package.json` para verificar la integridad del binario descargado:

```json
{
  "esbuild.binaryHashes": {
    "@esbuild/darwin-arm64/bin/esbuild": "e2dc9a52440a2a34f09434...",
    "@esbuild/linux-x64/bin/esbuild": "0c6588b092a2c291a72bab..."
  }
}
```

### Pros y Contras de npm

| Aspecto | Variante A (postinstall) | Variante B (optionalDeps) |
|---------|-------------------------|---------------------------|
| Complejidad | Baja | Alta |
| Paquetes a publicar | 1 | 5+ |
| Fiabilidad de descarga | Buena (propio código) | Excelente (npm engine) |
| Soporte offline | No | Sí (si ya se instaló antes) |
| Mantenimiento | Bajo | Medio |
| Verificación integridad | Manual | Checksums nativos |

---

## Opción 3: go install (nativo Go)

### Descripción

El mecanismo nativo de Go para instalar binarios:

```bash
go install github.com/yechua-silva/zyrocli@latest
```

Esto clona el repositorio, compila con la toolchain local de Go y coloca el binario en `$GOPATH/bin/zyrocli` (o `$HOME/go/bin/zyrocli`).

### Requisitos

- Go 1.26+ instalado
- `$GOPATH/bin` en el `PATH`

### Cómo funciona

1. `go install github.com/yechua-silva/zyrocli@latest` resuelve el último commit de `main` (o usa `@v0.1.0` para una versión específica).
2. Detecta `./cmd/zyrocli` como main package (Go lo infiere del module path).
3. Compila con `CGO_ENABLED=0` por defecto (binario estático).
4. Instala en `$GOPATH/bin/zyrocli`.

### Versionado semántico

```bash
go install github.com/yechua-silva/zyrocli@v0.1.0   # versión específica
go install github.com/yechua-silva/zyrocli@latest    # último commit de main
go install github.com/yechua-silva/zyrocli@main      # commit actual de main
```

Para que `@latest` funcione, se necesita un tag semver en el repositorio:

```bash
git tag v0.1.0
git push origin v0.1.0
```

### Pros

- ✅ **Cero config.** No requiere npm, Homebrew, ni repositorios externos.
- ✅ **Siempre actualizado.** Compila desde fuente, no hay binarios pre-compilados que mantener.
- ✅ **Seguro.** Verificación criptográfica vía Go module proxy (sum.golang.org).
- ✅ **Rápido.** `go install` es cacheable, segunda instalación es instantánea.

### Contras

- ❌ **Requiere Go toolchain.** El usuario debe tener Go instalado (aprox. 150 MB).
- ❌ **Compilación inicial lenta.** Compilar todo el proyecto puede tomar 10–30 segundos.
- ❌ **No funciona con `npx`.** Los usuarios Node.js no pueden usarlo directamente.

---

## Opción 4: Homebrew tap

### Descripción

Homebrew es el gestor de paquetes estándar en macOS (y cada vez más popular en Linux). Un **tap** es un repositorio de fórmulas Homebrew.

### Estructura

```
homebrew-tap/                    ← repositorio GitHub: secko/homebrew-tap
└── Formula/
    └── zyrocli.rb               ← fórmula Homebrew
```

### Fórmula

```ruby
# Formula/zyrocli.rb
class Zyrocli < Formula
  desc "Gentle AI orchestration for structured SDD workflows"
  homepage "https://github.com/yechua-silva/zyrocli"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/yechua-silva/zyrocli/releases/download/v#{version}/zyrocli-darwin-arm64"
      sha256 "abc123..."  # ← checksum SHA256 del binario
    else
      url "https://github.com/yechua-silva/zyrocli/releases/download/v#{version}/zyrocli-darwin-amd64"
      sha256 "def456..."
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/yechua-silva/zyrocli/releases/download/v#{version}/zyrocli-linux-arm64"
      sha256 "ghi789..."
    else
      url "https://github.com/yechua-silva/zyrocli/releases/download/v#{version}/zyrocli-linux-amd64"
      sha256 "jkl012..."
    end
  end

  def install
    bin.install Dir["*"].first => "zyrocli"
  end

  test do
    assert_match "ZyroCLI", shell_output("#{bin}/zyrocli --help")
  end
end
```

### Instalación

```bash
# Agregar el tap (solo una vez)
brew tap secko/tap

# Instalar
brew install zyrocli

# Actualizar
brew upgrade zyrocli
```

### Automatización con GitHub Actions

Homebrew tiene un sistema de **bump Formula** que permite actualizar la fórmula automáticamente cuando se crea una Release:

```yaml
# En el workflow de release, después de publicar:
- name: Update Homebrew formula
  uses: dawidd6/action-homebrew-bump-formula@v4
  with:
    token: ${{ secrets.HOMEBREW_TAP_TOKEN }}
    tap: secko/homebrew-tap
    formula: zyrocli
```

### Pros

- ✅ **Estándar en macOS.** Los usuarios macOS prefieren `brew install` sobre cualquier otra cosa.
- ✅ **Fácil de actualizar.** `brew upgrade zyrocli` es intuitivo.
- ✅ **Binarios pre-compilados.** No requiere compilar desde fuente.

### Contras

- ❌ **Requiere repositorio separado.** `secko/homebrew-tap` debe crearse y mantenerse.
- ❌ **Poco usado en Linux.** Los usuarios Linux prefieren otros métodos.
- ❌ **SHA256 manual.** Cada release requiere actualizar checksums en la fórmula (automatizable).

---

## Opción 5: curl | bash (script existente)

### Descripción

El script `scripts/install.sh` ya existe y funciona. Se puede exponer públicamente:

```bash
curl -sSL https://zyro.dev/install.sh | bash
```

O desde GitHub raw:

```bash
curl -sSL https://raw.githubusercontent.com/secko/zyrocli/main/scripts/install.sh | bash
```

### Estado actual

El script actual (`scripts/install.sh`) hace:
1. Compila el binario con `go build`
2. Lo mueve a `~/.local/bin/zyrocli`
3. Instala HelixDB si no está presente
4. Ejecuta `zyrocli install`

### Mejoras sugeridas

```bash
# scripts/install.sh — versión mejorada para distribución pública
set -euo pipefail

REPO="secko/zyrocli"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detectar plataforma
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS-$ARCH" in
  linux-x86_64)  BINARY="zyrocli-linux-amd64" ;;
  linux-aarch64) BINARY="zyrocli-linux-arm64" ;;
  darwin-x86_64) BINARY="zyrocli-darwin-amd64" ;;
  darwin-arm64)  BINARY="zyrocli-darwin-arm64" ;;
  *) echo "Unsupported: $OS-$ARCH"; exit 1 ;;
esac

# Descargar binario pre-compilado (en lugar de compilar)
if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/$BINARY"
else
  URL="https://github.com/$REPO/releases/download/v$VERSION/$BINARY"
fi

echo "⬇️  Downloading $BINARY..."
curl -sSL "$URL" -o "$INSTALL_DIR/zyrocli"
chmod +x "$INSTALL_DIR/zyrocli"
echo "✅ Installed to $INSTALL_DIR/zyrocli"
```

### Pros

- ✅ **Ya existe.** Solo requiere exponerlo públicamente.
- ✅ **Universal.** Funciona en cualquier Linux/macOS.
- ✅ **Simple.** Una línea para instalar.

### Contras

- ❌ **Seguridad.** `curl | bash` es considerado antipatrón por algunos (mitigable con checksums).
- ❌ **Requiere Go toolchain** en la versión actual (mejorable descargando binario).
- ❌ **No funciona en Windows** sin WSL.
- ❌ **No hay desinstalador.** El usuario debe eliminar manualmente.

---

## Comparativa y Veredicto

### Matriz de evaluación

| Opción | Facilidad instalación | Alcance (OS) | Mantenimiento | Velocidad instalación | Seguridad | Confiabilidad | Veredicto |
|--------|----------------------|--------------|---------------|----------------------|-----------|---------------|-----------|
| **GitHub Releases** | 🟡 Medio | 🌐 Todos | ✅ Automático | ✅ Rápido (binario) | ✅ Alta | ✅ Alta | ★ Fuente de verdad |
| **npm (postinstall)** | ✅ Fácil (`npx`) | 🌐 Todos | 🟡 Medio | ✅ Rápido (binario) | 🟡 Media | ✅ Alta | ★★ Primario |
| **go install** | 🟡 Medio (requiere Go) | 🟡 Solo con Go | ✅ Mínimo | ❌ Lento (compila) | ✅ Alta | ✅ Alta | ★ Complemento |
| **Homebrew tap** | ✅ Fácil (`brew install`) | 🟡 macOS/Linux | 🟡 Medio | ✅ Rápido (binario) | ✅ Alta | ✅ Alta | ★ Opcional |
| **curl \| bash** | ✅ Fácil | 🟡 Linux/macOS | ✅ Mínimo | ✅ Rápido (binario) | ❌ Baja | 🟡 Media | ★ Legado |

### Factores de decisión

| Factor | Peso | Decisión |
|--------|------|----------|
| Usuarios target (Node.js vs Go devs) | Alto | npm es el canal más universal |
| Esfuerzo de mantenimiento | Alto | Automatizar todo con Actions |
| Seguridad | Alto | Checksums + firmas |
| Velocidad de instalación | Medio | Binarios pre-compilados, no compilar |
| Descubrimiento | Medio | npm tiene más visibilidad que GitHub Releases |

### Veredicto

```
Recomendación: Implementar en orden de prioridad:

  1º npm package (Variante A - postinstall)   ← Canal principal (npx zyrocli)
  2º GitHub Releases + GoReleaser             ← Fuente de verdad para todos los canales
  3º go install                               ← Complemento para usuarios Go
  4º Homebrew tap                             ← Opcional (macOS)
  5º curl | bash                              ← Script legacy (mejorar)
```

**Justificación:** El npm package es el canal que más valor agrega porque permite `npx zyrocli` (el entry point más bajo de fricción para cualquier desarrollador). GitHub Releases es la infraestructura base que alimenta a npm, Homebrew y curl|bash.

---

## Plan de Implementación Recomendado

### Fase 1: Infraestructura base (GitHub Actions + GoReleaser)

**Estimación:** 1–2 días

```
[ ] Crear .github/workflows/release.yml (GoReleaser)
[ ] Crear .goreleaser.yaml
[ ] Agregar version ldflags (-X main.version=...)
[ ] Agregar variable Version en main.go
[ ] Probar: git tag v0.1.0-alpha → verify release
```

### Fase 2: npm package

**Estimación:** 1–2 días

```
[ ] Crear dist/npm/package.json
[ ] Crear dist/npm/bin/zyrocli.js (JS shim)
[ ] Crear dist/npm/scripts/postinstall.js (descarga desde GitHub Releases)
[ ] Agregar .npmignore
[ ] Probar: npm pack → npm install -g ./zyrocli-0.1.0.tgz
[ ] npm publish --access public (con cuenta personal)
```

### Fase 3: Mejora del script install.sh

**Estimación:** 0.5 días

```
[ ] Modificar scripts/install.sh para descargar binario pre-compilado
[ ] Agregar flag VERSION para elegir versión
[ ] Exponer en zyro.dev/install.sh
```

### Fase 4 (opcional): Homebrew tap

**Estimación:** 0.5 días

```
[ ] Crear repositorio secko/homebrew-tap
[ ] Crear Formula/zyrocli.rb
[ ] Agregar bump action al workflow de release
```

---

## Apéndice: Ejemplos Reales

### esbuild (Go → npm) — Arquitectura

```
npm package "esbuild"
├── package.json
│   ├── "bin": { "esbuild": "bin/esbuild" }
│   ├── "optionalDependencies": {
│   │   "@esbuild/darwin-arm64": "0.28.1",
│   │   "@esbuild/linux-x64": "0.28.1",
│   │   ...
│   │ }
│   └── "esbuild.binaryHashes": { ... }
├── bin/esbuild         ← JS shim: detecta OS/arch, ejecuta binario
├── install.js          ← postinstall: descarga si optionalDeps fallan
└── lib/main.js         ← API JS (no relevante para CLI puro)
```

Claves del éxito de esbuild:
- ~24 sub-paquetes de plataforma (cobertura total)
- JS shim liviano (~5 KB)
- Hash verification en postinstall
- Fallback: si `npm install --no-optional`, descarga directa desde npm registry

### sass (Dart → npm) — Arquitectura similar

```
npm package "sass"
├── package.json        ← optionalDependencies: sass-linux-x64, etc.
├── sass.js             ← entry point JS (shim)
└── dist/               ← binarios pre-compilados
```

Usan el mismo patrón de sub-paquetes de plataforma con `optionalDependencies`.

### @anthropic-ai/claude-code (npm + binario nativo)

Claude Code usa un patrón similar: el paquete npm descarga un binario nativo (escrito en Go/TypeScript compilado) durante la instalación.

### goreleaser/goreleaser-action

El action oficial de GoReleaser está en `goreleaser/goreleaser-action@v6`. Se integra con:

- GitHub Releases (nativo)
- Homebrew (genera fórmula automáticamente)
- Scoop (Windows)
- Snapcraft
- Docker images

---

## Apéndice: Publicación en npm con cuenta personal

### Situación actual

- El usuario tiene cuenta npm como "aivora" (no personal)
- Quiere usar su **cuenta personal** para publicar

### Pasos

```bash
# 1. Login con cuenta personal
npm login

# 2. Verificar cuenta actual
npm whoami
# → debe mostrar tu usuario personal, no "aivora"

# 3. Si necesitás cambiar de cuenta:
npm logout
npm login

# 4. El package name debe ser único en npm registry
npm search zyrocli
# → si ya existe, usar @tu-usuario/zyrocli o zyroagent-cli

# 5. Publicar
cd dist/npm
npm publish --access public

# 6. Verificar
npm view zyrocli
```

### Scoped packages vs unscoped

| Tipo | Ejemplo | Requiere org | Público gratis |
|------|---------|------------|----------------|
| Unscoped | `zyrocli` | No | ✅ Sí |
| Scoped | `@secko/zyrocli` | Sí (org) | ✅ Sí |
| Scoped personal | `@tuusuario/zyrocli` | No | ✅ Sí |

Si `zyrocli` ya está tomado en npm, la opción más limpia es:

```json
{
  "name": "@secko/zyrocli",
  "publishConfig": {
    "access": "public"
  }
}
```

### Automatización con GitHub Actions

```yaml
- name: Publish to npm
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
    NPM_CONFIG_REGISTRY: https://registry.npmjs.org
  run: |
    cd dist/npm
    echo "//registry.npmjs.org/:_authToken=$NODE_AUTH_TOKEN" > .npmrc
    npm publish --access public
```

El `NPM_TOKEN` se genera desde npmjs.com → Settings → Access Tokens → Generate New Token (tipo: `Automation`).

### Recomendación de naming

```
paquete npm:      @secko/zyrocli     (si "zyrocli" ya está ocupado)
                  zyrocli             (si está libre)
Homebrew formula: zyrocli
Go module:        github.com/yechua-silva/zyrocli
Binary:           zyrocli
```

---

## Apéndice: Versionado y Release Cycle

### Estrategia de versionado

```
Semver estricto (vMAJOR.MINOR.PATCH):
  v0.1.0    → Primer release público
  v0.2.0    → Nuevas features
  v0.2.1    → Bugfixes
  v1.0.0    → API estable
```

### Release automation

```bash
# Crear release (después de merge a main)
git checkout main
git pull
git tag v0.2.0
git push origin v0.2.0

# GitHub Actions:
#   1. Compila binarios (GoReleaser)
#   2. Sube a GitHub Releases
#   3. Publica npm package
#   4. (Opcional) Bump Homebrew formula
```

### release-please (opcional)

[release-please](https://github.com/googleapis/release-please) automatiza la creación de PRs de release basados en Conventional Commits:

```yaml
# .github/workflows/release-please.yml
on:
  push:
    branches: [main]

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          release-type: simple
          token: ${{ secrets.GITHUB_TOKEN }}
```

Esto genera automáticamente:
- Un PR con el changelog actualizado y version bump
- Al mergear el PR, crea un tag y la Release en GitHub

---

## Apéndice: Referencias

| Recurso | URL |
|---------|-----|
| esbuild npm package | https://www.npmjs.com/package/esbuild |
| esbuild install.js (código completo) | https://unpkg.com/esbuild/install.js |
| esbuild bin/esbuild (JS shim) | https://unpkg.com/esbuild/bin/esbuild |
| Homebrew Formula Cookbook | https://docs.brew.sh/Formula-Cookbook |
| gh release create manual | https://cli.github.com/manual/gh_release_create |
| GoReleaser docs | https://goreleaser.com |
| GoReleaser GitHub Action | https://github.com/goreleaser/goreleaser-action |
| npm publish docs | https://docs.npmjs.com/packages-and-modules/contributing-packages-to-the-registry |
| release-please action | https://github.com/googleapis/release-please-action |
| npm login / tokens | https://docs.npmjs.com/creating-and-viewing-access-tokens |

---

> **Próximos pasos:** Implementar Fase 1 (GitHub Actions + GoReleaser) para establecer la fuente de verdad, luego Fase 2 (npm package) para el canal de distribución principal. Esto permite que cualquier desarrollador use `npx zyrocli` sin instalar nada más que Node.js.
