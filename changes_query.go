package cloudant

// QueryBuilder implementation for the Changes() API call.
//
// Example:
// 	query := cloudant.NewChangesQuery().IncludeDocs().Build()
//
//	changes, err := db.Changes(query)

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// ChangesQueryBuilder defines the available parameter-setting functions.
type ChangesQueryBuilder interface {
	Conflicts() ChangesQueryBuilder
	Descending() ChangesQueryBuilder
	DocIDs([]string) ChangesQueryBuilder
	Feed(string) ChangesQueryBuilder
	Filter(string) ChangesQueryBuilder
	Heartbeat(int) ChangesQueryBuilder
	IncludeDocs() ChangesQueryBuilder
	Limit(int) ChangesQueryBuilder
	SeqInterval(int) ChangesQueryBuilder
	Since(string) ChangesQueryBuilder
	Style(string) ChangesQueryBuilder
	Timeout(int) ChangesQueryBuilder
	Build() *changesQuery
}

type changesQueryBuilder struct {
	conflicts   bool
	descending  bool
	docIDs      []string
	feed        string
	filter      string
	heartbeat   int
	includeDocs bool
	limit       int
	seqInterval int
	since       string
	style       string
	timeout     int
}

// changesQuery holds the implemented API call parameters. The doc_ids parameter
// is not yet implemented.
type changesQuery struct {
	Conflicts   bool
	Descending  bool
	DocIDs      []string
	Feed        string
	Filter      string
	Heartbeat   int
	IncludeDocs bool
	Limit       int
	SeqInterval int
	Since       string
	Style       string
	Timeout     int
}

// NewChangesQuery is the entry point.
func NewChangesQuery() ChangesQueryBuilder {
	return &changesQueryBuilder{}
}

func (c *changesQueryBuilder) Conflicts() ChangesQueryBuilder {
	c.conflicts = true
	return c
}

func (c *changesQueryBuilder) Descending() ChangesQueryBuilder {
	c.descending = true
	return c
}

func (c *changesQueryBuilder) DocIDs(docIDs []string) ChangesQueryBuilder {
	c.docIDs = docIDs
	return c
}

func (c *changesQueryBuilder) Feed(feed string) ChangesQueryBuilder {
	c.feed = feed
	return c
}

func (c *changesQueryBuilder) Filter(filter string) ChangesQueryBuilder {
	c.filter = filter
	return c
}

func (c *changesQueryBuilder) Heartbeat(hb int) ChangesQueryBuilder {
	c.heartbeat = hb
	return c
}

func (c *changesQueryBuilder) IncludeDocs() ChangesQueryBuilder {
	c.includeDocs = true
	return c
}

func (c *changesQueryBuilder) Limit(lim int) ChangesQueryBuilder {
	c.limit = lim
	return c
}

func (c *changesQueryBuilder) SeqInterval(interval int) ChangesQueryBuilder {
	c.seqInterval = interval
	return c
}

func (c *changesQueryBuilder) Since(seq string) ChangesQueryBuilder {
	if seq != "" {
		c.since = seq
	}
	return c
}

func (c *changesQueryBuilder) Style(style string) ChangesQueryBuilder {
	c.style = style
	return c
}

func (c *changesQueryBuilder) Timeout(secs int) ChangesQueryBuilder {
	c.timeout = secs
	return c
}

// GetQuery implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (cq *changesQuery) GetQuery() (url.Values, error) {
	vals := url.Values{}
	if cq.Conflicts {
		vals.Set("conflicts", "true")
	}
	if cq.Descending {
		vals.Set("descending", "true")
	}
	if len(cq.DocIDs) > 0 {
		data, err := json.Marshal(cq.DocIDs)
		if err != nil {
			return nil, err
		}
		vals.Set("doc_ids", string(data[:]))
	}
	if cq.IncludeDocs {
		vals.Set("include_docs", "true")
	}
	if cq.Feed != "" {
		vals.Set("feed", cq.Feed)
	}
	if cq.Filter != "" {
		vals.Set("filter", cq.Filter)
	}
	if cq.Heartbeat > 0 {
		vals.Set("heartbeat", strconv.Itoa(cq.Heartbeat))
	}
	if cq.Limit > 0 {
		vals.Set("limit", strconv.Itoa(cq.Limit))
	}
	if cq.SeqInterval > 0 {
		vals.Set("seq_interval", strconv.Itoa(cq.SeqInterval))
	}
	if cq.Style != "" {
		vals.Set("style", cq.Style)
	}
	if cq.Since != "" {
		vals.Set("since", cq.Since)
	}
	if cq.Timeout > 0 {
		vals.Set("timeout", strconv.Itoa(cq.Timeout))
	}
	return vals, nil
}

func (c *changesQueryBuilder) Build() *changesQuery {
	return &changesQuery{
		Conflicts:   c.conflicts,
		Descending:  c.descending,
		Feed:        c.feed,
		Filter:      c.filter,
		Heartbeat:   c.heartbeat,
		IncludeDocs: c.includeDocs,
		Limit:       c.limit,
		SeqInterval: c.seqInterval,
		Since:       c.since,
		Style:       c.style,
		Timeout:     c.timeout,
	}
}
