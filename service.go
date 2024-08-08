package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const DEFAULT_LIMIT = 100

type StatusResponse struct {
	Status string
}

func main() {

	fmt.Println("Started")
	db := connectDb()
	defer db.Close()

	go updaetTorNodes(db)

	r := mux.NewRouter()
	r.HandleFunc("/nodes", wrapDB(db, listNodes)).Methods("GET")
	r.HandleFunc("/nodes/time-range", wrapDB(db, listNodeTimeRange)).Methods("GET")
	r.HandleFunc("/allow-list", wrapDB(db, listAllowList)).Methods("GET")
	r.HandleFunc("/allow-list/{address}", wrapDB(db, addAllowList)).Methods("PUT")
	r.HandleFunc("/allow-list/{address}", wrapDB(db, deleteAllowList)).Methods("DELETE")
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

func extractPagingParameters(w http.ResponseWriter, r *http.Request) (int, string, error) {
	limit := DEFAULT_LIMIT
	var err error
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return 0, "", err
		}
	}
	pagingToken := r.URL.Query().Get("token")
	return limit, pagingToken, nil
}

func listNodes(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	limit, pagingToken, err := extractPagingParameters(w, r)
	if err != nil {
		return
	}

	page, error := ListCurrentNodes(db, pagingToken, limit)
	if error != nil {
		http.Error(w, "Got error looking up current ndoes", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(page)
}

func listNodeTimeRange(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	startTime := time.Now().Add(time.Duration(-168 * float64(time.Hour)))
	endTime := time.Now()

	limit, pagingToken, err := extractPagingParameters(w, r)
	if err != nil {
		return
	}

	if startTsStr := r.URL.Query().Get("start_ts"); startTsStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTsStr)
		if err != nil {
			http.Error(w, "Invalid start_ts parameter, use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	if endTsStr := r.URL.Query().Get("end_ts"); endTsStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTsStr)
		if err != nil {
			http.Error(w, "Invalid end_ts parameter, use RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	page, error := ListNodesInTimeRange(db, pagingToken, limit, startTime, endTime)
	if error != nil {
		http.Error(w, "Got error looking up current ndoes", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(page)
}

func listAllowList(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	limit, pagingToken, err := extractPagingParameters(w, r)
	if err != nil {
		return
	}
	page, error := ListAllowListNodes(db, pagingToken, limit)
	if error != nil {
		http.Error(w, "Got error looking up allow list", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(page)
}

func addAllowList(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	vars := mux.Vars(r)

	nodeAddress := vars["address"]

	if net.ParseIP(nodeAddress) == nil {
		http.Error(w, "Invalid IP address format", http.StatusBadRequest)
		return
	}

	err := AddAllowListNode(db, nodeAddress)
	if err != nil {
		http.Error(w, "Got error adding allow list entry", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(StatusResponse{Status: "OK"})
}

func deleteAllowList(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	vars := mux.Vars(r)

	nodeAddress := vars["address"]

	if net.ParseIP(nodeAddress) == nil {
		http.Error(w, "Invalid IP address format", http.StatusBadRequest)
		return
	}

	err := DeleteAllowListNode(db, nodeAddress)
	if err != nil {
		http.Error(w, "Got error deleting allow list entry", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(StatusResponse{Status: "OK"})
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
