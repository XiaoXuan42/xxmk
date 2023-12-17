package xxmk

import (
	"log"
	"strings"
	"unicode/utf8"
)

/*
 * Block should contain the last '\n' if it exists.
 */
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
	node := &AstNode{
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

	return node
}

func _parseWithPrefix(s string, notation string, allowSuffix bool, pos Pos) (bool, Pos, Pos, string) {
	if pos.Col != 0 {
		log.Panicf("_parseWithPrefix should be invoked at the beginning of a line: %s", pos)
		return false, Pos{}, Pos{}, ""
	}
	// two notation + '\n'
	if len(s) < len(notation)*2+1 {
		return false, Pos{}, Pos{}, ""
	}
	if s[:len(notation)] != notation {
		return false, Pos{}, Pos{}, ""
	}

	curPos := pos
	curPos.ConsumeStr(notation)
	endPos := pos
	found := false

	newS := s[len(notation):]
	newLineIdx := strings.Index(newS, "\n")
	if newLineIdx < 0 {
		return false, Pos{}, Pos{}, ""
	}
	if newLineIdx != 0 && !allowSuffix {
		return false, Pos{}, Pos{}, ""
	}
	suffix := newS[:newLineIdx]
	curPos.ConsumeStr(newS[:newLineIdx+1])

	newS = newS[newLineIdx+1:]
	for len(newS) > 0 {
		if curPos.Col != 0 {
			panic("The 'start of the line' invariance is broken")
		}
		if len(newS) < len(notation) {
			return false, Pos{}, Pos{}, ""
		}
		newLineIdx = strings.Index(newS, "\n")
		if newLineIdx < 0 {
			if newS != notation {
				return false, Pos{}, Pos{}, ""
			}
			curPos.ConsumeStr(newS)
			endPos = curPos
			found = true
			break
		} else {
			if newS[:newLineIdx] == notation {
				curPos.ConsumeStr(newS[:newLineIdx+1])
				endPos = curPos
				found = true
				break
			}
			curPos.ConsumeStr(newS[:newLineIdx+1])
			newS = newS[newLineIdx+1:]
		}
	}

	if !found {
		return false, Pos{}, Pos{}, suffix
	}
	if endPos.Offset <= pos.Offset {
		panic("endPos should be after pos")
	}
	return true, pos, endPos, suffix
}

func parseMathBlock(s string, ctx parseContext) *AstNode {
	ret, start, end, _ := _parseWithPrefix(s, "$$", true, ctx.p)
	if ret {
		node := &AstNode{
			Type:        MathBlock{},
			Start:       start,
			End:         end,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseCodeBlock(s string, ctx parseContext) *AstNode {
	ret, start, end, suffix := _parseWithPrefix(s, "```", true, ctx.p)
	if ret {
		node := &AstNode{
			Type:        CodeBlock{Suffix: suffix},
			Start:       start,
			End:         end,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseStrong(s string, ctx parseContext) *AstNode {
	if len(s) < 4 {
		return nil
	}
	symbol := s[0]
	if (symbol != '*' && symbol != '_') || symbol != s[1] {
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
	node := &AstNode{
		Type:        Strong{},
		Start:       ctx.p,
		End:         curCtx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseItalic(s string, ctx parseContext) *AstNode {
	if len(s) <= 2 {
		return nil
	}
	symbol := s[0]
	if symbol != '*' && symbol != '_' {
		return nil
	}
	foundEnd := false
	curCtx := ctx
	curCtx.p.Consume(rune(symbol))
	for _, c := range s[1:] {
		curCtx.p.Consume(c)
		if c == rune(symbol) {
			foundEnd = true
			break
		} else if c == '\n' {
			break
		}
	}
	if !foundEnd {
		return nil
	}
	if curCtx.p.Offset <= ctx.p.Offset+2 {
		// forbid empty italic
		return nil
	}
	node := &AstNode{
		Type:        Italic{},
		Start:       ctx.p,
		End:         curCtx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseCode(s string, ctx parseContext) *AstNode {
	// multiple ` at the start/end will be the same as just one `
	// multiple ` not at the start/end will be regarded as normal characters
	if len(s) < 2 {
		return nil
	}
	if s[0] != '`' {
		return nil
	}
	curCtx := ctx
	curCtx.p.Consume(rune('`'))
	foundEnd := false
	inSeq := true

	newS := s[1:]
	for i, c := range newS {
		curCtx.p.Consume(c)
		if c == '`' {
			if inSeq {
				continue
			}
			if i+1 >= len(newS) || !utf8.RuneStart(newS[i+1]) || newS[i+1] != '`' {
				foundEnd = true
				break
			}
			inSeq = true
		} else {
			inSeq = false
		}
	}
	if !foundEnd {
		return nil
	}
	node := &AstNode{
		Type:        Code{},
		Start:       ctx.p,
		End:         curCtx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseMath(s string, ctx parseContext) *AstNode {
	if len(s) <= 2 {
		return nil
	}
	if s[0] != '$' {
		return nil
	}
	if s[1] == '$' {
		return nil  // $$ is not a valid inline math
	}
	newS := s[1:]
	curCtx := ctx
	curCtx.p.Consume(rune('$'))
	foundEnd := false
	for i, c := range newS {
		curCtx.p.Consume(c)
		if c == '$' {
			if i+1 >= len(newS) || newS[i+1] != '$' {
				foundEnd = true
				break
			} else {
				return nil
			}
		}
	}
	if !foundEnd {
		return nil
	}
	node := &AstNode{
		Type:        Math{},
		Start:       ctx.p,
		End:         curCtx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
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

	isNewLine := true
	for {
		for _, c := range s[ctx.p.Offset:] {
			var subnode *AstNode
			for j := len(parser.BlockParserSeq) - 1; isNewLine && (j >= 0); j-- {
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
				isNewLine = c == '\n'
			}
		}
		fAddTextNode()
		if ctx.p.Offset >= len(s) {
			break
		}
		if ctx.p.Col != 0 {
			panic("Bug: should parse to a new line here")
		}
		isNewLine = true
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
	parser.BlockParserSeq = append(parser.BlockParserSeq, parseCodeBlock)
	parser.BlockParserSeq = append(parser.BlockParserSeq, parseMathBlock)

	parser.InlineParserSeq = make(map[rune][]InlineParser)
	// strong first, italic second
	parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseItalic)
	parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseItalic)
	parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseStrong)
	parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseStrong)
	parser.InlineParserSeq[rune('`')] = append(parser.InlineParserSeq[rune('`')], parseCode)
	parser.InlineParserSeq[rune('$')] = append(parser.InlineParserSeq[rune('$')], parseMath)
	return parser
}
