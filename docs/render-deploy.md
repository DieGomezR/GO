# Deploy En Render

Este proyecto ya esta preparado para desplegarse en Render con:

- `Dockerfile` en la raiz
- `render.yaml` en la raiz
- soporte automatico para `PORT`

## Opcion recomendada

Usa `Blueprint` en Render para que lea `render.yaml`.

## Pasos

1. Sube este repositorio a GitHub.
2. En Render crea `New > Blueprint`.
3. Selecciona el repositorio.
4. Render detectara `render.yaml` y creara un `Web Service`.
5. En el primer alta te pedira los valores marcados con `sync: false`.

## Variables secretas que debes cargar

- `DB_HOST`
- `DB_DATABASE`
- `DB_USERNAME`
- `DB_PASSWORD`
- `PROD_DB_HOST`
- `PROD_DB_DATABASE`
- `PROD_DB_USERNAME`
- `PROD_DB_PASSWORD`
- `PARTNER_API_USER`
- `PARTNER_API_PASS`

## Variables ya preconfiguradas en `render.yaml`

- `APP_ENV=production`
- `STORE_DRIVER=memory`
- `APP_SEED_ON_START=true`
- `AUTH_TOKEN_TTL=12h`
- `PARTNER_API_BASE_URL=https://nuplin.com/partner/api/`
- `PARTNER_LOGIN_BASE_URL=https://nuplin.com`
- `PARTNER_API_TIMEOUT=20s`
- `PARTNER_API_SKIP_TLS_VERIFY=true`

## Como funciona el puerto

Render expone la variable `PORT`. La app ahora usa ese valor automaticamente cuando `APP_ADDR` no esta definido, asi que no tienes que tocar el codigo para el deploy.

## Health check

Render puede verificar la app usando:

`/healthz`

## Primeras pruebas despues del deploy

1. `POST /v1/auth/login`
2. `GET /v1/partner-users/by-partner?partner=televvd&limit=5`

Headers para `partner-users`:

- `Authorization: Bearer <token>`
- `X-Partner-ID: televvd`
- `X-Partner-Country: co`

## Nota importante

`sync: false` solo te pide el valor en la creacion inicial del Blueprint. Si luego agregas nuevas variables secretas, cargalas manualmente en el dashboard de Render.
