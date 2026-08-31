package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: testdbcheck <postgres-url>")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL test database configuration failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL test database connection failed: %v\n", err)
		os.Exit(1)
	}
}
