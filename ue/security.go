package ue

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/free5gc/nas/ie"
	"github.com/free5gc/nas/message"
	"github.com/free5gc/util/milenage"
	"github.com/free5gc/util/ueauth"
)

func deriveKAmf(supi string, key []byte, snName string, SQN, AK []byte) ([]byte, error) {
	FC := ueauth.FC_FOR_KAUSF_DERIVATION
	P0 := []byte(snName)
	SQNxorAK := make([]byte, 6)
	for i := 0; i < len(SQN); i++ {
		SQNxorAK[i] = SQN[i] ^ AK[i]
	}
	P1 := SQNxorAK
	Kausf, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1))
	if err != nil {
		return nil, fmt.Errorf("GetKDFValue error: %+v", err)
	}
	P0 = []byte(snName)
	Kseaf, err := ueauth.GetKDFValue(Kausf, ueauth.FC_FOR_KSEAF_DERIVATION, P0, ueauth.KDFLen(P0))
	if err != nil {
		return nil, fmt.Errorf("GetKDFValue error: %+v", err)
	}

	supiRegexp, err := regexp.Compile("(?:imsi|supi)-([0-9]{5,15})")
	if err != nil {
		return nil, fmt.Errorf("regexp Compile error: %+v", err)
	}
	groups := supiRegexp.FindStringSubmatch(supi)

	P0 = []byte(groups[1])
	L0 := ueauth.KDFLen(P0)
	P1 = []byte{0x00, 0x00}
	L1 := ueauth.KDFLen(P1)

	return ueauth.GetKDFValue(Kseaf, ueauth.FC_FOR_KAMF_DERIVATION, P0, L0, P1, L1)
}

func deriveAlgorithmKey(kAmf []byte, cipheringAlgorithm ie.AlgCiphering, integrityAlgorithm ie.AlgIntegrity) ([]byte, []byte, error) {
	// Security Key
	P0 := []byte{message.NNASEncAlg}
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{byte(cipheringAlgorithm)}
	L1 := ueauth.KDFLen(P1)

	kenc, err := ueauth.GetKDFValue(kAmf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		return nil, nil, fmt.Errorf("GetKDFValue error: %+v", err)
	}

	// Integrity Key
	P0 = []byte{message.NNASIntAlg}
	L0 = ueauth.KDFLen(P0)
	P1 = []byte{byte(integrityAlgorithm)}
	L1 = ueauth.KDFLen(P1)

	kint, err := ueauth.GetKDFValue(kAmf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		return nil, nil, fmt.Errorf("GetKDFValue error: %+v", err)
	}

	return kenc, kint, nil
}

func deriveResStarAndSetKey(supi string, cipheringAlgorithm ie.AlgCiphering, integrityAlgorithm ie.AlgIntegrity, sqn, amf, encPermanentKey, encOpcKey string, rand []byte, autn []byte, snName string) ([]byte, []byte, []byte, []byte, string, error) {
	sqnHex, err := hex.DecodeString(sqn)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error decode sqn: %v", err)
	}

	_, err = hex.DecodeString(amf)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error decode amf: %v", err)
	}

	kHex, err := hex.DecodeString(encPermanentKey)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error decode encPermanentKey: %v", err)
	}

	opcHex, err := hex.DecodeString(encOpcKey)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error decode encOpcKey: %v", err)
	}

	sqnhe, ak, ik, ck, res, err := milenage.GenerateKeysWithAUTN(opcHex, kHex, rand, autn)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error GenerateKeysWithAUTN: %v", err)
	}

	if !bytes.Equal(sqnHex, sqnhe) {
		sqnHex = sqnhe
	}

	// derive RES*
	key := append(ck, ik...)
	FC := ueauth.FC_FOR_RES_STAR_XRES_STAR_DERIVATION
	P0 := []byte(snName)
	P1 := rand
	P2 := res

	kAmf, err := deriveKAmf(supi, key, snName, sqnHex, ak)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error deriveKAmf: %v", err)
	}
	kenc, kint, err := deriveAlgorithmKey(kAmf, cipheringAlgorithm, integrityAlgorithm)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error deriveAlgorithmKey: %v", err)
	}
	kdfVal_for_resStar, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1), P2, ueauth.KDFLen(P2))
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("error GetKDFValue: %v", err)
	}
	return kAmf, kenc, kint, kdfVal_for_resStar[len(kdfVal_for_resStar)/2:], hex.EncodeToString(sqnHex), nil
}
