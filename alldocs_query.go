package cloudant

// QueryBuilder implementation for the AllDocs() API call.
//
// Example:
// 	query := NewAllDocsQuery().
//     IncludeDocs().
//     Build()
//
//	changes, err := db.All(query)

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// AllDocsQueryBuilder defines the available parameter-setting functions.
type AllDocsQueryBuilder interface {
	Conflicts() AllDocsQueryBuilder
	DeletedConflicts() AllDocsQueryBuilder
	Descending() AllDocsQueryBuilder
	EndKey(string) AllDocsQueryBuilder
	IncludeDocs() AllDocsQueryBuilder
	InclusiveEnd() AllDocsQueryBuilder
	Key(string) AllDocsQueryBuilder
	Keys([]string) AllDocsQueryBuilder
	Limit(int) AllDocsQueryBuilder
	Meta() AllDocsQueryBuilder
	R(int) AllDocsQueryBuilder
	RevsInfo() AllDocsQueryBuilder
	Skip(int) AllDocsQueryBuilder
	StartKey(string) AllDocsQueryBuilder
	Build() *allDocsQuery
}

type allDocsQueryBuilder struct {
	conflicts        bool
	deletedConflicts bool
	descending       bool
	endKey           string
	includeDocs      bool
	inclusiveEnd     bool
	key              string
	keys             []string
	limit            int
	meta             bool
	r                int
	revsInfo         bool
	skip             int
	startKey         string
}

// allDocsQuery holds the implemented API call parameters.
type allDocsQuery struct {
	Conflicts        bool
	DeletedConflicts bool
	Descending       bool
	EndKey           string
	IncludeDocs      bool
	InclusiveEnd     bool
	Key              string
	Keys             []string
	Limit            int
	Meta             bool
	R                int
	RevsInfo         bool
	Skip             int
	StartKey         string
}

// NewAllDocsQuery is the entry point.
func NewAllDocsQuery() AllDocsQueryBuilder {
	return &allDocsQueryBuilder{}
}

func (a *allDocsQueryBuilder) Conflicts() AllDocsQueryBuilder {
	a.conflicts = true
	return a
}

func (a *allDocsQueryBuilder) DeletedConflicts() AllDocsQueryBuilder {
	a.deletedConflicts = true
	return a
}

func (a *allDocsQueryBuilder) Descending() AllDocsQueryBuilder {
	a.descending = true
	return a
}

func (a *allDocsQueryBuilder) EndKey(endKey string) AllDocsQueryBuilder {
	a.endKey = fmt.Sprintf("\"%s\"", endKey)
	return a
}

func (a *allDocsQueryBuilder) IncludeDocs() AllDocsQueryBuilder {
	a.includeDocs = true
	return a
}

func (a *allDocsQueryBuilder) InclusiveEnd() AllDocsQueryBuilder {
	a.inclusiveEnd = true
	return a
}

func (a *allDocsQueryBuilder) Key(key string) AllDocsQueryBuilder {
	a.key = fmt.Sprintf("\"%s\"", key)
	return a
}

func (a *allDocsQueryBuilder) Keys(keys []string) AllDocsQueryBuilder {
	a.keys = keys
	return a
}

func (a *allDocsQueryBuilder) Limit(lim int) AllDocsQueryBuilder {
	a.limit = lim
	return a
}

func (a *allDocsQueryBuilder) Meta() AllDocsQueryBuilder {
	a.meta = true
	return a
}

func (a *allDocsQueryBuilder) R(r int) AllDocsQueryBuilder {
	a.r = r
	return a
}

func (a *allDocsQueryBuilder) RevsInfo() AllDocsQueryBuilder {
	a.revsInfo = true
	return a
}

func (a *allDocsQueryBuilder) Skip(skip int) AllDocsQueryBuilder {
	a.skip = skip
	return a
}

func (a *allDocsQueryBuilder) StartKey(startKey string) AllDocsQueryBuilder {
	a.startKey = fmt.Sprintf("\"%s\"", startKey)
	return a
}

// GetQuery implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (aq *allDocsQuery) GetQuery() (url.Values, error) {
	vals := url.Values{}

	if aq.Conflicts {
		vals.Set("conflicts", "true")
	}
	if aq.Descending {
		vals.Set("descending", "true")
	}
	if aq.IncludeDocs {
		vals.Set("include_docs", "true")
	}
	if aq.DeletedConflicts {
		vals.Set("deleted_conflicts", "true")
	}
	if aq.InclusiveEnd {
		vals.Set("inclusive_end", "true")
	}
	if aq.RevsInfo {
		vals.Set("revs_info", "true")
	}
	if aq.Meta {
		vals.Set("meta", "true")
	}
	if aq.EndKey != "" {
		vals.Set("endkey", aq.EndKey)
	}
	if aq.Key != "" {
		vals.Set("key", aq.Key)
	}
	if len(aq.Keys) > 0 {
		data, err := json.Marshal(aq.Keys)
		if err != nil {
			return nil, err
		}
		vals.Set("keys", string(data[:]))
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
	if aq.R > 0 {
		vals.Set("r", strconv.Itoa(aq.R))
	}

	return vals, nil
}

func (a *allDocsQueryBuilder) Build() *allDocsQuery {
	return &allDocsQuery{
		Conflicts:        a.conflicts,
		DeletedConflicts: a.deletedConflicts,
		Descending:       a.descending,
		EndKey:           a.endKey,
		IncludeDocs:      a.includeDocs,
		InclusiveEnd:     a.inclusiveEnd,
		Key:              a.key,
		Keys:             a.keys,
		Limit:            a.limit,
		Meta:             a.meta,
		R:                a.r,
		RevsInfo:         a.revsInfo,
		Skip:             a.skip,
		StartKey:         a.startKey,
	}
}
