package xxmk

import "fmt"

type Document struct{}

func (doc Document) String() string {
	return "Document"
}

// AstNodeType

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

type QuoteBlock struct{}

func (quote QuoteBlock) String() string {
	return "QuoteBlock"
}

type Emphasis struct{}

func (text Emphasis) String() string {
	return "Emphasis"
}

type Italic struct{}

func (text Italic) String() string {
	return "Italic"
}

type Code struct{}

func (text Code) String() string {
	return "Code"
}

type Math struct{}

func (text Math) String() string {
	return "Math"
}

type Link struct {
	name string
	link string
}

func (link Link) String() string {
	return "Link"
}

type Image struct {
	name string
	link string
}

func (link Image) String() string {
	return "Image"
}
