package bn256

// This file contains extra functionality required/added by ethereum implementation.
// This is a separate file for easier update of library itself.

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	acc := new(gfP12)
	acc.SetOne()

	for i := 0; i < len(a); i++ {
		if a[i].p.IsInfinity() || b[i].p.IsInfinity() {
			continue
		}
		acc.Mul(acc, miller(b[i].p, a[i].p))
	}
	return finalExponentiation(acc).IsOne()
}
