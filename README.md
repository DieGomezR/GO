# Tienda Go

Backend educativo de una tienda construido con Go `1.26` y preparado para ejecutarse con el toolchain estable `go1.26.1`.

## Como leer este repositorio

Los archivos del proyecto tienen comentarios en el propio codigo para ayudarte a estudiar:

- `domain`: que existe en el negocio
- `service`: que reglas se aplican
- `api`: como se expone la logica por HTTP
- `main`: como se arma toda la aplicacion

El proyecto cubre un flujo tipico de negocio:

- autenticacion con token
- gestion de usuarios con roles
- catalogo de productos
- control de inventario
- registro de ventas
- resumen operativo para administracion

## Stack

- Go `1.26`
- `toolchain go1.26.1`
- `net/http` con `ServeMux` moderno
- `log/slog` para logging
- almacenamiento seleccionable: memoria o MySQL

## Roles

- `admin`: crea usuarios, productos, ajusta inventario y consulta resumen
- `manager`: consulta resumen, crea productos y ajusta inventario
- `cashier`: vende y consulta catalogo

## Estructura

```text
cmd/store-api           unico punto de entrada ejecutable
internal/config         configuracion por variables de entorno
internal/domain         entidades y errores de negocio
internal/store          persistencia en memoria y MySQL
internal/service        reglas de negocio
internal/api            rutas HTTP, middleware y serializacion JSON
internal/bootstrap      datos semilla
```

No hay mas binarios en `cmd/`: toda la API arranca desde `cmd/store-api/main.go`.

## Inicio rapido

1. Copia las variables de ejemplo de `.env.example` si quieres personalizar puertos, TTL o persistencia.
2. Ejecuta la API:

```powershell
New-Item -ItemType Directory -Force -Path .cache\go-build,.cache\gomod | Out-Null
$env:GOCACHE=(Resolve-Path '.cache\go-build')
$env:GOMODCACHE=(Resolve-Path '.cache\gomod')
.\.tools\go\bin\go.exe run ./cmd/store-api
```

Si ya tienes Go instalado globalmente, tambien sirve:

```powershell
go run ./cmd/store-api
```

La aplicacion carga automaticamente un archivo `.env` si existe en la raiz del proyecto.

Si `:8080` ya esta ocupado, cambia el puerto antes de arrancar:

```powershell
$env:APP_ADDR=':8091'
.\.tools\go\bin\go.exe run ./cmd/store-api
```

## Deploy en Render

El proyecto ya incluye:

- [Dockerfile](Dockerfile)
- [render.yaml](render.yaml)
- guia de despliegue en [docs/render-deploy.md](docs/render-deploy.md)

La app soporta `PORT` automaticamente, asi que Render puede arrancarla sin cambios extra de codigo.

## Ejecutar con MySQL

1. Crea una base de datos vacia, por ejemplo `tienda_go`.
2. Ajusta tu `.env`:

```env
STORE_DRIVER=mysql
MYSQL_DSN=root:secret@tcp(127.0.0.1:3306)/tienda_go?charset=utf8mb4
MYSQL_AUTO_MIGRATE=true
APP_SEED_ON_START=true
```

3. Arranca la API igual que siempre. Si `MYSQL_AUTO_MIGRATE=true`, la app crea estas tablas:

- `users`
- `sessions`
- `products`
- `inventory_movements`
- `orders`
- `order_items`

La primera vez tambien sembrara usuarios y productos demo. En arranques posteriores no volvera a duplicarlos.

## Usuarios demo

Al arrancar, la API siembra estos usuarios y los imprime en logs:

- `admin@store.local` / `Admin1234!`
- `manager@store.local` / `Manager1234!`
- `cashier@store.local` / `Cashier1234!`

## Flujo recomendado para probar

1. Inicia sesion.
2. Usa el token `Bearer` para consultar tu perfil.
3. Lista productos.
4. Registra una venta.
5. Consulta inventario y ordenes.
6. Si entras como `admin` o `manager`, revisa el resumen.

## Endpoints

### Publicos

- `GET /healthz`
- `POST /v1/auth/login`

### Autenticados

- `POST /v1/auth/logout`
- `GET /v1/me`
- `GET /v1/products`
- `GET /v1/products/{id}`
- `GET /v1/products/{id}/inventory`
- `GET /v1/orders`
- `GET /v1/orders/{id}`
- `POST /v1/orders`

### Solo `admin` o `manager`

- `GET /v1/dashboard/summary`
- `GET /v1/users`
- `POST /v1/products`
- `PUT /v1/products/{id}`
- `POST /v1/products/{id}/inventory`

### Solo `admin`

- `POST /v1/users`

### Partner Users

Estas rutas tambien salen del mismo servidor `cmd/store-api`; no hay otro `main.go`.

- `GET /v1/partner-users/details?partnerId=...`
- `GET /v1/partner-users/by-partner?partner=televvd&limit=...`
- `GET /v1/partner-users/prod/by-partner?partner=televvd&limit=...`
- `GET /v1/partner-users/generate-login-token-by-email?email=...`
- `GET /v1/partner-users/early-deactivation-status?country=...`
- `POST /v1/partner-users/generate-login-token`
- `POST /v1/partner-users/logout-all-devices`

Para el inventario completo revisa [docs/current-endpoints.md](docs/current-endpoints.md).

## Ejemplos

Login:

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/auth/login `
  -ContentType 'application/json' `
  -Body '{"email":"admin@store.local","password":"Admin1234!"}'

$token = $login.data.token
```

Listar productos:

```powershell
Invoke-RestMethod `
  -Method Get `
  -Uri http://localhost:8080/v1/products `
  -Headers @{ Authorization = "Bearer $token" }
```

Crear producto:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/products `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType 'application/json' `
  -Body '{
    "sku":"NB-005",
    "name":"Base Refrigerante",
    "description":"Base con ventilacion dual para laptop",
    "price_cents":1899000,
    "initial_stock":9
  }'
```

Registrar venta:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/v1/orders `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType 'application/json' `
  -Body '{
    "customer_name":"Cliente Mostrador",
    "items":[
      { "product_id":"REEMPLAZA_CON_ID", "quantity":2 }
    ]
  }'
```

Listar usuarios de un partner base:

```powershell
Invoke-RestMethod `
  -Method Get `
  -Uri 'http://localhost:8091/v1/partner-users/by-partner?partner=televvd&limit=5' `
  -Headers @{
    Authorization = "Bearer $token"
    'X-Partner-ID' = 'televvd'
    'X-Partner-Country' = 'co'
  }
```

Listar usuarios de un partner base usando solo `db_prod`:

```powershell
Invoke-RestMethod `
  -Method Get `
  -Uri 'http://localhost:8091/v1/partner-users/prod/by-partner?partner=televvd&limit=5' `
  -Headers @{
    Authorization = "Bearer $token"
  }
```

## Pruebas

```powershell
New-Item -ItemType Directory -Force -Path .cache\go-build,.cache\gomod | Out-Null
$env:GOCACHE=(Resolve-Path '.cache\go-build')
$env:GOMODCACHE=(Resolve-Path '.cache\gomod')
.\.tools\go\bin\go.exe test ./...
```

## Ruta de aprendizaje

Si quieres estudiar el proyecto de forma ordenada:

1. Empieza por `internal/domain`.
2. Sigue con `internal/service`.
3. Luego revisa `internal/api`.
4. Termina en `cmd/store-api/main.go` para ver el ensamblado completo.
5. Mira `internal/service/*_test.go` para aprender como se prueba la logica.

## Nota tecnica

La version base sigue siendo didactica: la API puede correr en memoria o con MySQL usando la misma capa de servicios.

El hash de contrasenas es deliberadamente simple para mantener el proyecto sin paquetes externos. Si quieres convertir esta base en algo productivo, cambia `internal/security/password.go` por Argon2id o bcrypt.
