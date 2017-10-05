package cloudant

// QueryBuilder implementation for the Get() API call.
//
// Example:
//  var doc interface{}
// 	query := NewGetQuery().
//     RevsInfo().
//     Conflicts().
//     Build()
//
//	err := db.Get(docId, query, &doc)

import (
	"encoding/json"
	"net/url"
)

// GetQueryBuilder defines the available parameter-setting functions.
type GetQueryBuilder interface {
	Attachments() GetQueryBuilder
	AttEncodingInfo() GetQueryBuilder
	AttsSince([]string) GetQueryBuilder
	Conflicts() GetQueryBuilder
	DeletedConflicts() GetQueryBuilder
	Latest() GetQueryBuilder
	LocalSeq() GetQueryBuilder
	Meta() GetQueryBuilder
	OpenRevs([]string) GetQueryBuilder
	Rev(string) GetQueryBuilder
	Revs() GetQueryBuilder
	RevsInfo() GetQueryBuilder
	Build() *getQuery
}

type getQueryBuilder struct {
	attachments      bool
	attEncodingInfo  bool
	attsSince        []string
	conflicts        bool
	deletedConflicts bool
	latest           bool
	localSeq         bool
	meta             bool
	openRevs         []string
	rev              string
	revs             bool
	revsInfo         bool
}

// getQuery holds the implemented API call parameters.
type getQuery struct {
	Attachments      bool
	AttEncodingInfo  bool
	AttsSince        []string
	Conflicts        bool
	DeletedConflicts bool
	Latest           bool
	LocalSeq         bool
	Meta             bool
	OpenRevs         []string
	Rev              string
	Revs             bool
	RevsInfo         bool
}

// NewGetQuery is the entry point.
func NewGetQuery() GetQueryBuilder {
	return &getQueryBuilder{}
}

func (g *getQueryBuilder) Attachments() GetQueryBuilder {
	g.attachments = true
	return g
}

func (g *getQueryBuilder) AttEncodingInfo() GetQueryBuilder {
	g.attEncodingInfo = true
	return g
}

func (g *getQueryBuilder) AttsSince(since []string) GetQueryBuilder {
	g.attsSince = since
	return g
}

func (g *getQueryBuilder) Conflicts() GetQueryBuilder {
	g.conflicts = true
	return g
}

func (g *getQueryBuilder) DeletedConflicts() GetQueryBuilder {
	g.deletedConflicts = true
	return g
}

func (g *getQueryBuilder) Latest() GetQueryBuilder {
	g.latest = true
	return g
}

func (g *getQueryBuilder) LocalSeq() GetQueryBuilder {
	g.localSeq = true
	return g
}

func (g *getQueryBuilder) Meta() GetQueryBuilder {
	g.meta = true
	return g
}

func (g *getQueryBuilder) OpenRevs(revs []string) GetQueryBuilder {
	g.openRevs = revs
	return g
}

func (g *getQueryBuilder) Rev(rev string) GetQueryBuilder {
	g.rev = rev
	return g
}

func (g *getQueryBuilder) Revs() GetQueryBuilder {
	g.revs = true
	return g
}

func (g *getQueryBuilder) RevsInfo() GetQueryBuilder {
	g.revsInfo = true
	return g
}

// GetQuery implements the QueryBuilder interface. It returns an
// url.Values map with the non-default values set.
func (gq *getQuery) GetQuery() (url.Values, error) {
	vals := url.Values{}

	if gq.Attachments {
		vals.Set("attachments", "true")
	}
	if gq.AttEncodingInfo {
		vals.Set("att_encoding_info", "true")
	}
	if len(gq.AttsSince) > 0 {
		data, err := json.Marshal(gq.AttsSince)
		if err != nil {
			return nil, err
		}
		vals.Set("attsSince", string(data[:]))
	}
	if gq.Conflicts {
		vals.Set("conflicts", "true")
	}
	if gq.DeletedConflicts {
		vals.Set("deleted_conflicts", "true")
	}
	if gq.Latest {
		vals.Set("latest", "true")
	}
	if gq.LocalSeq {
		vals.Set("local_seq", "true")
	}
	if gq.Meta {
		vals.Set("meta", "true")
	}
	if len(gq.OpenRevs) > 0 {
		data, err := json.Marshal(gq.OpenRevs)
		if err != nil {
			return nil, err
		}
		vals.Set("open_revs", string(data[:]))
	}
	if gq.Rev != "" {
		vals.Set("rev", gq.Rev)
	}
	if gq.Revs {
		vals.Set("revs", "true")
	}
	if gq.RevsInfo {
		vals.Set("revs_info", "true")
	}

	return vals, nil
}

func (g *getQueryBuilder) Build() *getQuery {
	return &getQuery{
		Attachments:      g.attachments,
		AttEncodingInfo:  g.attEncodingInfo,
		AttsSince:        g.attsSince,
		Conflicts:        g.conflicts,
		DeletedConflicts: g.deletedConflicts,
		Latest:           g.latest,
		LocalSeq:         g.localSeq,
		Meta:             g.meta,
		OpenRevs:         g.openRevs,
		Rev:              g.rev,
		Revs:             g.revs,
		RevsInfo:         g.revsInfo,
	}
}
