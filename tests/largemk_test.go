package tests

import (
	"os"
	"testing"

	"github.com/XiaoXuan42/xxmk/parserlib"
	"github.com/stretchr/testify/assert"
)

func TestDontPanic(t *testing.T) {
	res, err := os.ReadFile("./largemk.md")
	assert.Equal(t, nil, err)
	s := string(res)

	parser := parserlib.GetFullMKParser()
	parser.Parse(s)
}
