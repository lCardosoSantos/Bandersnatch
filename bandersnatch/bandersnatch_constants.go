package bandersnatch

import "math/big"

// Curve parameters

// GroupOrder is the order of the p253-subgroup of the Bandersnatch curve.
// This is a 253-bit prime number.
const (
	GroupOrder        = 0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1
	GroupOrder_string = "0x1cfb69d4ca675f520cce760202687600ff8f87007419047174fd06b52876e7e1"
)

// Cofactor is the cofactor of the Bandersnatch group, i.e. the size of the quotient of the group of rational curve points by the prime-order subgroup.
// The structure of this group is Z/2 x Z/2
const (
	Cofactor        = 4
	Cofactor_string = "4"
)

// CurveOrder denotes the non-prime size of the group of rational points of the Bandersnatch curve.
const (
	CurveOrder        = 52435875175126190479447740508185965837236623573762281007145613226918750691204 // = Cofactor * GroupOrder
	CurveOrder_string = "52435875175126190479447740508185965837236623573762281007145613226918750691204"
)

// GroupOrder_Int is the order of the relevant prime order subgroup of the Bandersnatch curve as a *big.Int
var GroupOrder_Int *big.Int = initIntFromString(GroupOrder_string)

// Cofactor_Int is the cofactor of the Bandersnatch group as a *big.Int
var Cofactor_Int *big.Int = big.NewInt(Cofactor)

// CurveOrder_Int is the (non-prime) order of the group of rational points of the Bandersnatch curve as a *big.Int
var CurveOrder_Int *big.Int = new(big.Int).Mul(GroupOrder_Int, Cofactor_Int)

// EndomorphismEigenvalue is a number, such that the efficient degree-2 endomorphism acts as multiplication by this constant on the p253-subgroup.
// This is a square root of -2 modulo GroupOrder
const (
	EndomorphismEivenvalue        = 0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05
	EndomorphismEigenvalue_string = "0x13b4f3dc4a39a493edf849562b38c72bcfc49db970a5056ed13d21408783df05"
)

const endomorphismEigenvalueIsOdd = true // we chose an odd representative above. This info is needed to get some test right.

// EndomorphismEigenvalue_Int is a *big.Int, such that the the efficient degree-2 endomorphism of the Bandersnatch curve acts as multiplication by this constant on the p253-subgroup.
var EndomorphismEigenvalue_Int *big.Int = initIntFromString(EndomorphismEigenvalue_string)

// parameters a, d in twisted Edwards form ax^2 + y^2 = 1 + dx^2y^2

// Note: both a and d are non-squares

// CurveParameterA denotes the constant a in the twisted Edwards representation ax^2+y^2 = 1+dx^2y^2 of the Bandersnatch curve
const (
	CurveParameterA        = -5
	CurveParameterA_string = "-5"
)

// CurveParameterD denotes the constant d in the twisted Edwards representation ax^2+y^2 = 1+dx^2y^2 of the Bandersnatch curve.
// Note that d == -15 - 10\sqrt{2}
const (
	CurveParameterD        = 0x6389c12633c267cbc66e3bf86be3b6d8cb66677177e54f92b369f2f5188d58e7
	CurveParameterD_string = "0x6389c12633c267cbc66e3bf86be3b6d8cb66677177e54f92b369f2f5188d58e7"
)

// CurveParameters as *big.Int's or FieldElements
var (
	CurveParameterD_Int *big.Int     = initIntFromString(CurveParameterD_string)
	CurveParameterD_fe  FieldElement = initFieldElementFromString(CurveParameterD_string)
	CurveParameterA_Int *big.Int     = initIntFromString(CurveParameterA_string)
	CurveParameterA_fe  FieldElement = initFieldElementFromString(CurveParameterA_string)
)

// squareRootDByA is a square root of d/a. Due to the way the bandersnatch curve was constructed, we have (sqrt(d/a) + 1)^2 == 2.
// This number appears in coordinates of the order-2 points at inifinity and in the formulae for the endomorphism.
// Note that there are two square roots of d/a; be sure to make consistent choices.
const (
	squareRootDByA        = 37446463827641770816307242315180085052603635617490163568005256780843403514038
	squareRootDByA_string = "37446463827641770816307242315180085052603635617490163568005256780843403514038"
)

// const, really
var (
	// squareRootDbyA_Int *big.Int     = initIntFromString(squareRootDByA_string) // TODO: Do we need this?
	squareRootDbyA_fe FieldElement = initFieldElementFromString(squareRootDByA_string)
)

// These parameters appear in the formulae for the endomorphism.
const (
	// endo_a1              = 0x23c58c92306dbb95960f739827ac195334fcd8fa17df036c692f7ddaa306c7d4
	// endo_a1_string       = "0x23c58c92306dbb95960f739827ac195334fcd8fa17df036c692f7ddaa306c7d4"
	// endo_a2              = 0x23c58c92306dbb96b0b30d3513b222f50d02d8ff03e5036c69317ddaa306c7d4
	// endo_a2_string       = "0x23c58c92306dbb96b0b30d3513b222f50d02d8ff03e5036c69317ddaa306c7d4"
	endo_b               = 0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b4 // == sqrt(2) - 1 == sqrt(a/d)
	endo_b_string        = "0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b4"
	endo_binverse        = 0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b6 // =1/endo_b == endo_b + 2 == sqrt(d/a). Equals sqrtDByA
	endo_binverse_string = "0x52c9f28b828426a561f00d3a63511a882ea712770d9af4d6ee0f014d172510b6"
	endo_bcd_string      = "36255886417209629651405037489028103282266637240540121152239675547668312569901" // == endo_b * endo_c * CurveParameterD
	endo_c               = 0x6cc624cf865457c3a97c6efd6c17d1078456abcfff36f4e9515c806cdf650b3d
	endo_c_string        = "0x6cc624cf865457c3a97c6efd6c17d1078456abcfff36f4e9515c806cdf650b3d"
	// endo_c1 == - endo_b
	//c1 = 0x2123b4c7a71956a2d149cacda650bd7d2516918bf263672811f0feb1e8daef4d
)

var (
	// endo_a1_fe       FieldElement = initFieldElementFromString(endo_a1_string)
	// endo_a2_fe       FieldElement = initFieldElementFromString(endo_a2_string)
	endo_b_fe        FieldElement = initFieldElementFromString(endo_b_string)
	endo_c_fe        FieldElement = initFieldElementFromString(endo_c_string)
	endo_binverse_fe FieldElement = initFieldElementFromString(endo_binverse_string) // Note == SqrtDDivA_fe
	endo_bcd_fe      FieldElement = initFieldElementFromString(endo_bcd_string)
)

/*

 */

// The point here is to force users to write Deserialize(..., TrustedInput, ...) rather than Deserialize(..., true, ...)
// in order to have better understandable semantics
// Golang does not have enum types, sadly, so we need to use structs: declaring a "type InPointTrusted bool" would cause Deserialze(..., true, ...)  to actually work due to implicit conversion.

// IsPointTrusted is a struct encapsulating a bool controlling whether some input is trusted or not.
// This is used to enforce better readable semantics in arguments.
// Users should use the predefined values TrustedInput and UntrustedInput of this type.
type IsPointTrusted struct {
	v bool
}

func (b IsPointTrusted) Bool() bool { return b.v }

// TrustedInput and UntrustedInput are used as arguments to Deserialization routines and in ToSubgroup.
var (
	TrustedInput   IsPointTrusted = IsPointTrusted{v: true}
	UntrustedInput IsPointTrusted = IsPointTrusted{v: false}
)