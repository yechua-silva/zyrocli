# Investigación: Documentación de CLIs Go publicados en npm

> **Fecha:** 2026-06-15
> **Propósito:** Referencia para mejorar README.md de ZyroCLI
> **Contexto:** ZyroCLI es un CLI en Go que se publica en npm como wrapper (el binario Go se descarga via postinstall). El README actual ya existe pero puede mejorarse siguiendo las mejores prácticas de proyectos exitosos.

---

## Proyectos analizados

### 1. esbuild (evanw/esbuild)
- **Lenguaje origen:** Go
- **Publicación npm:** Sí — el npm package `esbuild` es un wrapper que descarga el binario nativo via `postinstall.js`
- **Repo GitHub:** https://github.com/evanw/esbuild
- **npm:** https://www.npmjs.com/package/esbuild

#### Estructura del README
esbuild **no tiene un README clásico con badges en la parte superior**. En su lugar:
1. **Logo/wordmark** responsive (dark/light mode) — muy profesional
2. **Barra de navegación** con enlaces: Website | Getting started | Documentation | Plugins | FAQ
3. **Sección "Why?"** — problema que resuelve (build tools 10-100x más lentos)
4. **Benchmark** (gráfico comparativo de velocidad)
5. **Major features** (lista con viñetas)
6. **Call to action** — enlace a getting started

#### Badges
esbuild **no usa badges en el README de GitHub**. Esto es inusual. Pero en npm:
- El package page muestra versión, descargas, etc. automáticamente
- Tiene un `README.md` separado para npm (más corto, con foco en API JS)

#### Vinculación repo ↔ npm
```json
// npm/esbuild/package.json (package publicado)
{
  "repository": {
    "type": "git",
    "url": "git+https://github.com/evanw/esbuild.git"
  },
  "homepage": "https://github.com/evanw/esbuild#readme",
  "bugs": {
    "url": "https://github.com/evanw/esbuild/issues"
  }
}
```
El `repository` field usa formato objeto con `type` y `url`. Esto es lo **mínimo necesario** para que GitHub muestre el enlace "Visit npm" en la sidebar.

#### Lo que podemos copiar
- ✅ Logo responsive (dark/light mode) — da identidad profesional
- ✅ Navegación superior con enlaces a docs
- ✅ Empezar con el "problema que resuelve" (Why?)
- ✅ Separar README de GitHub del de npm si hay diferencias de audiencia
- ❌ La falta de badges no es recomendable para un proyecto más pequeño

---

### 2. dart-sass (sass/dart-sass)
- **Lenguaje origen:** Dart (compilado a JS para npm)
- **Publicación npm:** Sí — `npm install -g sass` / `npm install --save-dev sass`
- **Repo GitHub:** https://github.com/sass/dart-sass
- **npm:** https://www.npmjs.com/package/sass

#### Estructura del README
1. **Hero + tabla con logo + badges + enlaces sociales** (layout tabular visualmente denso)
2. **Tabla de contenidos** detallada
3. **"Using Dart Sass"** — múltiples métodos de instalación:
   - From Chocolatey or Scoop (Windows)
   - From Homebrew (macOS)
   - Standalone (descarga directa)
   - **From npm** (con JS API docs, browser support, legacy API, Jest)
   - From Pub (Dart)
   - From Source
   - In Docker
4. **Why Dart?** — justificación técnica
5. **Compatibility Policy** — semver, browser compat, Node.js compat
6. **Embedded Dart Sass**
7. **Behavioral Differences from Ruby Sass**
8. **Disclaimer**

#### Badges
```markdown
[npm statistics](https://nodei.co/npm/sass.png?downloads=true)
[![Pub version](https://img.shields.io/pub/v/sass.svg)](https://pub.dartlang.org/packages/sass)
[![GitHub actions build status](https://github.com/sass/dart-sass/workflows/CI/badge.svg)](https://github.com/sass/dart-sass/actions)
[![Mastodon](https://img.shields.io/mastodon/follow/...)]
[![Twitter](https://img.shields.io/twitter/follow/SassCSS?style=social)]
[![StackOverflow](https://img.shields.io/stackexchange/stackoverflow/t/sass)]
[![Gitter](https://img.shields.io/gitter/room/sass/sass)]
```
Usa **nodei.co** para el badge grande de npm statistics (descargas).

#### Vinculación repo ↔ npm
```json
// package.json publicado en npm
{
  "repository": {
    "type": "git",
    "url": "git+https://github.com/sass/dart-sass.git"
  },
  "homepage": "https://github.com/sass/dart-sass",
  "bugs": {
    "url": "https://github.com/sass/dart-sass/issues"
  }
}
```

#### Lo que podemos copiar
- ✅ **Tabla de contenidos** al inicio — crucial para READMEs largos
- ✅ **Múltiples métodos de instalación** ordenados por plataforma/gestor
- ✅ **Sección "From npm"** destacada como método principal para usuarios JS/Node
- ✅ Badge de npm con **nodei.co** (muestra estadísticas visuales)
- ✅ Badge de **build status** (GitHub Actions)
- ✅ Enlaces a redes sociales / comunidad

---

### 3. act (nektos/act)
- **Lenguaje origen:** Go
- **Publicación npm:** Sí tiene release en npm pero no es el foco principal
- **Repo GitHub:** https://github.com/nektos/act

#### Estructura del README
1. **Logo + badges** (fila superior compacta)
2. **Tagline** — "Think globally, act locally"
3. **Explicación** — qué hace y por qué (2 razones: Fast Feedback, Local Task Runner)
4. **How Does It Work?** — explicación técnica con GIF demo
5. **User Guide** — enlace a documentación externa
6. **Support** — enlace a discussions
7. **Contributing** — instrucciones breves + build desde source

#### Badges
```markdown
[![push](https://github.com/nektos/act/workflows/push/badge.svg?branch=master&event=push)](...)
[![Go Report Card](https://goreportcard.com/badge/github.com/nektos/act)](https://goreportcard.com/report/github.com/nektos/act)
[![awesome-runners](https://img.shields.io/badge/listed%20on-awesome--runners-blue.svg)](...)
```
- Usa **GitHub Actions badge** (workflow push)
- Usa **Go Report Card** (muy común en proyectos Go)
- No tiene badge de npm (no es su canal principal)

#### Lo que podemos copiar
- ✅ **GIF demo** al inicio — muestra el producto en acción
- ✅ **Tagline memorable** — "Think globally, act locally"
- ✅ **Go Report Card badge** — señal de calidad para proyectos Go
- ✅ Enfoque minimalista pero informativo

---

### 4. lazygit (jesseduffield/lazygit)
- **Lenguaje origen:** Go
- **Publicación npm:** No tiene package npm oficial (se distribuye por Homebrew, releases, etc.)
- **Repo GitHub:** https://github.com/jesseduffield/lazygit

#### Estructura del README
1. **Sponsors** (sección principal, muy prominente)
2. **Logo** + **badges**
3. **GIF demo** (feature principal)
4. **Elevator Pitch** (rant memorable y humorístico sobre git)
5. **Tabla de contenidos** detallada
6. **Features** (secciones individuales con GIFs para cada feature)
7. **Tutorials** (enlaces a YouTube)
8. **Installation** (MUY extensa — 20+ métodos)
9. **Usage** + **Keybindings** + **Config**
10. **Contributing** + **Donate** + **FAQ** + **Alternatives**

#### Badges
```markdown
[![GitHub Downloads](https://img.shields.io/github/downloads/jesseduffield/lazygit/total)](...)
[![Go Report Card](https://goreportcard.com/badge/github.com/jesseduffield/lazygit)](...)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/...)](...)
[![Coverage](https://app.codacy.com/project/badge/Coverage/...)](...)
[![golangci-lint](https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen)](...)
[![GitHub tag](https://img.shields.io/github/v/tag/jesseduffield/lazygit?color=blue)](...)
[![homebrew](https://img.shields.io/homebrew/v/lazygit?color=blue)](...)
```

#### Lo que podemos copiar
- ✅ **GIF demo por feature** — mejor que texto plano
- ✅ **Elevator Pitch** con personalidad — hace único el README
- ✅ **Packaging status** badge (repology.org) — muestra disponibilidad en todos los gestores
- ✅ **Tabla de contenidos** extremadamente detallada
- ✅ **Sección de alternativas** al final
- ❌ No aplica a npm directamente (no tiene package npm)

---

### 5. croc (schollz/croc)
- **Lenguaje origen:** Go
- **Publicación npm:** No tiene package npm oficial
- **Repo GitHub:** https://github.com/schollz/croc

#### Estructura del README
1. **Logo + badges** compactos
2. **Sponsor call** (notable: "This project's future depends on community support")
3. **About** — bullet points con capacidades clave
4. **Install** — 15+ métodos (curl, macOS, Windows, Nix, Alpine, Arch, Fedora, etc.)
5. **Usage** — ejemplos básicos + opciones avanzadas (tabla de flags)
6. **Acknowledgements**

#### Badges
```markdown
[![Version](https://img.shields.io/github/v/release/schollz/croc)](...)
[![Build Status](https://github.com/schollz/croc/actions/workflows/ci.yml/badge.svg)](...)
[![GitHub Sponsors](https://img.shields.io/github/sponsors/schollz)](...)
```

#### Lo que podemos copiar
- ✅ **Instalación con `curl | bash`** — el más directo
- ✅ **Badge de GitHub Sponsors** (si aplica)
- ✅ **Sección de "About" concisa** con bullets de差异化 (qué hace único al tool)
- ✅ **Tabla de opciones/uso** clara

---

### 6. quicktype (glideapps/quicktype)
- **Lenguaje origen:** TypeScript (no Go, pero relevante como CLI npm popular)
- **Publicación npm:** Sí — `npm install -g quicktype`
- **Repo GitHub:** https://github.com/glideapps/quicktype

#### Estructura del README
1. **Logo SVG**
2. **Badges** (npm version + build status)
3. **Descripción** + enlaces (web app, blog, FAQ)
4. **Supported Inputs** / **Target Languages** (tablas visuales con links)
5. **Installation** (breve — `npm install -g quicktype`)
6. **Usage** (ejemplos progresivos: simple → schema → TypeScript → JS API)
7. **Contributing** (build, edit, test)
8. **Custom rendering guide**

#### Badges
```markdown
[![npm version](https://badge.fury.io/js/quicktype.svg)](https://badge.fury.io/js/quicktype)
![Build status](https://github.com/quicktype/quicktype/actions/workflows/master.yaml/badge.svg)
```
Usa **badge.fury.io** para el badge de npm version (alternativa a shields.io).

#### Vinculación repo ↔ npm
```json
{
  "repository": "https://github.com/quicktype/quicktype",
  "bin": "dist/index.js"
}
```
Formato **string simple** (no objeto) para `repository`. Ambos formatos funcionan para npm.

#### Lo que podemos copiar
- ✅ **Ejemplos progresivos** (de simple a avanzado) — ideal para onboarding
- ✅ **Tabla de inputs/soportes** visual
- ✅ Enlace al **playground web** (si aplica)
- ✅ README más conciso que sass/lazygit

---

## Cómo funciona el enlace "npm" en GitHub

### 1. El `repository` field en package.json
GitHub detecta automáticamente que un repo tiene un paquete npm cuando:
- El package.json **publicado en npm** tiene un `repository` field que apunta al repo
- O el repo contiene un `package.json` en la raíz con el mismo nombre

### 2. El botón "Visit npm"
- Aparece en la **sidebar derecha** de GitHub (sección "Packages")
- GitHub lo genera automáticamente cuando detecta la conexión repo ↔ npm
- No requiere configuración manual — solo que el `repository` field en npm apunte al repo

### 3. Formato recomendado para repository
```json
{
  "repository": {
    "type": "git",
    "url": "git+https://github.com/secko/zyrocli.git"
  }
}
```
O el formato string simple:
```json
{
  "repository": "https://github.com/secko/zyrocli"
}
```
**Ambos funcionan.** El formato objeto es más explícito y es el estándar de npm.

### 4. Homepage field
```json
{
  "homepage": "https://github.com/secko/zyrocli#readme"
}
```
Esto hace que npm muestre el enlace "Readme" que apunta al README de GitHub.

---

## Mejores prácticas identificadas

| Práctica | Proyectos que la usan | Recomendación |
|----------|----------------------|---------------|
| **Logo/wordmark visual** | esbuild, lazygit, croc, quicktype | ✅ Agregar logo (idealmente responsive dark/light) |
| **Badges en la primera línea** | act, lazygit, croc, quicktype, sass | ✅ npm version + Go version + Build + License |
| **Tagline / One-liner** | act, croc, quicktype | ✅ "Orquestador autónomo para desarrollo asistido por IA" |
| **Tabla de contenidos** | sass, lazygit | ✅ Para READMEs largos (ZyroCLI ya tiene secciones, agregar TOC) |
| **GIF demo** | act, lazygit | ✅ si hay UI; ❌ ZyroCLI es CLI, screenshot de terminal basta |
| **Múltiples métodos de instalación** | sass, lazygit, croc | ✅ npm first, luego go install, luego script, luego releases |
| **npm como método #1** | sass, esbuild, quicktype | ✅ "npm install -g zyrocli" primero |
| **Sección "Why?" / Problema** | esbuild, lazygit, croc | ✅ ZyroCLI ya tiene "¿Por qué existe?" — mantenerlo |
| **Badge de npm version** | quicktype, sass | ✅ `https://img.shields.io/npm/v/zyrocli` |
| **Badge de npm downloads** | sass (nodei.co) | ✅ `https://img.shields.io/npm/dm/zyrocli` |
| **Badge de Go Report Card** | act, lazygit | ✅ `https://goreportcard.com/badge/...` |
| **Build status (GitHub Actions)** | act, lazygit, croc, sass | ✅ `https://github.com/.../workflows/.../badge.svg` |
| **Tabla de comandos/features** | ZyroCLI ya la tiene | ✅ Mantener tabla actual |
| **Elevator Pitch** | lazygit | ✅ Mantener sección "¿Qué hace ZyroCLI?" |
| **Referencia a docs externos** | esbuild, sass | ✅ ZyroCLI ya tiene enlaces a docs/ |
| **Sponsors / Donate** | lazygit, croc | ✅ Opcional si hay GitHub Sponsors |
| **Separar README GitHub vs npm** | esbuild | ✅ Evaluar si el público npm necesita doc diferente |
| **Enlace a "try it" / playground** | quicktype | ✅ `npx zyrocli setup` es el equivalente |

---

## Badges recomendados para ZyroCLI

Basado en el análisis, estos son los badges que ZyroCLI debería tener (en orden de prioridad):

```markdown
<!-- Línea 1: Estado del proyecto -->
[![npm version](https://img.shields.io/npm/v/zyrocli)](https://www.npmjs.com/package/zyrocli)
[![npm downloads](https://img.shields.io/npm/dm/zyrocli)](https://www.npmjs.com/package/zyrocli)
[![Go version](https://img.shields.io/github/go-mod/go-version/secko/zyrocli)](https://go.dev/)
[![License](https://img.shields.io/github/license/secko/zyrocli)](LICENSE)

<!-- Línea 2: Calidad y CI -->
[![Go Report Card](https://goreportcard.com/badge/github.com/secko/zyrocli)](https://goreportcard.com/report/github.com/secko/zyrocli)
[![CI](https://github.com/secko/zyrocli/actions/workflows/ci.yml/badge.svg)](https://github.com/secko/zyrocli/actions)
[![Tests](https://img.shields.io/badge/tests-383_✔️-brightgreen)]()

<!-- Opcional: npm install inline -->
[![npm install](https://img.shields.io/badge/npm_install-zyrocli-blue?logo=npm)](https://www.npmjs.com/package/zyrocli)
```

### URLs de badges (referencia rápida)

| Badge | URL |
|-------|-----|
| npm version | `https://img.shields.io/npm/v/zyrocli` |
| npm downloads (mensual) | `https://img.shields.io/npm/dm/zyrocli` |
| npm downloads (total) | `https://img.shields.io/npm/dt/zyrocli` |
| npm license | `https://img.shields.io/npm/l/zyrocli` |
| npm node version | `https://img.shields.io/node/v/zyrocli` |
| Go version (go.mod) | `https://img.shields.io/github/go-mod/go-version/secko/zyrocli` |
| GitHub release | `https://img.shields.io/github/v/release/secko/zyrocli` |
| GitHub tag | `https://img.shields.io/github/v/tag/secko/zyrocli` |
| GitHub license | `https://img.shields.io/github/license/secko/zyrocli` |
| GitHub Actions | `https://github.com/secko/zyrocli/actions/workflows/ci.yml/badge.svg` |
| Go Report Card | `https://goreportcard.com/badge/github.com/secko/zyrocli` |
| nodei.co (npm stats) | `https://nodei.co/npm/zyrocli.png?downloads=true` |

---

## Estructura de README recomendada para ZyroCLI

Basada en el análisis de todos los proyectos, este es el orden ideal de secciones:

1. **Logo + Tagline** + **Badges** (fila superior)
2. **¿Qué hace ZyroCLI?** → tabla de comandos (ya existe)
3. **Instalación rápida** → `npx zyrocli` (el primero, con badge npm)
4. **¿Por qué existe?** → problema que resuelve (ya existe, mantener)
5. **Innovaciones** → Memoria Causal, Agent-as-Validator, etc. (ya existe)
6. **Arquitectura** → diagrama (ya existe)
7. **Guía rápida** → ejemplos de uso progresivos (NUEVO — agregar)
8. **Documentación** → tabla de enlaces (ya existe)
9. **Estado del proyecto** → badges de tests, CI (ya existe, mejorar)
10. **Licencia** → MIT (ya existe)

### Mejoras clave sobre el README actual
- ✅ Agregar **badges** (npm version, Go version, build, license, Go Report Card)
- ✅ Poner **npm install** como primera opción de instalación y más destacada
- ✅ Agregar **tabla de contenidos** al inicio
- ✅ Agregar una **sección de "Guía rápida"** con ejemplos progresivos
- ✅ Mejorar el **formato de la tabla de comandos** con descripciones más detalladas
- ✅ Agregar **badge de npm downloads** (señal social de popularidad)
- ✅ Considerar un **logo/wordmark** (incluso texto estilizado)
- ✅ El package.json ya tiene el `repository` field correcto — GitHub debería mostrar "Visit npm" automáticamente

---

## Verificación del package.json actual de ZyroCLI

El package.json en `scripts/npm/package.json` **ya está correctamente configurado**:
```json
{
  "repository": {
    "type": "git",
    "url": "git+https://github.com/secko/zyrocli.git"
  },
  "homepage": "https://github.com/secko/zyrocli#readme",
  "bugs": {
    "url": "https://github.com/secko/zyrocli/issues"
  }
}
```
Esto es exactamente lo que npm/necesita para vincular el paquete con GitHub. Una vez publicado, GitHub mostrará automáticamente el enlace "Visit npm" en la sidebar del repo (sección Packages).

---

## Referencias

- https://github.com/evanw/esbuild
- https://github.com/sass/dart-sass
- https://github.com/nektos/act
- https://github.com/jesseduffield/lazygit
- https://github.com/schollz/croc
- https://github.com/glideapps/quicktype
- https://docs.npmjs.com/cli/v10/configuring-npm/package-json#repository
