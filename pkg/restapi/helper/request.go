/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helper

import (
	"errors"

	"github.com/trustbloc/sidetree-core-go/pkg/docutil"
	"github.com/trustbloc/sidetree-core-go/pkg/patch"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/model"
)

// CreateRequestInfo contains data for creating create payload
type CreateRequestInfo struct {

	// opaque document content
	// required
	OpaqueDocument string

	// the recovery public key as a HEX string
	// required
	RecoveryKey string

	// reveal value to be used for the next recovery
	NextRecoveryRevealValue []byte

	// reveal value to be used for the next update
	NextUpdateRevealValue []byte

	// latest hashing algorithm supported by protocol
	MultihashCode uint
}

//RevokeRequestInfo is the information required to create revoke request
type RevokeRequestInfo struct {

	// Unique Suffix
	DidUniqueSuffix string

	// reveal value for this revoke operation
	RecoveryRevealValue []byte
}

//RecoverRequestInfo is the information required to create recover request
type RecoverRequestInfo struct {

	// Unique Suffix of the did to be recovered
	DidUniqueSuffix string

	// reveal value for this recovery operation
	RecoveryRevealValue []byte

	// the new recovery public key as a HEX string
	RecoveryKey string

	// opaque content
	OpaqueDocument string

	// reveal value to be used for the next recovery
	NextRecoveryRevealValue []byte

	// reveal value to be used for the next update
	NextUpdateRevealValue []byte

	// latest hashing algorithm supported by protocol
	MultihashCode uint
}

//UpdateRequestInfo is the information required to create update request
type UpdateRequestInfo struct {

	// Unique Suffix
	DidUniqueSuffix string

	// Patch is one of standard patch actions
	Patch patch.Patch

	// reveal value for this update operation
	UpdateRevealValue []byte

	// reveal value to be used for the next update
	// optional if NextUpdateCommitmentHash is provided
	NextUpdateRevealValue []byte

	// latest hashing algorithm supported by protocol
	MultihashCode uint
}

// NewCreateRequest is utility function to create payload for 'create' request
func NewCreateRequest(info *CreateRequestInfo) ([]byte, error) {
	if info.OpaqueDocument == "" {
		return nil, errors.New("missing opaque document")
	}

	if info.RecoveryKey == "" {
		return nil, errors.New("missing recovery key")
	}

	replace, err := patch.NewReplacePatch(info.OpaqueDocument)
	if err != nil {
		return nil, err
	}

	patches := []patch.Patch{replace}
	patchDataBytes, err := getPatchDataBytes(info.MultihashCode, info.NextUpdateRevealValue, patches)
	if err != nil {
		return nil, err
	}

	mhPatchData, err := docutil.ComputeMultihash(info.MultihashCode, patchDataBytes)
	if err != nil {
		return nil, err
	}

	mhNextRecoveryCommitmentHash, err := getEncodedMultihash(info.MultihashCode, info.NextRecoveryRevealValue)
	if err != nil {
		return nil, err
	}

	suffixData := model.SuffixDataModel{
		PatchDataHash:              docutil.EncodeToString(mhPatchData),
		RecoveryKey:                model.PublicKey{PublicKeyHex: info.RecoveryKey},
		NextRecoveryCommitmentHash: mhNextRecoveryCommitmentHash,
	}

	suffixDataBytes, err := docutil.MarshalCanonical(suffixData)
	if err != nil {
		return nil, err
	}

	schema := &model.CreateRequest{
		Operation:  model.OperationTypeCreate,
		PatchData:  docutil.EncodeToString(patchDataBytes),
		SuffixData: docutil.EncodeToString(suffixDataBytes),
	}

	return docutil.MarshalCanonical(schema)
}

func getEncodedMultihash(mhCode uint, bytes []byte) (string, error) {
	hash, err := docutil.ComputeMultihash(mhCode, bytes)
	if err != nil {
		return "", err
	}

	return docutil.EncodeToString(hash), nil
}

// NewRevokeRequest is utility function to create payload for 'revoke' request
func NewRevokeRequest(info *RevokeRequestInfo) ([]byte, error) {
	if info.DidUniqueSuffix == "" {
		return nil, errors.New("missing did unique suffix")
	}

	signedDataModel := model.RevokeSignedDataModel{
		DidUniqueSuffix:     info.DidUniqueSuffix,
		RecoveryRevealValue: docutil.EncodeToString(info.RecoveryRevealValue),
	}

	// TODO: it should be signed using recovery key, so signer should be recovery key
	jws, err := signModel(signedDataModel)
	if err != nil {
		return nil, err
	}

	schema := &model.RevokeRequest{
		Operation:           model.OperationTypeRevoke,
		DidUniqueSuffix:     info.DidUniqueSuffix,
		RecoveryRevealValue: docutil.EncodeToString(info.RecoveryRevealValue),
		SignedData:          jws,
	}

	return docutil.MarshalCanonical(schema)
}

// NewUpdateRequest is utility function to create payload for 'update' request
func NewUpdateRequest(info *UpdateRequestInfo) ([]byte, error) {
	if info.DidUniqueSuffix == "" {
		return nil, errors.New("missing did unique suffix")
	}

	if info.Patch == nil {
		return nil, errors.New("missing update information")
	}

	patches := []patch.Patch{info.Patch}
	patchDataBytes, err := getPatchDataBytes(info.MultihashCode, info.NextUpdateRevealValue, patches)
	if err != nil {
		return nil, err
	}

	mhPatchData, err := getEncodedMultihash(info.MultihashCode, patchDataBytes)
	if err != nil {
		return nil, err
	}

	jws, err := signPayload(mhPatchData)
	if err != nil {
		return nil, err
	}

	schema := &model.UpdateRequest{
		Operation:         model.OperationTypeUpdate,
		DidUniqueSuffix:   info.DidUniqueSuffix,
		UpdateRevealValue: docutil.EncodeToString(info.UpdateRevealValue),
		PatchData:         docutil.EncodeToString(patchDataBytes),
		SignedData:        jws,
	}

	return docutil.MarshalCanonical(schema)
}

// NewRecoverRequest is utility function to create payload for 'recovery' request
func NewRecoverRequest(info *RecoverRequestInfo) ([]byte, error) {
	err := checkRequiredDataForRecovery(info)
	if err != nil {
		return nil, err
	}

	replacePatch, err := patch.NewReplacePatch(info.OpaqueDocument)
	if err != nil {
		return nil, err
	}

	patches := []patch.Patch{replacePatch}
	patchDataBytes, err := getPatchDataBytes(info.MultihashCode, info.NextUpdateRevealValue, patches)
	if err != nil {
		return nil, err
	}

	mhPatchData, err := docutil.ComputeMultihash(info.MultihashCode, patchDataBytes)
	if err != nil {
		return nil, err
	}

	mhNextRecoveryCommitmentHash, err := getEncodedMultihash(info.MultihashCode, info.NextRecoveryRevealValue)
	if err != nil {
		return nil, err
	}

	signedDataModel := model.RecoverSignedDataModel{
		PatchDataHash:              docutil.EncodeToString(mhPatchData),
		RecoveryKey:                model.PublicKey{PublicKeyHex: info.RecoveryKey},
		NextRecoveryCommitmentHash: mhNextRecoveryCommitmentHash,
	}

	// TODO: it should be signed using recovery key, so signer should be recovery key
	jws, err := signModel(signedDataModel)
	if err != nil {
		return nil, err
	}

	schema := &model.RecoverRequest{
		Operation:           model.OperationTypeRecover,
		DidUniqueSuffix:     info.DidUniqueSuffix,
		RecoveryRevealValue: docutil.EncodeToString(info.RecoveryRevealValue),
		PatchData:           docutil.EncodeToString(patchDataBytes),
		SignedData:          jws,
	}

	return docutil.MarshalCanonical(schema)
}

func signModel(data interface{}) (*model.JWS, error) {
	signedDataBytes, err := docutil.MarshalCanonical(data)
	if err != nil {
		return nil, err
	}

	payload := docutil.EncodeToString(signedDataBytes)

	return signPayload(payload)
}

// sign payload will sign payload
// TODO: Add signer here (part of JWS PR)
func signPayload(payload string) (*model.JWS, error) {
	return &model.JWS{
		// TODO: should be JWS encoded, encode for now
		Payload: payload,
	}, nil
}

func checkRequiredDataForRecovery(info *RecoverRequestInfo) error {
	if info.DidUniqueSuffix == "" {
		return errors.New("missing did unique suffix")
	}

	if info.OpaqueDocument == "" {
		return errors.New("missing opaque document")
	}

	if info.RecoveryKey == "" {
		return errors.New("missing recovery key")
	}

	return nil
}

func getPatchDataBytes(mhCode uint, reveal []byte, patches []patch.Patch) ([]byte, error) {
	mhNextUpdateCommitmentHash, err := getEncodedMultihash(mhCode, reveal)
	if err != nil {
		return nil, err
	}

	patchData := model.PatchDataModel{
		NextUpdateCommitmentHash: mhNextUpdateCommitmentHash,
		Patches:                  patches,
	}

	return docutil.MarshalCanonical(patchData)
}
