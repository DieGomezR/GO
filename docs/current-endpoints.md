# Endpoints Actuales

Este documento resume las rutas HTTP expuestas hoy por la API.

## Contexto

Hay dos grupos de endpoints:

- `store`: la tienda educativa original
- `partner-users`: el port del controller PHP `UserRegistrationController`

## Autenticacion

### Store

- Usa `Authorization: Bearer <token>`
- El token se obtiene con `POST /v1/auth/login`

### Partner Users

- Usa el mismo `Authorization: Bearer <token>` de la API principal
- Ademas, para dar contexto del partner al port PHP, conviene enviar:
  - `X-Partner-ID`: partner del actor autenticado
  - `X-Partner-Country`: por defecto `co`
  - `X-Partner-Role`: opcional; si no viene, se toma el rol del usuario autenticado en la tienda
  - `X-Partner-Email`: opcional; si no viene, se toma el email del usuario autenticado en la tienda
  - `X-Partner-Actor-ID`: opcional; util para auditoria

## Store API

| Metodo | Ruta | Que hace | Auth |
|---|---|---|---|
| GET | `/healthz` | Health check basico | Publico |
| POST | `/v1/auth/login` | Inicia sesion y devuelve token | Publico |
| POST | `/v1/auth/logout` | Cierra la sesion actual | Bearer |
| GET | `/v1/me` | Devuelve el usuario autenticado | Bearer |
| GET | `/v1/users` | Lista usuarios del sistema | `admin`, `manager` |
| POST | `/v1/users` | Crea un usuario nuevo | `admin` |
| GET | `/v1/dashboard/summary` | Resumen de usuarios, productos, ventas y stock critico | `admin`, `manager` |
| GET | `/v1/products` | Lista el catalogo | Bearer |
| GET | `/v1/products/{id}` | Devuelve un producto por ID | Bearer |
| POST | `/v1/products` | Crea un producto | `admin`, `manager` |
| PUT | `/v1/products/{id}` | Actualiza datos del producto | `admin`, `manager` |
| GET | `/v1/products/{id}/inventory` | Lista movimientos de inventario del producto | Bearer |
| POST | `/v1/products/{id}/inventory` | Ajusta stock manualmente | `admin`, `manager` |
| GET | `/v1/orders` | Lista ventas | Bearer |
| GET | `/v1/orders/{id}` | Devuelve una venta puntual | Bearer |
| POST | `/v1/orders` | Registra una venta y descuenta stock | `admin`, `manager`, `cashier` |

## Partner Users API

| Metodo | Ruta | Que hace | Notas |
|---|---|---|---|
| POST | `/v1/partner-users/register` | Registra un usuario nuevo en el flujo legacy | La parte de alta externa aun no esta implementada en el cliente Go |
| PUT | `/v1/partner-users/update` | Modifica nombre, password, paquete y canales | Usa MySQL real y partner API |
| POST | `/v1/partner-users/activate` | Activa o cambia paquete de un usuario | Usa MySQL real y partner API |
| DELETE | `/v1/partner-users/remove` | Elimina un usuario o programa su eliminacion | Si necesita scheduler diferido, hoy responde error de configuracion |
| POST | `/v1/partner-users/activate-channels` | Activa canales adicionales | Usa partner API |
| GET | `/v1/partner-users/details?partnerId=...` | Devuelve detalle del usuario, paquete y dispositivos | Validado con MySQL real y partner API real |
| GET | `/v1/partner-users/by-partner?partner=televvd&limit=...` | Lista usuarios de un partner base desde tablas `ISP_*_subscribers` | Validado con MySQL real |
| GET | `/v1/partner-users/early-deactivation-status?country=...` | Evalua si se permite desactivacion temprana | Usa policy Go simplificada |
| POST | `/v1/partner-users/deactivate` | Desactiva un usuario o programa la desactivacion | Si necesita scheduler diferido, hoy responde error de configuracion |
| POST | `/v1/partner-users/upload` | Recibe archivo JSON para carga masiva | Requiere storage y scheduler; hoy no estan integrados |
| POST | `/v1/partner-users/bulk-users` | Alias legacy de upload | Igual que `/upload` |
| POST | `/v1/partner-users/deactivate-multiple` | Desactiva varios usuarios | Depende de la misma logica de `/deactivate` |
| POST | `/v1/partner-users/change-package-multiple` | Cambia paquete a varios usuarios | Usa MySQL real y partner API |
| DELETE | `/v1/partner-users/remove-multiple` | Elimina varios usuarios | Depende de la misma logica de `/remove` |
| POST | `/v1/partner-users/generate-login-token` | Genera token de login directo para un usuario final | Usa partner API real |
| GET | `/v1/partner-users/generate-login-token-by-email?email=...` | Busca usuario por email y genera URL de login | Validado con MySQL real y partner API real |
| POST | `/v1/partner-users/logout-all-devices` | Cierra sesion de todos los dispositivos del usuario | Usa partner API real |
| POST | `/v1/partner-users/reactivate` | Reactiva usuario eliminado | La parte de alta externa aun no esta implementada en el cliente Go |

## Endpoints Verificados Con Integracion Real

Estos ya se probaron con:

- `moderntv`
- `nuplin_prod`
- `https://nuplin.com/partner/api/`

Verificados:

- `GET /v1/partner-users/details`
- `GET /v1/partner-users/by-partner`
- `GET /v1/partner-users/generate-login-token-by-email`
- `GET /v1/partner-users/early-deactivation-status`

## Recomendacion Para Postman

1. Haz login en `/v1/auth/login`
2. Usa `Authorization: Bearer <token>`
3. Para rutas `partner-users`, agrega por lo menos:
   - `X-Partner-ID: starnet`
   - `X-Partner-Country: co`
4. Prueba primero:
   - `GET /v1/partner-users/details?partnerId=starnet_1186`
   - `GET /v1/partner-users/by-partner?partner=televvd&limit=5`
   - `GET /v1/partner-users/generate-login-token-by-email?email=arenas3528@gmail.com`
