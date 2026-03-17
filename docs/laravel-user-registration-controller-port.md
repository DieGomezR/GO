# Port del `UserRegistrationController` a Go

Este documento explica como el controller PHP compartido en la conversación se repartió en una arquitectura Go.

## Problema del archivo original

El controller mezclaba muchas responsabilidades:

- request/response HTTP
- validaciones
- políticas de negocio
- queries SQL directas en dos conexiones
- resolución dinámica de tablas
- llamadas a una API externa
- jobs programados
- auditoría

En Go eso normalmente se separa para que cada capa tenga un objetivo concreto.

## Estructura creada

```text
internal/partnerusers/
  doc.go
  errors.go
  types.go
  ports.go
  helpers.go
  service_register.go
  service_lifecycle.go
  service_support.go
  handler.go
  helpers_test.go
```

## Mapeo de métodos PHP -> Go

- `register()` -> `Service.Register`
- `update()` -> `Service.Update`
- `activate()` -> `Service.ActivatePackage`
- `remove()` -> `Service.Remove`
- `activateChannels()` -> `Service.ActivateChannels`
- `getUserDetails()` -> `Service.GetUserDetails`
- `getEarlyDeactivationStatus()` -> `Service.GetEarlyDeactivationStatus`
- `deactivate()` -> `Service.Deactivate`
- `upload()` -> `Service.QueueBulkUpload`
- `bulkUsers()` -> mismo caso de uso de upload, distinto endpoint
- `deactivateMultiple()` -> `Service.DeactivateMultiple`
- `changePackageMultiple()` -> `Service.ChangePackageMultiple`
- `removeMultiple()` -> `Service.RemoveMultiple`
- `generateUserLoginToken()` -> `Service.GenerateUserLoginToken`
- `generateUserLoginTokenEmail()` -> `Service.GenerateUserLoginTokenByEmail`
- `logoutAllDevices()` -> `Service.LogoutAllDevices`
- `reactivateDeletedUser()` -> `Service.ReactivateDeletedUser`

## Qué se portó literalmente

Sí quedó trasladada la lógica privada más importante del controller:

- `basePartner`
- `normalize`
- `normalizeCountryCode`
- `mapTerritoryToCountryCode`
- `resolveCountryForPartnerId`
- `resolveCountryForUserId`
- `buildEarlyDeactivationSignal`
- `auditEarlyDeactivationAttempt`
- `pickTableForPartner`
- `computeNextPartnerIdFor`
- `parseApiError`
- `tryReadFromCandidates`

## Qué cambió respecto a Laravel

- Laravel `Request` y `response()->json(...)` se cambiaron por `Handler` HTTP.
- `DB::table(...)` y `Isp::query()` se movieron a interfaces en `ports.go`.
- `Http::get/post(...)` se movió a `PartnerAPIClient`.
- `VigenciaPolicy` y `EarlyDeactivationPolicyService` ahora son interfaces.
- `ScheduledDeactivateUser::dispatch(...)` ahora es `Scheduler`.
- La persistencia temporal del upload ahora es `UploadStorage`.

## Qué falta para usarlo en producción

Esta migración ya compila, pero aún necesita implementaciones concretas para:

- PostgreSQL/MySQL reales
- cliente HTTP real contra la API externa
- scheduler/cola real
- middleware real de autenticación para poblar `AuthUser`

La idea del ejemplo no es cerrar infraestructura, sino enseñarte cómo se transforma un controller monolítico de Laravel a una arquitectura Go mantenible.
