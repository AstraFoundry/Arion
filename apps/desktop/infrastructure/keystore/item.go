// Package keystore stores the hosted-sync refresh token in the OS credential
// store: the login Keychain on macOS and Credential Manager on Windows.
//
// Production items use service "Arion", account "account-refresh-token/<profile fingerprint>";
// cloud polishing keeps one item per provider at "cloud-polish/<provider>"
// (ADR 022). Account "cloud-polish" stores only which provider is active.
// A previous build stored the single credential in "cloud-polish"; cloudpolish
// copies that secret to the provider item on load.
// Unscoped refresh-token helpers address legacy account items only; production
// never reads or migrates those. The vocabulary
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
	// CloudPolishAccount is the active cloud-polishing provider. The item
	// JSON is {"provider":"<id>"} with no secret. CloudPolishProviderAccount
	// holds that provider's API key or ChatGPT refresh token.
	CloudPolishAccount = "cloud-polish"
	// MaxCredentialBlob is the Windows Credential Manager limit.
	MaxCredentialBlob = 5 * 512
)

// Target is the Windows credential name.
func Target() string { return Store{}.target() }

func (s Store) target() string { return s.serviceName() + "/" + s.accountName() }

func (s Store) accountName() string {
	if s.item != "" {
		return s.item
	}
	if s.profile == "" {
		return Account
	}
	return Account + "/" + s.profile
}

// PutArgs is the security(1) argument vector for a create-or-update.
func PutArgs(secret string) []string { return Store{}.putArgs(secret) }

func (s Store) putArgs(secret string) []string {
	return []string{"add-generic-password", "-U", "-s", s.serviceName(), "-a", s.accountName(), "-w", secret}
}

// GetArgs is the security(1) argument vector for a read.
func GetArgs() []string { return Store{}.getArgs() }

func (s Store) getArgs() []string {
	return []string{"find-generic-password", "-s", s.serviceName(), "-a", s.accountName(), "-w"}
}

// DeleteArgs is the security(1) argument vector for deletion.
func DeleteArgs() []string { return Store{}.deleteArgs() }

func (s Store) deleteArgs() []string {
	return []string{"delete-generic-password", "-s", s.serviceName(), "-a", s.accountName()}
}

func rejectEmpty(token string) error {
	if token == "" {
		return &account.Error{Code: account.CodeStore, Detail: "empty"}
	}
	return nil
}

// Open returns the platform store. It does not prompt or dial.
func Open() Store { return Store{} }

// Store is one platform credential item: the refresh token unless opened
// with OpenItem.
type Store struct{ profile, item, service string }

// WithService isolates a build's credentials while preserving the item identity.
func (s Store) WithService(service string) Store {
	s.service = service
	return s
}

func (s Store) serviceName() string {
	if s.service != "" {
		return s.service
	}
	return Service
}

// OpenForProfile isolates refresh tokens by the compiled public profile fingerprint.
func OpenForProfile(fingerprint string) Store { return Store{profile: fingerprint} }

// OpenItem addresses another item of the Arion service by account name, such
// as CloudPolishAccount. Delete removes that item only.
func OpenItem(account string) Store { return Store{item: account} }

// CloudPolishProviderAccount is one cloud provider's credential item.
func CloudPolishProviderAccount(provider string) string {
	return CloudPolishAccount + "/" + provider
}

// Catalog addresses every Arion credential item by account name.
type Catalog struct {
	// Service separates development credentials from installed releases.
	Service string
}

func (c Catalog) Put(account, secret string) error {
	return OpenItem(account).WithService(c.Service).Put(secret)
}
func (c Catalog) Get(account string) (string, error) {
	return OpenItem(account).WithService(c.Service).Get()
}
func (c Catalog) Delete(account string) error {
	return OpenItem(account).WithService(c.Service).Delete()
}
