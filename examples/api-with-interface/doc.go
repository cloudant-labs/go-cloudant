/**
 * doc
 * - Defines Doc structure and validation bindings
 */

package main

// Doc holds plan document
type Doc struct {
	ID  string `json:"_id,omitempty"`
	Rev string `json:"_rev,omitempty"`
	Foo string `json:"foo" binding:"required"`
}

// DocsView contains results of Cloudant view
type DocsView struct {
	Rows  []DocsViewRow `json:"rows"`
	Error string        `json:"error"`
}

// DocsViewRow Row contains one row from Cloudant view
type DocsViewRow struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Doc   Doc    `json:"doc"`
}
