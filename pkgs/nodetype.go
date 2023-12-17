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

type StrongText struct {}

func (text StrongText) String() string {
	return "StrongText"
}


// InlineAstNodeType
type Emphasis struct{}
