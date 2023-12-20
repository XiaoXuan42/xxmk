package parserlib

import (
	"log"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

/*
 * Block should contain the last '\n' if it exists.
 */
type InlineParser func(string, ParseContext) *AstNode
type BlockParser func(string, ParseContext) *AstNode
type ParseContext struct {
	P           Pos
	Parent      *AstNode
	LeftSibling *AstNode
	ParseText   func(string, ParseContext) *AstNode
}

// strings.Index() that take escape symbol \ into account
// pattern contains '\' is not supported
func _findInLine(s string, pattern string) int {
	if len(pattern) <= 0 || len(s) <= 0 {
		return -1
	}
	if len(pattern) == 1 {
		r := rune(pattern[0])
		lastEscape := false
		for i, c := range s {
			if c == r && !lastEscape {
				return i
			}
			if c == '\\' {
				lastEscape = true
			} else {
				lastEscape = false
			}
		}
		return -1
	} else {
		return strings.Index(s, pattern)
	}
}

func _matchUrl(s string) bool {
	urlRegex := regexp.MustCompile(`^\w+://[\w\.]+(:[0-9]+)?(/\w+)*(\?(\w+=\w+\&)*\w+=\w+)?(#\w+)?$`)
	res := urlRegex.MatchString(s)
	return res
}

func _matchEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}

func _textOrEmpty(text string, ctx ParseContext) *AstNode {
	if len(text) == 0 {
		return &AstNode{
			Type:        &Text{},
			Start:       ctx.P,
			End:         ctx.P,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
	} else {
		textnode := ctx.ParseText(text, ctx)
		if textnode == nil {
			log.Panicf("Failed to parse text: %s", text)
		}
		return textnode
	}
}

/* Block parsers */
func parseHeader(s string, ctx ParseContext) *AstNode {
	if len(s) == 0 || s[0] != '#' {
		return nil
	}
	head := Header{Level: 0}
	node := &AstNode{
		Start:       ctx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	i := 0
	for i < len(s) {
		if s[i] != '#' {
			break
		}
		head.Level = head.Level + 1
		ctx.P.Consume(rune(s[i]))
		i = i + 1
	}
	if i < len(s) && s[i] != ' ' && s[i] != '\n' {
		return nil
	}
	j := strings.Index(s, "\n")
	var text string
	endPos := ctx.P
	if j < 0 {
		text = s[i:]
		endPos.ConsumeStr(s[i:])
	} else {
		text = s[i:j]
		endPos.ConsumeStr(s[i : j+1])
	}
	textnode := _textOrEmpty(text, ctx)
	node.Children = append(node.Children, textnode)

	node.Type = &head
	node.End = endPos

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

func parseMathBlock(s string, ctx ParseContext) *AstNode {
	ret, start, end, _ := _parseWithPrefix(s, "$$", true, ctx.P)
	if ret {
		node := &AstNode{
			Type:        &MathBlock{},
			Start:       start,
			End:         end,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseCodeBlock(s string, ctx ParseContext) *AstNode {
	ret, start, end, suffix := _parseWithPrefix(s, "```", true, ctx.P)
	if ret {
		node := &AstNode{
			Type:        &CodeBlock{Suffix: suffix},
			Start:       start,
			End:         end,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseTable(s string, ctx ParseContext) *AstNode {
	type LineResult struct {
		valid     bool
		hasOrMark bool
		sep       int
		start     Pos
		end       Pos
		texts     []*AstNode
	}

	parseTableLine := func(s string, ctx ParseContext) LineResult {
		result := LineResult{start: ctx.P, end: ctx.P}

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
			ctx.P.Consume('|')
		}
		for cur < len(s) {
			curSep := _findInLine(s[cur:], "|")
			var textStr string
			nextP := ctx.P
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
				ctx.P.Consume(' ')
			}
			for ; tailingSpace >= 0; tailingSpace-- {
				if textStr[tailingSpace] != ' ' {
					break
				}
			}
			var curText *AstNode
			if tailingSpace < leadingSpace {
				curText = &AstNode{
					Type:        &Text{},
					Start:       ctx.P,
					End:         ctx.P,
					Parent:      ctx.Parent,
					LeftSibling: ctx.LeftSibling,
				}
			} else {
				curText = ctx.ParseText(textStr[leadingSpace:tailingSpace+1], ctx)
			}
			if curText == nil {
				log.Panicf("Failed to parse table line: %s", s)
			}
			ctx.LeftSibling = curText
			ctx.P = nextP
			result.texts = append(result.texts, curText)
			cur = curSep + 1
		}
		if isTailNewLine {
			ctx.P.Consume('\n')
		}
		result.valid = true
		if result.end != ctx.P {
			log.Panicf("Should agree on the end position: %s, %s", result.end.String(), ctx.P.String())
		}
		endSep := result.end.Offset - result.start.Offset
		if endSep != result.sep {
			log.Panicf("End and sep should agree on the end position: %d, %d", endSep, result.sep)
		}
		return result
	}

	tableNode := &AstNode{
		Type:        &Table{},
		Start:       ctx.P,
		End:         ctx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ctx
	curCtx.Parent = tableNode
	curCtx.LeftSibling = nil
	curRear := 0

	headerNode := &AstNode{
		Type:        &TableHead{},
		Start:       curCtx.P,
		End:         curCtx.P,
		Parent:      curCtx.Parent,
		LeftSibling: curCtx.LeftSibling,
	}
	curCtx.Parent = headerNode
	curCtx.LeftSibling = nil

	headResult := parseTableLine(s, curCtx)
	if !headResult.valid || !headResult.hasOrMark {
		return nil
	}
	headerNode.Children = append(headerNode.Children, headResult.texts...)
	headerNode.End = headResult.end
	curCtx.P = headResult.end
	curRear = headResult.sep
	if curRear >= len(s) {
		return nil
	}

	alignType := TableAlign{}
	alignNode := &AstNode{
		Type:        &TableAlign{},
		Start:       curCtx.P,
		End:         curCtx.P,
		Parent:      tableNode,
		LeftSibling: headerNode,
	}
	curCtx.Parent = alignNode
	curCtx.LeftSibling = nil
	alignResult := parseTableLine(s[curRear:], curCtx)
	if !alignResult.valid || len(alignResult.texts) != len(headResult.texts) {
		return nil
	}
	// convert texts to aligns
	for _, textnode := range alignResult.texts {
		startOff := textnode.Start.Offset - ctx.P.Offset
		endOff := textnode.End.Offset - ctx.P.Offset
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
				alignType.Aligns = append(alignType.Aligns, AlignMiddle)
			} else if isRight {
				alignType.Aligns = append(alignType.Aligns, AlignRight)
			} else {
				alignType.Aligns = append(alignType.Aligns, AlignLeft)
			}
		}
	}
	alignNode.Type = &alignType
	alignNode.End = alignResult.end
	curCtx.P = alignResult.end
	curRear += alignResult.sep

	curCtx.Parent = tableNode
	curCtx.LeftSibling = alignNode
	lineNodes := []*AstNode{}
	for curRear < len(s) {
		if s[curRear] == '\n' {
			curCtx.P.Consume('\n')
			break
		}
		lineNode := &AstNode{
			Type:        &TableLine{},
			Start:       curCtx.P,
			End:         curCtx.P,
			Parent:      tableNode,
			LeftSibling: curCtx.LeftSibling,
		}
		lineResult := parseTableLine(s[curRear:], curCtx)
		if !lineResult.valid {
			return nil
		}
		lineNode.Children = append(lineNode.Children, lineResult.texts...)
		lineNode.End = lineResult.end
		curCtx.P = lineResult.end
		curCtx.LeftSibling = lineNode
		curRear += lineResult.sep
		lineNodes = append(lineNodes, lineNode)
	}
	tableNode.Children = append(tableNode.Children, headerNode)
	tableNode.Children = append(tableNode.Children, alignNode)
	tableNode.Children = append(tableNode.Children, lineNodes...)
	tableNode.End = curCtx.P
	return tableNode
}

func parseQuoteBlock(s string, ctx ParseContext) *AstNode {
	if len(s) < 1 || s[0] != '>' {
		return nil
	}
	blkType := QuoteBlock{}
	node := &AstNode{
		Type:        &QuoteBlock{},
		Start:       ctx.P,
		End:         ctx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ctx
	curCtx.Parent = node
	curCtx.LeftSibling = nil
	end := strings.Index(s, "\n")
	if end < 0 {
		end = len(s)
		node.End.ConsumeStr(s)
	} else {
		node.End.ConsumeStr(s[:end+1])
	}

	i := 0
	for i < len(s) {
		if s[i] != '>' {
			break
		}
		curCtx.P.Consume('>')
		blkType.Level += 1
		i += 1
	}
	node.Type = &blkType
	for i < len(s) {
		if s[i] != ' ' {
			break
		}
		i += 1
		curCtx.P.Consume(' ')
	}
	var textnode *AstNode
	if i >= end {
		textnode = &AstNode{
			Type:        &Text{},
			Start:       curCtx.P,
			End:         curCtx.P,
			Parent:      curCtx.Parent,
			LeftSibling: curCtx.LeftSibling,
		}
	} else {
		textnode = ctx.ParseText(s[i:end], curCtx)
	}
	if textnode == nil {
		log.Panicf("Failed to parse quote: %s", s)
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseHorizontalRule(s string, ctx ParseContext) *AstNode {
	if len(s) < 3 {
		return nil
	}
	symbol := rune(s[0])
	symbolCnt := 0
	pos := ctx.P
	if symbol != '*' && symbol != '-' {
		return nil
	}
	for _, c := range s {
		pos.Consume(c)
		if c == '\n' {
			break
		} else if c != symbol {
			return nil
		} else {
			symbolCnt += 1
		}
	}
	if symbolCnt < 3 {
		return nil
	}
	node := &AstNode{
		Type:        &HorizontalRule{},
		Start:       ctx.P,
		End:         pos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseList(s string, ctx ParseContext) *AstNode {
	if len(s) == 0 {
		return nil
	}

	fParseListLine := func(s string, ctx ParseContext) *AstNode {
		if len(s) == 0 {
			return nil
		}
		itemType := ListItem{}
		node := &AstNode{
			Type:        &ListItem{},
			Start:       ctx.P,
			End:         ctx.P,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		start := 1
		taskPrefixLen := len("- [ ]")
		if len(s) >= taskPrefixLen && s[:taskPrefixLen] == "- [ ]" {
			itemType = ListItem{IsTask: true, IsFinished: false}
			start = taskPrefixLen
		} else if len(s) >= taskPrefixLen && s[:taskPrefixLen] == "- [x]" {
			itemType = ListItem{IsTask: true, IsFinished: true}
			start = taskPrefixLen
		} else if s[0] != '-' {
			dotPos := _findInLine(s, ".")
			if dotPos <= 0 {
				return nil
			}
			order, err := strconv.Atoi(s[:dotPos])
			if err != nil {
				return nil
			}
			start = dotPos + 1
			itemType = ListItem{IsOrdered: true, Order: uint32(order)}
		}
		node.End.ConsumeStr(s[:start])
		contentStart := node.End
		node.Type = &itemType

		lineBrk := -1
		text := ""
		if start < len(s) && s[start] != ' ' && s[start] != '\n' {
			return nil
		}
		if start < len(s) {
			lineBrk = strings.Index(s[start:], "\n")
			if lineBrk < 0 {
				text = s[start:]
				node.End.ConsumeStr(s[start:])
			} else {
				text = s[start : lineBrk+start]
				node.End.ConsumeStr(s[start : lineBrk+start+1])
			}
		}
		curCtx := ctx
		curCtx.P = contentStart
		curCtx.Parent = node
		curCtx.LeftSibling = nil
		textnode := _textOrEmpty(text, curCtx)
		node.Children = append(node.Children, textnode)
		return node
	}

	listType := List{}
	listnode := &AstNode{
		Type:        &List{},
		Start:       ctx.P,
		End:         ctx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ctx
	curCtx.Parent = listnode
	curCtx.LeftSibling = nil

	fstListItem := fParseListLine(s, curCtx)
	if fstListItem == nil {
		return nil
	}
	curOrder := fstListItem.Type.(*ListItem).Order + 1
	listType.IsOrdered = fstListItem.Type.(*ListItem).IsOrdered
	listType.IsTask = fstListItem.Type.(*ListItem).IsFinished
	listnode.Type = &listType
	listnode.Children = append(listnode.Children, fstListItem)
	curCtx.LeftSibling = fstListItem
	curCtx.P = fstListItem.End

	curIdx := curCtx.P.Offset - ctx.P.Offset
	for curIdx < len(s) {
		lstItem := fParseListLine(s[curIdx:], curCtx)

		if lstItem == nil {
			break
		} else if lstItem.Type.(*ListItem).IsOrdered != listType.IsOrdered {
			break
		} else if lstItem.Type.(*ListItem).IsTask != listType.IsTask {
			break
		} else {
			lstItemType := lstItem.Type.(*ListItem)
			lstItemType.Order = curOrder
			lstItem.Type = lstItemType

			curCtx.P = lstItem.End
			curCtx.LeftSibling = lstItem
			curIdx = curCtx.P.Offset - ctx.P.Offset
			curOrder += 1
			listnode.Children = append(listnode.Children, lstItem)
		}
	}
	listnode.End = curCtx.P
	return listnode
}

/* Inline parsers */
func parseEmphasis(s string, ctx ParseContext) *AstNode {
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
	curCtx.P.Consume(rune(symbol))
	curCtx.P.Consume(rune(symbol))
	for _, c := range s[2:] {
		curCtx.P.Consume(c)
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
		Type:        &Emphasis{},
		Start:       ctx.P,
		End:         curCtx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseItalic(s string, ctx ParseContext) *AstNode {
	if len(s) <= 2 {
		return nil
	}
	symbol := s[0]
	if symbol != '*' && symbol != '_' {
		return nil
	}
	foundEnd := false
	curCtx := ctx
	curCtx.P.Consume(rune(symbol))
	for _, c := range s[1:] {
		curCtx.P.Consume(c)
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
	if curCtx.P.Offset <= ctx.P.Offset+2 {
		// forbid empty italic
		return nil
	}
	node := &AstNode{
		Type:        &Italic{},
		Start:       ctx.P,
		End:         curCtx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseStrikeThrough(s string, ctx ParseContext) *AstNode {
	if len(s) < 4 {
		return nil
	}
	if s[:2] != "~~" {
		return nil
	}

	lastHit := false
	end := -1
	for i, c := range s[2:] {
		if c == '\n' {
			return nil
		} else if c == '~' {
			if lastHit {
				end = i + 2 // don't omit the ~~ at the beginning
				break
			}
			lastHit = true
		} else {
			lastHit = false
		}
	}
	if end < 0 {
		return nil
	}
	endPos := ctx.P
	endPos.ConsumeStr(s[:end+1])
	node := &AstNode{
		Type:        &StrikeThrough{},
		Start:       ctx.P,
		End:         endPos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ctx
	curCtx.P.ConsumeStr("~~")
	curCtx.Parent = node
	curCtx.LeftSibling = nil
	textnode := ctx.ParseText(s[2:end-1], curCtx)
	if textnode == nil {
		log.Panicf("Failed to parse text: %s", s[2:end-1])
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseCode(s string, ctx ParseContext) *AstNode {
	if len(s) < 2 {
		return nil
	}
	curCtx := ctx
	foundEnd := false
	inSeq := 0
	leadingBackticks := 0
	newS := s

	for len(newS) > 0 {
		if newS[0] == '`' {
			curCtx.P.Consume(rune('`'))
			leadingBackticks += 1
			newS = newS[1:]
		} else {
			break
		}
	}

	fRightNotBacktick := func(i int) bool {
		return i+1 >= len(newS) || !utf8.RuneStart(newS[i+1]) || newS[i+1] != '`'
	}
	for i, c := range newS {
		curCtx.P.Consume(c)
		if c == '`' {
			inSeq += 1
			if inSeq == leadingBackticks && fRightNotBacktick(i) {
				foundEnd = true
				break
			}
		} else {
			inSeq = 0
		}
	}
	if !foundEnd {
		return nil
	}
	node := &AstNode{
		Type:        &Code{},
		Start:       ctx.P,
		End:         curCtx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseMath(s string, ctx ParseContext) *AstNode {
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
	curCtx.P.Consume(rune('$'))
	foundEnd := false
	for i, c := range newS {
		curCtx.P.Consume(c)
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
		Type:        &Math{},
		Start:       ctx.P,
		End:         curCtx.P,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

// url "title" | <url> "title", input should contain no '\n'
func _parseLinkTitle(s string) (bool, string, string) {
	var link, title string
	linkStart, linkEnd := -1, -1
	for i, c := range s {
		if linkStart < 0 {
			if c != ' ' {
				linkStart = i
			}
		} else {
			if c == ' ' {
				break
			}
			linkEnd = i + 1 // to include the current character
		}
	}
	if linkStart < 0 || linkEnd < 0 {
		return false, "", ""
	}
	link = s[linkStart:linkEnd]
	titleStart, titleEnd := -1, -1
	for i, c := range s[linkEnd:] {
		if titleStart < 0 {
			if c == '"' {
				titleStart = i + linkEnd + 1
			} else if c != ' ' {
				return false, "", ""
			}
		} else if titleEnd < 0 {
			if c == '"' {
				titleEnd = i + linkEnd
			}
		} else if c != ' ' {
			return false, "", ""
		}
	}
	if titleStart >= 0 && titleEnd <= 0 {
		return false, "", ""
	}
	if titleStart >= 0 && titleEnd >= 0 {
		title = s[titleStart:titleEnd]
	}
	return true, link, title
}

func _parseLinkLike(s string, pos Pos) (bool, string, string, string, Pos) {
	if len(s) == 0 || s[0] != '[' {
		return false, "", "", "", Pos{}
	}
	curPos := pos
	newLineIdx := strings.Index(s, "\n")
	rightIdx := -1
	leftBracketMet := 0
	lastEscape := false
	for i, c := range s[1:] {
		if newLineIdx >= 0 && i >= newLineIdx {
			rightIdx = -1
			break
		}
		if !lastEscape && c == '[' {
			leftBracketMet += 1
		}
		if !lastEscape && c == ']' {
			if leftBracketMet == 0 {
				rightIdx = i + 1 // don't omit the [ at the beginning
				break
			}
			leftBracketMet -= 1
		}
		if c == '\\' {
			lastEscape = true
		} else {
			lastEscape = false
		}
	}
	if rightIdx < 0 || (newLineIdx >= 0 && newLineIdx < rightIdx) {
		return false, "", "", "", Pos{}
	}
	name := s[1:rightIdx]
	curPos.ConsumeStr(s[:rightIdx+1])

	// focus on the (<url> "title"?) part
	newS := s[rightIdx+1:]
	if len(newS) < 2 || newS[0] != '(' {
		return false, "", "", "", Pos{}
	}
	newLineIdx = strings.Index(newS, "\n")

	rightIdx = _findInLine(newS, ")")
	if rightIdx < 0 || (newLineIdx >= 0 && newLineIdx < rightIdx) {
		return false, "", "", "", Pos{}
	}

	ok, link, title := _parseLinkTitle(newS[1:rightIdx])
	if !ok {
		return false, "", "", "", Pos{}
	}

	curPos.ConsumeStr(s[:rightIdx+1])
	return true, name, link, title, curPos
}

func parseLink(s string, ctx ParseContext) *AstNode {
	ret, name, link, title, pos := _parseLinkLike(s, ctx.P)
	if ret {
		node := &AstNode{
			Type:        &Link{Link: link, Title: title},
			Start:       ctx.P,
			End:         pos,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		curCtx := ctx
		curCtx.LeftSibling = nil
		curCtx.Parent = node
		curCtx.P.Consume('[')
		textnode := ctx.ParseText(name, curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse link %s", s)
		}
		node.Children = append(node.Children, textnode)
		return node
	} else {
		return nil
	}
}

func parseSimpleLink(s string, ctx ParseContext) *AstNode {
	if len(s) <= 2 || s[0] != '<' {
		return nil
	}
	rightIdx := _findInLine(s, ">")

	if rightIdx <= 0 {
		return nil
	}
	link := s[1:rightIdx]
	if _matchUrl(link) || _matchEmail(link) {
		endPos := ctx.P
		endPos.ConsumeStr(s[:rightIdx+1])
		node := &AstNode{
			Type:        &SimpleLink{Link: link},
			Start:       ctx.P,
			End:         endPos,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseReferenceLink(s string, ctx ParseContext) *AstNode {
	lbr1, rbr1, lbr2, rbr2 := -1, -1, -1, -1
	if len(s) <= 4 {
		return nil
	}
	if s[0] != '[' {
		return nil
	}
	lbr1 = 0
	rbr1 = _findInLine(s[1:], "]") + 1
	if rbr1 < 0 || rbr1+2 >= len(s) {
		return nil
	}
	lbr2 = rbr1 + 1
	if s[lbr2] == ' ' {
		lbr2 += 1
	}
	if s[lbr2] != '[' {
		return nil
	}
	rbr2 = _findInLine(s[lbr2:], "]") + lbr2
	if rbr2 < 0 {
		return nil
	}
	endPos := ctx.P
	endPos.ConsumeStr(s[:rbr2+1])
	node := &AstNode{
		Type:        &ReferenceLink{Index: s[lbr2+1 : rbr2]},
		Start:       ctx.P,
		End:         endPos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	curCtx := ctx
	curCtx.LeftSibling = nil
	curCtx.Parent = node
	curCtx.P.Consume('[')
	text := s[lbr1+1 : rbr1]
	if len(text) == 0 {
		return nil
	}
	textnode := ctx.ParseText(text, curCtx)
	node.Children = append(node.Children, textnode)
	return node
}

func parseReferenceLinkIndex(s string, ctx ParseContext) *AstNode {
	if len(s) <= 3 || s[0] != '[' {
		return nil
	}
	lbr := 0
	rbr := _findInLine(s, "]")
	if rbr < 0 || rbr+1 >= len(s) || s[rbr+1] != ':' {
		return nil
	}
	indexType := ReferenceLinkIndex{Index: s[lbr+1 : rbr]}
	if rbr+2 >= len(s) {
		return nil
	}
	pos := ctx.P
	newLineIndex := strings.Index(s, "\n")
	if newLineIndex < 0 {
		pos.ConsumeStr(s)
		newLineIndex = len(s)
	} else {
		pos.ConsumeStr(s[:newLineIndex+1])
	}

	ok, link, title := _parseLinkTitle(s[rbr+2 : newLineIndex])
	if !ok {
		return nil
	}
	indexType.Link = link
	indexType.Title = title

	node := &AstNode{
		Type:        &indexType,
		Start:       ctx.P,
		End:         pos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseFootNote(s string, ctx ParseContext) *AstNode {
	if len(s) <= 3 || s[0] != '[' || s[1] != '^' {
		return nil
	}
	rbr := _findInLine(s, "]")
	if rbr <= 2 {
		return nil
	}
	index := s[2:rbr]
	endPos := ctx.P
	endPos.ConsumeStr(s[:rbr+1])
	node := &AstNode{
		Type:        &FootNote{Index: index},
		Start:       ctx.P,
		End:         endPos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	return node
}

func parseFootNoteIndex(s string, ctx ParseContext) *AstNode {
	if len(s) <= 4 || s[0] != '[' || s[1] != '^' {
		return nil
	}
	rbr := _findInLine(s, "]")
	if rbr <= 2 {
		return nil
	}
	endPos := ctx.P
	index := s[2:rbr]

	if rbr+1 >= len(s) || s[rbr+1] != ':' {
		return nil
	}

	endLine := strings.Index(s, "\n")
	if endLine < 0 {
		endLine = len(s)
		endPos.ConsumeStr(s)
	} else {
		endPos.ConsumeStr(s[:endLine+1])
	}
	node := &AstNode{
		Type:        &FootNoteIndex{Index: index},
		Start:       ctx.P,
		End:         endPos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	textStart := ctx.P
	textStart.ConsumeStr(s[:rbr+2])
	curCtx := ctx
	curCtx.P = textStart
	curCtx.Parent = node
	curCtx.LeftSibling = nil
	var textnode *AstNode
	if rbr+2 < endLine {
		textnode = ctx.ParseText(s[rbr+2:endLine], curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse text %s", s[:endLine])
		}
	} else {
		textnode = &AstNode{
			Type:        &Text{},
			Start:       textStart,
			End:         textStart,
			Parent:      node,
			LeftSibling: nil,
		}
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseImage(s string, ctx ParseContext) *AstNode {
	if len(s) < 1 && s[0] != '!' {
		return nil
	}
	ret, name, link, title, pos := _parseLinkLike(s[1:], ctx.P)
	if ret {
		node := &AstNode{
			Type:        &Image{Link: link, Title: title},
			Start:       ctx.P,
			End:         pos,
			Parent:      ctx.Parent,
			LeftSibling: ctx.LeftSibling,
		}
		curCtx := ctx
		curCtx.LeftSibling = nil
		curCtx.Parent = node
		curCtx.P.ConsumeStr("![")
		textnode := ctx.ParseText(name, curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse link %s", s)
		}
		node.Children = append(node.Children, textnode)
		return node
	} else {
		return nil
	}
}

func parseHtml(s string, ctx ParseContext) *AstNode {
	if len(s) < 2 || s[0] != '<' {
		return nil
	}
	start := 1
	isEnd := false
	if s[1] == '/' {
		start = 2
		isEnd = true
	}
	tagEnd := _findInLine(s, ">")
	if tagEnd < 0 {
		return nil
	}
	insideTag := s[start:tagEnd]
	if len(insideTag) == 0 {
		return nil
	}
	var tag, content string
	whitespace := _findInLine(insideTag, " ")
	if whitespace < 0 {
		tag = insideTag
	} else {
		tag = insideTag[:whitespace]
		content = insideTag[whitespace+1:]
	}
	if len(tag) == 0 {
		return nil
	}
	pos := ctx.P
	pos.ConsumeStr(s[:tagEnd+1])
	node := &AstNode{
		Type:        &HtmlStartTag{Tag: tag, Content: content},
		Start:       ctx.P,
		End:         pos,
		Parent:      ctx.Parent,
		LeftSibling: ctx.LeftSibling,
	}
	if isEnd {
		if len(content) > 0 {
			return nil
		}
		node.Type = &HtmlEndTag{Tag: tag}
	}
	return node
}
