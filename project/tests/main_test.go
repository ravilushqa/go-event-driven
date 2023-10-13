package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"

	"tickets/db"
)

var (
	startedContainers = make([]testcontainers.Container, 0)
	postgresURL       = os.Getenv("POSTGRES_URL")
	redisURL          = os.Getenv("REDIS_ADDR")
)

func TestMain(m *testing.M) {
	var code int
	// recover from panic if one occurred. Set exit code to 1 if there was a panic.
	defer func() {
		if r := recover(); r != nil {
			code = 1
			teardown(&code)
		}
	}()
	setup()
	defer teardown(&code)
	code = m.Run()
}

func teardown(i *int) {
	ctx := context.Background()
	for _, container := range startedContainers {
		err := container.Terminate(ctx)
		if err != nil {
			fmt.Printf("\033[1;31m%s\033[0m", "> Teardown failed\n")
		}
	}

	fmt.Printf("\033[1;33m%s\033[0m", "> Teardown completed\n")

	os.Exit(*i)
}

func setup() {
	// init test environment

	// Postgres
	if postgresURL == "" {
		fmt.Printf("\033[1;33m%s\033[0m", "> Setup postgres container\n")
		postgresContainer, connStr := db.StartPostgresContainer()
		postgresURL = connStr
		startedContainers = append(startedContainers, postgresContainer)
	}

	dbconn, err := sqlx.Open("postgres", postgresURL)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()

	fmt.Printf("\033[1;33m%s\033[0m", "> Setup database schema\n")
	err = db.InitializeDatabaseSchema(dbconn)
	if err != nil {
		panic(err)
	}

	// Redis
	if redisURL == "" {
		fmt.Printf("\033[1;33m%s\033[0m", "> Setup redis container\n")
		redisContainer, connStr := startRedisContainer()
		redisURL = connStr
		startedContainers = append(startedContainers, redisContainer)
	}

	fmt.Printf("\033[1;33m%s\033[0m", "> Setup completed\n")
}

func startRedisContainer() (testcontainers.Container, string) {
	ctx := context.Background()
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("docker.io/redis:7"),
		redis.WithSnapshotting(10, 1),
		redis.WithLogLevel(redis.LogLevelVerbose),
	)
	if err != nil {
		panic(err)
	}

	uri, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}

	return redisContainer, strings.Replace(uri, "redis://", "", 1)
}
