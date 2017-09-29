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
	Limit(int) AllDocsQueryBuilder
	Meta() AllDocsQueryBuilder
	R(int) AllDocsQueryBuilder
	RevsInfo() AllDocsQueryBuilder
	Skip(int) AllDocsQueryBuilder
	StartKey(string) AllDocsQueryBuilder
	Build() QueryBuilder
}

type allDocsQueryBuilder struct {
	conflicts        bool
	deletedConflicts bool
	descending       bool
	endKey           string
	includeDocs      bool
	inclusiveEnd     bool
	key              string
	limit            int
	meta             bool
	r                int
	revsInfo         bool
	skip             int
	startKey         string
}

// allDocsQuery holds the implemented API call parameters. The doc_ids parameter
// is not yet implemented.
type allDocsQuery struct {
	conflicts        bool
	deletedConflicts bool
	descending       bool
	endKey           string
	includeDocs      bool
	inclusiveEnd     bool
	key              string
	limit            int
	meta             bool
	r                int
	revsInfo         bool
	skip             int
	startKey         string
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
	a.endKey = endKey
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
	a.key = key
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
	a.startKey = startKey
	return a
}

// QueryString implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (aq *allDocsQuery) QueryString() (url.Values, error) {
	vals := url.Values{}

	if aq.conflicts {
		vals.Set("conflicts", "true")
	}
	if aq.descending {
		vals.Set("descending", "true")
	}
	if aq.includeDocs {
		vals.Set("include_docs", "true")
	}
	if aq.deletedConflicts {
		vals.Set("deleted_conflicts", "true")
	}
	if aq.inclusiveEnd {
		vals.Set("inclusive_end", "true")
	}
	if aq.revsInfo {
		vals.Set("revs_info", "true")
	}
	if aq.meta {
		vals.Set("meta", "true")
	}
	if aq.endKey != "" {
		vals.Set("endkey", aq.endKey)
	}
	if aq.key != "" {
		vals.Set("key", aq.key)
	}
	if aq.startKey != "" {
		vals.Set("startkey", aq.startKey)
	}
	if aq.limit > 0 {
		vals.Set("limit", strconv.Itoa(aq.limit))
	}
	if aq.skip > 0 {
		vals.Set("skip", strconv.Itoa(aq.skip))
	}
	if aq.r > 0 {
		vals.Set("r", strconv.Itoa(aq.r))
	}

	return vals, nil
}

func (a *allDocsQueryBuilder) Build() QueryBuilder {
	return &allDocsQuery{
		conflicts:        a.conflicts,
		deletedConflicts: a.deletedConflicts,
		descending:       a.descending,
		endKey:           a.endKey,
		includeDocs:      a.includeDocs,
		inclusiveEnd:     a.inclusiveEnd,
		key:              a.key,
		limit:            a.limit,
		meta:             a.meta,
		r:                a.r,
		revsInfo:         a.revsInfo,
		skip:             a.skip,
		startKey:         a.startKey,
	}
}
