package main

import "C"
import "github.com/XiaoXuan42/xxmk/parserlib"


//export ParseMarkDownToAstString
func ParseMarkDownToAstString(markdown string) *C.char {
	p := parser.GetHtmlMKParser()
	ast := p.Parse(markdown)
	return C.CString(ast.String())
}

//
func ParseMarkDownToAst(markdown string) *parser.AstNode {
	p := parser.GetHtmlMKParser()
	ast := p.Parse(markdown)
	return &ast.Root
}

func main() {

}