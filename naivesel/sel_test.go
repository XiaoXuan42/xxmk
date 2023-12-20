package naivesel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type S1 struct {
	Url   string
	Title string
	Link  string
}

func TestEncDecSimple(t *testing.T) {
	refLink := S1{Title: "link", Link: "http"}
	bytes := Serialize(&refLink)
	refLink2 := S1{}
	Deserialize(&refLink2, bytes)
	assert.Equal(t, 0, len(refLink2.Url))
	assert.Equal(t, "link", refLink2.Title)
	assert.Equal(t, "http", refLink2.Link)
}
