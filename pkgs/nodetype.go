package xxmk

import "fmt"

type Document struct{}

func (doc Document) String() string {
	return "Document"
}

type Text struct{}

func (text Text) String() string {
	return "Text"
}

type Header struct {
	Level int
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
	AlignLeft = iota
	AlignMiddle
	AlignRight
)

type TableHead struct{}
type TableAlign struct {
	aligns []int
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
	Level int
}

func (quote QuoteBlock) String() string {
	return "QuoteBlock"
}

/* List */
type List struct {
	IsOrdered bool
}

type ListItem struct {
	IsOrdered bool
	Order     int
}

func (list List) String() string {
	return "List"
}

func (item ListItem) String() string {
	return fmt.Sprintf("ListItem(%d)", item.Order)
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
	link string
}

func (link Link) String() string {
	return "Link"
}

// <link> link is url or email address
type SimpleLink struct {
	link string
}

func (link SimpleLink) String() string {
	return "SimpleLink"
}

/* end link */

type Image struct {
	name string
	link string
}

func (link Image) String() string {
	return "Image"
}

type HtmlStartTag struct {
	tag     string
	content string
}

func (html HtmlStartTag) String() string {
	return fmt.Sprintf("HtmlStartTag(%s)", html.tag)
}

type HtmlEndTag struct {
	tag string
}

func (html HtmlEndTag) String() string {
	return fmt.Sprintf("HtmlEndTag(%s)", html.tag)
}
