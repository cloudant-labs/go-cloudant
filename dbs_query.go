package cloudant

import (
	"fmt"
	"net/url"
	"strconv"
)

// NOTE: These parameters are ignored on CouchDB 1.6.X!
//
// DBsQuery is a helper utility to build Cloudant request parameters for database list (_all_dbs)
// Use .Values property (url.Values map) as params input to view functions
//
// Example:
// 	params := NewDBsQuery().
//     StartKey("db1").
//     Limit(5).
//     Values
//
//	dbList, err := db.AllDBs(params)

// DBsQuery object helps build Cloudant DBsQuery parameters
type DBsQuery struct {
	Values url.Values
}

// NewDBsQuery is a shortcut to create new Cloudant DBsQuery object with no parameters
func NewDBsQuery() *DBsQuery {
	return &DBsQuery{Values: url.Values{}}
}

// Descending applies descending=true parameter to Cloudant DBsQuery
func (q *DBsQuery) Descending() *DBsQuery {
	q.Values.Set("descending", "true")
	return q
}

// EndKey applies endkey=(key) parameter to Cloudant DBsQuery
func (q *DBsQuery) EndKey(endKey string) *DBsQuery {
	if endKey != "" {
		q.Values.Set("endkey", fmt.Sprintf("\"%s\"", endKey))
	}
	return q
}

// InclusiveEnd applies inclusive_end=true parameter to Cloudant DBsQuery
func (q *DBsQuery) InclusiveEnd() *DBsQuery {
	q.Values.Set("inclusive_end", "true")
	return q
}

// Limit applies limit parameter to Cloudant DBsQuery
func (q *DBsQuery) Limit(lim int) *DBsQuery {
	if lim > 0 {
		q.Values.Set("limit", strconv.Itoa(lim))
	}
	return q
}

// Skip applies skip=(number) parameter to Cloudant DBsQuery
func (q *DBsQuery) Skip(skip int) *DBsQuery {
	if skip > 0 {
		q.Values.Set("skip", strconv.Itoa(skip))
	}
	return q
}

// StartKey applies startkey=(key) parameter to Cloudant DBsQuery
func (q *DBsQuery) StartKey(startKey string) *DBsQuery {
	if startKey != "" {
		q.Values.Set("startkey", fmt.Sprintf("\"%s\"", startKey))
	}
	return q
}
