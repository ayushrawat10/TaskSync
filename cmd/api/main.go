package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"tasksync/internal/data"
	"tasksync/internal/jsonlog"
	"tasksync/internal/mailer"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  int
	}

	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
    cors struct {
        trustedOrigins []string
    }
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
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
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
    flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
        cfg.cors.trustedOrigins = strings.Fields(val)
        return nil
    })

	flag.Parse()

	cfg.db.dsn = os.Getenv("DB_DSN")
	if cfg.db.dsn == "" {
		log.Fatal("DB_DSN must be set in .env or as an environment variable")
	}
	maxOpenConns, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS"))
	if err != nil || maxOpenConns <= 0 {
		cfg.db.maxOpenConns = 10 // Set your default value here
	} else {
		cfg.db.maxOpenConns = maxOpenConns
	}

	// For maxIdleTime
	maxIdleTime, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_TIME"))
	if err != nil || maxIdleTime <= 0 {
		cfg.db.maxIdleTime = 10 * 60 * 1000 // Set your default value here
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

	cfg.smtp.host = os.Getenv("SMTP_HOST")
	if cfg.smtp.host == "" {
		cfg.smtp.host = "smtp.mailtrap.io" // default value
	}

	cfg.smtp.port, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil || cfg.smtp.port == 0 {
		cfg.smtp.port = 25 // default value
	}

	cfg.smtp.username = os.Getenv("SMTP_USERNAME")
	if cfg.smtp.username == "" {
		cfg.smtp.username = "0abf276416b183" // default value
	}

	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")
	if cfg.smtp.password == "" {
		cfg.smtp.password = "d8672aa2264bb5" // default value
	}

	cfg.smtp.sender = os.Getenv("SMTP_SENDER")
	if cfg.smtp.sender == "" {
		cfg.smtp.sender = "Greenlight <no-reply@greenlight.alexedwards.net>" // default value
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db.Database("tasksync")),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
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
