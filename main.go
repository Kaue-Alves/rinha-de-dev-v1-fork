package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	db *pgxpool.Pool
}

type Evento struct {
	ID                  int32  `json:"id"`
	Nome                string `json:"nome"`
	IngressosDisponiveis int32 `json:"ingressos_disponiveis"`
}

type ReservaRequest struct {
	EventoID  int32 `json:"evento_id"`
	UsuarioID int32 `json:"usuario_id"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := newDBPool(ctx)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer pool.Close()

	app := &App{db: pool}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /eventos", app.handleGetEventos)
	mux.HandleFunc("POST /reservas", app.handlePostReservas)

	server := &http.Server{
		Addr:              ":" + getEnvIntString("APP_PORT", "8080"),
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}
}

func newDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	host := getEnv("DB_HOST", getEnv("HOST", "localhost"))
	port := getEnv("DB_PORT", getEnv("PORT", "5432"))
	user := getEnv("DB_USER", getEnv("DBUSER", "admin"))
	password := getEnv("DB_PASS", getEnv("PASSWORD", "123"))
	database := getEnv("DB_NAME", getEnv("DATABASE", "rinha"))

	dsn := "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + database
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(getEnvInt("DB_POOL_MAX", 10))
	config.MinConns = int32(getEnvInt("DB_POOL_MIN", 2))
	config.HealthCheckPeriod = 15 * time.Second
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func (a *App) handleGetEventos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	rows, err := a.db.Query(ctx, "SELECT id, nome, ingressos_disponiveis FROM eventos")
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	eventos := make([]Evento, 0, 4)
	for rows.Next() {
		var e Evento
		if err := rows.Scan(&e.ID, &e.Nome, &e.IngressosDisponiveis); err != nil {
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		eventos = append(eventos, e)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(eventos); err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
	}
}

func (a *App) handlePostReservas(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var payload ReservaRequest
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "Você mandou algo errado.", http.StatusBadRequest)
		return
	}

	if payload.EventoID <= 0 || payload.UsuarioID <= 0 {
		http.Error(w, "Você mandou algo errado.", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 400*time.Millisecond)
	defer cancel()

	commandTag, err := a.db.Exec(ctx, `
		WITH updated AS (
			UPDATE eventos
			SET ingressos_disponiveis = ingressos_disponiveis - 1
			WHERE id = $1 AND ingressos_disponiveis > 0
			RETURNING id
		)
		INSERT INTO reservas (evento_id, usuario_id)
		SELECT id, $2 FROM updated;
	`, payload.EventoID, payload.UsuarioID)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}

	if commandTag.RowsAffected() == 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvIntString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	if _, err := strconv.Atoi(value); err != nil {
		return fallback
	}
	return value
}
