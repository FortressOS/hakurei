package stub_test

import (
	"reflect"
	"testing"

	"hakurei.app/container/stub"
)

func TestCallError(t *testing.T) {
	t.Parallel()

	t.Run("contains false", func(t *testing.T) {
		t.Parallel()
		if err := new(stub.Call).Error(true, false, true); !reflect.DeepEqual(err, stub.ErrCheck) {
			t.Errorf("Error: %#v, want %#v", err, stub.ErrCheck)
		}
	})

	t.Run("passthrough", func(t *testing.T) {
		t.Parallel()
		wantErr := stub.UniqueError(0xbabe)
		if err := (&stub.Call{Err: wantErr}).Error(true); !reflect.DeepEqual(err, wantErr) {
			t.Errorf("Error: %#v, want %#v", err, wantErr)
		}
	})
}
