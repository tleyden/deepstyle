package deepstylelib

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/tleyden/go-couch"
)

type ChangesFeedFollower struct {
	SyncGatewayUrl *url.URL
	Database       couch.Database
}

func NewChangesFeedFollower(syncGatewayUrl string) (*ChangesFeedFollower, error) {

	// if it has a trailing slash, remove it
	rawUrl := strings.TrimSuffix(syncGatewayUrl, "/")

	// url validation
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	db, err := couch.Connect(url.String())
	if err != nil {
		return nil, fmt.Errorf("Error connecting to db: %v.  Err: %v", syncGatewayUrl, err)
	}

	return &ChangesFeedFollower{
		SyncGatewayUrl: url,
		Database:       db,
	}, nil
}

func (f ChangesFeedFollower) Follow() {

	log.Printf("Follow feed: %v", f.SyncGatewayUrl)

	var since interface{}

	handleChange := func(reader io.Reader) interface{} {
		changes, err := decodeChanges(reader)
		if err != nil {
			// it's very common for this to timeout while waiting for new changes.
			// since we want to follow the changes feed forever, just log an error
			// TODO: don't even log an error if its an io.Timeout, just noise
			log.Printf("%T error decoding changes: %v.", err, err)
			return since
		}

		f.processChanges(changes)

		since = changes.LastSequence

		return since

	}

	options := map[string]interface{}{}
	options["feed"] = "longpoll"

	f.Database.Changes(handleChange, options)

}

func (f ChangesFeedFollower) processChanges(changes couch.Changes) {
	log.Printf("processChanges: %v", changes)
}

func decodeChanges(reader io.Reader) (couch.Changes, error) {

	changes := couch.Changes{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&changes)
	return changes, err

}
