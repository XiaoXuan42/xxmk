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

type MathBlock struct {}

func (math MathBlock) String() string {
	return "MathBlock"
}

type CodeBlock struct {
	Suffix string
}

func (code CodeBlock) String() string {
	return fmt.Sprintf("CodeBlock(%s)", code.Suffix)
}

type Strong struct{}

func (text Strong) String() string {
	return "Strong"
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

// InlineAstNodeType
type Emphasis struct{}
