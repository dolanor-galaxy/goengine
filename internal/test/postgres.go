// +build integration

package test

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/lib/pq"
)

// PostgresDatabase provides a database connection to the callback for conducting integration tests
func PostgresDatabase(t *testing.T, testCase func(db *sql.DB)) {
	ctrl := postgresController(t)

	dsn := postgresDSN(t)
	dsnMatches := postgresDSNDatabaseMatch(dsn)
	databaseName := dsn[dsnMatches[2]:dsnMatches[3]]

	// Create the schema to use
	ctrl.Create(t, databaseName)
	defer ctrl.Drop(t, databaseName)

	// Open db connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("test.postgres: Connection failed: %+v", err)
	}
	defer db.Close()

	testCase(db)
}

var postgresControl *dbController

type dbController struct {
	db *sql.DB
}

func postgresController(t *testing.T) *dbController {
	if postgresControl == nil {
		dsn := postgresDSN(t)

		// Extract the postgres db name and replace it with postgres
		matches := postgresDSNDatabaseMatch(dsn)
		postgresDSN := dsn[:matches[0]] + "dbname=postgres" + dsn[matches[1]:]

		// Open the connection
		postgresDB, err := sql.Open("postgres", postgresDSN)
		if err != nil {
			t.Fatalf("test.postgres: failed to connect to postgres db: %+v", err)
		}

		postgresControl = &dbController{postgresDB}
	}

	return postgresControl
}

func (c *dbController) Drop(t *testing.T, databaseName string) {
	c.disableDatabaseAccess(t, databaseName)

	_, err := c.db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, databaseName))
	if err != nil {
		t.Fatalf("test.postgres: Fail to drop database. %+v", err)
	}
}

func (c *dbController) Create(t *testing.T, databaseName string) {
	_, err := c.db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, databaseName))
	if err != nil {
		// If the database already exists continue
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P04" {
			c.enableDatabaseAccess(t, databaseName)
			return
		}

		t.Fatalf("test.postgres: Fail to create database. %+v", err)
	}
	c.enableDatabaseAccess(t, databaseName)
}

func (c *dbController) disableDatabaseAccess(t *testing.T, databaseName string) {
	// Making sure the database exists
	row := c.db.QueryRow("SELECT datname FROM pg_database WHERE datname = $1", databaseName)
	if row == nil {
		// No database so no one has access
		return
	}
	// Disallow new connections
	_, err := c.db.Exec(fmt.Sprintf(`ALTER DATABASE "%s" WITH ALLOW_CONNECTIONS false`, databaseName))
	if err != nil {
		t.Fatalf("test.postgres: Unable to disallow connections to the db (%v)", err)
	}

	// Terminate existing connections
	row = c.db.QueryRow("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", databaseName)
}

func (c *dbController) enableDatabaseAccess(t *testing.T, databaseName string) {
	_, err := c.db.Exec(fmt.Sprintf(`ALTER DATABASE "%s" WITH ALLOW_CONNECTIONS true`, databaseName))
	if err != nil {
		t.Fatalf("test.postgres: Unable to allow connections to the db (%v)", err)
	}
}

// postgresDSN returns a parsed postgres dsn
func postgresDSN(t *testing.T) string {
	// Fetch the postgres dsn from the env var
	osDSN, exists := os.LookupEnv("POSTGRES_DSN")
	if !exists {
		t.Fatalf("test.postgres: missing POSTGRES_DSN enviroment variable")
	}

	// Parse the postgres dsn
	parsedDSN, err := pq.ParseURL(osDSN)
	if err != nil {
		t.Fatalf("test.postgres: failed to parse postgres dsn (%v)\n", err)
	}

	return parsedDSN
}

// postgresDSNDatabaseMatch locate the dbname within the dsn and return the indexes
func postgresDSNDatabaseMatch(dsn string) []int {
	r := regexp.MustCompile(`dbname=(((\\ )|[^ ])+)`)
	matches := r.FindStringSubmatchIndex(dsn)

	return matches
}