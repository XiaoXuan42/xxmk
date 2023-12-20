package parserlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPbMarshal(t *testing.T) {
	proto := AstNodeProto{}
	doc := Document{}
	proto.Type = _astNodeTypeToProtobuf(&doc)
	assert.Equal(t, proto.Type.Which, AstNodeTypeEnumProto_Document)
}

func TestPbEncDec(t *testing.T) {
	hd := Header{ Level: 2 }
	hdBuf := _astNodeTypeToProtobuf(&hd)
	hdBack := _astNodeTypeFromProtobuf(hdBuf)
	assert.Equal(t, hdBack, &hd)

	refLink := ReferenceLinkIndex{ Title: "link", Link: "http" }
	refBuf := _astNodeTypeToProtobuf(&refLink)
	refBack := _astNodeTypeFromProtobuf(refBuf)
	assert.Equal(t, refBack, &refLink)

	align := TableAlign{ aligns: []uint32{0, 1, 2, 3, 4} }
	alignBuf := _astNodeTypeToProtobuf(&align)
	alignBack := _astNodeTypeFromProtobuf(alignBuf)
	assert.Equal(t, alignBack, &align)

	for _, v := range str2NodeType {
		tpProto := _astNodeTypeToProtobuf(v)
		backV := _astNodeTypeFromProtobuf(tpProto)
		assert.Equal(t, v, backV)
	}
}
