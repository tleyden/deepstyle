package deepstylelib

import (
	"testing"

	"github.com/tleyden/go-couch"
)

/*

TODO:
   - spin up local database
   - add job document
   - etc ..
func TestExecuteDeepStyleJob(t *testing.T) {

	config := configuration{
		Database:     couch.Database{},
		TempDir:      "/tmp",
		UnitTestMode: true,
	}

	jobDoc := JobDocument{}

	err := executeDeepStyleJob(config, jobDoc)
	assert.True(t, err == nil)

}
*/

func TestAddAttachment(t *testing.T) {
	config := configuration{
		Database:     couch.Database{},
		TempDir:      "/tmp",
		UnitTestMode: true,
	}

	jobDoc := JobDocument{}
	jobDoc.SetConfiguration(config)
	jobDoc.AddAttachment("foo", "/tmp/foo.png")

}
