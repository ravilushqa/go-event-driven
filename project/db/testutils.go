package db

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	db        *sqlx.DB
	getDbOnce sync.Once
)

func GetDb(t *testing.T) *sqlx.DB {
	getDbOnce.Do(func() {
		var err error
		db, err = sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
		assert.NoError(t, err)
		t.Cleanup(func() {
			db.Close()
		})

		err = InitializeDatabaseSchema(db)
		assert.NoError(t, err)
	})
	return db
}

func StartPostgresContainer() (testcontainers.Container, string) {
	ctx := context.Background()
	dbName := "db"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/postgres:15.2-alpine"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		panic(err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable", "application_name=test")

	return postgresContainer, connStr
}
