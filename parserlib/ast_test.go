package parserlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestAstProtoMarshalWithoutErr(t *testing.T) {
	start := Pos{}
	end := Pos{}
	end.ConsumeStr("hello world\n")
	ast := Ast{
		Root: AstNode{
			Type:  &Document{},
			Start: start,
			End:   end,
		},
	}
	buf := ast.ToProtoBuf()
	_, err := proto.Marshal(buf)
	assert.Equal(t, nil, err)
}

func TestAstEqual(t *testing.T) {
	mk := `# Title1
- item1
- item2

hello world!
## Title2
1. item1
2. item2
`
	parser := GetFullMKParser()
	ast := parser.Parse(mk)
	astProto := ast.ToProtoBuf()
	buf, err := proto.Marshal(astProto)
	assert.Equal(t, nil, err)
	astProto2 := &AstProto{}
	err = proto.Unmarshal(buf, astProto2)
	assert.Equal(t, nil, err)
	ast2 := &Ast{}
	ast2.FromProtobuf(astProto2)
	assert.Equal(t, true, ast.Eq(ast2))
}
