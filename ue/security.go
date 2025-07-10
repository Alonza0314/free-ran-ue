package ue

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"github.com/free5gc/nas"
	"github.com/free5gc/nas/nasMessage"
	"github.com/free5gc/nas/security"
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
		return nil, errors.New(fmt.Sprintf("GetKDFValue error: %+v", err))
	}
	P0 = []byte(snName)
	Kseaf, err := ueauth.GetKDFValue(Kausf, ueauth.FC_FOR_KSEAF_DERIVATION, P0, ueauth.KDFLen(P0))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GetKDFValue error: %+v", err))
	}

	supiRegexp, err := regexp.Compile("(?:imsi|supi)-([0-9]{5,15})")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("regexp Compile error: %+v", err))
	}
	groups := supiRegexp.FindStringSubmatch(supi)

	P0 = []byte(groups[1])
	L0 := ueauth.KDFLen(P0)
	P1 = []byte{0x00, 0x00}
	L1 := ueauth.KDFLen(P1)

	return ueauth.GetKDFValue(Kseaf, ueauth.FC_FOR_KAMF_DERIVATION, P0, L0, P1, L1)
}

func deriveAlgorithmKey(kAmf []byte, cipheringAlgorithm, integrityAlgorithm uint8) ([]byte, []byte, error) {
	// Security Key
	P0 := []byte{security.NNASEncAlg}
	L0 := ueauth.KDFLen(P0)
	P1 := []byte{cipheringAlgorithm}
	L1 := ueauth.KDFLen(P1)

	kenc, err := ueauth.GetKDFValue(kAmf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("GetKDFValue error: %+v", err))
	}

	// Integrity Key
	P0 = []byte{security.NNASIntAlg}
	L0 = ueauth.KDFLen(P0)
	P1 = []byte{integrityAlgorithm}
	L1 = ueauth.KDFLen(P1)

	kint, err := ueauth.GetKDFValue(kAmf, ueauth.FC_FOR_ALGORITHM_KEY_DERIVATION, P0, L0, P1, L1)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("GetKDFValue error: %+v", err))
	}

	return kenc, kint, nil
}

func deriveResStarAndSetKey(supi string, cipheringAlgorithm, integrityAlgorithm uint8, sqn, amf, encPermanentKey, encOpcKey string, rand []byte, snName string) ([]byte, []byte, []byte, []byte, error) {
	sqnHex, err := hex.DecodeString(sqn)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error decode sqn: %v", err))
	}

	amfHex, err := hex.DecodeString(amf)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error decode amf: %v", err))
	}

	kHex, err := hex.DecodeString(encPermanentKey)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error decode encPermanentKey: %v", err))
	}

	opcHex, err := hex.DecodeString(encOpcKey)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error decode encOpcKey: %v", err))
	}

	macA, macS := make([]byte, 8), make([]byte, 8)
	ck, ik := make([]byte, 16), make([]byte, 16)
	res := make([]byte, 8)
	ak, akStar := make([]byte, 6), make([]byte, 6)

	// generate macA and macS
	if err := milenage.F1(opcHex, kHex, rand, sqnHex, amfHex, macA, macS); err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error F1: %v", err))
	}

	//generate res, ck, ik, ak, akstar
	if err := milenage.F2345(opcHex, kHex, rand, res, ck, ik, ak, akStar); err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error F2345: %v", err))
	}

	// derive RES*
	key := append(ck, ik...)
	FC := ueauth.FC_FOR_RES_STAR_XRES_STAR_DERIVATION
	P0 := []byte(snName)
	P1 := rand
	P2 := res

	kAmf, err := deriveKAmf(supi, key, snName, sqnHex, ak)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error deriveKAmf: %v", err))
	}
	kenc, kint, err := deriveAlgorithmKey(kAmf, cipheringAlgorithm, integrityAlgorithm)
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error deriveAlgorithmKey: %v", err))
	}
	kdfVal_for_resStar, err := ueauth.GetKDFValue(key, FC, P0, ueauth.KDFLen(P0), P1, ueauth.KDFLen(P1), P2, ueauth.KDFLen(P2))
	if err != nil {
		return nil, nil, nil, nil, errors.New(fmt.Sprintf("Error GetKDFValue: %v", err))
	}
	return kAmf, kenc, kint, kdfVal_for_resStar[len(kdfVal_for_resStar)/2:], nil
}

func encodeNasPduWithSecurity(nasPdu []byte, securityHeaderType uint8, ue *Ue, securityContextAvailable bool, newSecurityContext bool) ([]byte, error) {
	m := nas.NewMessage()
	if err := m.PlainNasDecode(&nasPdu); err != nil {
		return nil, err
	}

	m.SecurityHeader = nas.SecurityHeader{
		ProtocolDiscriminator: nasMessage.Epd5GSMobilityManagementMessage,
		SecurityHeaderType:    securityHeaderType,
	}

	return nasEncode(m, securityContextAvailable, newSecurityContext, ue)
}
