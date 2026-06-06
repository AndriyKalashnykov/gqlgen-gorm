// Command server runs the schema-first GraphQL todo API.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/AndriyKalashnykov/gqlgen-gorm/graph/generated"
	resolvers "github.com/AndriyKalashnykov/gqlgen-gorm/graph/resolvers"
	common2 "github.com/AndriyKalashnykov/gqlgen-gorm/internal/common"
)

const defaultPort = 4000

func main() {
	port := resolvePort()

	// `server -healthcheck` is used by the container HEALTHCHECK; it probes
	// the running server and exits 0 (healthy) or 1 (unhealthy).
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck(port))
	}

	db, err := common2.InitDb()
	if err != nil {
		log.Fatal(err)
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{}}))

	customCtx := &common2.CustomContext{
		Database: db,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", common2.CreateContext(customCtx, srv))

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("connect to http://localhost:%d/ for GraphQL playground", port)
	log.Fatal(httpServer.ListenAndServe())
}

// resolvePort reads the PORT env var, falling back to defaultPort.
func resolvePort() int {
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal("PORT must be a valid integer")
		}
		return p
	}
	return defaultPort
}

// healthcheck probes /healthz and returns a process exit code.
func healthcheck(port int) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	url := "http://localhost:" + strconv.Itoa(port) + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
