package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleProvisioningProfile(t *testing.T) {
	for _, name := range []string{"", "profiles/App.provisionprofile", "absolute", "missing", "directory"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, map[string]string{"app": "executable", "profiles/App.provisionprofile": "signed profile bytes"})
			profile := name
			if name == "absolute" {
				profile = filepath.Join(root, "profiles", "App.provisionprofile")
			} else if name == "directory" {
				profile = "profiles"
			}
			data, err := json.Marshal(map[string]any{"name": "App", "macos": map[string]string{"provisioningProfile": profile}})
			if err != nil {
				t.Fatal(err)
			}
			writeFiles(t, root, map[string]string{jsonConfig: string(data)})
			c, err := loadConfig(root)
			if err != nil {
				t.Fatal(err)
			}
			app, err := writeBundle(c, t.TempDir(), filepath.Join(root, "app"), nil, nil, nil)
			if name == "missing" || name == "directory" {
				if err == nil || !strings.Contains(err.Error(), "provisioning profile") {
					t.Fatalf("invalid profile: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(app, "Contents", "embedded.provisionprofile"))
			if profile == "" {
				if !os.IsNotExist(err) {
					t.Fatalf("unconfigured profile was embedded: %v", err)
				}
			} else if err != nil || string(got) != "signed profile bytes" {
				t.Fatalf("embedded profile = %q, %v", got, err)
			}
		})
	}
}

func TestProvisioningProfileBuildInputs(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c := &Config{root: root, Main: "./cmd/mygo"}
	if err := json.Unmarshal([]byte(`{"macos":{"provisioningProfile":"App.provisionprofile"}}`), c); err != nil {
		t.Fatal(err)
	}
	in, err := listBuildInputs(c)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, "App.provisionprofile")
	found := false
	for _, f := range in.files {
		found = found || f == profile
	}
	if !found {
		t.Fatal("the provisioning profile is not watched")
	}
	root = t.TempDir()
	c.root = root
	writeFiles(t, root, map[string]string{"App.provisionprofile": "first profile"})
	var before, after bytes.Buffer
	if err := hashEntitlements(&before, c); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"App.provisionprofile": "renewed profile"})
	if err := hashEntitlements(&after, c); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatal("renewing a profile does not change the development build fingerprint")
	}
	if err := os.Remove(filepath.Join(root, "App.provisionprofile")); err != nil {
		t.Fatal(err)
	}
	if err := hashEntitlements(&after, c); err == nil {
		t.Fatal("a missing profile was hashed")
	}
}

func TestProvisioningProfileNeedsSigningIdentity(t *testing.T) {
	c := &Config{}
	if err := json.Unmarshal([]byte(`{"macos":{"provisioningProfile":"App.provisionprofile"}}`), c); err != nil {
		t.Fatal(err)
	}
	if err := codesign(c, t.TempDir(), "-", true); err != errProfileAdHoc {
		t.Fatalf("ad-hoc signing with a profile: %v", err)
	}
	// dev does not read macos.signingIdentity.
	if err := codesign(c, t.TempDir(), "-", false); err == nil || !strings.Contains(err.Error(), "-sign") || strings.Contains(err.Error(), "macos.signingIdentity") {
		t.Fatalf("ad-hoc development signing with a profile: %v", err)
	}
}
