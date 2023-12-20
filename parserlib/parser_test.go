package parserlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func _astCheck(node *AstNode) bool {
	var lastCh *AstNode
	for _, ch := range node.Children {
		if ch.LeftSibling != lastCh {
			return false
		}
		lastCh = ch
		if !_astCheck(ch) {
			return false
		}
	}
	return true
}

func TestSucc(t *testing.T) {
	parser := GetFullMKParser()
	mks := []string{
		`# Title1
- item1
- item2

hello world!
## Title2
1. item1
2. item2
`}
	for _, mk := range mks {
		parser.Parse(mk)
	}
}

func TestHeader(t *testing.T) {
	simplemk := `# TestHeader
this is a simple test
##  Section1
###Section2
sometexts...
## Section3
sometexts...`
	parser := GetFullMKParser()
	ast := parser.Parse(simplemk)
	if ast.Root.End.Line != 6 {
		t.Fatalf("Wrong line count: %d(expect 6)", ast.Root.End.Line)
	}
	headerCount := map[uint32]int{}
	headerTot := 0
	textCount := 0
	textTotalLen := 0
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *Text:
			textTotalLen += node.End.Offset - node.Start.Offset
			textCount += 1
		case *Header:
			headerCount[tp.Level] += 1
			headerTot += 1
		case *Document:
		default:
			t.Fatalf("Wrong node type: %s", node.Type.String())
		}
	})
	// note: ###Section2 is not a valid header
	if textCount != 6 {
		t.Fatalf("Wrong text count: %d", textCount)
	}
	if textTotalLen != 89 {
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
	parser := GetFullMKParser()
	ast := parser.Parse(simplemk)
	if ast.Root.End.Line != 5 {
		t.Fatalf("Wrong line count: %d", ast.Root.End.Line)
	}
	strongStrs := map[string]int{}
	italicStrs := map[string]int{}
	codeStrs := map[string]int{}
	mathStrs := map[string]int{}
	t.Logf("%s", ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case *Emphasis:
			t.Logf("Strong: %s", simplemk[node.Start.Offset:node.End.Offset])
			strongStrs[simplemk[node.Start.Offset+2:node.End.Offset-2]] += 1
		case *Italic:
			t.Logf("Italic: %s", simplemk[node.Start.Offset:node.End.Offset])
			italicStrs[simplemk[node.Start.Offset+1:node.End.Offset-1]] += 1
		case *Text:
			t.Logf("Text: %s", simplemk[node.Start.Offset:node.End.Offset])
		case *Code:
			t.Logf("Code: %s", simplemk[node.Start.Offset:node.End.Offset])
			codeStrs[simplemk[node.Start.Offset:node.End.Offset]] += 1
		case *Math:
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
	assert.Equal(t, 1, codeStrs["```good ` `job```"])
	assert.Equal(t, 1, codeStrs["` job`"])
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
	parser := GetFullMKParser()
	ast := parser.Parse(simplemk)
	var mathContent, codeContent []string
	var codeSuffix []string

	t.Logf("%s\n%s", simplemk, ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *MathBlock:
			mathContent = append(mathContent, simplemk[node.Start.Offset:node.End.Offset])
		case *CodeBlock:
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
	mk := `[hello](hello.com "hello link  "   ), this is a good image ![image](image.com)
[![image](imagePath)](imageLink) [images\[good](imagelink) <https://www.baidu.com> <link class="hello"></link>
<a><09@gmail.com>`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)

	var linkType []Link
	var linkNode []*AstNode
	var imageType []Image
	var imageNode []*AstNode
	var simpleLinkType []SimpleLink
	var simpleLinkNode []*AstNode
	var htmlStartType []HtmlStartTag
	t.Logf("%s\n%s", mk, ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *Link:
			linkNode = append(linkNode, node)
			linkType = append(linkType, *tp)
		case *Image:
			imageType = append(imageType, *tp)
			imageNode = append(imageNode, node)
		case *SimpleLink:
			simpleLinkType = append(simpleLinkType, *tp)
			simpleLinkNode = append(simpleLinkNode, node)
		case *HtmlStartTag:
			htmlStartType = append(htmlStartType, *tp)
		default:
		}
	})
	assert.Equal(t, 3, len(linkType))
	assert.Equal(t, 2, len(imageType))
	assert.Equal(t, 2, len(simpleLinkType))
	assert.Equal(t, 2, len(htmlStartType))
	assert.Equal(t, "hello", linkNode[0].Children[0].Text(mk))
	assert.Equal(t, "hello.com", linkType[0].Link)
	assert.Equal(t, "hello link  ", linkType[0].Title)
	assert.Equal(t, "image", imageNode[0].Children[0].Text(mk))
	assert.Equal(t, "image.com", imageType[0].Link)
	assert.Equal(t, "![image](imagePath)", linkNode[1].Children[0].Text(mk))
	assert.Equal(t, "imageLink", linkType[1].Link)
	assert.Equal(t, "image", imageNode[1].Children[0].Text(mk))
	assert.Equal(t, "imagePath", imageType[1].Link)
	assert.Equal(t, `images\[good`, linkNode[2].Children[0].Text(mk))
	assert.Equal(t, "https://www.baidu.com", simpleLinkType[0].Link)
	assert.Equal(t, "<https://www.baidu.com>", simpleLinkNode[0].Text(mk))
	assert.Equal(t, "09@gmail.com", simpleLinkType[1].Link)
	assert.Equal(t, "<09@gmail.com>", simpleLinkNode[1].Text(mk))
	assert.Equal(t, "link", htmlStartType[0].Tag)
	assert.Equal(t, "a", htmlStartType[1].Tag)
	assert.Equal(t, `class="hello"`, htmlStartType[0].Content)
}

func TestTable(t *testing.T) {
	mk := `|  hello |  world| |
|:--: | --: | -- |
abc
|d | e   | f|
| g |  |  i`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	t.Logf("%s", ast.String())
	_astCheck(&ast.Root)
	assert.Equal(t, 1, len(ast.Root.Children))
	assert.Equal(t, "Table", ast.Root.Children[0].Type.String())
	assert.Equal(t, 5, len(ast.Root.Children[0].Children))
	headerNode := ast.Root.Children[0].Children[0]
	alignNode := ast.Root.Children[0].Children[1]
	lineNodes := ast.Root.Children[0].Children[2:]
	assert.Equal(t, "TableHead", headerNode.Type.String())
	assert.Equal(t, 3, len(headerNode.Children))
	assert.Equal(t, "hello", headerNode.Children[0].Text(mk))
	assert.Equal(t, "world", headerNode.Children[1].Text(mk))
	assert.Equal(t, "", headerNode.Children[2].Text(mk))

	assert.Equal(t, "TableAlign", alignNode.Type.String())
	tbAlign := alignNode.Type.(*TableAlign)
	assert.Equal(t, 3, len(tbAlign.Aligns))
	assert.Equal(t, AlignMiddle, tbAlign.Aligns[0])
	assert.Equal(t, AlignRight, tbAlign.Aligns[1])
	assert.Equal(t, AlignLeft, tbAlign.Aligns[2])

	line0 := lineNodes[0]
	assert.Equal(t, 1, len(line0.Children))
	assert.Equal(t, "abc", line0.Children[0].Text(mk))
	line1 := lineNodes[1]
	assert.Equal(t, 3, len(line1.Children))
	assert.Equal(t, "d", line1.Children[0].Text(mk))
	assert.Equal(t, "e", line1.Children[1].Text(mk))
	assert.Equal(t, "f", line1.Children[2].Text(mk))
	line2 := lineNodes[2]
	assert.Equal(t, 3, len(line2.Children))
	assert.Equal(t, "g", line2.Children[0].Text(mk))
	assert.Equal(t, "", line2.Children[1].Text(mk))
	assert.Equal(t, "i", line2.Children[2].Text(mk))
}

func TestQuoteBlock(t *testing.T) {
	mk := `>    hello world

>>    
>
>>nice to meet you!`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	assert.Equal(t, 4, len(ast.Root.Children))

	texts := []string{
		"hello world", "", "", "nice to meet you!",
	}
	levels := []uint32{1, 2, 1, 2}
	for i := 0; i < 4; i++ {
		assert.Equal(t, "QuoteBlock", ast.Root.Children[i].Type.String())
		assert.Equal(t, levels[i], ast.Root.Children[i].Type.(*QuoteBlock).Level)
		assert.Equal(t, 1, len(ast.Root.Children[i].Children))
		assert.Equal(t, "Text", ast.Root.Children[i].Children[0].Type.String())
		assert.Equal(t, texts[i], ast.Root.Children[i].Children[0].Text(mk))
	}
}

func TestHorizontalRule(t *testing.T) {
	mk := `# good
--
---
*
***`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	t.Logf(ast.String())
	_astCheck(&ast.Root)

	hcnt := 0
	hLines := []int{}
	ast.Root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case *HorizontalRule:
			hcnt += 1
			hLines = append(hLines, node.Start.Line)
		}
	})
	assert.Equal(t, 2, hcnt)
	assert.Equal(t, []int{2, 4}, hLines)
}

func TestStrikeThrough(t *testing.T) {
	mk := `~~good job

~~nice to meet you~
~~nice to ~ ~ meet you!~~`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	collects := []string{}
	ast.Root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case *StrikeThrough:
			collects = append(collects, node.Children[0].Text(mk))
		}
	})
	assert.Equal(t, 1, len(collects))
	assert.Equal(t, "nice to ~ ~ meet you!", collects[0])
}

func TestList(t *testing.T) {
	mk := `- item1
- item2
- item3
1.    item4
- item5
3. item6
6. item7`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	assert.Equal(t, 4, len(ast.Root.Children))
	lst1 := ast.Root.Children[0]
	lst2 := ast.Root.Children[1]
	lst3 := ast.Root.Children[2]
	lst4 := ast.Root.Children[3]
	assert.Equal(t, false, lst1.Type.(*List).IsOrdered)
	assert.Equal(t, 3, len(lst1.Children))
	itemNames := []string{" item1", " item2", " item3"}
	for i, ch := range lst1.Children {
		assert.Equal(t, 1, len(ch.Children))
		assert.Equal(t, "Text", ch.Children[0].Type.String())
		assert.Equal(t, itemNames[i], ch.Children[0].Text(mk))
	}

	assert.Equal(t, true, lst2.Type.(*List).IsOrdered)
	assert.Equal(t, 1, len(lst2.Children))
	itemNames = []string{"    item4"}
	orders := []uint32{1}
	for i, ch := range lst2.Children {
		assert.Equal(t, orders[i], ch.Type.(*ListItem).Order)
		assert.Equal(t, 1, len(ch.Children))
		assert.Equal(t, "Text", ch.Children[0].Type.String())
		assert.Equal(t, itemNames[i], ch.Children[0].Text(mk))
	}

	assert.Equal(t, false, lst3.Type.(*List).IsOrdered)
	assert.Equal(t, 1, len(lst3.Children))
	itemNames = []string{" item5"}
	for i, ch := range lst3.Children {
		assert.Equal(t, 1, len(ch.Children))
		assert.Equal(t, "Text", ch.Children[0].Type.String())
		assert.Equal(t, itemNames[i], ch.Children[0].Text(mk))
	}

	assert.Equal(t, true, lst4.Type.(*List).IsOrdered)
	assert.Equal(t, 2, len(lst4.Children))
	itemNames = []string{" item6", " item7"}
	orders = []uint32{3, 4}
	for i, ch := range lst4.Children {
		assert.Equal(t, orders[i], ch.Type.(*ListItem).Order)
		assert.Equal(t, 1, len(ch.Children))
		assert.Equal(t, "Text", ch.Children[0].Type.String())
		assert.Equal(t, itemNames[i], ch.Children[0].Text(mk))
	}
}

func TestParseLinkTitle(t *testing.T) {
	inputs := []string{" https://somewhere.com", `http://nice.com "nice"  `}
	trueMap := map[int][]string{
		0: {"https://somewhere.com", ""},
		1: {"http://nice.com", "nice"},
	}
	for i := 0; i < len(inputs); i++ {
		ok, link, title := _parseLinkTitle(inputs[i])
		assert.Equal(t, true, ok)
		assert.Equal(t, trueMap[i][0], link)
		assert.Equal(t, trueMap[i][1], title)
	}
}

func TestReferenceLink(t *testing.T) {
	mk := `reference link: [reflink ][1]
[link][ link ]: reference link
[ link ]: https://somewhere.com
[1]: http://nice.com "nice"
`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)

	var refLink []ReferenceLink
	var refNode []*AstNode
	var refLinkIndex []ReferenceLinkIndex
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *ReferenceLink:
			refLink = append(refLink, *tp)
			refNode = append(refNode, node)
		case *ReferenceLinkIndex:
			refLinkIndex = append(refLinkIndex, *tp)
		}
	})
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	assert.Equal(t, 2, len(refLink))
	assert.Equal(t, 2, len(refLinkIndex))
	trueRefMap := map[int][]string{
		0: {"1", "reflink ", "https://somewhere.com", ""},
		1: {" link ", "link", "http://nice.com", "nice"},
	}
	trueIndexMap := map[int][]string{
		0: {" link ", "https://somewhere.com", ""},
		1: {"1", "http://nice.com", "nice"},
	}
	for i := 0; i < 2; i++ {
		assert.Equal(t, trueRefMap[i][0], refLink[i].Index)
		assert.Equal(t, trueRefMap[i][1], refNode[i].Children[0].Text(mk))
		assert.Equal(t, trueIndexMap[i][0], refLinkIndex[i].Index)
		assert.Equal(t, trueIndexMap[i][1], refLinkIndex[i].Link)
		assert.Equal(t, trueIndexMap[i][2], refLinkIndex[i].Title)
	}
}

func TestFootNote(t *testing.T) {
	mk := `hello [^1], this is me![^MyDescription]

[^1]: world
[^MyDescription]: David`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)

	var footType []FootNote
	var footIndexType []FootNoteIndex
	var footIndexNode []*AstNode

	t.Logf(ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *FootNote:
			footType = append(footType, *tp)
		case *FootNoteIndex:
			footIndexNode = append(footIndexNode, node)
			footIndexType = append(footIndexType, *tp)
		}
	})
	trueMap := map[int][]string{
		0: {"1", " world"},
		1: {"MyDescription", " David"},
	}
	assert.Equal(t, 2, len(footType))
	assert.Equal(t, 2, len(footIndexType))
	for i := 0; i < 2; i++ {
		assert.Equal(t, trueMap[i][0], footType[i].Index)
		assert.Equal(t, trueMap[i][0], footIndexType[i].Index)
		assert.Equal(t, trueMap[i][1], footIndexNode[i].Children[0].Text(mk))
	}
}

func TestTaskList(t *testing.T) {
	mk := `- [] hello world!
- [ ] good morning
- [x] nice to meet you! 
- [ ] how are you?`
	parser := GetFullMKParser()
	itemCnt := 0
	var listItemType []ListItem
	var listItemNode []*AstNode
	ast := parser.Parse(mk)
	t.Logf(ast.String())
	_astCheck(&ast.Root)
	ast.Root.PreVisit(func(node *AstNode) {
		switch tp := node.Type.(type) {
		case *ListItem:
			itemCnt += 1
			listItemType = append(listItemType, *tp)
			listItemNode = append(listItemNode, node)
		}
	})
	assert.Equal(t, 4, itemCnt)
	trueMap := map[int]string{
		0: " [] hello world!",
		1: " good morning",
		2: " nice to meet you! ",
		3: " how are you?",
	}
	finishMap := map[int][]bool{
		0: {false, false},
		1: {true, false},
		2: {true, true},
		3: {true, false},
	}
	for i := 1; i < 4; i++ {
		assert.Equal(t, trueMap[i], listItemNode[i].Children[0].Text(mk))
		assert.Equal(t, finishMap[i][0], listItemType[i].IsTask)
		assert.Equal(t, finishMap[i][1], listItemType[i].IsFinished)
	}
}

func TestSimpleAST(t *testing.T) {
	mk := `Part1

Part2`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	s := ast.String()
	t.Logf("%s", s)
	_astCheck(&ast.Root)
	textCnt := 0
	ast.Root.PreVisit(func(node *AstNode) {
		switch node.Type.(type) {
		case *Text:
			textCnt += 1
		}
	})
	assert.Equal(t, 2, textCnt)
}
