package parserlib

import (
	"log"
)

type MKParser struct {
	BlockParserSeq  []BlockParser
	InlineParserSeq map[rune][]InlineParser
}

func (parser *MKParser) parseText(s string, ctx parseContext) *AstNode {
	node := AstNode{
		Type:        &Text{},
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
		curCtx.p.Consume(rune(s[curIdx]))
		curIdx += 1
	}
	textStartPos = curCtx.p
	node.Start = curCtx.p
	if curIdx == len(s) {
		return nil
	}
	doubleEnter := false
	for !doubleEnter {
		lastEscape, lastEnter := false, false
		for _, c := range s[curIdx:] {
			var subnode *AstNode
			if c == '\n' {
				if lastEnter {
					doubleEnter = true
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
						Type:        &Text{},
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

	if !doubleEnter && node.End.Offset-ctx.p.Offset != len(s) {
		panic("Bug: Text should contain all characters of the string")
	}
	return &node
}

func (parser *MKParser) Parse(s string) Ast {
	ast := Ast{
		Root: AstNode{
			Type:  &Document{},
			Start: Pos{Line: 0, Col: 0, Offset: 0},
		},
	}
	ctx := parseContext{
		p:           Pos{Line: 0, Col: 0, Offset: 0},
		parent:      &ast.Root,
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
				ast.Root.Children = append(ast.Root.Children, textnode)
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
				ast.Root.LeftSibling = subnode
				ast.Root.Children = append(ast.Root.Children, subnode)
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
						Type:        &Text{},
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
	ast.Root.End = ctx.p

	return ast
}

func (parser *MKParser) addDefaultInlineParser(name string) {
	switch name {
	case "Emphasis":
		parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseEmphasis)
		parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseEmphasis)
	case "Italic":
		parser.InlineParserSeq[rune('*')] = append(parser.InlineParserSeq[rune('*')], parseItalic)
		parser.InlineParserSeq[rune('_')] = append(parser.InlineParserSeq[rune('_')], parseItalic)
	case "StrikeThrough":
		parser.InlineParserSeq[rune('~')] = append(parser.InlineParserSeq[rune('~')], parseStrikeThrough)
	case "Code":
		parser.InlineParserSeq[rune('`')] = append(parser.InlineParserSeq[rune('`')], parseCode)
	case "Math":
		parser.InlineParserSeq[rune('$')] = append(parser.InlineParserSeq[rune('$')], parseMath)
	case "Link":
		parser.InlineParserSeq[rune('[')] = append(parser.InlineParserSeq[rune('[')], parseLink)
	case "SimpleLink":
		parser.InlineParserSeq[rune('<')] = append(parser.InlineParserSeq[rune('<')], parseSimpleLink)
	case "Image":
		parser.InlineParserSeq[rune('!')] = append(parser.InlineParserSeq[rune('!')], parseImage)
	case "Html":
		parser.InlineParserSeq[rune('<')] = append(parser.InlineParserSeq[rune('<')], parseHtml)
	case "ReferenceLink":
		parser.InlineParserSeq[rune('[')] = append(parser.InlineParserSeq[rune('[')], parseReferenceLink)
	case "FootNote":
		parser.InlineParserSeq[rune('[')] = append(parser.InlineParserSeq[rune('[')], parseFootNote)
	default:
		log.Panicf("%s is not supported", name)
	}
}

func _addAllDefaultInlineParsers(parser *MKParser) {
	// Emphasis before Italic
	// FootNote before ReferenceLink
	parser.AddDefaultInlineParsers([]string{
		"Emphasis", "Italic", "StrikeThrough", "Code", "Math", "Link", "SimpleLink", "Image", "Html", "FootNote", "ReferenceLink",
	})
}

func (parser *MKParser) AddDefaultInlineParsers(names []string) {
	// add in reverse order
	for i := len(names) - 1; i >= 0; i-- {
		parser.addDefaultInlineParser(names[i])
	}
}

func (parser *MKParser) AddExtensionInlineParser(lookAhead rune, method InlineParser) {
	parser.InlineParserSeq[lookAhead] = append(parser.InlineParserSeq[lookAhead], method)
}

func (parser *MKParser) addDefaultBlockParser(name string) {
	switch name {
	case "Header":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseHeader)
	case "QuoteBlock":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseQuoteBlock)
	case "CodeBlock":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseCodeBlock)
	case "MathBlock":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseMathBlock)
	case "Table":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseTable)
	case "HorizontalRule":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseHorizontalRule)
	case "List":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseList)
	case "ReferenceLinkIndex":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseReferenceLinkIndex)
	case "FootNoteIndex":
		parser.BlockParserSeq = append(parser.BlockParserSeq, parseFootNoteIndex)
	default:
		log.Panicf("%s is not supported", name)
	}
}

func _addAllDefaultBlockParsers(parser *MKParser) {
	parser.AddDefaultBlockParsers([]string{
		"HorizontalRule", "Header", "QuoteBlock", "CodeBlock", "MathBlock", "Table", "List", "FootNoteIndex", "ReferenceLinkIndex",
	})
}

func (parser *MKParser) AddDefaultBlockParsers(names []string) {
	for i := len(names) - 1; i >= 0; i-- {
		parser.addDefaultBlockParser(names[i])
	}
}

func (parser *MKParser) AddExtensionBlockParser(method BlockParser) {
	parser.BlockParserSeq = append(parser.BlockParserSeq, method)
}

func GetBaseParser() MKParser {
	parser := MKParser{}
	parser.InlineParserSeq = make(map[rune][]InlineParser)
	return parser
}

func GetBlockOnlyParser() MKParser {
	parser := GetBaseParser()
	_addAllDefaultBlockParsers(&parser)
	return parser
}

func GetInlineOnlyParser() MKParser {
	parser := GetBaseParser()
	_addAllDefaultInlineParsers(&parser)
	return parser
}

func GetFullMKParser() MKParser {
	parser := GetBaseParser()
	_addAllDefaultBlockParsers(&parser)
	_addAllDefaultInlineParsers(&parser)
	return parser
}
