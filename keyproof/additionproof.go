package keyproof

import (
	"strings"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/zkproof"
)

type (
	additionProofStructure struct {
		a1                string
		a2                string
		mod               string
		result            string
		myname            string
		addRepresentation zkproof.RepresentationProofStructure
		addRange          rangeProofStructure
	}

	AdditionProof struct {
		ModAddProof Proof
		HiderProof  Proof
		RangeProof  RangeProof
	}

	additionProofCommit struct {
		modAdd      secret
		hider       secret
		rangeCommit rangeCommit
	}
)

func newAdditionProofStructure(a1, a2, mod, result string, l uint) additionProofStructure {
	structure := additionProofStructure{
		a1:     a1,
		a2:     a2,
		mod:    mod,
		result: result,
		myname: strings.Join([]string{a1, a2, mod, result, "add"}, "_"),
	}
	structure.addRepresentation = zkproof.RepresentationProofStructure{
		Lhs: []zkproof.LhsContribution{
			{result, big.NewInt(1)},
			{a1, big.NewInt(-1)},
			{a2, big.NewInt(-1)},
		},
		Rhs: []zkproof.RhsContribution{
			{mod, strings.Join([]string{structure.myname, "mod"}, "_"), 1},
			{"h", strings.Join([]string{structure.myname, "hider"}, "_"), 1},
		},
	}
	structure.addRange = rangeProofStructure{
		structure.addRepresentation,
		strings.Join([]string{structure.myname, "mod"}, "_"),
		0,
		l,
	}
	return structure
}

func (s *additionProofStructure) commitmentsFromSecrets(g zkproof.Group, list []*big.Int, bases zkproof.BaseLookup, secretdata zkproof.SecretLookup) ([]*big.Int, additionProofCommit) {
	var commit additionProofCommit

	// Generate needed commit data
	commit.modAdd = newSecret(g, strings.Join([]string{s.myname, "mod"}, "_"),
		new(big.Int).Div(
			new(big.Int).Sub(
				secretdata.Secret(s.result),
				new(big.Int).Add(
					secretdata.Secret(s.a1),
					secretdata.Secret(s.a2))),
			secretdata.Secret(s.mod)))
	commit.hider = newSecret(g, strings.Join([]string{s.myname, "hider"}, "_"),
		new(big.Int).Mod(
			new(big.Int).Sub(
				secretdata.Secret(strings.Join([]string{s.result, "hider"}, "_")),
				new(big.Int).Add(
					new(big.Int).Add(
						secretdata.Secret(strings.Join([]string{s.a1, "hider"}, "_")),
						secretdata.Secret(strings.Join([]string{s.a2, "hider"}, "_"))),
					new(big.Int).Mul(
						secretdata.Secret(strings.Join([]string{s.mod, "hider"}, "_")),
						commit.modAdd.secretv))),
			g.Order))

	// build inner secrets
	secrets := zkproof.NewSecretMerge(&commit.hider, &commit.modAdd, secretdata)

	// and build commits
	list = s.addRepresentation.CommitmentsFromSecrets(g, list, bases, &secrets)
	list, commit.rangeCommit = s.addRange.commitmentsFromSecrets(g, list, bases, &secrets)

	return list, commit
}

func (s *additionProofStructure) buildProof(g zkproof.Group, challenge *big.Int, commit additionProofCommit, secretdata zkproof.SecretLookup) AdditionProof {
	rangeSecrets := zkproof.NewSecretMerge(&commit.hider, &commit.modAdd, secretdata)
	return AdditionProof{
		RangeProof:  s.addRange.buildProof(g, challenge, commit.rangeCommit, &rangeSecrets),
		ModAddProof: commit.modAdd.buildProof(g, challenge),
		HiderProof:  commit.hider.buildProof(g, challenge),
	}
}

func (s *additionProofStructure) fakeProof(g zkproof.Group) AdditionProof {
	return AdditionProof{
		RangeProof:  s.addRange.fakeProof(g),
		ModAddProof: fakeProof(g),
		HiderProof:  fakeProof(g),
	}
}

func (s *additionProofStructure) verifyProofStructure(proof AdditionProof) bool {
	if !s.addRange.verifyProofStructure(proof.RangeProof) {
		return false
	}
	if !proof.HiderProof.verifyStructure() || !proof.ModAddProof.verifyStructure() {
		return false
	}
	return true
}

func (s *additionProofStructure) commitmentsFromProof(g zkproof.Group, list []*big.Int, challenge *big.Int, bases zkproof.BaseLookup, proofdata zkproof.ProofLookup, proof AdditionProof) []*big.Int {
	// build inner proof lookup
	proof.ModAddProof.setName(strings.Join([]string{s.myname, "mod"}, "_"))
	proof.HiderProof.setName(strings.Join([]string{s.myname, "hider"}, "_"))
	proofs := zkproof.NewProofMerge(&proof.HiderProof, &proof.ModAddProof, proofdata)

	// build commitments
	list = s.addRepresentation.CommitmentsFromProof(g, list, challenge, bases, &proofs)
	list = s.addRange.commitmentsFromProof(g, list, challenge, bases, proof.RangeProof)

	return list
}

func (s *additionProofStructure) isTrue(secretdata zkproof.SecretLookup) bool {
	div := new(big.Int)
	mod := new(big.Int)

	div.DivMod(
		new(big.Int).Sub(
			secretdata.Secret(s.result),
			new(big.Int).Add(
				secretdata.Secret(s.a1),
				secretdata.Secret(s.a2))),
		secretdata.Secret(s.mod),
		mod)

	return mod.Cmp(big.NewInt(0)) == 0 && uint(div.BitLen()) <= s.addRange.l2
}

func (s *additionProofStructure) numRangeProofs() int {
	return 1
}

func (s *additionProofStructure) numCommitments() int {
	return s.addRepresentation.NumCommitments() + s.addRange.numCommitments()
}
