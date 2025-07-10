package util

import "strconv"

func encodeMcc(mcc string) []byte {
	// MCC "208" -> digits: 2, 0, 8
	d1, _ := strconv.Atoi(string(mcc[0])) // 2
	d2, _ := strconv.Atoi(string(mcc[1])) // 0
	d3, _ := strconv.Atoi(string(mcc[2])) // 8

	// according to 3GPP
	// Byte 1: (MCC[1] << 4) | MCC[0] = (0 << 4) | 2 = 0x02
	// Byte 2: (f << 4) | MCC[2] = (15 << 4) | 8 = 0xf8
	byte1 := uint8((d2 << 4) | d1)
	byte2 := uint8((0xf << 4) | d3) // f is padding

	return []byte{byte1, byte2}
}

func encodeMnc(mnc string) []byte {
	// MNC "93" -> digits: 9, 3
	d1, _ := strconv.Atoi(string(mnc[0])) // 9
	d2, _ := strconv.Atoi(string(mnc[1])) // 3

	// Byte 3: (MNC[1] << 4) | MNC[0] = (3 << 4) | 9 = 0x39
	byte3 := uint8((d2 << 4) | d1)

	return []byte{byte3}
}

func encodeMsin(msin string) []byte {
	// always take 12 digits
	if len(msin) < 14 {
		for i := len(msin); i < 14; i++ {
			msin = "0" + msin
		}
	}

	result := make([]byte, len(msin)/2)

	for i := 0; i < len(msin); i += 2 {
		var d1, d2 uint8

		tmpD1, err := strconv.Atoi(string(msin[i]))
		if err != nil {
			panic(err)
		}
		d1 = uint8(tmpD1)

		tmpD2, err := strconv.Atoi(string(msin[i+1]))
		if err != nil {
			panic(err)
		}
		d2 = uint8(tmpD2)

		result[i/2] = (d2 << 4) | d1
	}

	return result
}

func SupiToBytes(supi string) []byte {
	var mcc, mnc, msin string

	mcc = supi[0:3]
	mnc = supi[3:5]
	msin = supi[5:]

	mccBytes := encodeMcc(mcc)
	mncBytes := encodeMnc(mnc)
	msinBytes := encodeMsin(msin)

	buffer := make([]byte, 0)

	// Byte 0: SUPI Type (0x01 = IMSI)
	buffer = append(buffer, 0x01)

	// Bytes 1-3: MCC + MNC 編碼
	buffer = append(buffer, mccBytes...)
	buffer = append(buffer, mncBytes...)

	// Byte 4: Routing Indicator (0xf0)
	buffer = append(buffer, 0xf0)

	// Byte 5: Protection Scheme ID (f) + Home Network PKI (f)
	// use null protection scheme
	buffer = append(buffer, 0xff)

	// Add MSIN BCD encoded
	buffer = append(buffer, msinBytes...)

	return buffer
}
