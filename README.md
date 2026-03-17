# helper

`helper` es un módulo Go con utilidades reutilizables para tres áreas comunes:

- logging con `zap`
- acceso a PostgreSQL con `gorm`
- consumo HTTP con reintentos y respuesta paginada

## Instalación

```bash
go get github.com/hugomode/helper
```

## Paquetes disponibles

- `github.com/hugomode/helper/logger`
- `github.com/hugomode/helper/db`
- `github.com/hugomode/helper/myHttp`

## Logger

El paquete `logger` expone un logger global y una fábrica para crear loggers con nivel específico.

### Variables de entorno

- `APP_ENV=development`: usa salida tipo consola, más legible para desarrollo.
- cualquier otro valor: mantiene la configuración base de producción.

### Uso básico

```go
package main

import (
	"github.com/hugomode/helper/logger"
)

func main() {
	logger.InitLogger()

	logger.Log.Info("aplicacion iniciada")

	debugLog := logger.NewLoggerWithLevel("debug")
	debugLog.Debug("mensaje de debug")
}
```

### Integración con GORM

Si inicializas la base de datos con `db.GetDBPostgres()`, el módulo ya conecta GORM con `zap` usando `logger.NewZapGormLogger(...)`.

## Base de datos

El paquete `db` crea una conexión singleton a PostgreSQL y trae helpers para filtros y paginación.

### Variables de entorno

Requeridas:

- `DB_POSTGRES_HOST`
- `DB_POSTGRES_PORT`
- `DB_POSTGRES_USER`
- `DB_POSTGRES_PASS`
- `DB_POSTGRES_NAME`
- `DB_POSTGRES_SCHEMA`

Opcionales para pool de conexiones:

- `DB_POSTGRES_MAX_OPEN_CONNECTIONS`
- `DB_POSTGRES_MAX_IDLE_CONNECTIONS`
- `DB_POSTGRES_MAX_OPEN_CONNECTIONS_TIMEOUT`
- `DB_POSTGRES_MAX_IDLE_CONNECTIONS_TIMEOUT`

### Crear la conexión

```go
package main

import (
	"github.com/hugomode/helper/db"
	"github.com/hugomode/helper/logger"
)

func main() {
	logger.InitLogger()

	conn, err := db.GetDBPostgres()
	if err != nil {
		panic(err)
	}

	if err := db.HealthheckPostgresHandler(); err != nil {
		panic(err)
	}

	_ = conn
}
```

### Paginación

`GetDataQueryPagination` aplica `OFFSET` y `LIMIT` sobre un `*gorm.DB`.

```go
var users []User

conn, err := db.GetDBPostgres()
if err != nil {
	return err
}

query := conn.Model(&User{}).Order("id desc")

if err := db.GetDataQueryPagination(20, 1, &users, query); err != nil {
	return err
}
```

Reglas relevantes:

- `pageSize` debe ser mayor a `0`
- si `pageNumber` es `0`, internamente se usa `1`

### Consultas en paralelo

`RunQueriesInParallel` sirve para ejecutar en paralelo la consulta de datos y la consulta de conteo.
Para el conteo, usa `GetCountQuery` para clonar la query base y limpiar `ORDER`, `LIMIT` y `OFFSET`.

```go
package service

import (
	"context"

	"github.com/hugomode/helper/db"
	"github.com/hugomode/helper/myHttp"
	"gorm.io/gorm"
)

func GetUsersPage(ctx context.Context, tx *gorm.DB, pageSize, pageNumber uint) (*myHttp.JSONData, error) {
	var users []User
	var total int64

	baseQuery := tx.Model(&User{})

	err := db.RunQueriesInParallel(
		ctx,
		func() error {
			return db.GetDataQueryPagination(pageSize, pageNumber, &users, baseQuery.Order("id desc"))
		},
		func() error {
			return db.GetCountQuery(baseQuery).Count(&total).Error
		},
	)
	if err != nil {
		return nil, err
	}

	return myHttp.ResponseWithPagination(users, total, pageSize, pageNumber), nil
}
```

### Filtros auxiliares

#### `LikesQuery`

Construye una condición agrupada:

```sql
AND (cond1 OR cond2 OR ...)
```

Ejemplo:

```go
names := []string{"hugo", "mode"}

query := db.LikesQuery(
	conn.Model(&User{}),
	names,
	"name",
	db.LikeMatchContains,
)
```

Modos soportados:

- `db.LikeMatchExact`
- `db.LikeMatchPrefix`
- `db.LikeMatchSuffix`
- `db.LikeMatchContains`

Para `string` usa `lower(columna) LIKE ?`. Para tipos numéricos y `bool`, hace `cast(... as varchar) LIKE ?`.

#### `BetweenDates`

Aplica filtros por rango sobre una columna de fecha:

```go
query := db.BetweenDates(
	conn.Model(&Order{}),
	startDate,
	endDate,
	"created_at",
)
```

Comportamiento:

- si `desde` y `hasta` existen, usa `BETWEEN`
- si solo existe `hasta`, usa `<`
- si solo existe `desde`, usa `>`

## HTTP

El paquete `myHttp` envuelve `net/http` y agrega:

- headers por defecto en JSON
- logging
- reintentos
- impresión opcional del request como `curl`
- auth básica

### Crear un cliente

```go
package main

import (
	"context"
	"time"

	"github.com/hugomode/helper/myHttp"
)

func main() {
	client := myHttp.New(context.Background())
	client.SetCallRetry(3)
	client.SetPrintCurl(true)
	client.SetLevel("debug")

	resp, err := client.GetRest("https://api.example.com/users", 5*time.Second)
	if err != nil {
		panic(err)
	}

	_ = resp
}
```

### Métodos disponibles

- `GetRest(url, timeout)`
- `PostRest(url, body, timeout)`
- `PutRest(url, body, timeout)`
- `PatchRest(url, body, timeout)`

### Configuración adicional

```go
import "net/http"

client := myHttp.New(ctx)

client.SetHeader(http.Header{
	"Authorization": []string{"Bearer <token>"},
})

client.AddHeader(http.Header{
	"X-Trace-ID": []string{"abc-123"},
})

client.SetBasicAuthenticacion("user", "pass")
client.SetCallRetry(2)
client.SetPrintCurl(false)
```

### Respuesta estándar paginada

`myHttp.ResponseWithPagination(...)` devuelve:

```go
type JSONData struct {
	Data       any       `json:"data"`
	TotalPages *uint     `json:"total_pages,omitempty"`
	PageNumber *uint     `json:"page_number,omitempty"`
	PageSize   *uint     `json:"page_size,omitempty"`
	Count      *int64    `json:"count,omitempty"`
	Errors     []*string `json:"errors,omitempty"`
}
```

## Tests

Para ejecutar la suite:

```bash
go test ./...
```

Actualmente hay cobertura sobre:

- inicialización del logger
- validación de paginación y ejecución paralela
- cliente HTTP con `httptest`

## Notas

- la conexión PostgreSQL se inicializa una sola vez por proceso usando `sync.Once`
- `logger.InitLogger()` debe ejecutarse antes de usar `logger.Log` o antes de abrir la conexión de base de datos
