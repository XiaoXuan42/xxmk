package xxmkproto

import (
	"testing"

	"github.com/XiaoXuan42/xxmk/parserlib"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestPbMarshal(t *testing.T) {
	proto := AstNodeProto{}
	doc := parserlib.Document{}
	proto.Type = _astNodeTypeToProtobuf(&doc)
	assert.Equal(t, proto.Type.Which, AstNodeTypeEnumProto_Document)
}

func TestPbEncDec(t *testing.T) {
	hd := parserlib.Header{Level: 2}
	hdBuf := _astNodeTypeToProtobuf(&hd)
	hdBack := _astNodeTypeFromProtobuf(hdBuf)
	assert.Equal(t, hdBack, &hd)

	refLink := parserlib.ReferenceLinkIndex{Title: "link", Link: "http"}
	refBuf := _astNodeTypeToProtobuf(&refLink)
	refBack := _astNodeTypeFromProtobuf(refBuf)
	assert.Equal(t, refBack, &refLink)

	align := parserlib.TableAlign{Aligns: []uint32{0, 1, 2, 3, 4}}
	alignBuf := _astNodeTypeToProtobuf(&align)
	alignBack := _astNodeTypeFromProtobuf(alignBuf)
	assert.Equal(t, alignBack, &align)
}

func TestAstProtoMarshalWithoutErr(t *testing.T) {
	start := parserlib.Pos{}
	end := parserlib.Pos{}
	end.ConsumeStr("hello world\n")
	ast := parserlib.Ast{
		Root: parserlib.AstNode{
			Type:  &parserlib.Document{},
			Start: start,
			End:   end,
		},
	}
	buf := AstToProtoBuf(&ast)
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
	parser := parserlib.GetFullMKParser()
	ast := parser.Parse(mk)
	astProto := AstToProtoBuf(&ast)
	buf, err := proto.Marshal(astProto)
	assert.Equal(t, nil, err)
	astProto2 := &AstProto{}
	err = proto.Unmarshal(buf, astProto2)
	assert.Equal(t, nil, err)
	ast2 := &parserlib.Ast{}
	AstFromProtobuf(ast2, astProto2)
	assert.Equal(t, true, ast.Eq(ast2))
}
