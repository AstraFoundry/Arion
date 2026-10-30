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

func TestCenterProfilesHaveSeparateCredentials(t *testing.T) {
	a, b := OpenForProfile("profile-a"), OpenForProfile("profile-b")
	if a.target() == b.target() || a.target() == Target() {
		t.Fatal("profile credential collision")
	}
	for _, args := range [][]string{a.putArgs("secret"), a.getArgs(), a.deleteArgs()} {
		if !hasPair(args, "-a", Account+"/profile-a") {
			t.Fatalf("unscoped credential args: %q", args)
		}
		if hasPair(args, "-a", VocabularyKeyAccount) {
			t.Fatal("center profile touched vocabulary key")
		}
	}
}

func TestCloudPolishItemIsSeparate(t *testing.T) {
	cloud := OpenItem(CloudPolishAccount)
	if cloud.target() == Target() || cloud.target() == OpenForProfile("p").target() {
		t.Fatal("cloud credential shares the refresh-token item")
	}
	for _, args := range [][]string{cloud.putArgs("secret"), cloud.getArgs(), cloud.deleteArgs()} {
		if !hasPair(args, "-s", Service) || !hasPair(args, "-a", CloudPolishAccount) {
			t.Fatalf("args = %q", args)
		}
	}
	deep := OpenItem(CloudPolishProviderAccount("deepseek"))
	route := OpenItem(CloudPolishProviderAccount("openrouter"))
	if deep.target() == route.target() || deep.target() == cloud.target() || deep.target() == Target() {
		t.Fatal("provider credentials collide")
	}
	for _, args := range [][]string{deep.putArgs("secret"), deep.getArgs(), deep.deleteArgs()} {
		if !hasPair(args, "-s", Service) || !hasPair(args, "-a", "cloud-polish/deepseek") {
			t.Fatalf("args = %q", args)
		}
	}
}

func TestDevelopmentCredentialsCannotAddressProductionItems(t *testing.T) {
	for _, production := range []Store{OpenForProfile("center"), OpenItem(CloudPolishAccount), OpenItem(CloudPolishProviderAccount("chatgpt"))} {
		development := production.WithService("Arion Dev")
		if development.target() == production.target() {
			t.Fatal("development can change production credentials")
		}
		for _, args := range [][]string{development.putArgs("synthetic"), development.getArgs(), development.deleteArgs()} {
			if !hasPair(args, "-s", "Arion Dev") || !hasPair(args, "-a", production.accountName()) {
				t.Fatalf("development item escaped its service: %q", args)
			}
		}
	}
}
