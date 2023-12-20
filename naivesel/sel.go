package naivesel

import "reflect"

func _getValList(tp interface{}) []interface{} {
	var result []interface{}
	val := reflect.ValueOf(tp).Elem()
	for i := 0; i < val.NumField(); i++ {
		addr := val.Field(i).Addr().Interface()
		result = append(result, addr)
	}
	return result
}

func _valueListToBytes(valList []interface{}) []byte {
	var result []byte
	for _, v := range valList {
		switch val := v.(type) {
		case *int:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *uint:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *int32:
			result = append(result, _encodeUint32(uint32(*val))...)
		case *uint32:
			result = append(result, _encodeUint32(*val)...)
		case *string:
			result = append(result, _encodeStrs(*val)...)
		case *bool:
			result = append(result, _encodeBools(*val)...)
		case *[]uint32:
			result = append(result, _encodeUint32Slice(*val)...)
		default:
			panic("Unknown type to encode")
		}
	}
	return result
}

func _valueListFromBytes(valList []interface{}, bytes []byte) {
	rear := 0
	for _, v := range valList {
		switch val := v.(type) {
		case *int:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = int(dv)
			rear += offset
		case *uint:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = uint(dv)
			rear += offset
		case *int32:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = int32(dv)
			rear += offset
		case *uint32:
			dv, offset := _decodeUint32(bytes[rear:])
			*val = uint32(dv)
			rear += offset
		case *string:
			dv, offset := _decodeStrs(bytes[rear:])
			*val = dv
			rear += offset
		case *bool:
			dv, offset := _decodeBools(bytes[rear:])
			*val = dv
			rear += offset
		case *[]uint32:
			dv, offset := _decodeUint32Slice(bytes[rear:])
			*val = dv
			rear += offset
		default:
			panic("Unknown type to decode")
		}
	}
}

func Serialize(tp interface{}) []byte {
	valList := _getValList(tp)
	return _valueListToBytes(valList)
}

func Deserialize(tp interface{}, bytes []byte) {
	valList := _getValList(tp)
	_valueListFromBytes(valList, bytes)
}
