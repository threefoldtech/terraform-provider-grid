package provider

type Number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64
}

type Chars interface {
	byte | string
}

func includes[N Number | Chars](l []N, i N) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}
