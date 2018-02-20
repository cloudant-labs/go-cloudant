package cloudant

// NOTE: These parameters are ignored on CouchDB 1.6.X!
//
// QueryBuilder implementation for the AllDBs() API call.
//
// Example:
// 	query := NewAllDBsQuery().
//     StartKey("db1").
//     Limit(5).
//     Build()
//
//	dbList, err := db.AllDBs(query)

import (
	"fmt"
	"net/url"
	"strconv"
)

// AllDBsQueryBuilder defines the available parameter-setting functions.
type AllDBsQueryBuilder interface {
	EndKey(string) AllDBsQueryBuilder
	InclusiveEnd() AllDBsQueryBuilder
	Limit(int) AllDBsQueryBuilder
	Skip(int) AllDBsQueryBuilder
	StartKey(string) AllDBsQueryBuilder
	Build() *allDBsQuery
}

type allDBsQueryBuilder struct {
	endKey       string
	inclusiveEnd bool
	limit        int
	skip         int
	startKey     string
}

// allDBsQuery holds the implemented API call parameters.
type allDBsQuery struct {
	EndKey       string
	InclusiveEnd bool
	Limit        int
	Skip         int
	StartKey     string
}

// NewAllDBsQuery is the entry point.
func NewAllDBsQuery() AllDBsQueryBuilder {
	return &allDBsQueryBuilder{}
}

func (a *allDBsQueryBuilder) EndKey(endKey string) AllDBsQueryBuilder {
	a.endKey = fmt.Sprintf("\"%s\"", endKey)
	return a
}

func (a *allDBsQueryBuilder) InclusiveEnd() AllDBsQueryBuilder {
	a.inclusiveEnd = true
	return a
}

func (a *allDBsQueryBuilder) Limit(lim int) AllDBsQueryBuilder {
	a.limit = lim
	return a
}

func (a *allDBsQueryBuilder) Skip(skip int) AllDBsQueryBuilder {
	a.skip = skip
	return a
}

func (a *allDBsQueryBuilder) StartKey(startKey string) AllDBsQueryBuilder {
	a.startKey = fmt.Sprintf("\"%s\"", startKey)
	return a
}

// GetQuery implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (aq *allDBsQuery) GetQuery() (url.Values, error) {
	vals := url.Values{}

	if aq.InclusiveEnd {
		vals.Set("inclusive_end", "true")
	}
	if aq.EndKey != "" {
		vals.Set("endkey", aq.EndKey)
	}
	if aq.StartKey != "" {
		vals.Set("startkey", aq.StartKey)
	}
	if aq.Limit > 0 {
		vals.Set("limit", strconv.Itoa(aq.Limit))
	}
	if aq.Skip > 0 {
		vals.Set("skip", strconv.Itoa(aq.Skip))
	}
	return vals, nil
}

func (a *allDBsQueryBuilder) Build() *allDBsQuery {
	return &allDBsQuery{
		EndKey:       a.endKey,
		InclusiveEnd: a.inclusiveEnd,
		Limit:        a.limit,
		Skip:         a.skip,
		StartKey:     a.startKey,
	}
}
