package xxmk

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type AstNodeType interface {
	String() string
}

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

type Ast struct {
	root AstNode
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

func (astnode *AstNode) PreVisit(f func (*AstNode)) {
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

func (ast *Ast) String() string {
	if ast == nil {
		return "None"
	}
	return ast.root.String()
}
