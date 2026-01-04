package fetch

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidRequestError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			name:        "simple error",
			err:         errors.New("test error"),
			wantMessage: "test error",
		},
		{
			name:        "empty error",
			err:         errors.New(""),
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ire := &InvalidRequestError{err: tt.err}
			assert.Equal(t, tt.wantMessage, ire.Error())
		})
	}
}

func TestInvalidRequestError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "unwrap simple error",
			err:     errors.New("wrapped error"),
			wantErr: errors.New("wrapped error"),
		},
		{
			name:    "unwrap nil",
			err:     nil,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ire := &InvalidRequestError{err: tt.err}
			unwrapped := ire.Unwrap()

			if tt.wantErr == nil {
				assert.Nil(t, unwrapped)
			} else {
				assert.Equal(t, tt.wantErr.Error(), unwrapped.Error())
			}
		})
	}
}
