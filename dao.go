package main

import (
	"database/sql"
	"log"
	"prophet/takehome/.gen/prophet/public/model"
	. "prophet/takehome/.gen/prophet/public/table"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/samber/lo"
)

type PagedResult[T any] struct {
	Page           []T
	HasMoreResults bool
	PagingToken    string
}

func paginate[T any](models []T, limit int, tokenExtractor func(T) string) PagedResult[T] {

	hasMoreResults := len(models) > limit
	if hasMoreResults {
		models = models[0:limit]
	}

	token := ""
	if len(models) > 0 {
		token = tokenExtractor(models[len(models)-1])
	}
	return PagedResult[T]{
		Page:           models,
		HasMoreResults: hasMoreResults,
		PagingToken:    token,
	}
}

func ListCurrentNodes(db *sql.DB, pagingToken string, limit int) (PagedResult[model.ExitNodes], error) {
	var models []model.ExitNodes

	maxQuery := (SELECT(MAX(NodeFetchHistory.FetchTime).AS("max_time")).FROM(NodeFetchHistory)).AsTable("fetchHistory")

	query := SELECT(ExitNodes.NodeAddress, ExitNodes.FetchTime).
		FROM(maxQuery.INNER_JOIN(ExitNodes, TimestampColumn("max_time").From(maxQuery).EQ(ExitNodes.FetchTime)).
			LEFT_JOIN(AllowList, ExitNodes.NodeAddress.EQ(AllowList.NodeAddress))).
		WHERE(ExitNodes.NodeAddress.GT(String(pagingToken)).AND(AllowList.NodeAddress.IS_NULL())).
		ORDER_BY(ExitNodes.NodeAddress.ASC()).
		LIMIT(int64(limit + 1))

	log.Println(query.Sql())

	err := query.Query(db, &models)
	if err != nil {
		log.Println(err)
		return PagedResult[model.ExitNodes]{}, err
	}

	return paginate(models, limit, func(m model.ExitNodes) string {
		return m.NodeAddress
	}), nil
}

func ListNodesInTimeRange(db *sql.DB, pagingToken string, limit int, startTime time.Time, endTime time.Time) (PagedResult[model.ExitNodes], error) {
	var models []model.ExitNodes

	query := SELECT(ExitNodes.NodeAddress, MAX(ExitNodes.FetchTime)).
		FROM(ExitNodes.LEFT_JOIN(AllowList, ExitNodes.NodeAddress.EQ(AllowList.NodeAddress))).
		WHERE(ExitNodes.NodeAddress.GT(String(pagingToken)).
			AND(ExitNodes.FetchTime.BETWEEN(TimestampT(startTime), TimestampT(endTime))).
			AND(AllowList.NodeAddress.IS_NULL())).
		GROUP_BY(ExitNodes.NodeAddress).
		ORDER_BY(ExitNodes.NodeAddress.ASC()).
		LIMIT(int64(limit + 1))

	log.Println(query.Sql())
	err := query.Query(db, &models)
	if err != nil {
		log.Println(err)
		return PagedResult[model.ExitNodes]{}, err
	}

	return paginate(models, limit, func(m model.ExitNodes) string {
		return m.NodeAddress
	}), nil
}

func ListAllowListNodes(db *sql.DB, pagingToken string, limit int) (PagedResult[string], error) {
	var models []model.AllowList

	query := SELECT(AllowList.NodeAddress).
		FROM(AllowList).
		WHERE(AllowList.NodeAddress.GT(String(pagingToken))).
		ORDER_BY(AllowList.NodeAddress).LIMIT(int64(limit + 1))

	err := query.Query(db, &models)

	if err != nil {
		log.Println(err)
		return PagedResult[string]{}, err
	}

	nodes := lo.Map(models, func(m model.AllowList, i int) string {
		return m.NodeAddress
	})

	return paginate(nodes, limit, func(s string) string {
		return s
	}), nil
}

func AddAllowListNode(db *sql.DB, nodeAddress string) error {
	query := AllowList.INSERT(AllowList.NodeAddress).
		VALUES(nodeAddress).
		ON_CONFLICT(AllowList.NodeAddress).
		DO_NOTHING()

	_, err := query.Exec(db)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func DeleteAllowListNode(db *sql.DB, nodeAddress string) error {
	query := AllowList.DELETE().WHERE(AllowList.NodeAddress.EQ(String(nodeAddress)))

	_, err := query.Exec(db)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
