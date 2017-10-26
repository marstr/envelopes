package envelopes_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
)

func TestState_MarshalJSON(t *testing.T) {
	testCases := []struct {
		envelopes.State
		want string
	}{
		{envelopes.State{}, `{"budget":{"name":"","balance":0},"accounts":null,"parent":"0000000000000000000000000000000000000000"}`},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got, err := json.Marshal(tc.State); err != nil {
				t.Error(err)
			} else if cast := string(got); cast != tc.want {
				t.Logf("\ngot:  %q\nwant: %q", cast, tc.want)
				t.Fail()
			}
		})
	}
}

func TestState_ID(t *testing.T) {
	testCases := []struct {
		envelopes.State
		expected [20]byte
	}{
		// Calculated here: https://play.golang.org/p/r_q6EAZ-MT
		{envelopes.State{}, [20]byte{41, 249, 117, 10, 100, 40, 62, 194, 174, 210, 230, 58, 103, 204, 146, 64, 123, 79, 180, 179}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprint(tc.State), func(t *testing.T) {

		})
	}
}
