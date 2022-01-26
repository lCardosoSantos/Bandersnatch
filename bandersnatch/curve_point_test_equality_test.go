package bandersnatch

import (
	"fmt"
	"testing"
)

// checks whether IsNeutralElement correctly recognized neutral elements
func checkfun_recognize_neutral(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(1)
	var singular = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.Flags[0].CheckFlag(Case_zero_exact) && !singular
	return guardForInvalidPoints(expected, singular, "Neutral point not recognized", s.Points[0].IsNeutralElement)
}

// checks whether IsEqual correctly recognizes pairs of equal points
func checkfun_recognize_equality(s *TestSample) (bool, string) {
	s.AssertNumberOfPoints(2)
	var singular bool = s.AnyFlags().CheckFlag(Case_singular)
	var expected bool = s.AnyFlags().CheckFlag(Case_equal_exact) && !singular
	result1, result2 := guardForInvalidPoints(expected, singular, "equality testing failed", s.Points[0].IsEqual, s.Points[1])
	if !result1 {
		fmt.Println(expected)
	}
	return result1, result2
}

func test_equality_properties(t *testing.T, receiverType PointType, excludedFlags PointFlags) {
	point_string := PointTypeToString(receiverType)
	make_samples1_and_run_tests(t, checkfun_recognize_neutral, "Did not recognize neutral element for "+point_string, receiverType, 10, excludedFlags)
	// make_samples1_and_run_tests(t, checkfun_recognize_neutral_exact, "Did not recognize exact neutral element for "+point_string, receiverType, 10, excludedFlags)
	make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality "+point_string, receiverType, receiverType, 10, excludedFlags)
	// make_samples2_and_run_tests(t, checkfun_recognize_equality_exact, "Did not recognize exact equality "+point_string, receiverType, receiverType, 10, excludedFlags)
	for _, type1 := range allTestPointTypes {
		if type1 == receiverType {
			continue // already checked
		}
		other_string := PointTypeToString(type1)
		make_samples2_and_run_tests(t, checkfun_recognize_equality, "Did not recognize equality for "+point_string+" and "+other_string, receiverType, type1, 10, excludedFlags)
		// make_samples2_and_run_tests(t, checkfun_recognize_equality_exact, "Did not recognize exact equality "+point_string, receiverType, type1, 10, excludedFlags)
	}
}

func TestEqualityForXTW(t *testing.T) {
	test_equality_properties(t, pointTypeXTWSubgroup, excludeNoPoints)
	test_equality_properties(t, pointTypeXTWFull, excludeNoPoints)
}

func TestEqualityForAXTW(t *testing.T) {
	test_equality_properties(t, pointTypeAXTWSubgroup, excludeNoPoints)
	test_equality_properties(t, pointTypeAXTWFull, excludeNoPoints)
}

func TestEqualityForEFGH(t *testing.T) {
	test_equality_properties(t, pointTypeEFGHSubgroup, excludeNoPoints)
	test_equality_properties(t, pointTypeEFGHFull, excludeNoPoints)
}
