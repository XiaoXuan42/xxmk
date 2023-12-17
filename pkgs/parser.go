package xxmk

import (
	"log"
)

type InlineParser func(string, parseContext) *AstNode
type BlockParser func(string, parseContext) *AstNode
type parseContext struct {
	p           Pos
	parent      *AstNode
	leftSibling *AstNode
	parseText   func(string, parseContext) *AstNode
}

func parseHeader(s string, ctx parseContext) *AstNode {
	if len(s) == 0 || s[0] != '#' {
		return nil
	}
	head := Header{Level: 0}
	node := AstNode{
		Start:       ctx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	i := 0
	for i < len(s) {
		if s[i] != '#' {
			break
		}
		head.Level = head.Level + 1
		ctx.p.Consume(rune(s[i]))
		i = i + 1
	}
	if i < len(s) && s[i] != ' ' {
		return nil
	}
	for i < len(s) {
		if s[i] != ' ' {
			break
		}
		ctx.p.Consume(rune(s[i]))
		i = i + 1
	}
	j := i
	for j < len(s) {
		if s[j] == '\n' {
			j = j + 1
			break
		}
		j = j + 1
	}

	textnode := ctx.parseText(s[i:j], ctx)
	if textnode != nil {
		node.Children = append(node.Children, textnode)
	} else {
		log.Panicf("Failed to parse string. (start: %s)", ctx.p.String())
	}
	ctx.p.ConsumeStr(s[i:j])

	node.Type = head
	node.End = ctx.p

	return &node
}

func parseStrong(s string, ctx parseContext) *AstNode {
	if len(s) < 4 {
		return nil
	}
	symbol := s[0]
	if symbol != '*' && symbol != '_' && symbol != s[1] {
		return nil
	}
	lastSymbol := false
	foundEnd := false
	curCtx := ctx
	curCtx.p.Consume(rune(symbol))
	curCtx.p.Consume(rune(symbol))
	for _, c := range s[2:] {
		curCtx.p.Consume(c)
		if c == rune(symbol) && lastSymbol {
			foundEnd = true
			break
		}
		if lastSymbol && c == ' ' {
			break
		}
		if c == '\n' {
			break
		} else if c == rune(symbol) {
			lastSymbol = true
		} else {
			lastSymbol = false
		}
	}
	if !foundEnd {
		return nil
	}
	node := AstNode{
		Type:        StrongText{},
		Start:       ctx.p,
		End:         curCtx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return &node
}

type MKParser struct {
	BlockParserSeq  []BlockParser
	InlineParserSeq map[rune][]InlineParser
}

func (parser *MKParser) parseText(s string, ctx parseContext) *AstNode {
	node := AstNode{
		Type:        Text{},
		Start:       ctx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := parseContext{
		p:           ctx.p,
		parent:      &node,
		leftSibling: nil,
		parseText:   ctx.parseText,
	}

	textNodeAdded := false
	textStartPos := curCtx.p
	fAddPrevTextNode := func() bool {
		if textNodeAdded {
			if curCtx.p.Offset == textStartPos.Offset {
				panic("Bug: should not add an empty text node")
			}
			if curCtx.leftSibling == nil || curCtx.leftSibling.Type.String() != "Text" {
				panic("Bug: text node is not really added")
			}
			if curCtx.leftSibling.End != curCtx.p {
				panic("Bug: text node added is not complete")
			}
			node.Children = append(node.Children, curCtx.leftSibling)
			return true
		} else {
			if curCtx.p.Offset != textStartPos.Offset {
				panic("Bug: should add a new text node")
			}
			return false
		}
	}

	curIdx := 0
	// remove leading \n
	// '\n' in utf-8 is 1 byte
	for curIdx < len(s) {
		if s[curIdx] != '\n' {
			break
		}
		ctx.p.Consume(rune(s[curIdx]))
		curIdx += 1
	}
	textStartPos = ctx.p
	if curIdx == len(s) {
		return nil
	}
	for {
		lastEscape, lastEnter := false, false
		for _, c := range s[curIdx:] {
			var subnode *AstNode
			if c == '\n' {
				if lastEnter {
					break
				}
				lastEscape = false
				lastEnter = true
			} else if c == '\\' {
				lastEnter = false
				lastEscape = !lastEscape
			} else {
				lastEnter = false
				if lastEscape {
					lastEscape = false
				} else if parsers, ok := parser.InlineParserSeq[c]; ok {
					// in reverse order
					for i := len(parsers) - 1; i >= 0; i-- {
						offset := curCtx.p.Offset - ctx.p.Offset
						subnode = parsers[i](s[offset:], curCtx)
						if subnode != nil {
							break
						}
					}
				}
			}

			if subnode != nil {
				if subnode.End.Offset <= curCtx.p.Offset {
					panic("Bug: subnode's offset should be larger")
				}
				fAddPrevTextNode()
				curCtx.leftSibling = subnode
				curCtx.p = subnode.End
				textStartPos = curCtx.p
				textNodeAdded = false
				node.Children = append(node.Children, subnode)
				break
			} else {
				curCtx.p.Consume(c)
				if textNodeAdded {
					curCtx.leftSibling.End = curCtx.p
				} else {
					curCtx.leftSibling = &AstNode{
						Type:        Text{},
						Start:       textStartPos,
						End:         curCtx.p,
						Parent:      curCtx.parent,
						LeftSibling: curCtx.leftSibling,
					}
					textNodeAdded = true
				}
				curCtx.leftSibling.End = curCtx.p
			}
		}

		fAddPrevTextNode()
		textStartPos = curCtx.p
		textNodeAdded = false

		curIdx = curCtx.p.Offset - ctx.p.Offset
		if curIdx >= len(s) {
			break
		}
	}

	node.End = curCtx.p
	if len(node.Children) == 1 && node.Children[0].Type.String() == "Text" {
		if node.Children[0].LeftSibling != nil {
			panic("Bug: children's left sibling must be nil")
		}
		if node.Children[0].Start != node.Start || node.Children[0].End != node.End {
			panic("Bug: children's range mismatches with parent's")
		}
		node.Start = node.Children[0].Start
		node.End = node.Children[0].End
		node.Children = nil
	}

	if node.End.Offset-node.Start.Offset != len(s) {
		panic("Bug: Text should contain all characters of the string")
	}
	return &node
}

func (parser *MKParser) Parse(s string) Ast {
	ast := Ast{
		root: AstNode{
			Type:  Document{},
			Start: Pos{Line: 0, Col: 0, Offset: 0},
		},
	}
	ctx := parseContext{
		p:           Pos{Line: 0, Col: 0, Offset: 0},
		parent:      &ast.root,
		leftSibling: nil,
		parseText:   parser.parseText,
	}

	textStartPos := ctx.p
	textNodeAdded := false
	fAddTextNode := func() {
		if textNodeAdded {
			if textStartPos == ctx.p {
				panic("Bug: should not add an empty text node")
			}
			if ctx.leftSibling == nil || ctx.leftSibling.Type.String() != "Text" {
				panic("Bug: text node is not really added")
			}
			if ctx.leftSibling.End != ctx.p {
				panic("Bug: text node added is not complete")
			}

			// provide correct context for parsing text node
			endPoint := ctx.p
			ctx.leftSibling = ctx.leftSibling.LeftSibling
			ctx.p = textStartPos

			for ctx.p != endPoint {
				if ctx.p.Offset > endPoint.Offset {
					panic("Bug: ctx's offset should not exceed endPoint's")
				}
				textnode := ctx.parseText(s[ctx.p.Offset:endPoint.Offset], ctx)
				if textnode == nil {
					// trailing '\n'
					ctx.p = endPoint
					break
				}
				ast.root.Children = append(ast.root.Children, textnode)
				ctx.leftSibling = textnode
				ctx.p = textnode.End
			}
			textStartPos = ctx.p
			textNodeAdded = false
		} else {
			if textStartPos != ctx.p {
				panic("Bug: should add a new text node")
			}
		}
	}

	for {
		for _, c := range s[ctx.p.Offset:] {
			var subnode *AstNode
			for j := len(parser.BlockParserSeq) - 1; j >= 0; j-- {
				blkParser := parser.BlockParserSeq[j]
				if subnode = blkParser(s[ctx.p.Offset:], ctx); subnode != nil {
					break
				}
			}

			if subnode != nil {
				fAddTextNode()
				if subnode.End.Offset <= ctx.p.Offset {
					panic("Bug: subnode's offset should be larger")
				}
				ast.root.LeftSibling = subnode
				ast.root.Children = append(ast.root.Children, subnode)
				ctx.leftSibling = subnode
				ctx.p = subnode.End
				textStartPos = ctx.p
				break
			} else {
				if textNodeAdded {
					ctx.p.Consume(c)
					if ctx.leftSibling.Type.String() != "Text" {
						panic("Bug: should add a 'Text' node")
					}
					ctx.leftSibling.End = ctx.p
				} else {
					// create phony text node to "simulate" the context
					textStartPos = ctx.p
					ctx.p.Consume(c)
					ctx.leftSibling = &AstNode{
						Type:        Text{},
						Start:       textStartPos,
						End:         ctx.p,
						Parent:      ctx.parent,
						LeftSibling: ctx.leftSibling,
					}
					textNodeAdded = true
				}
			}
		}
		fAddTextNode()
		if ctx.p.Offset >= len(s) {
			break
		}
	}

	if ctx.p.Offset < len(s) {
		panic("Bug: parser should read all characters")
	}
	ast.root.End = ctx.p

	return ast
}

func GetBaseMKParser() MKParser {
	parser := MKParser{}
	parser.BlockParserSeq = append(parser.BlockParserSeq, parseHeader)
	parser.InlineParserSeq = make(map[rune][]InlineParser)
	parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseStrong)
	parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseStrong)
	return parser
}
