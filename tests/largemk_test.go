package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/XiaoXuan42/xxmk/parserlib"
)

func TestDontPanic(t *testing.T) {
	res, err := os.ReadFile("./largemk.md")
	assert.Equal(t, nil, err)
	s := string(res)

	parser := parserlib.GetHtmlMKParser()
	parser.Parse(s)
}
