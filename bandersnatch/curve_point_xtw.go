package bandersnatch

import (
	"math/big"
	"math/rand"
)

type point_xtw_base struct {
	thisCurvePointCanRepresentFullCurve
	thisCurvePointCanRepresentInfinity
	x FieldElement
	y FieldElement
	z FieldElement
	t FieldElement
}

// Point_xtw describes points on the p253-subgroup of the Bandersnatch curve in extended twisted Edwards coordinates.
// Extended means that we additionally store T with T = X*Y/Z. Note that Z is never 0 for points in the subgroup, but see code comment about desingularisation.)
// cf. https://iacr.org/archive/asiacrypt2008/53500329/53500329.pdf
type Point_xtw_subgroup struct {
	thisCurvePointCanOnlyRepresentSubgroup
	point_xtw_base
}

type Point_xtw_full struct {
	point_xtw_base
}

/*
	NOTE: Points described by Point_xtw should really been seen as solutions to the set of homogeneous equations

	ax^2 + y^2 = z^2 + dt^2
	x*y = z*t

	with addition law for P3 = P1 + P2 given by:
	X3 = (X1Y2 + Y1X2)(Z1Z2 - dT1T2)
	Y3 = (Y1Y2 - aX1X2)(Z1Z2 + dT1T2)
	T3 = (X1Y2 + Y1X2)(Y1Y2-aX1X2)
	Z3 = (Z1Z2 - dT1T2)(Z1Z2 + dT1T2)

	which we call the extended twisted Edwards model. We treat this as a curve model (like Weierstrass, Montgomery, (twisted) Edwards) rather than a redundant coordinate representaion.

	Clearly, the set of affine solutions corresponds exactly to the set of affine solutions of the usual twisted Edwards equation ax^2 + y^2 = 1+dx^2y^2
	(with z==1, t==x*y), but there are differences in the behaviour at infinity:
	Notably, the twisted Edwards curve has 2+2 points at infinity and the curve is actually singular there:
	Those are double (in the sense that a desingularization results in two points) points at (1:0:0), (0:1:0) each.
	By contrast, the extended twisted Edwards model has no singularities for a != +/-d (over the algebraic closure, to be clear).
	In fact, the additional t coordinate both improves efficiency and is a very convenient desingularization, where things become more clear.
	The (not neccessarily rational) points at infinity (z==0) of this model are in (x:y:t:z) coordinates:
	(0:sqrtz(d):1:0), (0:-sqrt(d):1:0), (sqrt(d/a):0:1:0), (-sqrt(d/a):0:1:0)
	The first two point have order 4 (doubling them gives (0:-1:0:1)), the latter two points have order 2.
	Now, in the case usually considered in the literature, d is a non-square and a is a square.
	Then all these points at infinity are actually not rational and we even get a unified point addition law that works for all rational points.

	In the bandersnatch case, both a and d are non-squares. This means we get two bona-fide rational(!) points at infinity of order 2.
	The addition law above no longer works in all cases. A lengthy analysis (TODO: Make a pdf and write up the proof or find one in literature) shows that the following holds

	Theorem:
	for P1,P2 rational, the extended Edwards addition law given above for P1 + P2 does not work if and only if P1 - P2 is a (rational, order-2) point at infinity.

	Consequences:
	The addition law works for all points in the subgroup of size 2*p253, generated by the large-prime p253 subgroup and the affine point of order 2.
	If P1,P2 are both contained in a cyclic subgroup generated by Q, then the addition law can only fail in the following cases:
		One of P1,P2 is the neutral element, the other one is equal to Q and is a point at infinity.
		Q has order 2*p253, P1 = alpha * Q, P2 = beta * Q with alpha-beta == p253 mod 2*p253. We can actually ensure that never happens in our exponentiation algorithms.
*/

// example point on the subgroup specified in the bandersnatch paper
var example_generator_x *big.Int = initIntFromString("0x29c132cc2c0b34c5743711777bbe42f32b79c022ad998465e1e71866a252ae18")
var example_generator_y *big.Int = initIntFromString("0x2a6c669eda123e0f157d8b50badcd586358cad81eee464605e3167b6cc974166")
var example_generator_t *big.Int = new(big.Int).Mul(example_generator_x, example_generator_y)
var example_generator_xtw point_xtw_base = func() (ret point_xtw_base) {
	ret.x.SetInt(example_generator_x)
	ret.y.SetInt(example_generator_y)
	ret.t.SetInt(example_generator_t)
	ret.z.SetOne()
	return
}()

/*
	Basic functions for Point_xtw
*/

// NeutralElement_<foo> denotes the Neutral Element of the Bandersnatch curve in <foo> coordinates.
var (
	NeutralElement_xtw point_xtw_base = point_xtw_base{x: FieldElementZero, y: FieldElementOne, t: FieldElementZero, z: FieldElementOne}
)

// These are the three points of order 2 that we can represent with extended twisted coordinates. None of these is in the p253-subgroup, of course.
// Although we do not need or use this, note that SqrtDDivA_fe := sqrt(d/a) == sqrt(2) - 1 due to the way the bandersnatch curve was constructed.
var (
	orderTwoPoint_xtw      point_xtw_base = point_xtw_base{x: FieldElementZero, y: FieldElementMinusOne, t: FieldElementZero, z: FieldElementOne}
	exceptionalPoint_1_xtw point_xtw_base = point_xtw_base{x: squareRootDbyA_fe, y: FieldElementZero, t: FieldElementOne, z: FieldElementZero}
	exceptionalPoint_2_xtw point_xtw_base = point_xtw_base{x: squareRootDbyA_fe, y: FieldElementZero, t: FieldElementMinusOne, z: FieldElementZero}
)

// normalizeAffineZ replaces the internal representation with an equivalent one with Z==1, unless the point is at infinity (in which case we panic).
// This is used to convert to or output affine coordinates.
func (p *point_xtw_base) normalizeAffineZ() {
	if p.IsNaP() {
		napEncountered("Try to converting invalid point xtw to coos with z==1", false, p)
		// If the above did not panic, we replace the NaP p by an default NaP with x==y==t==0, z==1.
		*p = point_xtw_base{z: FieldElementOne} // invalid point
		return
	}

	// We reasonably likely call normalizeAffineZ several times in a row on the same point. If Z==1 to start with, do nothing.
	if p.z.IsOne() {
		return
	}
	if p.z.IsZero() {
		// division by zero error
		panic("Trying to make point at infinity affine")
	}
	var zInverse FieldElement
	zInverse.Inv(&p.z)
	p.x.MulEq(&zInverse)
	p.y.MulEq(&zInverse)
	p.t.MulEq(&zInverse)
	p.z.SetOne()
}

func (p *Point_xtw_subgroup) normalizeSubgroup() {
	if !legendreCheckE1_projectiveYZ(p.y, p.z) {
		p.flipDecaf()
	}
}

func (p *point_xtw_base) flipDecaf() {
	p.x.NegEq()
	p.y.NegEq()
}

func (p *Point_xtw_subgroup) HasDecaf() bool {
	return true
}

func (p *point_xtw_base) rerandomizeRepresentation(rnd *rand.Rand) {
	var m FieldElement
	m.setRandomUnsafeNonZero(rnd)
	p.x.MulEq(&m)
	p.y.MulEq(&m)
	p.t.MulEq(&m)
	p.z.MulEq(&m)
}

func (p *point_xtw_base) IsE1() bool {
	var tmp FieldElement
	tmp.Mul(&p.t, &squareRootDbyA_fe)
	return tmp.IsEqual(&p.x)
}

func (p *Point_xtw_subgroup) rerandomizeRepresentation(rnd *rand.Rand) {
	p.point_xtw_base.rerandomizeRepresentation(rnd)
	if rnd.Intn(2) == 0 {
		p.flipDecaf()
	}
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_subgroup) X_affine() FieldElement {
	p.normalizeAffineZ()
	p.normalizeSubgroup()
	return p.x
}

// X_affine returns the X coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_full) X_affine() FieldElement {
	p.normalizeAffineZ()
	return p.x
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_subgroup) Y_affine() FieldElement {
	p.normalizeAffineZ()
	p.normalizeSubgroup()
	return p.y
}

// Y_affine returns the Y coordinate of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_full) Y_affine() FieldElement {
	p.normalizeAffineZ()
	return p.y
}

// T_affine returns the T coordinate (i.e. T=XY) of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_subgroup) T_affine() FieldElement {
	p.normalizeAffineZ()
	p.normalizeSubgroup()
	return p.t
}

// T_affine returns the T coordinate (i.e. T=XY) of the given point in affine twisted Edwards coordinates.
func (p *Point_xtw_full) T_affine() FieldElement {
	p.normalizeAffineZ()
	return p.t
}

func (p *Point_xtw_subgroup) XY_affine() (FieldElement, FieldElement) {
	p.normalizeAffineZ()
	p.normalizeSubgroup()
	return p.x, p.y
}

func (p *Point_xtw_full) XY_affine() (FieldElement, FieldElement) {
	p.normalizeAffineZ()
	return p.x, p.y
}

func (p *Point_xtw_subgroup) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	p.normalizeAffineZ()
	p.normalizeSubgroup()
	return p.x, p.y, p.t
}

func (p *Point_xtw_full) XYT_affine() (FieldElement, FieldElement, FieldElement) {
	p.normalizeAffineZ()
	return p.x, p.y, p.t
}

// X_projective returns the X coordinate of the given point p in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), T_projective() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw_subgroup) X_projective() FieldElement {
	p.normalizeSubgroup()
	return p.x
}

func (p *Point_xtw_full) X_projective() FieldElement {
	return p.x
}

// Y_projective returns the Y coordinate of the given point P in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw_subgroup) Y_projective() FieldElement {
	p.normalizeSubgroup()
	return p.y
}

func (p *Point_xtw_full) Y_projective() FieldElement {
	return p.y
}

// Z_projective returns the Z coordinate of the given point P in projective twisted Edwards coordinates.
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw_subgroup) Z_projective() FieldElement {
	p.normalizeSubgroup()
	return p.z
}

func (p *Point_xtw_full) Z_projective() FieldElement {
	return p.z
}

func (p *Point_xtw_subgroup) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.z
}

func (p *Point_xtw_full) XYZ_projective() (FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.z
}

// T_projective returns the T coordinate of the given point P in projective twisted Edwards coordinates (i.e. T = XY/Z).
// Note that calling functions on P other than X_projective(), Y_projective(), Z() might change the representations of P at will,
// so callers must not interleave calling other functions.
func (p *Point_xtw_subgroup) T_projective() FieldElement {
	p.normalizeSubgroup()
	return p.t
}

func (p *Point_xtw_full) T_projective() FieldElement {
	return p.t
}

func (p *Point_xtw_subgroup) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	p.normalizeSubgroup()
	return p.x, p.y, p.t, p.z
}

func (p *Point_xtw_full) XYTZ_projective() (FieldElement, FieldElement, FieldElement, FieldElement) {
	return p.x, p.y, p.t, p.z
}

func (p *point_xtw_base) X_decaf_projective() FieldElement {
	return p.x
}

func (p *point_xtw_base) Y_decaf_projective() FieldElement {
	return p.y
}

func (p *point_xtw_base) T_decaf_projective() FieldElement {
	return p.t
}

func (p *point_xtw_base) Z_decaf_projective() FieldElement {
	return p.z
}

func (p *point_xtw_base) X_decaf_affine() FieldElement {
	p.normalizeAffineZ()
	return p.x
}

func (p *point_xtw_base) Y_decaf_affine() FieldElement {
	p.normalizeAffineZ()
	return p.y
}

func (p *point_xtw_base) T_decaf_affine() FieldElement {
	p.normalizeAffineZ()
	return p.t
}

func (p *Point_xtw_full) IsInSubgroup() bool {
	return legendreCheckA_projectiveXZ(p.x, p.z) && legendreCheckE1_projectiveYZ(p.y, p.z)
}

// TODO
/*
// SerializeLong serialize the given point in long serialization format. err==nil iff everything worked OK.
func (p *Point_xtw) SerializeLong(output io.Writer) (bytes_written int, err error) {
	return default_SerializeLong(p, output)
}

// SerializeShort serialize the given point in short serialization format. err==nil iff everything worked OK.
func (p *Point_xtw) SerializeShort(output io.Writer) (bytes_written int, err error) {
	return default_SerializeShort(p, output)
}
*/

// TODO !

/*
// DeserializeShort deserialize from the given input byte stream (expecting it to start with a point in short serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeShort(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeShort(p, input, trusted)
}

// DeserializeLong deserialize from the given input byte stream (expecting it to start with a point in long serialization format) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeLong(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeLong(p, input, trusted)
}

// DeserializeAuto deserialize from the given input byte stream (expecting it to start with a point in either short or long serialization format -- it autodetects that) and store the result in the receiver.
// err==nil iff no error occured. trusted should be one of the constants TrustedInput or UntrustedInput.
// For UntrustedInput, we perform a specially-tailored efficient curve and subgroup membership tests.
// Note that long format is considerably more efficient to deserialize.
func (p *Point_xtw) DeserializeAuto(input io.Reader, trusted IsPointTrusted) (bytes_read int, err error) {
	return default_DeserializeAuto(p, input, trusted)
}

*/

// String prints the point in X:Y:T:Z - format
func (p *point_xtw_base) String() string {
	// Not the most efficient way to concatenate strings, but good enough.
	return p.x.String() + ":" + p.y.String() + ":" + p.t.String() + ":" + p.z.String()
}

func (p *Point_xtw_subgroup) String() (ret string) {
	ret = p.point_xtw_base.String()
	if !legendreCheckE1_projectiveYZ(p.y, p.z) {
		ret += " [+A]"
	}
	return
}

/*
// AffineExtended returns a copy of the point in affine extended coordinates.
func (p *Point_xtw) AffineExtended() Point_axtw {
	panic("Needs to change")
	p.normalizeAffineZ()
	return Point_axtw{x: p.x, y: p.y, t: p.t}
}
*/

func (p *point_xtw_base) ToDecaf_xtw() point_xtw_base {
	return *p
}

func (p *point_xtw_base) ToDecaf_axtw() point_axtw_base {
	p.normalizeAffineZ()
	return point_axtw_base{x: p.x, y: p.y, t: p.t}
}

/*
// ExtendedTwistedEdwards() returns a copy of the given point in extended twisted Edwards coordinates.
func (p *Point_xtw) ExtendedTwistedEdwards() Point_xtw {
	panic("Needs to change")
	return *p // Note that Go forces the caller to make a copy.
}
*/

func (p *point_xtw_base) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

func (p *Point_xtw_full) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

func (p *Point_xtw_subgroup) Clone() interface{} {
	p_copy := *p
	return &p_copy
}

// IsNeutralElement checks if the point P is the neutral element of the curve (modulo the identification of P with P+A).
// Use IsNeutralElement_FullCurve if you do not want this identification.
func (p *Point_xtw_subgroup) IsNeutralElement() bool {
	// The only point with x==0 are the neutral element N and the affine order-two point A, which we work modulo.
	if p.x.IsZero() {
		if p.y.IsZero() {
			return napEncountered("compared invalid xtw point to zero", true, p)
		}
		return true
	}
	if !p.t.IsZero() {
		panic("Non-NaP xtw with x==0, t!=0")
	}
	return false
}

func (p *Point_xtw_full) IsNeutralElement() bool {
	if !p.x.IsZero() {
		return false
	}
	if p.IsNaP() {
		return napEncountered("compared invalid xtw point to zero exactly", true, p)
	}
	if !p.t.IsZero() {
		panic("Non-NaP xtw point with x==0, but t!=0 encountered.")
	}
	// we know x==0, y!=0 (because otherwise, we have a NaP), t==0.
	// This implies z == +/- y
	return p.y.IsEqual(&p.z)
}

// SetNeutral sets the Point P to the neutral element of the curve.
func (p *point_xtw_base) SetNeutral() {
	*p = NeutralElement_xtw
}

// IsNaP checks whether the point is singular (x==y==0, indeed most likely x==y==t==z==0). Singular points must never appear if the library is used correctly. They can appear by
// a) performing operations on points that are not in the correct subgroup
// b) zero-initialized points are singular (Go lacks constructors to fix that).
// The reason why we check x==y==0 and do not check t,z is due to what happens if we perform mixed additions.
func (p *point_xtw_base) IsNaP() bool {
	return p.x.IsZero() && p.y.IsZero()
}

// z.Add(x,y) computes z = x+y according to the elliptic curve group law.
func (p *Point_xtw_subgroup) Add(x CurvePointPtrInterfaceRead, y CurvePointPtrInterfaceRead) {
	// TODO: Optimize. For certain edge cases, going directly to xtw is slightly more efficient.
	var result Point_efgh_subgroup
	result.Add(x, y)
	p.SetFrom(&result)

	/*
		panic(0)
		switch x := x.(type) {
		case *Point_xtw:
			switch y := y.(type) {
			case *Point_xtw:
				p.add_ttt(x, y)
			case *Point_axtw:
				p.add_tta(x, y)
			default:
				var y_converted Point_xtw = convertToPoint_xtw(y)
				p.add_ttt(x, &y_converted)
			}
		case *Point_axtw:
			switch y := y.(type) {
			case *Point_xtw:
				p.add_tta(y, x)
			case *Point_axtw:
				p.add_taa(x, y)
			default:
				var y_converted Point_xtw = convertToPoint_xtw(y)
				p.add_tta(&y_converted, x)

			}
		default: // for x
			var x_converted Point_xtw = convertToPoint_xtw(x)

			switch y := y.(type) {
			case *Point_xtw:
				p.add_ttt(&x_converted, y)
			case *Point_axtw:
				p.add_tta(&x_converted, y)
			default:
				var y_converted Point_xtw = convertToPoint_xtw(y)
				p.add_ttt(&x_converted, &y_converted)
			}
		}
	*/
}

func (p *Point_xtw_full) Add(x, y CurvePointPtrInterfaceRead) {
	var result_efgh Point_efgh_full
	result_efgh.Add(x, y)
	p.SetFrom(&result_efgh)
}

func (p *Point_xtw_subgroup) Sub(x, y CurvePointPtrInterfaceRead) {
	var result_efgh Point_efgh_subgroup
	result_efgh.Sub(x, y)
	p.SetFrom(&result_efgh)
}

func (p *Point_xtw_full) Sub(x, y CurvePointPtrInterfaceRead) {
	var result_efgh Point_efgh_full
	result_efgh.Sub(x, y)
	p.SetFrom(&result_efgh)
}

// z.Double(x) computes z = x+x according to the elliptic curve group law.
func (p *point_xtw_base) Double(input CurvePointPtrInterfaceRead) {
	var result_efgh point_efgh_base
	result_efgh.Double(input)
	*p = result_efgh.ToDecaf_xtw()
}

func (p *Point_xtw_subgroup) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

func (p *Point_xtw_full) Neg(input CurvePointPtrInterfaceRead) {
	p.SetFrom(input)
	p.NegEq()
}

func (p *Point_xtw_subgroup) Endo(input CurvePointPtrInterfaceRead) {
	var result_efgh Point_efgh_subgroup
	result_efgh.Endo(input)
	p.SetFrom(&result_efgh)
}

func (p *Point_xtw_full) Endo(input CurvePointPtrInterfaceRead) {
	var result_efgh Point_efgh_full
	result_efgh.Endo(input)
	p.SetFrom(&result_efgh)
}

func (p *point_xtw_base) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("checking whether NaP point is at infinity", false, p)
	}
	if p.z.IsZero() {
		// The only valid points (albeit not in subgroup) with z == 0 are the two exceptional points with z==y==0
		// We catch x==y==0 above (which already means the user of the library screwed up).

		// None of these can ever happen unless the library author messed up.
		if !p.y.IsZero() {
			panic("xtw point with z==0, but y!=0 encountered.")
		}
		// TODO: Remove?
		if p.t.IsZero() {
			panic("xtw point with z==t==0 encountered, but (x,y) != (0,0), so this was not NaP. This must never happen.")
		}
		// impossible, because y==0 and no NaP
		if p.x.IsZero() {
			panic("Non-NaP xtw point with z==0 and, y==0 and x==0 encountered. This is impossible")
		}
		return true
	}
	return false
}

func (p *Point_xtw_subgroup) IsAtInfinity() bool {
	if p.IsNaP() {
		return napEncountered("checking whether NaP point is at infinity", false, p)
	}
	return false
}

func (p *Point_xtw_subgroup) IsEqual(other CurvePointPtrInterfaceRead) bool {
	switch other := other.(type) {
	case *Point_xtw_subgroup:
		ret, potentialNaP := p.isEqual_moduloA_tt(&other.point_xtw_base)
		if potentialNaP && (p.IsNaP() || other.IsNaP()) {
			return napEncountered("NaP detected during comparison of xtw points", true, p, other)
		}
		return ret
	case *Point_xtw_full:
		p.normalizeSubgroup()
		ret, potentialNaP := p.isEqual_exact_tt(&other.point_xtw_base)
		if potentialNaP && (p.IsNaP() || other.IsNaP()) {
			return napEncountered("NaP detected during comparison of xtw_subgroup and xtw points", true, p, other)
		}
		return ret
	case *Point_axtw_subgroup:
		ret, potentialNaP := p.isEqual_moduloA_ta(&other.point_axtw_base)
		if potentialNaP && (p.IsNaP() || other.IsNaP()) {
			return napEncountered("NaP detected during comparison of xtw_subgroup and axtw_subgroup points", true, p, other)
		}
		return ret
	case *Point_axtw_full:
		p.normalizeSubgroup()
		if p.IsNaP() || other.IsNaP() {
			return napEncountered("NaP detected during comparison of xtw_subgroup and axtw_full points", true, p, other)
		}
		return p.isEqual_exact_ta(&other.point_axtw_base)
	default:
		if p.IsNaP() || other.IsNaP() {
			return napEncountered("NaP detected during comparison of xtw_subgroup and other point", true, p, other)
		}
		if other.CanOnlyRepresentSubgroup() {
			ret, _ := p.isEqual_moduloA_tany(other)
			return ret
		} else {
			p.normalizeSubgroup()
			return p.isEqual_exact_tany(other)
		}
	}
}

func (p *Point_xtw_full) IsEqual(other CurvePointPtrInterfaceRead) bool {
	if p.IsNaP() || other.IsNaP() {
		return napEncountered("NaP detected during comparison of xtw_full and other point", true, p, other)
	}
	switch other := other.(type) {
	case *Point_xtw_full:
		ret, _ := p.isEqual_exact_tt(&other.point_xtw_base)
		return ret
	case *Point_xtw_subgroup:
		other.normalizeSubgroup()
		ret, _ := p.isEqual_exact_tt(&other.point_xtw_base)
		return ret
	case *Point_axtw_full:
		return p.isEqual_exact_ta(&other.point_axtw_base)
	case *Point_axtw_subgroup:
		other.normalizeSubgroup()
		return p.isEqual_exact_ta(&other.point_axtw_base)
	default:
		if other.CanOnlyRepresentSubgroup() {
			ret, _ := p.isEqual_moduloA_tany(other)
			return ret
		} else {
			return p.isEqual_exact_tany(other)
		}
	}
}

// EndoEq applies the endomorphism on the given point. p.EndoEq() is shorthand for p.Endo(&p).
func (p *Point_xtw_subgroup) EndoEq() {
	p.Endo(p)
}

func (p *Point_xtw_full) EndoEq() {
	p.Endo(p)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_xtw_subgroup) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// AddEq adds (via the elliptic curve group addition law) the given curve point x (in any coordinate format) to the received p, overwriting p.
func (p *Point_xtw_full) AddEq(x CurvePointPtrInterfaceRead) {
	p.Add(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_xtw_subgroup) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// SubEq subtracts (via the elliptic curve group addition law) the given curve point x (in any coordinate format) from the received p, overwriting p.
func (p *Point_xtw_full) SubEq(x CurvePointPtrInterfaceRead) {
	p.Sub(p, x)
}

// DoubleEq doubles the received point p, overwriting p.
func (p *point_xtw_base) DoubleEq() {
	var result_efgh point_efgh_base
	result_efgh.double_st(p)
	*p = result_efgh.ToDecaf_xtw()
}

// NeqEq replaces the given point by its negative (wrt the elliptic curve group addition law)
func (p *point_xtw_base) NegEq() {
	p.x.NegEq()
	p.t.NegEq()
}

func (p *Point_xtw_subgroup) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw_subgroup:
		*p = *input
	case *Point_axtw_subgroup:
		p.x = input.x
		p.y = input.y
		p.t = input.t
	case *Point_efgh_subgroup:
		p.point_xtw_base = input.ToDecaf_xtw()
	default:
		ensureSubgroupOnly(input)
		p.x = input.X_decaf_projective()
		p.y = input.Y_decaf_projective()
		p.t = input.T_decaf_projective()
		p.z = input.Z_decaf_projective()
	}
}

// SetFrom initializes the point from the given input point (which may have a different coordinate format)
func (p *Point_xtw_full) SetFrom(input CurvePointPtrInterfaceRead) {
	switch input := input.(type) {
	case *Point_xtw_full:
		*p = *input
	case *Point_xtw_subgroup:
		input.normalizeSubgroup()
		p.point_xtw_base = input.point_xtw_base
	case *Point_axtw_full:
		p.x = input.x
		p.y = input.y
		p.t = input.t
		p.z.SetOne()
	case *Point_axtw_subgroup:
		input.normalizeSubgroup()
		p.x = input.x
		p.y = input.y
		p.t = input.t
		p.z.SetOne()
	case *Point_efgh_subgroup:
		input.normalizeSubgroup()
		p.point_xtw_base = input.ToDecaf_xtw()
	case *Point_efgh_full:
		p.point_xtw_base = input.ToDecaf_xtw()
	case CurvePointPtrInterfaceCooReadProjectiveT:
		p.x, p.y, p.t, p.z = input.XYTZ_projective()
	default:
		p.x, p.y, p.z = input.XYZ_projective()
		p.t.Mul(&p.x, &p.y)
		p.x.MulEq(&p.z)
		p.y.MulEq(&p.z)
		p.z.SquareEq()
	}
}

func (p *point_xtw_base) Validate() bool {
	return p.isPointOnCurve()
}

func (p *Point_xtw_subgroup) Validate() bool {
	return p.point_xtw_base.isPointOnCurve() && legendreCheckA_projectiveXZ(p.x, p.z)
}

func (p *Point_xtw_full) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_xtw_base = makeRandomPointOnCurve_t(rnd)
}

func (p *Point_xtw_subgroup) sampleRandomUnsafe(rnd *rand.Rand) {
	p.point_xtw_base = makeRandomPointOnCurve_t(rnd)
	p.point_xtw_base.DoubleEq()
}

func (p *Point_xtw_full) SetE1() {
	p.point_xtw_base = exceptionalPoint_1_xtw
}

func (p *Point_xtw_full) SetE2() {
	p.point_xtw_base = exceptionalPoint_2_xtw
}

func (p *Point_xtw_full) SetAffineTwoTorsion() {
	p.point_xtw_base = orderTwoPoint_xtw
}
