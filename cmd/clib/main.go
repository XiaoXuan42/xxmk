package main

/*
#include <stdlib.h>

typedef struct ByteSlice {
	int len;
	void *p;
} ByteSlice;
*/
import "C"
import (

	"github.com/XiaoXuan42/xxmk/parserlib"
	"google.golang.org/protobuf/proto"
)

//export ParseMarkDownToAstString
func ParseMarkDownToAstString(markdown string) *C.char {
	p := parserlib.GetHtmlMKParser()
	ast := p.Parse(markdown)
	return C.CString(ast.String())
}

//export ParseMarkDownToAstProto
func ParseMarkDownToAstProto(markdown string) C.struct_ByteSlice {
	p := parserlib.GetHtmlMKParser()
	ast := p.Parse(markdown)
	astProto := ast.ToProtoBuf()
	buf, err := proto.Marshal(astProto)
	var slice C.struct_ByteSlice
	if err != nil {
		slice.len = 0
		slice.p = nil
		return slice
	}
	slice.len = C.int(len(buf))
	slice.p = C.CBytes(buf)
	return slice
}

func main() {

}
