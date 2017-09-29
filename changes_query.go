package cloudant

// QueryBuilder implementation for the Changes() API call.
//
// Example:
// 	query := cloudant.NewChangesQuery().IncludeDocs().Build()
//
//	changes, err := db.Changes(query)

import (
	"net/url"
	"strconv"
)

// ChangesQueryBuilder defines the available parameter-setting functions.
type ChangesQueryBuilder interface {
	Conflicts() ChangesQueryBuilder
	Descending() ChangesQueryBuilder
	Feed(string) ChangesQueryBuilder
	Filter(string) ChangesQueryBuilder
	Heartbeat(int) ChangesQueryBuilder
	IncludeDocs() ChangesQueryBuilder
	Limit(int) ChangesQueryBuilder
	Since(string) ChangesQueryBuilder
	Style(string) ChangesQueryBuilder
	Timeout(int) ChangesQueryBuilder
	Build() QueryBuilder
}

type changesQueryBuilder struct {
	conflicts   bool
	descending  bool
	feed        string
	filter      string
	heartbeat   int
	includeDocs bool
	limit       int
	since       string
	style       string
	timeout     int
}

// changesQuery holds the implemented API call parameters. The doc_ids parameter
// is not yet implemented.
type changesQuery struct {
	conflicts   bool
	descending  bool
	feed        string
	filter      string
	heartbeat   int
	includeDocs bool
	limit       int
	since       string
	style       string
	timeout     int
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

func (c *changesQueryBuilder) Since(seq string) ChangesQueryBuilder {
	c.since = seq
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

// QueryString implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (cq *changesQuery) QueryString() (url.Values, error) {
	vals := url.Values{}
	if cq.conflicts {
		vals.Set("conflicts", "true")
	}
	if cq.descending {
		vals.Set("descending", "true")
	}
	if cq.includeDocs {
		vals.Set("include_docs", "true")
	}
	if cq.feed != "" {
		vals.Set("feed", cq.feed)
	}
	if cq.filter != "" {
		vals.Set("filter", cq.filter)
	}
	if cq.heartbeat > 0 {
		vals.Set("heartbeat", strconv.Itoa(cq.heartbeat))
	}
	if cq.style != "" {
		vals.Set("style", cq.style)
	}
	if cq.since != "" {
		vals.Set("since", cq.since)
	}
	if cq.timeout > 0 {
		vals.Set("timeout", strconv.Itoa(cq.timeout))
	}
	return vals, nil
}

func (c *changesQueryBuilder) Build() QueryBuilder {
	return &changesQuery{
		conflicts:   c.conflicts,
		descending:  c.descending,
		feed:        c.feed,
		filter:      c.filter,
		heartbeat:   c.heartbeat,
		includeDocs: c.includeDocs,
		limit:       c.limit,
		since:       c.since,
		style:       c.style,
		timeout:     c.timeout,
	}
}
