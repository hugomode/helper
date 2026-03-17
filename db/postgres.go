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
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbname)
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
				TablePrefix: schemaname + ".",
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

// LikesQuery builds a GORM query with OR conditions using the LIKE operator for the specified slice of elements.
// It supports string and int types, applying lower() to strings and casting ints to varchar.
func LikesQuery[T *int | *string | *bool](query *gorm.DB, slice []T, nameColumn string) *gorm.DB {
	for _, element := range slice {
		switch any(element).(type) {
		case *string:
			var str *string
			str = any(element).(*string)
			query = query.Or(fmt.Sprintf("lower(%s) LIKE ?", nameColumn), strings.ToLower(*str))
		case *int:
			var str *int
			str = any(element).(*int)
			query = query.Or(fmt.Sprintf("cast(%s as varchar) LIKE ?", nameColumn), strconv.Itoa(*str))
		}
	}
	return query
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
