package parserlib

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

type Pos struct {
	Line   int
	Col    int
	Offset int
}

func (pos Pos) String() string {
	return fmt.Sprintf("[Line %d, Col %d, Offset %d]", pos.Line, pos.Col, pos.Offset)
}

func (pos *Pos) Consume(c rune) {
	if c == '\n' {
		pos.Line = pos.Line + 1
		pos.Col = 0
	} else {
		pos.Col = pos.Col + 1
	}
	pos.Offset = pos.Offset + utf8.RuneLen(c)
}

func (pos *Pos) ConsumeStr(s string) {
	for _, c := range s {
		pos.Consume(c)
	}
}

func (pos *Pos) ForwardInlineByInt(cnt int) {
	pos.Col += cnt
	pos.Offset += cnt
}

func (pos *Pos) Back(c rune) {
	l := utf8.RuneLen(c)
	pos.Col -= l
	pos.Offset -= l
	if pos.Col < 0 || pos.Offset < 0 {
		panic("Invalid position")
	}
}

type AstNode struct {
	Type  AstNodeType
	Start Pos
	End   Pos

	Parent      *AstNode
	LeftSibling *AstNode
	Children    []*AstNode
}

func (astnode *AstNode) StringLines() ([]string, []int) {
	if astnode == nil {
		return []string{}, []int{}
	}
	subs := make([]string, 0)
	paddings := make([]int, 0)
	subs = append(subs, astnode.Type.String()+astnode.Start.String())
	paddings = append(paddings, 0)

	for _, node := range astnode.Children {
		childStrs, childPads := node.StringLines()
		subs = append(subs, childStrs...)
		for _, p := range childPads {
			paddings = append(paddings, p+1)
		}
	}
	return subs, paddings
}

func (astnode *AstNode) String() string {
	if astnode == nil {
		return "None"
	}
	lines, paddings := astnode.StringLines()
	paddedLines := make([]string, 0)
	for i := 0; i < len(lines); i++ {
		pad := strings.Repeat("  ", paddings[i])
		var curLine string
		if i == 0 {
			curLine = pad + lines[i]
		} else {
			curLine = pad + "|-" + lines[i]
		}
		paddedLines = append(paddedLines, curLine)
	}
	return strings.Join(paddedLines, "\n")
}

func (astnode *AstNode) PreVisit(f func(*AstNode)) {
	if astnode == nil {
		return
	}
	f(astnode)
	for _, ch := range astnode.Children {
		ch.PreVisit(f)
	}
}

func (astnode *AstNode) Text(s string) string {
	return s[astnode.Start.Offset:astnode.End.Offset]
}

func (astnode *AstNode) _eq(other *AstNode) bool {
	if !reflect.DeepEqual(astnode.Type, other.Type) {
		return false
	}
	if len(astnode.Children) != len(other.Children) {
		return false
	}
	for i := 0; i < len(astnode.Children); i++ {
		if !astnode.Children[i]._eq(other.Children[i]) {
			return false
		}
	}
	return true
}

type Ast struct {
	Root AstNode
}

func (ast *Ast) String() string {
	if ast == nil {
		return "None"
	}
	return ast.Root.String()
}

func (ast *Ast) Eq(other *Ast) bool {
	return ast.Root._eq(&other.Root)
}

type AstIterator struct {
	Cur *AstNode
	Ch  int
}

func (it AstIterator) AstItToRight() AstIterator {
	if it.Cur == nil {
		return AstIterator{}
	}
	parent := it.Cur.Parent
	if parent == nil {
		return AstIterator{}
	} else if len(parent.Children) <= it.Ch+1 {
		return AstIterator{}
	} else {
		return AstIterator{
			Cur: parent.Children[it.Ch+1],
			Ch:  it.Ch + 1,
		}
	}
}

func (it AstIterator) AstItToLeft() AstIterator {
	if it.Cur == nil {
		return AstIterator{}
	}
	parent := it.Cur.Parent
	if parent == nil {
		return AstIterator{}
	} else if it.Ch <= 0 {
		return AstIterator{}
	} else {
		return AstIterator{
			Cur: parent.Children[it.Ch-1],
			Ch:  it.Ch - 1,
		}
	}
}

func (it AstIterator) AstItToFstChild() AstIterator {
	if it.Cur == nil {
		return AstIterator{}
	} else if len(it.Cur.Children) == 0 {
		return AstIterator{}
	} else {
		return AstIterator{
			Cur: it.Cur.Children[0],
			Ch:  0,
		}
	}
}

func (it AstIterator) AstFindRightFirstType(targetId int, until *AstNode) AstIterator {
	failedRes := AstIterator{}
	if targetId == GetNodeTypeId(it.Cur.Type) {
		return it
	}
	if it.Cur.Parent == nil {
		return failedRes
	}
	for i := it.Ch; i < len(it.Cur.Parent.Children); i++ {
		curNode := it.Cur.Parent.Children[i]
		if curNode == until {
			return failedRes
		}
		curId := GetNodeTypeId(curNode.Type)
		if curId == targetId {
			return AstIterator{
				Cur: curNode,
				Ch:  i,
			}
		}
	}
	return failedRes
}

func (it AstIterator) AstFindRightFirstTypeByStr(tp string, until *AstNode) AstIterator {
	targetId := GetNodeTypeIdFromStr(tp)
	return it.AstFindRightFirstType(targetId, until)
}
