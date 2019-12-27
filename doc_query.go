package cloudant

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// DocQuery is a helper utility to build Cloudant request parameters for document
//
// Example:
//  var doc interface{}
// 	q := NewDocQuery().
//     RevsInfo().
//     Conflicts()
//
//	err := db.Get(docId, q, &doc)

// DocQuery object helps build Cloudant DocQuery parameters
type DocQuery struct {
	URLValues url.Values
}

// NewDocQuery is a shortcut to create new Cloudant DocQuery object with no parameters
func NewDocQuery() *DocQuery {
	return &DocQuery{URLValues: url.Values{}}
}

// Attachments applies attachments=true parameter to Cloudant DocQuery
func (q *DocQuery) Attachments() *DocQuery {
	q.URLValues.Set("attachments", "true")
	return q
}

// AttEncodingInfo applies att_encoding_info=true parameter to Cloudant DocQuery
func (q *DocQuery) AttEncodingInfo() *DocQuery {
	q.URLValues.Set("att_encoding_info", "true")
	return q
}

// AttsSince applies attsSince=(since) parameter to Cloudant ViewQuery
func (q *DocQuery) AttsSince(since []string) *DocQuery {
	if len(since) > 0 {
		data, err := json.Marshal(since)
		if err == nil {
			q.URLValues.Set("attsSince", string(data[:]))
		}
	}
	return q
}

// Conflicts applies conflicts=true parameter to Cloudant DocQuery
func (q *DocQuery) Conflicts() *DocQuery {
	q.URLValues.Set("conflicts", "true")
	return q
}

// DeletedConflicts applies deleted_conflicts=true parameter to Cloudant DocQuery
func (q *DocQuery) DeletedConflicts() *DocQuery {
	q.URLValues.Set("deleted_conflicts", "true")
	return q
}

// Latest applies latest=true parameter to Cloudant DocQuery
func (q *DocQuery) Latest() *DocQuery {
	q.URLValues.Set("latest", "true")
	return q
}

// LocalSeq applies local_seq=true parameter to Cloudant DocQuery
func (q *DocQuery) LocalSeq() *DocQuery {
	q.URLValues.Set("local_seq", "true")
	return q
}

// Meta applies meta=true parameter to Cloudant DocQuery
func (q *DocQuery) Meta() *DocQuery {
	q.URLValues.Set("meta", "true")
	return q
}

// Skip applies skip=(number) parameter to Cloudant DocQuery
func (q *DocQuery) Skip(skip int) *DocQuery {
	if skip > 0 {
		q.URLValues.Set("skip", strconv.Itoa(skip))
	}
	return q
}

// OpenRevs applies open_revs=(revs) parameter to Cloudant DocQuery
func (q *DocQuery) OpenRevs(revs []string) *DocQuery {
	if len(revs) > 0 {
		data, err := json.Marshal(revs)
		if err == nil {
			q.URLValues.Set("open_revs", string(data[:]))
		}
	}
	return q
}

// Rev applies rev=(rev) parameter to Cloudant DocQuery
func (q *DocQuery) Rev(rev string) *DocQuery {
	if rev != "" {
		q.URLValues.Set("rev", rev)
	}
	return q
}

// Revs applies revs=true parameter to Cloudant DocQuery
func (q *DocQuery) Revs() *DocQuery {
	q.URLValues.Set("revs", "true")
	return q
}

// RevsInfo applies revs_info=true parameter to Cloudant DocQuery
func (q *DocQuery) RevsInfo() *DocQuery {
	q.URLValues.Set("revs_info", "true")
	return q
}
