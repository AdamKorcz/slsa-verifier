package slsaprovenance

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	dsselib "github.com/secure-systems-lab/go-securesystemslib/dsse"

	serrors "github.com/slsa-framework/slsa-verifier/v2/errors"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/internal/gha/slsaprovenance/common"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/internal/gha/slsaprovenance/iface"
	slsav02 "github.com/slsa-framework/slsa-verifier/v2/verifiers/internal/gha/slsaprovenance/v0.2"
	slsav1 "github.com/slsa-framework/slsa-verifier/v2/verifiers/internal/gha/slsaprovenance/v1.0"
)

// provenanceConstructor creates a new Provenance instance for the given payload as a json Decoder.
type provenanceConstructor func(payload []byte) (iface.Provenance, error)

// predicateTypeMap stores the different provenance version types. It is a map of
// predicate type -> ProvenanceConstructor.
var predicateTypeMap = map[string]provenanceConstructor{
	common.ProvenanceV02Type:      slsav02.New,
	slsa1.PredicateSLSAProvenance: slsav1.New,
}

// ProvenanceFromEnvelope returns a Provenance instance for the given DSSE Envelope.
func ProvenanceFromEnvelope(env *dsselib.Envelope) (iface.Provenance, error) {
	if env.PayloadType != intoto.PayloadType {
		return nil, fmt.Errorf("%w: expected payload type %q, got '%s'",
			serrors.ErrorInvalidDssePayload, intoto.PayloadType, env.PayloadType)
	}
	pyld, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", serrors.ErrorInvalidDssePayload, err.Error())
	}

	// Load the in-toto attestation statement header.
	pred := intoto.StatementHeader{}
	if err := json.Unmarshal(pyld, &pred); err != nil {
		return nil, fmt.Errorf("%w: decoding json: %v", serrors.ErrorInvalidDssePayload, err)
	}

	// Verify the predicate type is one we can handle.
	newProv, ok := predicateTypeMap[pred.PredicateType]
	if !ok {
		return nil, fmt.Errorf("%w: unexpected predicate type '%s'", serrors.ErrorInvalidDssePayload, pred.PredicateType)
	}
	prov, err := newProv(pyld)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", serrors.ErrorInvalidDssePayload, err)
	}

	return prov, nil
}
