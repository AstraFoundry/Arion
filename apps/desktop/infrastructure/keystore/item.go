// Package keystore stores the hosted-sync refresh token in the OS credential
// store: the login Keychain on macOS and Credential Manager on Windows.
//
// The item is service "Arion", account "account-refresh-token". The vocabulary
// encryption key is a different account ("vocabulary-key") created by
// infrastructure/secrets. Delete addresses only the refresh-token account.
//
// macOS uses /usr/bin/security, the same mechanism as
// github.com/zalando/go-keyring, instead of adding that module. The secret is
// an argument of that process for the call, visible to other processes of the
// same user; at rest it is a Keychain generic password. The vocabulary key
// stays on the cgo SecItem path in infrastructure/secrets so this package
// cannot delete it, and so GOOS=darwin go vet does not need the macOS SDK.
// Windows calls CredWrite, CredRead, and CredDelete. Credential Manager blobs
// are at most 2560 bytes; a longer token is an error.
package keystore

import "arion.local/desktop/internal/account"

const (
	// Service is the credential service. It matches the vocabulary key's
	// service name; Account keeps the items distinct.
	Service = "Arion"
	// Account is the refresh-token item sign-out deletes.
	Account = "account-refresh-token"
	// VocabularyKeyAccount is the ADR 005 encryption key. This package never
	// reads or writes it.
	VocabularyKeyAccount = "vocabulary-key"
	// MaxCredentialBlob is the Windows Credential Manager limit.
	MaxCredentialBlob = 5 * 512
)

// Target is the Windows credential name.
func Target() string { return Service + "/" + Account }

// PutArgs is the security(1) argument vector for a create-or-update.
func PutArgs(secret string) []string {
	return []string{"add-generic-password", "-U", "-s", Service, "-a", Account, "-w", secret}
}

// GetArgs is the security(1) argument vector for a read.
func GetArgs() []string {
	return []string{"find-generic-password", "-s", Service, "-a", Account, "-w"}
}

// DeleteArgs is the security(1) argument vector for deletion.
func DeleteArgs() []string {
	return []string{"delete-generic-password", "-s", Service, "-a", Account}
}

func rejectEmpty(token string) error {
	if token == "" {
		return &account.Error{Code: account.CodeStore, Detail: "empty"}
	}
	return nil
}

// Open returns the platform store. It does not prompt or dial.
func Open() Store { return Store{} }

// Store is the platform refresh-token item.
type Store struct{}
