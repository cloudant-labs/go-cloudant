package cloudant

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// DocQuery is a helper utility to build Cloudant request parameters for document
// Use .Values property (url.Values map) as params input to document functions
//
// Example:
//  var doc interface{}
// 	params := NewDocQuery().
//     RevsInfo().
//     Conflicts().
//     Values
//
//	err := db.Get(docId, params, &doc)

// DocQuery object helps build Cloudant DocQuery parameters
type DocQuery struct {
	Values url.Values
}

// NewDocQuery is a shortcut to create new Cloudant DocQuery object with no parameters
func NewDocQuery() *DocQuery {
	return &DocQuery{Values: url.Values{}}
}

// Attachments applies attachments=true parameter to Cloudant DocQuery
func (q *DocQuery) Attachments() *DocQuery {
	q.Values.Set("attachments", "true")
	return q
}

// AttEncodingInfo applies att_encoding_info=true parameter to Cloudant DocQuery
func (q *DocQuery) AttEncodingInfo() *DocQuery {
	q.Values.Set("att_encoding_info", "true")
	return q
}

// AttsSince applies attsSince=(since) parameter to Cloudant ViewQuery
func (q *ViewQuery) AttsSince(since []string) *ViewQuery {
	if len(since) > 0 {
		data, err := json.Marshal(since)
		if err == nil {
			q.Values.Set("attsSince", string(data[:]))
		}
	}
	return q
}

// Conflicts applies conflicts=true parameter to Cloudant DocQuery
func (q *DocQuery) Conflicts() *DocQuery {
	q.Values.Set("conflicts", "true")
	return q
}

// DeletedConflicts applies deleted_conflicts=true parameter to Cloudant DocQuery
func (q *DocQuery) DeletedConflicts() *DocQuery {
	q.Values.Set("deleted_conflicts", "true")
	return q
}

// Latest applies latest=true parameter to Cloudant DocQuery
func (q *DocQuery) Latest() *DocQuery {
	q.Values.Set("latest", "true")
	return q
}

// LocalSeq applies local_seq=true parameter to Cloudant DocQuery
func (q *DocQuery) LocalSeq() *DocQuery {
	q.Values.Set("local_seq", "true")
	return q
}

// Meta applies meta=true parameter to Cloudant DocQuery
func (q *DocQuery) Meta() *DocQuery {
	q.Values.Set("meta", "true")
	return q
}

// Skip applies skip=(number) parameter to Cloudant DocQuery
func (q *DocQuery) Skip(skip int) *DocQuery {
	if skip > 0 {
		q.Values.Set("skip", strconv.Itoa(skip))
	}
	return q
}

// OpenRevs applies open_revs=(revs) parameter to Cloudant ViewQuery
func (q *ViewQuery) OpenRevs(revs []string) *ViewQuery {
	if len(revs) > 0 {
		data, err := json.Marshal(revs)
		if err == nil {
			q.Values.Set("open_revs", string(data[:]))
		}
	}
	return q
}

// Rev applies rev=(rev) parameter to Cloudant DocQuery
func (q *DocQuery) Rev(rev string) *DocQuery {
	if rev != "" {
		q.Values.Set("rev", rev)
	}
	return q
}

// Revs applies revs=true parameter to Cloudant DocQuery
func (q *DocQuery) Revs() *DocQuery {
	q.Values.Set("revs", "true")
	return q
}

// RevsInfo applies revs_info=true parameter to Cloudant DocQuery
func (q *DocQuery) RevsInfo() *DocQuery {
	q.Values.Set("revs_info", "true")
	return q
}
