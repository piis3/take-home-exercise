package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"prophet/takehome/.gen/prophet/public/model"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {

	fmt.Println("Started")
	db := connectDb()
	defer db.Close()

	go updaetTorNodes(db)

	r := mux.NewRouter()
	r.HandleFunc("/nodes", wrapDB(db, listNodes)).Methods("GET")
	http.ListenAndServe(":8080", r)
}

func updaetTorNodes(db *sql.DB) {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			WriteTorNodes(db)
		}
	}
}

func wrapDB(db *sql.DB, f func(*sql.DB, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		f(db, w, r)
	}
}

type ListNodeResponse struct {
	ExitNodes []model.ExitNodes
}

func listNodes(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	nodes, error := ListCurrentNodes(db)
	if error != nil {
		http.Error(w, "Got error looking up current ndoes", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ListNodeResponse{ExitNodes: nodes})
}

func connectDb() *sql.DB {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbUser := "prophet"
	dbPassword := "password"
	dbName := "prophet"
	dbPort := "5432"

	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}
