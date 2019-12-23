package cloudant

// ChangesQuery is a helper utility to build Cloudant request parameters for changes feeds
// Use .Values property (url.Values map) as params input to changes functions
//
// Example:
// 	params := cloudant.NewChangesQuery().IncludeDocs().Values
//
//	changes, err := db.Changes(params)

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// ChangesQuery object helps build Cloudant ChangesQuery parameters
type ChangesQuery struct {
	Values url.Values
}

// NewChangesQuery is a shortcut to create new Cloudant ChangesQuery object with no parameters
func NewChangesQuery() *ChangesQuery {
	return &ChangesQuery{Values: url.Values{}}
}

// Conflicts applies conflicts=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Conflicts() *ChangesQuery {
	q.Values.Set("conflicts", "true")
	return q
}

// Descending applies descending=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Descending() *ChangesQuery {
	q.Values.Set("descending", "true")
	return q
}

// DocIDs applies doc_ids=(doc_ids) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) DocIDs(docIDs []string) *ChangesQuery {
	if len(docIDs) > 0 {
		data, err := json.Marshal(docIDs)
		if err == nil {
			q.Values.Set("doc_ids", string(data[:]))
		}
	}
	return q
}

// Feed applies feed=(feed) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Feed(feed string) *ChangesQuery {
	if feed != "" {
		q.Values.Set("feed", feed)
	}
	return q
}

// Filter applies filter=(filter) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Filter(filter string) *ChangesQuery {
	if filter != "" {
		q.Values.Set("filter", filter)
	}
	return q
}

// Heartbeat applies heartbeat parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Heartbeat(heartbeat int) *ChangesQuery {
	if heartbeat > 0 {
		q.Values.Set("heartbeat", strconv.Itoa(heartbeat))
	}
	return q
}

// IncludeDocs applies include_docs=true parameter to Cloudant ChangesQuery
func (q *ChangesQuery) IncludeDocs() *ChangesQuery {
	q.Values.Set("include_docs", "true")
	return q
}

// Limit applies limit parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Limit(lim int) *ChangesQuery {
	if lim > 0 {
		q.Values.Set("limit", strconv.Itoa(lim))
	}
	return q
}

// SeqInterval applies seq_interval parameter to Cloudant ChangesQuery
func (q *ChangesQuery) SeqInterval(interval int) *ChangesQuery {
	if interval > 0 {
		q.Values.Set("seq_interval", strconv.Itoa(interval))
	}
	return q
}

// Since applies since=(since) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Since(since string) *ChangesQuery {
	if since != "" {
		q.Values.Set("since", since)
	}
	return q
}

// Style applies style=(style) parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Style(style string) *ChangesQuery {
	if style != "" {
		q.Values.Set("style", style)
	}
	return q
}

// Timeout applies seq_interval parameter to Cloudant ChangesQuery
func (q *ChangesQuery) Timeout(timeout int) *ChangesQuery {
	if timeout > 0 {
		q.Values.Set("timeout", strconv.Itoa(timeout))
	}
	return q
}
