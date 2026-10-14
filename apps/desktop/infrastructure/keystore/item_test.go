package keystore

import (
	"strings"
	"testing"
)

func TestRefreshItemIsNotTheVocabularyKey(t *testing.T) {
	if Account == VocabularyKeyAccount {
		t.Fatal("refresh token account collides with the vocabulary key")
	}
	if strings.Contains(Target(), VocabularyKeyAccount) {
		t.Fatal("credential target names the vocabulary key")
	}
	for _, args := range [][]string{PutArgs("secret"), GetArgs(), DeleteArgs()} {
		if !hasPair(args, "-s", Service) || !hasPair(args, "-a", Account) {
			t.Fatalf("args = %q, want service and account", args)
		}
		for _, arg := range args {
			if arg == VocabularyKeyAccount {
				t.Fatalf("args name the vocabulary key: %q", args)
			}
		}
	}
	if MaxCredentialBlob != 2560 {
		t.Fatalf("blob limit = %d", MaxCredentialBlob)
	}
}

func hasPair(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
