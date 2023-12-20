package parserlib

import (
	"log"
)

type MKParser struct {
	BlockParserSeq  []BlockParser
	InlineParserSeq map[rune][]InlineParser
}

func (parser *MKParser) parseText(s string, ctx ParseContext) *AstNode {
	node := AstNode{
		Type:        &Text{},
		Start:       ctx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ParseContext{
		P:           ctx.P,
		Parent:      &node,
		LeftSibling: nil,
		ParseText:   ctx.ParseText,
	}

	textNodeAdded := false
	textStartPos := curCtx.P
	fAddPrevTextNode := func() bool {
		if textNodeAdded {
			if curCtx.P.Offset == textStartPos.Offset {
				panic("Bug: should not add an empty text node")
			}
			if curCtx.LeftSibling == nil || curCtx.LeftSibling.Type.String() != "Text" {
				panic("Bug: text node is not really added")
			}
			if curCtx.LeftSibling.End != curCtx.P {
				panic("Bug: text node added is not complete")
			}
			node.Children = append(node.Children, curCtx.LeftSibling)
			return true
		} else {
			if curCtx.P.Offset != textStartPos.Offset {
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
		curCtx.P.Consume(rune(s[curIdx]))
		curIdx += 1
	}
	textStartPos = curCtx.P
	node.Start = curCtx.P
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
						offset := curCtx.P.Offset - ctx.P.Offset
						subnode = parsers[i](s[offset:], curCtx)
						if subnode != nil {
							break
						}
					}
				}
			}

			if subnode != nil {
				if subnode.End.Offset <= curCtx.P.Offset {
					panic("Bug: subnode's offset should be larger")
				}
				fAddPrevTextNode()
				curCtx.LeftSibling = subnode
				curCtx.P = subnode.End
				textStartPos = curCtx.P
				textNodeAdded = false
				node.Children = append(node.Children, subnode)
				break
			} else {
				curCtx.P.Consume(c)
				if textNodeAdded {
					curCtx.LeftSibling.End = curCtx.P
				} else {
					curCtx.LeftSibling = &AstNode{
						Type:        &Text{},
						Start:       textStartPos,
						End:         curCtx.P,
						Parent:      curCtx.Parent,
						LeftSibling: curCtx.LeftSibling,
					}
					textNodeAdded = true
				}
				curCtx.LeftSibling.End = curCtx.P
			}
		}

		fAddPrevTextNode()
		textStartPos = curCtx.P
		textNodeAdded = false

		curIdx = curCtx.P.Offset - ctx.P.Offset
		if curIdx >= len(s) {
			break
		}
	}

	node.End = curCtx.P
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

	if !doubleEnter && node.End.Offset-ctx.P.Offset != len(s) {
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
	ctx := ParseContext{
		P:           Pos{Line: 0, Col: 0, Offset: 0},
		Parent:      &ast.Root,
		LeftSibling: nil,
		ParseText:   parser.parseText,
	}

	textStartPos := ctx.P
	textNodeAdded := false
	fAddTextNode := func() {
		if textNodeAdded {
			if textStartPos == ctx.P {
				panic("Bug: should not add an empty text node")
			}
			if ctx.LeftSibling == nil || ctx.LeftSibling.Type.String() != "Text" {
				panic("Bug: text node is not really added")
			}
			if ctx.LeftSibling.End != ctx.P {
				panic("Bug: text node added is not complete")
			}

			// provide correct context for parsing text node
			endPoint := ctx.P
			ctx.LeftSibling = ctx.LeftSibling.LeftSibling
			ctx.P = textStartPos

			for ctx.P != endPoint {
				if ctx.P.Offset > endPoint.Offset {
					panic("Bug: ctx's offset should not exceed endPoint's")
				}
				textnode := ctx.ParseText(s[ctx.P.Offset:endPoint.Offset], ctx)
				if textnode == nil {
					// trailing '\n'
					ctx.P = endPoint
					break
				}
				ast.Root.Children = append(ast.Root.Children, textnode)
				ctx.LeftSibling = textnode
				ctx.P = textnode.End
			}
			textStartPos = ctx.P
			textNodeAdded = false
		} else {
			if textStartPos != ctx.P {
				panic("Bug: should add a new text node")
			}
		}
	}

	isNewLine := true
	for {
		for _, c := range s[ctx.P.Offset:] {
			var subnode *AstNode
			for j := len(parser.BlockParserSeq) - 1; isNewLine && (j >= 0); j-- {
				blkParser := parser.BlockParserSeq[j]
				if subnode = blkParser(s[ctx.P.Offset:], ctx); subnode != nil {
					break
				}
			}

			if subnode != nil {
				fAddTextNode()
				if subnode.End.Offset <= ctx.P.Offset {
					panic("Bug: subnode's offset should be larger")
				}
				ast.Root.LeftSibling = subnode
				ast.Root.Children = append(ast.Root.Children, subnode)
				ctx.LeftSibling = subnode
				ctx.P = subnode.End
				textStartPos = ctx.P
				break
			} else {
				if textNodeAdded {
					ctx.P.Consume(c)
					if ctx.LeftSibling.Type.String() != "Text" {
						panic("Bug: should add a 'Text' node")
					}
					ctx.LeftSibling.End = ctx.P
				} else {
					// create phony text node to "simulate" the context
					textStartPos = ctx.P
					ctx.P.Consume(c)
					ctx.LeftSibling = &AstNode{
						Type:        &Text{},
						Start:       textStartPos,
						End:         ctx.P,
						Parent:      ctx.Parent,
						LeftSibling: ctx.LeftSibling,
					}
					textNodeAdded = true
				}
				isNewLine = c == '\n'
			}
		}
		fAddTextNode()
		if ctx.P.Offset >= len(s) {
			break
		}
		if ctx.P.Col != 0 {
			panic("Bug: should parse to a new line here")
		}
		isNewLine = true
	}

	if ctx.P.Offset < len(s) {
		panic("Bug: parser should read all characters")
	}
	ast.Root.End = ctx.P

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
