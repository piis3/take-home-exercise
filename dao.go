package main

import (
	"database/sql"
	"log"
	"prophet/takehome/.gen/prophet/public/model"
	. "prophet/takehome/.gen/prophet/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

func ListCurrentNodes(db *sql.DB) ([]model.ExitNodes, error) {
	var models []model.ExitNodes

	maxQuery := (SELECT(MAX(NodeFetchHistory.FetchTime).AS("max_time")).FROM(NodeFetchHistory)).AsTable("fetchHistory")

	query := SELECT(ExitNodes.NodeAddress, ExitNodes.FetchTime).
		FROM(maxQuery.INNER_JOIN(ExitNodes, TimestampColumn("max_time").From(maxQuery).EQ(ExitNodes.FetchTime)))
	log.Println(query.Sql())

	err := query.Query(db, &models)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return models, nil
}
