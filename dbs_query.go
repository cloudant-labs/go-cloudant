package cloudant

import (
	"fmt"
	"net/url"
	"strconv"
)

// NOTE: These parameters are ignored on CouchDB 1.6.X!
//
// DBsQuery is a helper utility to build Cloudant request parameters for database list (_all_dbs)
//
// Example:
// 	q := NewDBsQuery().
//     StartKey("db1").
//     Limit(5)
//
//	dbList, err := db.AllDBs(q)

// DBsQuery object helps build Cloudant DBsQuery parameters
type DBsQuery struct {
	URLValues url.Values
}

// NewDBsQuery is a shortcut to create new Cloudant DBsQuery object with no parameters
func NewDBsQuery() *DBsQuery {
	return &DBsQuery{URLValues: url.Values{}}
}

// Descending applies descending=true parameter to Cloudant DBsQuery
func (q *DBsQuery) Descending() *DBsQuery {
	q.URLValues.Set("descending", "true")
	return q
}

// EndKey applies endkey=(key) parameter to Cloudant DBsQuery
func (q *DBsQuery) EndKey(endKey string) *DBsQuery {
	if endKey != "" {
		q.URLValues.Set("endkey", fmt.Sprintf("\"%s\"", endKey))
	}
	return q
}

// InclusiveEnd applies inclusive_end=true parameter to Cloudant DBsQuery
func (q *DBsQuery) InclusiveEnd() *DBsQuery {
	q.URLValues.Set("inclusive_end", "true")
	return q
}

// Limit applies limit parameter to Cloudant DBsQuery
func (q *DBsQuery) Limit(lim int) *DBsQuery {
	if lim > 0 {
		q.URLValues.Set("limit", strconv.Itoa(lim))
	}
	return q
}

// Skip applies skip=(number) parameter to Cloudant DBsQuery
func (q *DBsQuery) Skip(skip int) *DBsQuery {
	if skip > 0 {
		q.URLValues.Set("skip", strconv.Itoa(skip))
	}
	return q
}

// StartKey applies startkey=(key) parameter to Cloudant DBsQuery
func (q *DBsQuery) StartKey(startKey string) *DBsQuery {
	if startKey != "" {
		q.URLValues.Set("startkey", fmt.Sprintf("\"%s\"", startKey))
	}
	return q
}
