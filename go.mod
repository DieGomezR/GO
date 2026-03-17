// Nombre del modulo para importar paquetes internos desde el propio proyecto.
module tienda-go

// Version minima del lenguaje que espera este proyecto.
go 1.26

// Toolchain sugerido para garantizar que el codigo se compile con la version estable validada.
toolchain go1.26.1

require github.com/go-sql-driver/mysql v1.9.3

require filippo.io/edwards25519 v1.1.0 // indirect
