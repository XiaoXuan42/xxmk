package xxmk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	simplemk := `# TestHeader
this is a simple test
##  Section1
###Section2
sometexts...
## Section3
sometexts...`
	parser := GetBaseMKParser()
	ast := parser.Parse(simplemk)
	if ast.root.End.Line != 6 {
		t.Fatalf("Wrong line count: %d(expect 6)", ast.root.End.Line)
	}
	headerCount := map[int]int{}
	headerTot := 0
	textCount := 0
	textTotalLen := 0
	ast.root.PreVisit(func (node *AstNode) {
		switch tp := node.Type.(type) {
		case Text:
			textTotalLen += node.End.Offset - node.Start.Offset
			textCount += 1
		case Header:
			headerCount[tp.Level] += 1
			headerTot += 1
		case Document:
		default:
			t.Fatalf("Wrong node type: %s", node.Type.String())
		}
	})
	// note: ###Section2 is not a valid header
	if textCount != 6 {
		t.Fatalf("Wrong text count: %d", textCount)
	}
	if textTotalLen != 88 {
		t.Fatalf("Wrong text total length: %d", textTotalLen)
	}
	if headerTot != 3 {
		t.Fatalf("Wrong total header count: %d", headerTot)
	}
	if headerCount[1] != 1 {
		t.Fatalf("Wrong level 1 header count: %d", headerCount[1])
	}
	if headerCount[2] != 2 {
		t.Fatalf("Wrong level 2 header count: %d", headerCount[2])
	}
	if headerCount[3] != 0 {
		t.Fatalf("Wrong level 3 header count: %d", headerCount[3])
	}
}

func TestStrong(t *testing.T) {
	simplemk := `**helloworld**
# **hello*world**
__hello_ world__
__hello_world__
__helloworld__
__nice to meet you!__`
	parser := GetBaseMKParser()
	ast := parser.Parse(simplemk)
	if ast.root.End.Line != 5 {
		t.Fatalf("Wrong line count: %d", ast.root.End.Line)
	}
	strongStrs := map[string]int{}
	ast.root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case StrongText:
			t.Logf("%s %d", simplemk[node.Start.Offset:node.End.Offset], simplemk[node.End.Offset-1])
			strongStrs[simplemk[node.Start.Offset+2:node.End.Offset-2]] += 1
		default:
		}
	})
	for k, v := range strongStrs {
		t.Logf("key: %s, value: %d", k, v)
	}
	assert.Equal(t, 2, strongStrs["helloworld"])
	assert.Equal(t, 1, strongStrs["hello*world"])
	assert.Equal(t, 1, strongStrs["hello_world"])
	assert.Equal(t, 1, strongStrs["nice to meet you!"])
	assert.Equal(t, 0, strongStrs["hello_ world"])
}
