package parser

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type AstNodeType interface {
	String() string
	GetValueList() []interface{}
}

func _encodeUint32(vs ...uint32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[i*4:i*4+4], v)
	}
	return b
}

func _decodeUint32(bytes []byte) (uint32, int) {
	return binary.LittleEndian.Uint32(bytes), 4
}

func _encodeBools(bs ...bool) []byte {
	var result []byte
	for _, b := range bs {
		if b {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	}
	return result
}

func _decodeBools(bytes []byte) (bool, int) {
	if bytes[0] != 0 {
		return true, 1
	}
	return false, 1
}

func _encodeStrs(strs ...string) []byte {
	var result []byte
	for _, s := range strs {
		result = append(result, _encodeUint32(uint32(len(s)))...)
		result = append(result, []byte(s)...)
	}
	return result
}

func _decodeStrs(bytes []byte) (string, int) {
	len, offset := _decodeUint32(bytes)
	totOffset := offset + int(len)
	s := string(bytes[offset:totOffset])
	return s, totOffset
}

func _encodeUint32Slice(slice []uint32) []byte {
	var result []byte
	result = append(result, _encodeUint32(uint32(len(slice)))...)
	for _, u := range slice {
		result = append(result, _encodeUint32(u)...)
	}
	return result
}

func _decodeUint32Slice(bytes []byte) ([]uint32, int) {
	len, offset := _decodeUint32(bytes)
	var result []uint32
	for i := 0; i < int(len); i++ {
		v, curOff := _decodeUint32(bytes[offset:])
		offset += curOff
		result = append(result, v)
	}
	return result, offset
}

func _setWhich(s string, buf *AstNodeTypeProto) {
	buf.Which = AstNodeTypeEnumProto(AstNodeTypeEnumProto_value[s])
}

func _valueListToBytes(valList []interface{}) []byte {
	var result []byte
	for _, v := range valList {
		switch val := v.(type) {
		case *int:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *uint:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *int32:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *uint32:
			result = append(result, _encodeUint32(*val)...)
		case *string:
			result = append(result, _encodeStrs(*val)...)
		case *bool:
			result = append(result, _encodeBools(*val)...)
		case *[]uint32:
			result = append(result, _encodeUint32Slice(*val)...)
		default:
			panic("Unknown type to encode")
		}
	}
	return result
}

func _valueListFromBytes(valList []interface{}, bytes []byte) {
	rear := 0
	for _, v := range valList {
		switch val := v.(type) {
		case *int:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = int(dv)
			rear += offset
		case *uint:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = uint(dv)
			rear += offset
		case *int32:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = int32(dv)
			rear += offset
		case *uint32:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = uint32(dv)
			rear += offset
		case *string:
			dv, offset := _decodeStrs(bytes[rear:])
			*val = dv
			rear += offset
		case *bool:
			dv, offset := _decodeBools(bytes[rear:])
			*val = dv
			rear += offset
		case *[]uint32:
			dv, offset := _decodeUint32Slice(bytes[rear:])
			*val = dv
			rear += offset
		default:
			panic("Unknown type to decode")
		}
	}
}

func _astNodeTypeToProtobuf(tp AstNodeType) *AstNodeTypeProto {
	typeProto := &AstNodeTypeProto{}
	tpName := reflect.TypeOf(tp).Elem().Name()
	_setWhich(tpName, typeProto)
	valList := tp.GetValueList()
	typeProto.Encode = append(typeProto.Encode, _valueListToBytes(valList)...)
	return typeProto
}

func _getAstNodeType(buf *AstNodeTypeProto) AstNodeType {
	str := AstNodeTypeEnumProto_name[int32(buf.Which)]
	intf := reflect.New(reflect.TypeOf(str2NodeType[str]).Elem()).Interface()
	return intf.(AstNodeType)
}

func _astNodeTypeFromProtobuf(buf *AstNodeTypeProto) AstNodeType {
	nodeTp := _getAstNodeType(buf)
	valList := nodeTp.GetValueList()
	_valueListFromBytes(valList, buf.Encode)
	return nodeTp
}

type Document struct{}

func (doc Document) String() string {
	return "Document"
}

func (doc *Document) GetValueList() []interface{} {
	return nil
}

type Text struct{}

func (text Text) String() string {
	return "Text"
}

func (text *Text) GetValueList() []interface{} {
	return nil
}

type Header struct {
	Level uint32
}

func (header Header) String() string {
	return fmt.Sprintf("Header(%d)", header.Level)
}

func (head *Header) GetValueList() []interface{} {
	return []interface{}{&head.Level}
}

type MathBlock struct{}

func (math MathBlock) String() string {
	return "MathBlock"
}

func (math *MathBlock) GetValueList() []interface{} {
	return nil
}

type CodeBlock struct {
	Suffix string
}

func (code CodeBlock) String() string {
	return fmt.Sprintf("CodeBlock(%s)", code.Suffix)
}

func (code *CodeBlock) GetValueList() []interface{} {
	return []interface{}{&code.Suffix}
}

type HorizontalRule struct{}

func (rule HorizontalRule) String() string {
	return "HorizontalRule"
}

func (rule *HorizontalRule) GetValueList() []interface{} {
	return nil
}

/* Table */
const (
	AlignLeft uint32 = iota
	AlignMiddle
	AlignRight
)

type TableHead struct{}
type TableAlign struct {
	aligns []uint32
}
type TableLine struct{}
type Table struct{}

func (head TableHead) String() string {
	return "TableHead"
}

func (head *TableHead) GetValueList() []interface{} {
	return nil
}

func (align TableAlign) String() string {
	return "TableAlign"
}

func (align *TableAlign) GetValueList() []interface{} {
	return []interface{}{&align.aligns}
}

func (line TableLine) String() string {
	return "TableLine"
}

func (line *TableLine) GetValueList() []interface{} {
	return nil
}

func (table Table) String() string {
	return "Table"
}

func (table *Table) GetValueList() []interface{} {
	return nil
}

/* end Table */

type QuoteBlock struct {
	Level uint32
}

func (quote QuoteBlock) String() string {
	return "QuoteBlock"
}

func (quote *QuoteBlock) GetValueList() []interface{} {
	return []interface{}{&quote.Level}
}

/* List */
type List struct {
	IsOrdered bool
	IsTask    bool
}

func (list List) String() string {
	return "List"
}

func (list *List) GetValueList() []interface{} {
	return []interface{}{&list.IsOrdered, &list.IsTask}
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

func (item *ListItem) GetValueList() []interface{} {
	return []interface{}{&item.IsOrdered, &item.IsTask, &item.IsFinished, &item.Order}
}

/* end List */

/****** inline ast nodes ******/
type Emphasis struct{}

func (text Emphasis) String() string {
	return "Emphasis"
}

func (text *Emphasis) GetValueList() []interface{} {
	return nil
}

type Italic struct{}

func (text Italic) String() string {
	return "Italic"
}

func (text *Italic) GetValueList() []interface{} {
	return nil
}

type StrikeThrough struct{}

func (text StrikeThrough) String() string {
	return "StrikeThrough"
}

func (text *StrikeThrough) GetValueList() []interface{} {
	return nil
}

type Code struct{}

func (text Code) String() string {
	return "Code"
}

func (text *Code) GetValueList() []interface{} {
	return nil
}

type Math struct{}

func (text Math) String() string {
	return "Math"
}

func (text *Math) GetValueList() []interface{} {
	return nil
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

func (link *Link) GetValueList() []interface{} {
	return []interface{}{&link.Link, &link.Title}
}

// <link> link is url or email address
type SimpleLink struct {
	Link string
}

func (link SimpleLink) String() string {
	return "SimpleLink"
}

func (link *SimpleLink) GetValueList() []interface{} {
	return []interface{}{&link.Link}
}

type ReferenceLink struct {
	Index string
}

func (link ReferenceLink) String() string {
	return fmt.Sprintf("ReferenceLink(%s)", link.Index)
}

func (link *ReferenceLink) GetValueList() []interface{} {
	return []interface{}{&link.Index}
}

type ReferenceLinkIndex struct {
	Index string
	Link  string
	Title string
}

func (link ReferenceLinkIndex) String() string {
	return fmt.Sprintf("ReferenceLinkIndex(%s)", link.Index)
}

func (link *ReferenceLinkIndex) GetValueList() []interface{} {
	return []interface{}{&link.Index, &link.Link, &link.Title}
}

type FootNote struct {
	Index string
}

func (footnote FootNote) String() string {
	return fmt.Sprintf("FootNote(%s)", footnote.Index)
}

func (footnote *FootNote) GetValueList() []interface{} {
	return []interface{}{&footnote.Index}
}

type FootNoteIndex struct {
	Index string
}

func (footnote FootNoteIndex) String() string {
	return fmt.Sprintf("FootNoteIndex(%s)", footnote.Index)
}

func (footnote *FootNoteIndex) GetValueList() []interface{} {
	return []interface{}{&footnote.Index}
}

/* end link */

type Image struct {
	Link  string
	Title string
}

func (image Image) String() string {
	return "Image"
}

func (image *Image) GetValueList() []interface{} {
	return []interface{}{&image.Link, &image.Title}
}

type HtmlStartTag struct {
	Tag     string
	Content string
}

func (html HtmlStartTag) String() string {
	return fmt.Sprintf("HtmlStartTag(%s)", html.Tag)
}

func (html *HtmlStartTag) GetValueList() []interface{} {
	return []interface{}{&html.Tag, &html.Content}
}

type HtmlEndTag struct {
	Tag string
}

func (html HtmlEndTag) String() string {
	return fmt.Sprintf("HtmlEndTag(%s)", html.Tag)
}

func (html *HtmlEndTag) GetValueList() []interface{} {
	return []interface{}{&html.Tag}
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
