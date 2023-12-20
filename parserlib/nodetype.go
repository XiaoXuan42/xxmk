package parserlib

import (
	"fmt"
	"reflect"

	"github.com/XiaoXuan42/xxmk/naivesel"
)

type AstNodeType interface {
	String() string
}


func _setWhich(s string, buf *AstNodeTypeProto) {
	buf.Which = AstNodeTypeEnumProto(AstNodeTypeEnumProto_value[s])
}

func _getAstNodeType(buf *AstNodeTypeProto) AstNodeType {
	str := AstNodeTypeEnumProto_name[int32(buf.Which)]
	intf := reflect.New(reflect.TypeOf(str2NodeType[str]).Elem()).Interface()
	return intf.(AstNodeType)
}

func _astNodeTypeToProtobuf(tp AstNodeType) *AstNodeTypeProto {
	typeProto := &AstNodeTypeProto{}
	tpName := reflect.TypeOf(tp).Elem().Name()
	_setWhich(tpName, typeProto)
	typeProto.Encode = append(typeProto.Encode, naivesel.Serialize(tp)...)
	return typeProto
}

func _astNodeTypeFromProtobuf(buf *AstNodeTypeProto) AstNodeType {
	nodeTp := _getAstNodeType(buf)
	naivesel.Deserialize(nodeTp, buf.Encode)
	return nodeTp
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
