package bootstrap

import (
	"errors"
	"testing"

	"arion.local/desktop/internal/account"
	arion "arion.local/desktop/ui/shared/runtime"
)

// A credential-store read lands only while it is still the current check;
// a failed read is an error, not "nothing stored".
func TestFinishCredential(t *testing.T) {
	unreadable := account.Wrap(account.CodeStore, "read", errors.New("keychain locked"))
	for _, tc := range []struct {
		name   string
		stale  bool
		stored bool
		err    error
		want   arion.AccountCredential
		code   string
	}{
		{"stored", false, true, nil, arion.CredentialStored, ""},
		{"absent", false, false, nil, arion.CredentialAbsent, ""},
		{"unreadable", false, false, unreadable, arion.CredentialUnreadable, string(account.CodeStore)},
		{"stale", true, true, nil, arion.CredentialUnknown, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{}
			a.state.Account = arion.Account{Phase: arion.AccountSignedOut, Credential: arion.CredentialChecking}
			seq := a.accountGeneration()
			if tc.stale {
				a.beginAccount()
			}
			a.finishCredential(seq, tc.stored, tc.err)
			if got := a.state.Account; got.Credential != tc.want || got.ErrorCode != tc.code {
				t.Fatalf("account = %+v", got)
			}
		})
	}
}
