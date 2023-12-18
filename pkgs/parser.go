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

func parseTable(s string, ctx parseContext) *AstNode {
	type LineResult struct {
		valid     bool
		hasOrMark bool
		sep       int
		start     Pos
		end       Pos
		texts     []*AstNode
	}

	parseTableLine := func(s string, ctx parseContext) LineResult {
		result := LineResult{start: ctx.p, end: ctx.p}

		sep := strings.Index(s, "\n")

		isTailNewLine := false
		if sep == 0 {
			return result
		} else if sep < 0 {
			sep = len(s)
			result.end.ConsumeStr(s)
			result.sep = sep
		} else {
			isTailNewLine = true
			result.end.ConsumeStr(s[:sep+1])
			s = s[:sep]
			result.sep = sep + 1
		}

		cur := 0
		if s[0] == '|' {
			cur = 1
			result.hasOrMark = true
			ctx.p.Consume('|')
		}
		for ; cur < len(s); {
			curSep := strings.Index(s[cur:], "|")
			var textStr string
			nextP := ctx.p
			if curSep < 0 {
				curSep = sep
				textStr = s[cur:]
				nextP.ConsumeStr(s[cur:])
			} else {
				curSep += cur
				result.hasOrMark = true
				textStr = s[cur:curSep]
				nextP.ConsumeStr(s[cur : curSep+1])
			}

			leadingSpace, tailingSpace := 0, len(textStr)-1
			for ; leadingSpace < len(textStr); leadingSpace++ {
				if textStr[leadingSpace] != ' ' {
					break
				}
				ctx.p.Consume(' ')
			}
			for ; tailingSpace >= 0; tailingSpace-- {
				if textStr[tailingSpace] != ' ' {
					break
				}
			}
			var curText *AstNode
			if tailingSpace < leadingSpace {
				curText = &AstNode{
					Type:        Text{},
					Start:       ctx.p,
					End:         ctx.p,
					Parent:      ctx.parent,
					LeftSibling: ctx.leftSibling,
				}
			} else {
				curText = ctx.parseText(textStr[leadingSpace:tailingSpace+1], ctx)
			}
			if curText == nil {
				log.Panicf("Failed to parse table line: %s", s)
			}
			ctx.leftSibling = curText
			ctx.p = nextP
			result.texts = append(result.texts, curText)
			cur = curSep + 1
		}
		if isTailNewLine {
			ctx.p.Consume('\n')
		}
		result.valid = true
		if result.end != ctx.p {
			log.Panicf("Should agree on the end position: %s, %s", result.end.String(), ctx.p.String())
		}
		endSep := result.end.Offset - result.start.Offset
		if endSep != result.sep {
			log.Panicf("End and sep should agree on the end position: %d, %d", endSep, result.sep)
		}
		return result
	}

	tableNode := &AstNode{
		Type:        Table{},
		Start:       ctx.p,
		End:         ctx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := ctx
	curCtx.parent = tableNode
	curCtx.leftSibling = nil
	curRear := 0

	headerNode := &AstNode{
		Type:        TableHead{},
		Start:       curCtx.p,
		End:         curCtx.p,
		Parent:      curCtx.parent,
		LeftSibling: curCtx.leftSibling,
	}
	curCtx.parent = headerNode
	curCtx.leftSibling = nil

	headResult := parseTableLine(s, curCtx)
	if !headResult.valid || !headResult.hasOrMark {
		return nil
	}
	headerNode.Children = append(headerNode.Children, headResult.texts...)
	headerNode.End = headResult.end
	curCtx.p = headResult.end
	curRear = headResult.sep
	if curRear >= len(s) {
		return nil
	}

	alignType := TableAlign{}
	alignNode := &AstNode{
		Type:        TableAlign{},
		Start:       curCtx.p,
		End:         curCtx.p,
		Parent:      tableNode,
		LeftSibling: headerNode,
	}
	curCtx.parent = alignNode
	curCtx.leftSibling = nil
	alignResult := parseTableLine(s[curRear:], curCtx)
	if !alignResult.valid || len(alignResult.texts) != len(headResult.texts) {
		return nil
	}
	// convert texts to aligns
	for _, textnode := range alignResult.texts {
		startOff := textnode.Start.Offset - ctx.p.Offset
		endOff := textnode.End.Offset - ctx.p.Offset
		sAlign := s[startOff:endOff]
		isLeft, isRight := false, false
		if len(sAlign) == 0 {
			return nil
		} else if len(sAlign) == 1 && sAlign[0] != '-' {
			return nil
		} else {
			if sAlign[0] == ':' {
				isLeft = true
				if sAlign[1] != '-' {
					return nil
				}
			}
			if sAlign[len(sAlign)-1] == ':' {
				isRight = true
			}
			if isLeft && isRight {
				alignType.aligns = append(alignType.aligns, AlignMiddle)
			} else if isRight {
				alignType.aligns = append(alignType.aligns, AlignRight)
			} else {
				alignType.aligns = append(alignType.aligns, AlignLeft)
			}
		}
	}
	alignNode.Type = alignType
	alignNode.End = alignResult.end
	curCtx.p = alignResult.end
	curRear += alignResult.sep

	curCtx.parent = tableNode
	curCtx.leftSibling = alignNode
	lineNodes := []*AstNode{}
	for curRear < len(s) {
		if s[curRear] == '\n' {
			curCtx.p.Consume('\n')
			break
		}
		lineNode := &AstNode{
			Type:        TableLine{},
			Start:       curCtx.p,
			End:         curCtx.p,
			Parent:      tableNode,
			LeftSibling: curCtx.leftSibling,
		}
		lineResult := parseTableLine(s[curRear:], curCtx)
		if !lineResult.valid {
			return nil
		}
		lineNode.Children = append(lineNode.Children, lineResult.texts...)
		lineNode.End = lineResult.end
		curCtx.p = lineResult.end
		curCtx.leftSibling = lineNode
		curRear += lineResult.sep
		lineNodes = append(lineNodes, lineNode)
	}
	tableNode.Children = append(tableNode.Children, headerNode)
	tableNode.Children = append(tableNode.Children, alignNode)
	tableNode.Children = append(tableNode.Children, lineNodes...)
	tableNode.End = curCtx.p
	return tableNode
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
		return nil // $$ is not a valid inline math
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

func _parseLinkLike(s string, pos Pos) (bool, string, string, Pos) {
	if len(s) == 0 || s[0] != '[' {
		return false, "", "", Pos{}
	}
	curPos := pos
	newLineIdx := strings.Index(s, "\n")
	rightIdx := strings.Index(s, "]")
	if rightIdx < 0 || (newLineIdx >= 0 && newLineIdx < rightIdx) {
		return false, "", "", Pos{}
	}
	name := s[1:rightIdx]
	curPos.ConsumeStr(s[:rightIdx+1])

	newS := s[rightIdx+1:]
	if len(newS) < 2 || newS[0] != '(' {
		return false, "", "", Pos{}
	}
	newLineIdx = strings.Index(newS, "\n")
	rightIdx = strings.Index(newS, ")")
	if rightIdx < 0 || (newLineIdx >= 0 && newLineIdx < rightIdx) {
		return false, "", "", Pos{}
	}
	link := newS[1:rightIdx]
	curPos.ConsumeStr(s[:rightIdx+1])
	return true, name, link, curPos
}

func parseLink(s string, ctx parseContext) *AstNode {
	ret, name, link, pos := _parseLinkLike(s, ctx.p)
	if ret {
		return &AstNode{
			Type:        Link{name: name, link: link},
			Start:       ctx.p,
			End:         pos,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
	} else {
		return nil
	}
}

func parseImage(s string, ctx parseContext) *AstNode {
	if len(s) < 1 && s[0] != '!' {
		return nil
	}
	curPos := ctx.p
	curPos.Consume(rune(s[0]))
	ret, name, link, pos := _parseLinkLike(s[1:], ctx.p)
	if ret {
		return &AstNode{
			Type:        Image{name: name, link: link},
			Start:       ctx.p,
			End:         pos,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
	} else {
		return nil
	}
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
	parser.BlockParserSeq = append(parser.BlockParserSeq, parseTable)

	parser.InlineParserSeq = make(map[rune][]InlineParser)
	// strong first, italic second
	parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseItalic)
	parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseItalic)
	parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseStrong)
	parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseStrong)
	parser.InlineParserSeq[rune('`')] = append(parser.InlineParserSeq[rune('`')], parseCode)
	parser.InlineParserSeq[rune('$')] = append(parser.InlineParserSeq[rune('$')], parseMath)
	parser.InlineParserSeq[rune('[')] = append(parser.InlineParserSeq[rune('[')], parseLink)
	parser.InlineParserSeq[rune('!')] = append(parser.InlineParserSeq[rune('!')], parseImage)
	return parser
}
