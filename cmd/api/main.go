package main

import (
	"context"
	"flag"
    "strconv"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
    "tasksync/internal/data"
    "tasksync/internal/jsonlog"
)

const version = "1.0.0"

type config struct {
    port int
    env string
    db struct {
        dsn string
        maxOpenConns int
        maxIdleConns int
        maxIdleTime int
    }
}

type application struct {
    config config
    logger *jsonlog.Logger
    models data.Models
}

func main() {
    logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
    err := godotenv.Load()
    if err != nil {
        log.Println("Error loading .env file, proceeding with default settings or command-line flags")
    }

    var cfg config

    flag.IntVar(&cfg.port, "port", 4000, "API server port")
    flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
    flag.Parse()

    cfg.db.dsn = os.Getenv("DB_DSN")
    if cfg.db.dsn == "" {
        log.Fatal("DB_DSN must be set in .env or as an environment variable")
    }
    maxOpenConns, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
    if err != nil || maxOpenConns <= 0 {
        cfg.db.maxOpenConns = 10  // Set your default value here
    } else {
        cfg.db.maxOpenConns = maxOpenConns
    }

    // For maxIdleTime
    maxIdleTime, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_TIME"))
    if err != nil || maxIdleTime <= 0 {
        cfg.db.maxIdleTime = 10*60*1000  // Set your default value here
    } else {
        cfg.db.maxIdleTime = maxIdleTime * 60 * 1000
    }

    db, err := openDB(cfg)
    if err != nil {
        log.Println(err)
    }
    defer func() {
        if err := db.Disconnect(context.Background()); err != nil {
            log.Println(err)
        }
    }()
    logger.PrintInfo("Database connection pool established", nil)

    app := &application{
        config: cfg,
        logger: logger,
        models: data.NewModels(db.Database("tasksync")),
    }

    srv := &http.Server{
        Addr: fmt.Sprintf(":%d", cfg.port),
        Handler: app.routes(),
        ErrorLog: log.New(logger, "", 0),
        IdleTimeout: time.Minute,
        ReadTimeout: 10*time.Second,
        WriteTimeout: 10*time.Second,
    }

    logger.PrintInfo("Starting server", map[string]string{
        "addr": srv.Addr,
        "env": cfg.env,
    })

    err = srv.ListenAndServe()
    logger.PrintFatal(err, nil)
}

func openDB(cfg config) (*mongo.Client, error) {
    clientOptions := options.Client().ApplyURI(cfg.db.dsn)
    clientOptions.SetMaxPoolSize(uint64(cfg.db.maxOpenConns))
    clientOptions.SetMaxConnIdleTime(time.Duration(cfg.db.maxIdleTime) * time.Millisecond)

    client, err := mongo.Connect(context.Background(), clientOptions)
    if err != nil {
        return nil, err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err = client.Ping(ctx, nil)
    if err != nil {
        client.Disconnect(context.Background())
        return nil, err
    }

    return client, nil
}