# Helper Utility

Este proyecto es una colección de utilitarios en Go para simplificar tareas comunes como el manejo de logs con Zap, la conexión y consultas a bases de datos Postgres mediante GORM, y la realización de peticiones HTTP con reintentos y logging integrado.

## Tabla de Contenidos
- [Instalación](#instalación)
- [Logger](#logger)
- [Base de Datos (Postgres)](#base-de-datos-postgres)
- [Paginación y Consultas Paralelas](#paginación-y-consultas-paralelas)
- [HTTP Util](#http-util)
- [Pruebas (Testing)](#pruebas-testing)

---

## Instalación

Para importar este utilitario en tu proyecto Go:

```bash
go get -u github.com/hugomode/helper
```

---

## Logger

El paquete `logger` proporciona una configuración base para `uber-go/zap` con soporte para diferentes entornos y niveles de log.

### Configuración
Puede configurarse mediante las siguientes variables de entorno:
- `APP_ENV`: Si es `development`, los logs se muestran con formato amigable y colores (console encoding). Por defecto usa configuración de producción (JSON).

### Ejemplo de Uso
```go
import "go.hugomode/helper/logger"

func main() {
    // Inicializar el logger global
    logger.InitLogger()
    
    // Usar el logger global
    logger.Log.Info("Iniciando aplicación...")
    
    // Crear un logger con un nivel específico
    customLog := logger.NewLoggerWithLevel("debug")
    customLog.Debug("Este es un mensaje de debug")
}
```

---

## Base de Datos (Postgres)

El paquete `db` facilita la conexión a PostgreSQL utilizando GORM, manejando automáticamente el pool de conexiones y el esquema.

### Variables de Entorno Requeridas
- `DB_POSTGRES_HOST`, `DB_POSTGRES_PORT`, `DB_POSTGRES_USER`, `DB_POSTGRES_PASS`, `DB_POSTGRES_NAME`, `DB_POSTGRES_SCHEMA`

---

## Paginación y Consultas Paralelas

El paquete `db` incluye funciones para simplificar la paginación y la ejecución de consultas concurrentes (útil para obtener datos y conteo total).

### Ejemplo de Paginación y Consultas Paralelas
```go
import (
    "context"
    "go.hugomode/helper/db"
    "go.hugomode/helper/http"
)

func ObtenerUsuariosPaginados(pageSize, pageNumber uint) (*http.JSONData, error) {
    var usuarios []Usuario
    var total int64
    ctx := context.Background()
    
    tx, _ := db.GetDBPostgres()

    err := db.RunQueriesInParallel(ctx,
        func() error {
            // Consulta de datos con paginación
            return db.GetDataQueryPagination(pageSize, pageNumber, &usuarios, tx)
        },
        func() error {
            // Consulta de conteo total
            return tx.Model(&Usuario{}).Count(&total).Error
        },
    )

    if err != nil {
        return nil, err
    }

    // Generar respuesta estandarizada
    return http.ResponseWithPagination(usuarios, total, pageSize, pageNumber), nil
}
```

---

## HTTP Util

El paquete `http` proporciona un cliente robusto con soporte para retries, logging y manejo de respuestas JSON.

### Ejemplo de Uso
```go
import (
    "context"
    "time"
    "go.hugomode/helper/http"
)

func main() {
    ctx := context.Background()
    client := http.New(ctx)

    client.SetCallRetry(3)
    client.SetPrintCurl(true)

    resp, err := client.GetRest("https://api.ejemplo.com/data", 5*time.Second)
    if err == nil {
        fmt.Printf("Status: %d, Body: %s\n", resp.StatusCode, string(resp.Data))
    }
}
```

#### Estructura de Respuesta Estándar
```go
type JSONData struct {
    Data       any       `json:"data"`
    TotalPages *uint     `json:"total_pages,omitempty"`
    PageNumber *uint     `json:"page_number,omitempty"`
    PageSize   *uint     `json:"page_size,omitempty"`
    Count      *uint64   `json:"count,omitempty"`
    Errors     []*string `json:"errors,omitempty"`
}
```

---

## Pruebas (Testing)

El proyecto incluye tests unitarios para los paquetes `logger`, `db` y `http`. Para ejecutar las pruebas, asegúrate de haber instalado las dependencias (especialmente `testify`) y corre el siguiente comando desde la raíz del proyecto:

```bash
go test -v ./...
```

Estas pruebas verifican:
- Inicialización y niveles del logger.
- Lógica de paginación y ejecución paralela segura.
- Configuración de cabeceras y peticiones HTTP (usando `httptest`).
