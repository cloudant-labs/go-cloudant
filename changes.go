package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Change represents a part returned by _changes.
type Change struct {
	ID      string
	Rev     string
	Seq     string
	Deleted bool
	Doc     map[string]interface{} // Only present if Changes() called with include_docs=true
}

// ChangeRow represents a part returned by _changes.
type ChangeRow struct {
	ID      string                 `json:"id"`
	Seq     string                 `json:"seq"` // If using CouchDB1.6, this is a number
	Changes []ChangeRowChanges     `json:"changes"`
	Deleted bool                   `json:"deleted"`
	Doc     map[string]interface{} `json:"doc"`
}

// Constants defining the possible event types in a changes feed.
const (
	// ChangesInsert is a new document, with _rev starting with "1-"
	ChangesInsert = iota
	// ChangesUpdate is a new revison of an existing document
	ChangesUpdate
	// ChangesDelete is a document deletion
	ChangesDelete
	// ChangesHeartbeat is an empty line sent to keep the connection open
	ChangesHeartbeat
	// ChangesTerminated means far end closed the connection
	ChangesTerminated
	ChangesError
)

// ChangeEvent is the message structure delivered by the Read function.
type ChangeEvent struct {
	EventType int
	Meta      *DocumentMeta
	Seq       string
	Doc       map[string]interface{}
	Err       error
}

// Follower is the orchestrator
type Follower struct {
	db          *Database
	stop        chan struct{}
	stopped     chan struct{}
	since       string
	seqInterval int
}

// UnmarshalJSON is here for coping with CouchDB1.6's sequence IDs being
// numbers, not strings as in Cloudant and CouchDB2.X.
//
// See https://play.golang.org/p/BytXCeHMvt
func (c *ChangeRow) UnmarshalJSON(data []byte) error {
	// Create a new type with same structure as ChangeRow but without its method set
	// to avoid an infinite `UnmarshalJSON` call stack
	type ChangeRow16 ChangeRow
	changeRow := struct {
		ChangeRow16
		Seq json.Number `json:"seq"`
	}{ChangeRow16: ChangeRow16(*c)}

	if err := json.Unmarshal(data, &changeRow); err != nil {
		return err
	}

	*c = ChangeRow(changeRow.ChangeRow16)
	c.Seq = changeRow.Seq.String()

	return nil
}

// ChangeRowChanges represents a part returned by _changes.
type ChangeRowChanges struct {
	Rev string `json:"rev"`
}

// Changes returns a channel in which Change types can be received.
// See: https://console.bluemix.net/docs/services/Cloudant/api/database.html#get-changes
func (d *Database) Changes(q *ChangesQuery) (<-chan *Change, error) {
	verb := "GET"
	var body []byte
	var err error
	if len(q.DocIDValues) > 0 {
		// If we're given a "doc_ids" argument, we're better off with a POST
		body, err = json.Marshal(map[string][]string{"doc_ids": q.DocIDValues})
		if err != nil {
			return nil, err
		}
		verb = "POST"
	}

	urlStr, err := Endpoint(*d.URL, "/_changes", q.URLValues)
	if err != nil {
		return nil, err
	}
	job, err := d.client.request(verb, urlStr, bytes.NewReader(body))
	if err != nil {
		job.done()
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		job.done()
		return nil, err
	}

	changes := make(chan *Change, 1000)

	go func(job *Job, changes chan<- *Change) {
		defer job.Close()
		defer close(changes)

		reader := bufio.NewReader(job.response.Body)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			lineStr := string(line)
			lineStr = strings.TrimSpace(lineStr)      // remove whitespace
			lineStr = strings.TrimRight(lineStr, ",") // remove trailing comma

			if len(lineStr) > 7 && lineStr[0:7] == "{\"seq\":" {
				var change = new(ChangeRow)

				err := json.Unmarshal([]byte(lineStr), change)
				if err == nil && len(change.Changes) == 1 {
					changes <- &Change{
						ID:      change.ID,
						Rev:     change.Changes[0].Rev,
						Seq:     change.Seq,
						Doc:     change.Doc,
						Deleted: change.Deleted,
					}
				} else {
					fmt.Println(err)
				}
			}
		}
	}(job, changes)

	return changes, nil
}

// eventType tries to classify the current event as insert, delete or update.
// This is problematic: https://pouchdb.com/guides/changes.html#understanding-changes
// Under certain circumstances, the INSERT may be missed.
func eventType(change *ChangeRow) int {
	if change.Deleted {
		return ChangesDelete
	}
	if strings.HasPrefix(change.Changes[0].Rev, "1-") {
		return ChangesInsert
	}
	return ChangesUpdate
}

// NewFollower creates a Follower on database's changes.
func (d *Database) NewFollower(interval int) *Follower {
	follower := &Follower{
		db:          d,
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		seqInterval: interval,
	}
	return follower
}

// Close will terminate the Follower.
func (f *Follower) Close() {
	close(f.stop)
	<-f.stopped
}

// Follow starts listening to the changes feed.
func (f *Follower) Follow() (<-chan *ChangeEvent, error) {
	q := NewChangesQuery().
		IncludeDocs().
		Feed("continuous").
		Since(f.since).
		Heartbeat(10).
		Timeout(30).
		SeqInterval(f.seqInterval)

	urlStr, err := Endpoint(*f.db.URL, "/_changes", q.URLValues)
	if err != nil {
		return nil, err
	}

	job, err := f.db.client.request("GET", urlStr, nil)
	if err != nil {
		job.Close()
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		job.Close()
		return nil, err
	}

	changes := make(chan *ChangeEvent, 1000)
	go func() {
		defer job.Close()
		defer close(f.stopped) // This lets consumers block until terminated

		reader := bufio.NewReader(job.response.Body)

		for {
			select {
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					changes <- &ChangeEvent{EventType: ChangesTerminated}
					return
				}
				lineStr := strings.TrimSpace(string(line))
				if lineStr == "" {
					changes <- &ChangeEvent{EventType: ChangesHeartbeat}
					continue
				}
				if len(lineStr) > 7 && lineStr[0:7] == "{\"seq\":" {
					change := &ChangeRow{}

					err := json.Unmarshal([]byte(lineStr), change)
					if err == nil && len(change.Changes) == 1 {
						// Save the sequence ID so that we can resume from the
						// last processed event if asked to. The sequence ID will
						// be null if we're between seq_intervals.
						if change.Seq != "null" {
							f.since = change.Seq
						}
						changes <- &ChangeEvent{
							EventType: eventType(change),
							Meta: &DocumentMeta{
								ID:  change.ID,
								Rev: change.Changes[0].Rev,
							},
							Seq: change.Seq,
							Doc: change.Doc,
						}
					} else {
						changes <- &ChangeEvent{
							EventType: ChangesError,
							Err:       err,
						}
					}
				}
			case <-f.stop:
				return
			}
		}
	}()

	return changes, nil
}
