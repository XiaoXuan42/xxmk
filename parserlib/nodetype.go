package parserlib

import (
	"fmt"
	"reflect"
	"sync"
)

type AstNodeType interface {
	String() string
}

type Document struct{}

func (doc Document) String() string {
	return "Document"
}

type Text struct{}

func (text Text) String() string {
	return "Text"
}

type Header struct {
	Level uint32
}

func (header Header) String() string {
	return fmt.Sprintf("Header(%d)", header.Level)
}

type MathBlock struct{}

func (math MathBlock) String() string {
	return "MathBlock"
}

type CodeBlock struct {
	Suffix string
}

func (code CodeBlock) String() string {
	return fmt.Sprintf("CodeBlock(%s)", code.Suffix)
}

type HorizontalRule struct{}

func (rule HorizontalRule) String() string {
	return "HorizontalRule"
}

/* Table */
const (
	AlignLeft uint32 = iota
	AlignMiddle
	AlignRight
)

type TableHead struct{}
type TableAlign struct {
	Aligns []uint32
}
type TableLine struct{}
type Table struct{}

func (head TableHead) String() string {
	return "TableHead"
}

func (align TableAlign) String() string {
	return "TableAlign"
}

func (line TableLine) String() string {
	return "TableLine"
}

func (table Table) String() string {
	return "Table"
}

/* end Table */

type QuoteBlock struct {
	Level uint32
}

func (quote QuoteBlock) String() string {
	return "QuoteBlock"
}

/* List */
type List struct {
	IsOrdered bool
	IsTask    bool
}

func (list List) String() string {
	return "List"
}

type ListItem struct {
	IsOrdered  bool
	IsTask     bool
	IsFinished bool
	Order      uint32
}

func (item ListItem) String() string {
	if item.IsTask {
		s := "-"
		if item.IsFinished {
			s = "x"
		}
		return fmt.Sprintf("ListItem(task: %s)", s)
	} else if item.IsOrdered {
		return fmt.Sprintf("ListItem(%d)", item.Order)
	} else {
		return "ListItem"
	}
}

/* end List */

/****** inline ast nodes ******/
type Emphasis struct{}

func (text Emphasis) String() string {
	return "Emphasis"
}

type Italic struct{}

func (text Italic) String() string {
	return "Italic"
}

type StrikeThrough struct{}

func (text StrikeThrough) String() string {
	return "StrikeThrough"
}

type Code struct{}

func (text Code) String() string {
	return "Code"
}

type Math struct{}

func (text Math) String() string {
	return "Math"
}

/* Link */
// [title](link)
type Link struct {
	Link  string
	Title string
}

func (link Link) String() string {
	return "Link"
}

// <link> link is url or email address
type SimpleLink struct {
	Link string
}

func (link SimpleLink) String() string {
	return "SimpleLink"
}

type ReferenceLink struct {
	Index string
}

func (link ReferenceLink) String() string {
	return fmt.Sprintf("ReferenceLink(%s)", link.Index)
}

type ReferenceLinkIndex struct {
	Index string
	Link  string
	Title string
}

func (link ReferenceLinkIndex) String() string {
	return fmt.Sprintf("ReferenceLinkIndex(%s)", link.Index)
}

type FootNote struct {
	Index string
}

func (footnote FootNote) String() string {
	return fmt.Sprintf("FootNote(%s)", footnote.Index)
}

type FootNoteIndex struct {
	Index string
}

func (footnote FootNoteIndex) String() string {
	return fmt.Sprintf("FootNoteIndex(%s)", footnote.Index)
}

/* end link */

type Image struct {
	Link  string
	Title string
}

func (image Image) String() string {
	return "Image"
}

type HtmlStartTag struct {
	Tag     string
	Content string
}

func (html HtmlStartTag) String() string {
	return fmt.Sprintf("HtmlStartTag(%s)", html.Tag)
}

type HtmlEndTag struct {
	Tag string
}

func (html HtmlEndTag) String() string {
	return fmt.Sprintf("HtmlEndTag(%s)", html.Tag)
}

var str2NodeType = map[string]AstNodeType{
	"Document":           &Document{},
	"Text":               &Text{},
	"Header":             &Header{},
	"MathBlock":          &MathBlock{},
	"CodeBlock":          &CodeBlock{},
	"HorizontalRule":     &HorizontalRule{},
	"TableHead":          &TableHead{},
	"TableAlign":         &TableAlign{},
	"TableLine":          &TableLine{},
	"Table":              &Table{},
	"QuoteBlock":         &QuoteBlock{},
	"List":               &List{},
	"ListItem":           &ListItem{},
	"Emphasis":           &Emphasis{},
	"Italic":             &Italic{},
	"StrikeThrough":      &StrikeThrough{},
	"Code":               &Code{},
	"Math":               &Math{},
	"Link":               &Link{},
	"SimpleLink":         &SimpleLink{},
	"ReferenceLink":      &ReferenceLink{},
	"ReferenceLinkIndex": &ReferenceLinkIndex{},
	"FootNote":           &FootNote{},
	"FootNoteIndex":      &FootNoteIndex{},
	"Image":              &Image{},
	"HtmlStartTag":       &HtmlStartTag{},
	"HtmlEndTag":         &HtmlEndTag{},
}

var str2NodeID = map[string]int{
	"Document":           1,
	"Text":               2,
	"Header":             3,
	"MathBlock":          4,
	"CodeBlock":          5,
	"HorizontalRule":     6,
	"TableHead":          7,
	"TableAlign":         8,
	"TableLine":          9,
	"Table":              10,
	"QuoteBlock":         11,
	"List":               12,
	"ListItem":           13,
	"Emphasis":           14,
	"Italic":             15,
	"StrikeThrough":      16,
	"Code":               17,
	"Math":               18,
	"Link":               19,
	"SimpleLink":         20,
	"ReferenceLink":      21,
	"ReferenceLinkIndex": 22,
	"FootNote":           23,
	"FootNoteIndex":      24,
	"Image":              25,
	"HtmlStartTag":       26,
	"HtmlEndTag":         27,
}
var str2NodeIDLock sync.RWMutex

func GetNodeTypeName(nodeTp AstNodeType) string {
	return reflect.TypeOf(nodeTp).Elem().Name()
}

func GetDefaultTypePtr(key string) AstNodeType {
	res, ok := str2NodeType[key]
	if ok {
		return res
	} else {
		return nil
	}
}

func GetNodeTypeIdFromStr(key string) int {
	str2NodeIDLock.RLock()

	val, ok := str2NodeID[key]
	if !ok {
		str2NodeIDLock.RUnlock()
		str2NodeIDLock.Lock()
		defer str2NodeIDLock.Unlock()
		val, ok := str2NodeID[key]
		if ok {
			return val
		}
		id := len(str2NodeID) + 1
		str2NodeID[key] = id
		return id
	} else {
		str2NodeIDLock.RUnlock()
		return val
	}
}

func GetNodeTypeId(tp AstNodeType) int {
	tpName := GetNodeTypeName(tp)
	return GetNodeTypeIdFromStr(tpName)
}
