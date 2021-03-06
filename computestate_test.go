package hoff

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_ComputeState_Call(t *testing.T) {
	testCases := []struct {
		name                  string
		givenComputeStateCall func() ComputeState
		expectedState         StateType
		expectedNodeBranch    *bool
		expectedError         error
		expectedString        string
	}{
		{
			name:                  "Should generate a continue state",
			givenComputeStateCall: func() ComputeState { return NewContinueComputeState() },
			expectedState:         ContinueState,
			expectedString:        "'Continue'",
		},
		{
			name:                  "Should generate a continue state on branch 'true'",
			givenComputeStateCall: func() ComputeState { return NewContinueOnBranchComputeState(true) },
			expectedState:         ContinueState,
			expectedNodeBranch:    boolPointer(true),
			expectedString:        "'Continue on true'",
		},
		{
			name:                  "Should generate a skip state",
			givenComputeStateCall: func() ComputeState { return NewSkipComputeState() },
			expectedState:         SkipState,
			expectedString:        "'Skip'",
		},
		{
			name:                  "Should generate a abort state",
			givenComputeStateCall: func() ComputeState { return NewAbortComputeState(errors.New("error")) },
			expectedState:         AbortState,
			expectedError:         errors.New("error"),
			expectedString:        "'Abort on error'",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			computeState := testCase.givenComputeStateCall()
			if computeState.Value != testCase.expectedState {
				t.Errorf("state - got: %+v, want: %+v", computeState.Value, testCase.expectedState)
			}
			if !cmp.Equal(computeState.Branch, testCase.expectedNodeBranch) {
				t.Errorf("branch - got: %+v, want: %+v", computeState.Branch, testCase.expectedNodeBranch)
			}
			if !cmp.Equal(computeState.Error, testCase.expectedError, errorComparator) {
				t.Errorf("error - got: %+v, want: %+v", computeState.Error, testCase.expectedError)
			}
			if computeState.String() != testCase.expectedString {
				t.Errorf("string - got: %+v, want: %+v", computeState.String(), testCase.expectedString)
			}
		})
	}
}
