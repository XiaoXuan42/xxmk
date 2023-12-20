package naivesel

import "encoding/binary"

func _encodeUint32(vs ...uint32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[i*4:i*4+4], v)
	}
	return b
}

func _decodeUint32(bytes []byte) (uint32, int) {
	return binary.LittleEndian.Uint32(bytes), 4
}

func _encodeBools(bs ...bool) []byte {
	var result []byte
	for _, b := range bs {
		if b {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	}
	return result
}

func _decodeBools(bytes []byte) (bool, int) {
	if bytes[0] != 0 {
		return true, 1
	}
	return false, 1
}

func _encodeStrs(strs ...string) []byte {
	var result []byte
	for _, s := range strs {
		result = append(result, _encodeUint32(uint32(len(s)))...)
		result = append(result, []byte(s)...)
	}
	return result
}

func _decodeStrs(bytes []byte) (string, int) {
	len, offset := _decodeUint32(bytes)
	totOffset := offset + int(len)
	s := string(bytes[offset:totOffset])
	return s, totOffset
}

func _encodeUint32Slice(slice []uint32) []byte {
	var result []byte
	result = append(result, _encodeUint32(uint32(len(slice)))...)
	for _, u := range slice {
		result = append(result, _encodeUint32(u)...)
	}
	return result
}

func _decodeUint32Slice(bytes []byte) ([]uint32, int) {
	len, offset := _decodeUint32(bytes)
	var result []uint32
	for i := 0; i < int(len); i++ {
		v, curOff := _decodeUint32(bytes[offset:])
		offset += curOff
		result = append(result, v)
	}
	return result, offset
}
