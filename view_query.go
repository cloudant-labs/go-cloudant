package cloudant

import (
	"fmt"
	"net/url"
	"strconv"
)

// ViewQuery is a helper utility to build Cloudant request parameters for views (including _all_docs)
//
// Example:
// 	q := NewViewQuery().
//     Conflicts()
//
//	changes, err := db.All(q)

// ViewQuery object helps build Cloudant ViewQuery parameters
type ViewQuery struct {
	URLValues url.Values
	KeyValues []string
}

// NewViewQuery is a shortcut to create new Cloudant ViewQuery object with no parameters
func NewViewQuery() *ViewQuery {
	return &ViewQuery{URLValues: url.Values{}}
}

// Conflicts applies conflicts=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Conflicts() *ViewQuery {
	q.URLValues.Set("conflicts", "true")
	return q
}

// DeletedConflicts applies deleted_conflicts=true parameter to Cloudant ViewQuery
func (q *ViewQuery) DeletedConflicts() *ViewQuery {
	q.URLValues.Set("deleted_conflicts", "true")
	return q
}

// Descending applies descending=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Descending() *ViewQuery {
	q.URLValues.Set("descending", "true")
	return q
}

// EndKey applies endkey=(key) parameter to Cloudant ViewQuery
func (q *ViewQuery) EndKey(endKey string) *ViewQuery {
	if endKey != "" {
		q.URLValues.Set("endkey", fmt.Sprintf("\"%s\"", endKey))
	}
	return q
}

// EndKeyDocID applies endkey_docid=(key) parameter to Cloudant ViewQuery
func (q *ViewQuery) EndKeyDocID(endKeyDocID string) *ViewQuery {
	if endKeyDocID != "" {
		q.URLValues.Set("endkey_docid", fmt.Sprintf("\"%s\"", endKeyDocID))
	}
	return q
}

// Group applies group=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Group() *ViewQuery {
	q.URLValues.Set("group", "true")
	return q
}

// GroupLevel applies group_level=(number) parameter to Cloudant ViewQuery
func (q *ViewQuery) GroupLevel(groupLevel int) *ViewQuery {
	if groupLevel > 0 {
		q.URLValues.Set("group_level", strconv.Itoa(groupLevel))
	}
	return q
}

// IncludeDocs applies include_docs=true parameter to Cloudant ViewQuery
func (q *ViewQuery) IncludeDocs() *ViewQuery {
	q.URLValues.Set("include_docs", "true")
	return q
}

// InclusiveEnd applies inclusive_end=true parameter to Cloudant ViewQuery
func (q *ViewQuery) InclusiveEnd() *ViewQuery {
	q.URLValues.Set("inclusive_end", "true")
	return q
}

// Key applies key=(key) parameter to Cloudant ViewQuery
func (q *ViewQuery) Key(key string) *ViewQuery {
	if key != "" {
		q.URLValues.Set("key", fmt.Sprintf("\"%s\"", key))
	}
	return q
}

// Keys applies keys=(keys) parameter to Cloudant ViewQuery
func (q *ViewQuery) Keys(keys []string) *ViewQuery {
	q.KeyValues = keys
	return q
}

// Limit applies limit parameter to Cloudant ViewQuery
func (q *ViewQuery) Limit(lim int) *ViewQuery {
	if lim > 0 {
		q.URLValues.Set("limit", strconv.Itoa(lim))
	}
	return q
}

// Meta applies meta=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Meta() *ViewQuery {
	q.URLValues.Set("meta", "true")
	return q
}

// R applies r=(number) parameter to Cloudant ViewQuery
func (q *ViewQuery) R(r int) *ViewQuery {
	if r > 0 {
		q.URLValues.Set("r", strconv.Itoa(r))
	}
	return q
}

// Reduce applies reduce=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Reduce() *ViewQuery {
	q.URLValues.Set("reduce", "true")
	return q
}

// RevsInfo applies revs_info=true parameter to Cloudant ViewQuery
func (q *ViewQuery) RevsInfo() *ViewQuery {
	q.URLValues.Set("revs_info", "true")
	return q
}

// Skip applies skip=(number) parameter to Cloudant ViewQuery
func (q *ViewQuery) Skip(skip int) *ViewQuery {
	if skip > 0 {
		q.URLValues.Set("skip", strconv.Itoa(skip))
	}
	return q
}

// Stable applies stable=true parameter to Cloudant ViewQuery
func (q *ViewQuery) Stable() *ViewQuery {
	q.URLValues.Set("stable", "true")
	return q
}

// Stale applies stale=ok parameter to Cloudant ViewQuery
func (q *ViewQuery) Stale() *ViewQuery {
	q.URLValues.Set("stale", "ok")
	return q
}

// StaleUpdateAfter applies stale=update_after parameter to Cloudant ViewQuery
func (q *ViewQuery) StaleUpdateAfter() *ViewQuery {
	q.URLValues.Set("stale", "update_after")
	return q
}

// StartKey applies startkey=(key) parameter to Cloudant ViewQuery
func (q *ViewQuery) StartKey(startKey string) *ViewQuery {
	if startKey != "" {
		q.URLValues.Set("startkey", fmt.Sprintf("\"%s\"", startKey))
	}
	return q
}

// StartKeyDocID applies startkey_docid=(key) parameter to Cloudant ViewQuery
func (q *ViewQuery) StartKeyDocID(startKeyDocID string) *ViewQuery {
	if startKeyDocID != "" {
		q.URLValues.Set("startkey_docid", fmt.Sprintf("\"%s\"", startKeyDocID))
	}
	return q
}

// DoNotUpdate applies update=false parameter to Cloudant ViewQuery - return results without updating the view
func (q *ViewQuery) DoNotUpdate() *ViewQuery {
	q.URLValues.Set("update", "false")
	return q
}

// UpdateLazy applies update=lazy parameter to Cloudant ViewQuery - return the view results without waiting for an update, but update them immediately after the request
func (q *ViewQuery) UpdateLazy() *ViewQuery {
	q.URLValues.Set("update", "lazy")
	return q
}
