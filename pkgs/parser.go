package xxmk

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
type InlineParser func(string, parseContext) *AstNode
type BlockParser func(string, parseContext) *AstNode
type parseContext struct {
	p           Pos
	parent      *AstNode
	leftSibling *AstNode
	parseText   func(string, parseContext) *AstNode
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

func _textOrEmpty(text string, ctx parseContext) *AstNode {
	if len(text) == 0 {
		return &AstNode{
			Type:        Text{},
			Start:       ctx.p,
			End:         ctx.p,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
	} else {
		textnode := ctx.parseText(text, ctx)
		if textnode == nil {
			log.Panicf("Failed to parse text: %s", text)
		}
		return textnode
	}
}

/* Block parsers */
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
	if i < len(s) && s[i] != ' ' && s[i] != '\n' {
		return nil
	}
	j := strings.Index(s, "\n")
	var text string
	endPos := ctx.p
	if j < 0 {
		text = s[i:]
		endPos.ConsumeStr(s[i:])
	} else {
		text = s[i:j]
		endPos.ConsumeStr(s[i:j+1])
	}
	textnode := _textOrEmpty(text, ctx)
	node.Children = append(node.Children, textnode)

	node.Type = head
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
		for cur < len(s) {
			curSep := _findInLine(s[cur:], "|")
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

func parseQuoteBlock(s string, ctx parseContext) *AstNode {
	if len(s) < 1 || s[0] != '>' {
		return nil
	}
	blkType := QuoteBlock{}
	node := &AstNode{
		Type:        QuoteBlock{},
		Start:       ctx.p,
		End:         ctx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := ctx
	curCtx.parent = node
	curCtx.leftSibling = nil
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
		curCtx.p.Consume('>')
		blkType.Level += 1
		i += 1
	}
	node.Type = blkType
	for i < len(s) {
		if s[i] != ' ' {
			break
		}
		i += 1
		curCtx.p.Consume(' ')
	}
	var textnode *AstNode
	if i >= end {
		textnode = &AstNode{
			Type:        Text{},
			Start:       curCtx.p,
			End:         curCtx.p,
			Parent:      curCtx.parent,
			LeftSibling: curCtx.leftSibling,
		}
	} else {
		textnode = ctx.parseText(s[i:end], curCtx)
	}
	if textnode == nil {
		log.Panicf("Failed to parse quote: %s", s)
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseHorizontalRule(s string, ctx parseContext) *AstNode {
	if len(s) < 3 {
		return nil
	}
	symbol := rune(s[0])
	symbolCnt := 0
	pos := ctx.p
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
		Type:        HorizontalRule{},
		Start:       ctx.p,
		End:         pos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseList(s string, ctx parseContext) *AstNode {
	if len(s) == 0 {
		return nil
	}

	fParseListLine := func(s string, ctx parseContext) *AstNode {
		if len(s) == 0 {
			return nil
		}
		itemType := ListItem{}
		node := &AstNode{
			Type:        ListItem{},
			Start:       ctx.p,
			End:         ctx.p,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
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
			itemType = ListItem{IsOrdered: true, Order: order}
		}
		node.End.ConsumeStr(s[:start])
		contentStart := node.End
		node.Type = itemType

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
		curCtx.p = contentStart
		curCtx.parent = node
		curCtx.leftSibling = nil
		textnode := _textOrEmpty(text, curCtx)
		node.Children = append(node.Children, textnode)
		return node
	}

	listType := List{}
	listnode := &AstNode{
		Type:        List{},
		Start:       ctx.p,
		End:         ctx.p,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := ctx
	curCtx.parent = listnode
	curCtx.leftSibling = nil

	fstListItem := fParseListLine(s, curCtx)
	if fstListItem == nil {
		return nil
	}
	curOrder := fstListItem.Type.(ListItem).Order + 1
	listType.IsOrdered = fstListItem.Type.(ListItem).IsOrdered
	listType.IsTask = fstListItem.Type.(ListItem).IsFinished
	listnode.Type = listType
	listnode.Children = append(listnode.Children, fstListItem)
	curCtx.leftSibling = fstListItem
	curCtx.p = fstListItem.End

	curIdx := curCtx.p.Offset - ctx.p.Offset
	for curIdx < len(s) {
		lstItem := fParseListLine(s[curIdx:], curCtx)

		if lstItem == nil {
			break
		} else if lstItem.Type.(ListItem).IsOrdered != listType.IsOrdered {
			break
		} else if lstItem.Type.(ListItem).IsTask != listType.IsTask {
			break
		} else {
			lstItemType := lstItem.Type.(ListItem)
			lstItemType.Order = curOrder
			lstItem.Type = lstItemType

			curCtx.p = lstItem.End
			curCtx.leftSibling = lstItem
			curIdx = curCtx.p.Offset - ctx.p.Offset
			curOrder += 1
			listnode.Children = append(listnode.Children, lstItem)
		}
	}
	listnode.End = curCtx.p
	return listnode
}

/* Inline parsers */
func parseEmphasis(s string, ctx parseContext) *AstNode {
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
		Type:        Emphasis{},
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

func parseStrikeThrough(s string, ctx parseContext) *AstNode {
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
	endPos := ctx.p
	endPos.ConsumeStr(s[:end+1])
	node := &AstNode{
		Type:        StrikeThrough{},
		Start:       ctx.p,
		End:         endPos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := ctx
	curCtx.p.ConsumeStr("~~")
	curCtx.parent = node
	curCtx.leftSibling = nil
	textnode := ctx.parseText(s[2:end-1], curCtx)
	if textnode == nil {
		log.Panicf("Failed to parse text: %s", s[2:end-1])
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseCode(s string, ctx parseContext) *AstNode {
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
			curCtx.p.Consume(rune('`'))
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
		curCtx.p.Consume(c)
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

func parseLink(s string, ctx parseContext) *AstNode {
	ret, name, link, title, pos := _parseLinkLike(s, ctx.p)
	if ret {
		node := &AstNode{
			Type:        Link{Link: link, Title: title},
			Start:       ctx.p,
			End:         pos,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
		curCtx := ctx
		curCtx.leftSibling = nil
		curCtx.parent = node
		curCtx.p.Consume('[')
		textnode := ctx.parseText(name, curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse link %s", s)
		}
		node.Children = append(node.Children, textnode)
		return node
	} else {
		return nil
	}
}

func parseSimpleLink(s string, ctx parseContext) *AstNode {
	if len(s) <= 2 || s[0] != '<' {
		return nil
	}
	rightIdx := _findInLine(s, ">")

	if rightIdx <= 0 {
		return nil
	}
	link := s[1:rightIdx]
	if _matchUrl(link) || _matchEmail(link) {
		endPos := ctx.p
		endPos.ConsumeStr(s[:rightIdx+1])
		node := &AstNode{
			Type:        SimpleLink{Link: link},
			Start:       ctx.p,
			End:         endPos,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
		return node
	} else {
		return nil
	}
}

func parseReferenceLink(s string, ctx parseContext) *AstNode {
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
	endPos := ctx.p
	endPos.ConsumeStr(s[:rbr2+1])
	node := &AstNode{
		Type:        ReferenceLink{Index: s[lbr2+1 : rbr2]},
		Start:       ctx.p,
		End:         endPos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	curCtx := ctx
	curCtx.leftSibling = nil
	curCtx.parent = node
	curCtx.p.Consume('[')
	text := s[lbr1+1 : rbr1]
	if len(text) == 0 {
		return nil
	}
	textnode := ctx.parseText(text, curCtx)
	node.Children = append(node.Children, textnode)
	return node
}

func parseReferenceLinkIndex(s string, ctx parseContext) *AstNode {
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
	pos := ctx.p
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
		Type:        indexType,
		Start:       ctx.p,
		End:         pos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseFootNote(s string, ctx parseContext) *AstNode {
	if len(s) <= 3 || s[0] != '[' || s[1] != '^' {
		return nil
	}
	rbr := _findInLine(s, "]")
	if rbr <= 2 {
		return nil
	}
	index := s[2:rbr]
	endPos := ctx.p
	endPos.ConsumeStr(s[:rbr+1])
	node := &AstNode{
		Type:        FootNote{Index: index},
		Start:       ctx.p,
		End:         endPos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	return node
}

func parseFootNoteIndex(s string, ctx parseContext) *AstNode {
	if len(s) <= 4 || s[0] != '[' || s[1] != '^' {
		return nil
	}
	rbr := _findInLine(s, "]")
	if rbr <= 2 {
		return nil
	}
	endPos := ctx.p
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
		Type:        FootNoteIndex{Index: index},
		Start:       ctx.p,
		End:         endPos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	textStart := ctx.p
	textStart.ConsumeStr(s[:rbr+2])
	curCtx := ctx
	curCtx.p = textStart
	curCtx.parent = node
	curCtx.leftSibling = nil
	var textnode *AstNode
	if rbr+2 < endLine {
		textnode = ctx.parseText(s[rbr+2:endLine], curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse text %s", s[:endLine])
		}
	} else {
		textnode = &AstNode{
			Type:        Text{},
			Start:       textStart,
			End:         textStart,
			Parent:      node,
			LeftSibling: nil,
		}
	}
	node.Children = append(node.Children, textnode)
	return node
}

func parseImage(s string, ctx parseContext) *AstNode {
	if len(s) < 1 && s[0] != '!' {
		return nil
	}
	ret, name, link, title, pos := _parseLinkLike(s[1:], ctx.p)
	if ret {
		node := &AstNode{
			Type:        Image{Link: link, Title: title},
			Start:       ctx.p,
			End:         pos,
			Parent:      ctx.parent,
			LeftSibling: ctx.leftSibling,
		}
		curCtx := ctx
		curCtx.leftSibling = nil
		curCtx.parent = node
		curCtx.p.ConsumeStr("![")
		textnode := ctx.parseText(name, curCtx)
		if textnode == nil {
			log.Panicf("Failed to parse link %s", s)
		}
		node.Children = append(node.Children, textnode)
		return node
	} else {
		return nil
	}
}

func parseHtml(s string, ctx parseContext) *AstNode {
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
	pos := ctx.p
	pos.ConsumeStr(s[:tagEnd+1])
	node := &AstNode{
		Type:        HtmlStartTag{Tag: tag, Content: content},
		Start:       ctx.p,
		End:         pos,
		Parent:      ctx.parent,
		LeftSibling: ctx.leftSibling,
	}
	if isEnd {
		if len(content) > 0 {
			return nil
		}
		node.Type = HtmlEndTag{Tag: tag}
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

func (parser *MKParser) addDefaultInlineParsers(names []string) {
	// add in reverse order
	for i := len(names) - 1; i >= 0; i-- {
		parser.addDefaultInlineParser(names[i])
	}
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

func (parser *MKParser) addDefaultBlockParsers(names []string) {
	for i := len(names) - 1; i >= 0; i-- {
		parser.addDefaultBlockParser(names[i])
	}
}

func GetHtmlMKParser() MKParser {
	// Emphasis before Italic
	// FootNote before ReferenceLink
	parser := MKParser{}
	parser.addDefaultBlockParsers([]string{
		"HorizontalRule", "Header", "QuoteBlock", "CodeBlock", "MathBlock", "Table", "List", "FootNoteIndex", "ReferenceLinkIndex",
	})
	parser.InlineParserSeq = make(map[rune][]InlineParser)
	parser.addDefaultInlineParsers([]string{
		"Emphasis", "Italic", "StrikeThrough", "Code", "Math", "Link", "SimpleLink", "Image", "Html", "FootNote", "ReferenceLink",
	})
	return parser
}
