package domain

import "testing"

func TestRegisterCmdValidateRejectsInvalidContactFormats(t *testing.T) {
	tests := []struct {
		name string
		cmd  RegisterCmd
	}{
		{
			name: "email without dotted domain",
			cmd:  RegisterCmd{Email: "n0_s7st3m_ar3_S@F3"},
		},
		{
			name: "short non e164 phone",
			cmd:  RegisterCmd{Phone: "+12"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cmd.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
