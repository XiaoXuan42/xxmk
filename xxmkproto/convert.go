package xxmkproto

import (
	"reflect"

	"github.com/XiaoXuan42/xxmk/naivesel"
	"github.com/XiaoXuan42/xxmk/parserlib"
)

func _setWhich(s string, buf *AstNodeTypeProto) {
	buf.Which = AstNodeTypeEnumProto(AstNodeTypeEnumProto_value[s])
}

func _getAstNodeType(buf *AstNodeTypeProto) parserlib.AstNodeType {
	str := AstNodeTypeEnumProto_name[int32(buf.Which)]
	intf := reflect.New(reflect.TypeOf(parserlib.GetDefaultTypePtr(str)).Elem()).Interface()
	return intf.(parserlib.AstNodeType)
}

func _astNodeTypeToProtobuf(tp parserlib.AstNodeType) *AstNodeTypeProto {
	typeProto := &AstNodeTypeProto{}
	tpName := reflect.TypeOf(tp).Elem().Name()
	_setWhich(tpName, typeProto)
	typeProto.Encode = append(typeProto.Encode, naivesel.Serialize(tp)...)
	return typeProto
}

func _astNodeTypeFromProtobuf(buf *AstNodeTypeProto) parserlib.AstNodeType {
	nodeTp := _getAstNodeType(buf)
	naivesel.Deserialize(nodeTp, buf.Encode)
	return nodeTp
}

func _posToProtobuf(pos *parserlib.Pos) *PosProto {
	res := &PosProto{}
	res.Line = int32(pos.Line)
	res.Col = int32(pos.Col)
	res.Offset = int32(pos.Offset)
	return res
}

func _posFromProtobuf(pos *parserlib.Pos, buf *PosProto) {
	pos.Line = int(buf.Line)
	pos.Col = int(buf.Col)
	pos.Offset = int(buf.Offset)
}

func _nodeToProtobuf(astnode *parserlib.AstNode, ar *[]*AstNodeProto) int32 {
	nodeProto := &AstNodeProto{}
	curId := int32(len(*ar))
	*ar = append(*ar, nodeProto)

	nodeProto.Type = &AstNodeTypeProto{}
	nodeProto.Type = _astNodeTypeToProtobuf(astnode.Type)
	nodeProto.Start = _posToProtobuf(&astnode.Start)
	nodeProto.End = _posToProtobuf(&astnode.End)

	for _, ch := range astnode.Children {
		chId := _nodeToProtobuf(ch, ar)
		nodeProto.Children = append(nodeProto.Children, chId)
	}
	leftsib := int32(-1)
	for _, chId := range nodeProto.Children {
		(*ar)[chId].Parent = curId
		(*ar)[chId].Leftsibling = leftsib
		leftsib = chId
	}
	return int32(curId)
}

func _nodeFromProtobuf(astnode *parserlib.AstNode, ar []*AstNodeProto, curId int32) {
	curBuf := ar[curId]
	astnode.Type = _astNodeTypeFromProtobuf(curBuf.Type)
	_posFromProtobuf(&astnode.Start, curBuf.Start)
	_posFromProtobuf(&astnode.End, curBuf.End)

	var leftSib *parserlib.AstNode
	for _, chId := range curBuf.Children {
		chNode := &parserlib.AstNode{}
		_nodeFromProtobuf(chNode, ar, chId)
		chNode.Parent = astnode
		chNode.LeftSibling = leftSib
		leftSib = chNode
		astnode.Children = append(astnode.Children, chNode)
	}
}

func AstToProtoBuf(ast *parserlib.Ast) *AstProto {
	buf := &AstProto{}
	_nodeToProtobuf(&ast.Root, &buf.Nodes)
	return buf
}

func AstFromProtobuf(ast *parserlib.Ast, astProto *AstProto) {
	if astProto == nil {
		return
	}
	_nodeFromProtobuf(&ast.Root, astProto.Nodes, 0)
}
