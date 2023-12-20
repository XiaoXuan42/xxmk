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

func (pos *Pos) Back(c rune) {
	l := utf8.RuneLen(c)
	pos.Col -= l
	pos.Offset -= l
	if pos.Col < 0 || pos.Offset < 0 {
		panic("Invalid position")
	}
}

func (pos *Pos) _toProtobuf() *PosProto {
	res := &PosProto{}
	res.Line = int32(pos.Line)
	res.Col = int32(pos.Col)
	res.Offset = int32(pos.Offset)
	return res
}

func (pos *Pos) _fromProtobuf(buf *PosProto) {
	pos.Line = int(buf.Line)
	pos.Col = int(buf.Col)
	pos.Offset = int(buf.Offset)
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

func (astnode *AstNode) _toProtobuf(ar *[]*AstNodeProto) int32 {
	nodeProto := &AstNodeProto{}
	curId := int32(len(*ar))
	*ar = append(*ar, nodeProto)

	nodeProto.Type = &AstNodeTypeProto{}
	nodeProto.Type = _astNodeTypeToProtobuf(astnode.Type)
	nodeProto.Start = astnode.Start._toProtobuf()
	nodeProto.End = astnode.End._toProtobuf()

	for _, ch := range astnode.Children {
		chId := ch._toProtobuf(ar)
		nodeProto.Children = append(nodeProto.Children, chId)
	}
	leftsib := int32(-1)
	for _, chId := range nodeProto.Children {
		(*ar)[chId].Parent = curId
		(*ar)[chId].Leftsibling = leftsib
		leftsib = chId
	}
	return int32(curId)
}

func (astnode *AstNode) _fromProtobuf(ar []*AstNodeProto, curId int32) {
	curBuf := ar[curId]
	astnode.Type = _astNodeTypeFromProtobuf(curBuf.Type)
	astnode.Start._fromProtobuf(curBuf.Start)
	astnode.End._fromProtobuf(curBuf.End)

	var leftSib *AstNode
	for _, chId := range curBuf.Children {
		chNode := &AstNode{}
		chNode._fromProtobuf(ar, chId)
		chNode.Parent = astnode
		chNode.LeftSibling = leftSib
		leftSib = chNode
		astnode.Children = append(astnode.Children, chNode)
	}
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

func (ast *Ast) ToProtoBuf() *AstProto {
	buf := &AstProto{}
	ast.Root._toProtobuf(&buf.Nodes)
	return buf
}

func (ast *Ast) FromProtobuf(astProto *AstProto) {
	if astProto == nil {
		return
	}
	ast.Root._fromProtobuf(astProto.Nodes, 0)
}

func (ast *Ast) Eq(other *Ast) bool {
	return ast.Root._eq(&other.Root)
}
