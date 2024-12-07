package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"
	"github.com/obynonwane/inventory-service/data"
)

const (
	webPort  = "80"
	rpcPort  = "5001"
	gRpcPort = "50001"
)

var counts int64

type Config struct {
	Repo   data.Repository
	Client *http.Client
}

func main() {

	log.Println("Starting inventory service")

	//Connect to DB
	conn := connectToDB()
	if conn == nil {
		log.Panic("can't connect to Postgres")
	}

	// Setup config with an initialized Repo
	app := Config{
		Repo:   data.NewPostgresRepository(conn),
		Client: &http.Client{},
	}

	// Pass the initialized Config to RPCServer
	rpcServer := &RPCServer{
		App: &app,
	}

	// Register RPC server: tell teh app e will be accepting rpc request
	err := rpc.Register(rpcServer)
	if err != nil {
		log.Panic("failed to register RPC server:", err)
	}

	//register gRPC: and start listening
	go app.grpcListen()

	// define http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	// start the server
	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return db, nil
}

func connectToDB() *sql.DB {

	dsn := DbConnectionDetails()
	for {
		connection, err := openDB(dsn)
		if err != nil {
			log.Println(err)
			log.Println("Postgres not yet ready ...")
			counts++
		} else {
			log.Println("Inventory Service Connected to Postgres ...")
			return connection
		}

		if counts > 10 {
			log.Println(err)
			return nil
		}

		log.Println("backing off for 2 seconds")
		time.Sleep(2 * time.Second)
		continue
	}
}

func DbConnectionDetails() string {

	environment := os.Getenv("DEV_ENV")

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found")
	}

	// Load environment-specific .env file
	goEnv := os.Getenv("DEV_ENV")
	if goEnv == "test" {
		err = godotenv.Load(".env.test")
		if err != nil {
			log.Println("No .env.test file found")
		}
	}

	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	dbname := os.Getenv("DATABASE_NAME")
	sslmode := os.Getenv("DATABASE_SSLMODE")
	timezone := os.Getenv("DATABASE_TIMEZONE")
	connectTimeout := os.Getenv("DATABASE_CONNECT_TIMEOUT")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s connect_timeout=%s",
		host, port, user, password, dbname, sslmode, timezone, connectTimeout,
	)

	log.Println(environment, "GO ENVIRONMENT")
	return connStr
}

func (app *Config) setupRepo(conn *sql.DB) {
	db := data.NewPostgresRepository(conn)
	app.Repo = db
}

func (app *Config) rpcListen() error {
	log.Println("starting RPC server on port", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		return err
	}

	// schduled to be executed after the execution of the rpcListen
	defer listen.Close()

	// a loop that executes forever that keeps listenning for connection
	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			return err
		}

		// start the rpcConn in a different thread using goroutine to avoid waiting
		// in line for other processes to complete
		go rpc.ServeConn(rpcConn)
	}
}
