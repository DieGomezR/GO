// Package partnerusers muestra como portar un controller grande de Laravel
// hacia una arquitectura Go basada en capas y dependencias explicitas.
//
// El controller PHP original mezclaba en una sola clase:
//   - validacion HTTP
//   - autenticacion
//   - logica de negocio
//   - consultas SQL directas
//   - llamadas a una API externa
//   - jobs en segundo plano
//   - politicas de fechas
//
// En esta version Go se separa en:
//   - types.go: contratos de entrada/salida y modelos del caso de uso
//   - ports.go: interfaces de infraestructura y servicios externos
//   - helpers.go: logica transversal que el controller original tenia como metodos privados
//   - service_*.go: orquestacion por caso de uso
//   - handler.go: adaptador HTTP
package partnerusers
