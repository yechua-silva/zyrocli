# Tarea: Agregar autenticación JWT a API Go

## Código base
Repositorio Go con:
- `main.go` — servidor HTTP básico
- `go.mod` — módulo Go
- `handler.go` — 2 endpoints: GET /ping, GET /status

## Requerimiento
Agregar autenticación JWT a la API:

1. Agregar endpoint `POST /login` que recibe `{"username":"admin","password":"secret"}` y devuelve un JWT
2. Agregar middleware de autenticación que valide el JWT en endpoints protegidos
3. Agregar endpoint protegido `GET /profile` que devuelva el usuario del JWT
4. Tests: `go test ./...` debe pasar

## Criterios de éxito
- `go build ./...` compila sin errores
- `go test ./...` pasa todos los tests
- `POST /login` devuelve un JWT válido
- `GET /profile` funciona con token válido
- `GET /profile` devuelve 401 sin token o token inválido

## Stack tecnológico
- Go 1.24+
- github.com/golang-jwt/jwt/v5
- Solo stdlib para el HTTP server
