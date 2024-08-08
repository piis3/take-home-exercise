package main

import (
	"bufio"
	"database/sql"
	"log"
	"net/http"
	"os"
	"prophet/takehome/.gen/prophet/public/model"
	. "prophet/takehome/.gen/prophet/public/table"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
)

const (
	// DEFAULT_URL = "http://localhost:8081/dan.me.torlist"
	DEFAULT_URL = "https://www.dan.me.uk/torlist/?exit"
)

func fetchTorNodes() []string {
	url := DEFAULT_URL
	if envUrl := os.Getenv("TOR_NODE_URL"); envUrl != "" {
		url = envUrl
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var nodes []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		nodes = append(nodes, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Got error fetching tor nodes from %s, %v", url, err)
		return nil
	}

	return nodes
}

func WriteTorNodes(db *sql.DB) {
	tx, err := db.Begin()

	if err != nil {
		log.Printf("Got error beginning db transaction, %v", err)
		return
	}

	var lockDest []model.NodeFetchHistory

	// get an exclusive lock on the DB
	lock := SELECT(NodeFetchHistory.ID, NodeFetchHistory.FetchTime).
		FROM(NodeFetchHistory).
		WHERE(NodeFetchHistory.FetchTime.
			GT(LOCALTIMESTAMP().SUB(INTERVAL(1, HOUR)))).
		FOR(UPDATE().NOWAIT())

	err = lock.Query(tx, &lockDest)
	if err != nil {
		// This just means we didn't get the lock
		return
	}

	if len(lockDest) > 0 {
		// No need to run
		log.Print("Skipping fetch, this has been updated recently")
		return
	}

	var currentTime time.Time

	SELECT(LOCALTIMESTAMP()).Query(tx, &currentTime)

	nodes := fetchTorNodes()
	models := make([]model.ExitNodes, len(nodes))
	for i, v := range nodes {
		models[i] = model.ExitNodes{NodeAddress: v, FetchTime: currentTime}
	}

	_, err = ExitNodes.INSERT(ExitNodes.NodeAddress, ExitNodes.FetchTime).MODELS(models).Exec(tx)
	if err != nil {
		log.Println(err)
	}

	_, err = NodeFetchHistory.INSERT(NodeFetchHistory.FetchTime).VALUES(currentTime).Exec(tx)
	if err != nil {
		log.Println(err)
	}

	tx.Commit()
}
