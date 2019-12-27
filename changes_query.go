package cloudant

// ChangesQuery is a helper utility to build Cloudant request parameters for changes feeds
//
// Example:
// 	q := cloudant.NewChangesQuery().IncludeDocs()
//
//	changes, err := db.Changes(q)

import (
	"net/url"
	"strconv"
)

// ChangesQuery object helps build Cloudant ChangesQuery parameters
type ChangesQuery struct {
	URLValues   url.Values
	DocIDValues []string
}

// NewChangesQuery is a shortcut to create new Cloudant ChangesQuery object with no parameters
func NewChangesQuery() *ChangesQuery {
	return &ChangesQuery{URLValues: url.Values{}}
}

// Conflicts applies conflicts=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Conflicts() *ChangesQuery {
	q.URLValues.Set("conflicts", "true")
	return q
}

// Descending applies descending=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Descending() *ChangesQuery {
	q.URLValues.Set("descending", "true")
	return q
}

// DocIDs applies doc_ids=(doc_ids) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) DocIDs(docIDs []string) *ChangesQuery {
	q.DocIDValues = docIDs
	return q
}

// Feed applies feed=(feed) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Feed(feed string) *ChangesQuery {
	if feed != "" {
		q.URLValues.Set("feed", feed)
	}
	return q
}

// Filter applies filter=(filter) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Filter(filter string) *ChangesQuery {
	if filter != "" {
		q.URLValues.Set("filter", filter)
	}
	return q
}

// Heartbeat applies heartbeat parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Heartbeat(heartbeat int) *ChangesQuery {
	if heartbeat > 0 {
		q.URLValues.Set("heartbeat", strconv.Itoa(heartbeat))
	}
	return q
}

// IncludeDocs applies include_docs=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) IncludeDocs() *ChangesQuery {
	q.URLValues.Set("include_docs", "true")
	return q
}

// Limit applies limit parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Limit(lim int) *ChangesQuery {
	if lim > 0 {
		q.URLValues.Set("limit", strconv.Itoa(lim))
	}
	return q
}

// SeqInterval applies seq_interval parameter to Cloudant ChangesQuery
func (q *ChangesQuery) SeqInterval(interval int) *ChangesQuery {
	if interval > 0 {
		q.URLValues.Set("seq_interval", strconv.Itoa(interval))
	}
	return q
}

// Since applies since=(since) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Since(since string) *ChangesQuery {
	if since != "" {
		q.URLValues.Set("since", since)
	}
	return q
}

// Style applies style=(style) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Style(style string) *ChangesQuery {
	if style != "" {
		q.URLValues.Set("style", style)
	}
	return q
}

// Timeout applies seq_interval parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Timeout(timeout int) *ChangesQuery {
	if timeout > 0 {
		q.URLValues.Set("timeout", strconv.Itoa(timeout))
	}
	return q
}
