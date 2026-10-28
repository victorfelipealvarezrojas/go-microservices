package main

import (
	"authentication/data"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	// Por qué hay que agregarlos a mano:
	// database/sql solo define interfaces; no sabe hablar con ninguna base de datos.
	// Cada driver se registra a sí mismo en su init() llamando a sql.Register("nombre", driver).
	// al ejecutar sql.Open("pgx", dsn) elige el driver por ese string.
	// Como no hay ninguna referencia en el código a un identificador del paquete,
	// ni el compilador ni goimports pueden deducir que hace falta. Sin el import el
	// driver nunca se registra, asi que se ejecuta su init() por medio de los import manuales

	_ "github.com/jackc/pgconn"        // capa de conexión de bajo nivel a Postgres (protocolo, auth, TLS). Redundante aquí: pgx ya la importa
	_ "github.com/jackc/pgx/v4"        // driver/cliente nativo de Postgres. Redundante aquí: stdlib ya lo importa
	_ "github.com/jackc/pgx/v4/stdlib" // adaptador de pgx para database/sql. Su init() registra el driver con el nombre "pgx". ESTE es el necesario
)

const webPort = "80"

var counts int64

type Config struct {
	DB     *sql.DB
	Models data.Models
}

func main() {
	log.Println("Starting authentication service")

	conn := conectToDB()
	if conn == nil {
		log.Panic("Can't connect to Postgres!")
	}

	app := &Config{
		DB:     conn,
		Models: data.New(conn),
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func conectToDB() *sql.DB {
	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)
		if err != nil {
			log.Println("Postgres not yet ready...")
			counts++
		} else {
			log.Println("Connected to postgres!")
			return connection
		}

		if counts > 10 {
			log.Panicln(err)
			return nil
		}

		log.Println("Backing off for two seconds...")
		time.Sleep(2 * time.Second)
		continue
	}
}
