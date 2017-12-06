package cloudant

import (
	"bufio"
	"encoding/json"
	"strings"
)

// Constants defining the possible event types in a changes feed
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

// ChangeEvent is the message structure delivered by the Read function
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

// NewFollower creates a Follower on database's changes
func NewFollower(database *Database, interval int) *Follower {
	follower := &Follower{
		db:          database,
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		seqInterval: interval,
	}
	return follower
}

// Close will terminate the Follower
func (f *Follower) Close() {
	close(f.stop)
	<-f.stopped
}

// Follow starts listening to the changes feed
func (f *Follower) Follow() (<-chan *ChangeEvent, error) {
	query := NewChangesQuery().
		IncludeDocs().
		Feed("continuous").
		Since(f.since).
		Heartbeat(10).
		Timeout(30)

	if f.seqInterval > 0 {
		query = query.SeqInterval(f.seqInterval)
	}

	params, _ := query.Build().GetQuery()

	urlStr, err := Endpoint(*f.db.URL, "/_changes", params)
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
