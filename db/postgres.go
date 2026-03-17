package db

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hugomode/helper/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	dbPostgres     *gorm.DB
	dbOncePostgres sync.Once
)

type LikeMatchMode string
type LikeQueryValue interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~bool
}

const (
	LikeMatchExact      LikeMatchMode = "exact"
	LikeMatchPrefix     LikeMatchMode = "prefix"
	LikeMatchSuffix     LikeMatchMode = "suffix"
	LikeMatchContains   LikeMatchMode = "contains"
	LikeMatchPrefixOnly LikeMatchMode = LikeMatchPrefix
	LikeMatchSuffixOnly LikeMatchMode = LikeMatchSuffix
)

// GetDBPostgres initializes and returns a singleton instance of the GORM Postgres database connection.
// It configures connection pooling based on environment variables.
func GetDBPostgres() (*gorm.DB, error) {
	var err error
	host := os.Getenv("DB_POSTGRES_HOST")
	port := os.Getenv("DB_POSTGRES_PORT")
	user := os.Getenv("DB_POSTGRES_USER")
	pass := os.Getenv("DB_POSTGRES_PASS")
	dbname := os.Getenv("DB_POSTGRES_NAME")
	schemaname := os.Getenv("DB_POSTGRES_SCHEMA")
	// Utiliza la función sync.Once para garantizar que la conexión se establezca solo una vez.
	dbOncePostgres.Do(func() {
		// Configura la cadena de conexión de PostgreSQL
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s", host, port, user, pass, dbname, schemaname)
		newLogger := logger.NewZapGormLogger(logger.Log)
		// Abre la conexión a la base de datos
		dbPostgres, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   fmt.Sprintf("%s.", schemaname), // schema name
				SingularTable: true,
			},
			PrepareStmt: true,
			Logger:      newLogger,
		})
		if err != nil {
			panic("Error connecting to the database: " + err.Error())
		}
		debe, err := dbPostgres.DB()
		if err != nil {
			panic("Error obtaining/configuring connection pool: " + err.Error())
		}
		if schemaname != "" {
			logger.Log.Info("Using database schema", zap.String("schema", schemaname))
			// We can't easily change the DB connection's naming strategy after it's created
			// if GetDBPostgres doesn't allow it.
			// Actually, we can session into it with new config.
			dbPostgres = dbPostgres.Session(&gorm.Session{
				NewDB: true,
			})
			dbPostgres.Config.NamingStrategy = schema.NamingStrategy{
				TablePrefix:   fmt.Sprintf("%s.", schemaname),
				SingularTable: true,
			}
		}
		maxOpenConn := os.Getenv("DB_POSTGRES_MAX_OPEN_CONNECTIONS")
		maxIdleConn := os.Getenv("DB_POSTGRES_MAX_IDLE_CONNECTIONS")
		maxOpenConnTime := os.Getenv("DB_POSTGRES_MAX_OPEN_CONNECTIONS_TIMEOUT")
		maxIdleConnTime := os.Getenv("DB_POSTGRES_MAX_IDLE_CONNECTIONS_TIMEOUT")
		if maxOpenConn != "" {
			maxOpenConn, _ := strconv.Atoi(maxOpenConn)
			debe.SetMaxOpenConns(maxOpenConn)
		}
		if maxIdleConn != "" {
			maxIdleConn, _ := strconv.Atoi(maxIdleConn)
			debe.SetMaxIdleConns(maxIdleConn)
		}
		if maxOpenConnTime != "" {
			maxOpenConnTime, _ := strconv.Atoi(maxOpenConnTime)
			debe.SetConnMaxLifetime(time.Duration(maxOpenConnTime) * time.Second)
		}
		if maxIdleConnTime != "" {
			maxIdleConnTime, _ := strconv.Atoi(maxIdleConnTime)
			debe.SetConnMaxIdleTime(time.Duration(maxIdleConnTime) * time.Second)
		}
	})

	return dbPostgres, err
}

// HealthcheckPostgresHandler performs a ping to the Postgres database to verify connectivity.
func HealthheckPostgresHandler() error {
	sqlDB, err := dbPostgres.DB()
	if err != nil {
		return err
	}
	// Ping a la base de datos
	err = sqlDB.Ping()
	if err != nil {
		return err
	}
	return nil
}

// LikesQuery adds a grouped LIKE filter to the query:
// AND (cond1 OR cond2 ...)
func LikesQuery[T LikeQueryValue](query *gorm.DB, slice []T, nameColumn string, matchMode LikeMatchMode) *gorm.DB {
	conditions := make([]string, 0, len(slice))
	args := make([]any, 0, len(slice))
	columnExpr := buildLikeColumnExpr[T](nameColumn)

	for _, element := range slice {
		conditions = append(conditions, columnExpr)
		args = append(args, buildLikePattern(normalizeLikeValue(element), matchMode))
	}

	if len(conditions) == 0 {
		return query
	}

	return query.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func buildLikePattern(value string, matchMode LikeMatchMode) string {
	switch matchMode {
	case LikeMatchPrefix:
		return value + "%"
	case LikeMatchSuffix:
		return "%" + value
	case LikeMatchContains:
		return "%" + value + "%"
	case LikeMatchExact:
		fallthrough
	default:
		return value
	}
}

func buildLikeColumnExpr[T LikeQueryValue](nameColumn string) string {
	var zero T

	switch any(zero).(type) {
	case string:
		return fmt.Sprintf("lower(%s) LIKE ?", nameColumn)
	default:
		return fmt.Sprintf("cast(%s as varchar) LIKE ?", nameColumn)
	}
}

func normalizeLikeValue[T LikeQueryValue](value T) string {
	switch v := any(value).(type) {
	case string:
		return strings.ToLower(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.ToLower(fmt.Sprint(v))
	}
}

// BetweenDates filters a query based on a date range (start and end times) for a given column.
func BetweenDates(query *gorm.DB, desde, hasta *time.Time, nameColumn string) *gorm.DB {
	if desde != nil && hasta != nil {
		query = query.Where(fmt.Sprintf("%s BETWEEN ? AND ?", nameColumn), desde, hasta)
	} else if hasta != nil {
		query = query.Where(fmt.Sprintf("%s < ?", nameColumn), hasta)
	} else if desde != nil {
		query = query.Where(fmt.Sprintf("%s > ?", nameColumn), desde)
	}
	return query
}
