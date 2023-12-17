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
	ast.root.PreVisit(func(node *AstNode) {
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
	simplemk += "```good ` `job``` `` job`"
	simplemk += "$hello$ $$world$ $world$$"
	parser := GetBaseMKParser()
	ast := parser.Parse(simplemk)
	if ast.root.End.Line != 5 {
		t.Fatalf("Wrong line count: %d", ast.root.End.Line)
	}
	strongStrs := map[string]int{}
	italicStrs := map[string]int{}
	codeStrs := map[string]int{}
	mathStrs := map[string]int{}
	t.Logf("%s", ast.String())
	ast.root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case Strong:
			t.Logf("Strong: %s", simplemk[node.Start.Offset:node.End.Offset])
			strongStrs[simplemk[node.Start.Offset+2:node.End.Offset-2]] += 1
		case Italic:
			t.Logf("Italic: %s", simplemk[node.Start.Offset:node.End.Offset])
			italicStrs[simplemk[node.Start.Offset+1:node.End.Offset-1]] += 1
		case Text:
			t.Logf("Text: %s", simplemk[node.Start.Offset:node.End.Offset])
		case Code:
			t.Logf("Code: %s", simplemk[node.Start.Offset:node.End.Offset])
			codeStrs[simplemk[node.Start.Offset:node.End.Offset]] += 1
		case Math:
			t.Logf("Math: %s", simplemk[node.Start.Offset:node.End.Offset])
			mathStrs[simplemk[node.Start.Offset:node.End.Offset]] += 1
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
	assert.Equal(t, 1, italicStrs["hello"], 1)
	assert.Equal(t, 0, italicStrs["helloworld"])
	assert.Equal(t, 0, italicStrs["hello world"])
	assert.Equal(t, 0, italicStrs["world"])
	assert.Equal(t, 1, codeStrs["```good `"])
	assert.Equal(t, 1, codeStrs["`job``` `` job`"])
	assert.Equal(t, 1, mathStrs["$hello$"])
	assert.Equal(t, 0, mathStrs["$$world$"])
	assert.Equal(t, 0, mathStrs["$world$$"])
}

func TestBlock1(t *testing.T) {
	mathsnippt1 := `$$a
a + b = 2
$$`
	mathsnippt2 := `$$
e^{i\theta} = cos\theta + i sin\theta
$$`
	codesnippt := "```c++\n" + `#include<iostream>
using namespace std;
int main() {
	return 0;
}` + "\n```"

	simplemk := mathsnippt1
	simplemk += "\n" + codesnippt
	simplemk += "\n" + mathsnippt2
	parser := GetBaseMKParser()
	ast := parser.Parse(simplemk)
	var mathContent, codeContent []string
	var codeSuffix []string

	t.Logf("%s\n%s", simplemk, ast.String())
	ast.root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case MathBlock:
			mathContent = append(mathContent, simplemk[node.Start.Offset:node.End.Offset])
		case CodeBlock:
			codeSuffix = append(codeSuffix, tp.Suffix)
			codeContent = append(codeContent, simplemk[node.Start.Offset:node.End.Offset])
		default:
		}
	})
	assert.Equal(t, 1, len(codeSuffix))
	assert.Equal(t, "c++", codeSuffix[0])
	assert.Equal(t, 2, len(mathContent))
	assert.Equal(t, mathsnippt1+"\n", mathContent[0])
	assert.Equal(t, mathsnippt2, mathContent[1])
	assert.Equal(t, 1, len(codeContent))
	assert.Equal(t, codesnippt+"\n", codeContent[0])
}

func TestLink(t *testing.T) {
	mk := `[hello](hello.com), this is a good image ![image](image.com)`
	parser := GetBaseMKParser()
	ast := parser.Parse(mk)

	var linkType Link
	var imageType Image
	linkCnt, imageCnt := 0, 0
	t.Logf("%s\n%s", mk, ast.String())
	ast.root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case Link:
			linkType = tp
			linkCnt += 1
		case Image:
			imageType = tp
			imageCnt += 1
		default:
		}
	})
	assert.Equal(t, 1, linkCnt)
	assert.Equal(t, 1, imageCnt)
	assert.Equal(t, "hello", linkType.name)
	assert.Equal(t, "hello.com", linkType.link)
	assert.Equal(t, "image", imageType.name)
	assert.Equal(t, "image.com", imageType.link)
}
