package copy

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestCatalogsMatchDeclaredKeys(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "copy.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	keys := make(map[Key]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok || len(value.Values) != 1 {
			return true
		}
		literal, ok := value.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		key, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		keys[Key(key)] = true
		return true
	})
	for locale, catalog := range catalogs {
		if len(keys) != len(catalog.messages) {
			t.Errorf("%s: keys=%d messages=%d", locale, len(keys), len(catalog.messages))
		}
		for key := range keys {
			if strings.TrimSpace(catalog.Text(key)) == "" {
				t.Errorf("%s: empty %s", locale, key)
			}
		}
		for key := range catalog.messages {
			if !keys[key] {
				t.Errorf("%s: undeclared %s", locale, key)
			}
		}
	}
}

func TestCatalogRejectsInvalidData(t *testing.T) {
	for _, data := range []string{
		`[]`, `{"a":"one","a":"two"}`, `{"a":""}`, `{"":"value"}`,
		`{"a":42}`, `{"a":"{value"}`, `{"a":"value}"}`, `{"a":"{}"}`,
		`{"a":"{nested{value}}"}`, `{"a":"ok"} {}`, `{"a":"ok"`,
	} {
		t.Run(data, func(t *testing.T) {
			if _, err := decodeCatalog(data); err == nil {
				t.Fatal("invalid catalog accepted")
			}
		})
	}
}

func TestTranslationRejectsMissingExtraAndChangedPlaceholders(t *testing.T) {
	base := map[Key]string{"a": "{first} then {second}"}
	for _, translation := range []map[Key]string{
		{}, {"b": "{first} {second}"}, {"a": "{first}"},
		{"a": "{first} {second}", "extra": "extra"}, {"a": "{first} {second} {second}"},
	} {
		if err := validateTranslation(base, translation); err == nil {
			t.Fatalf("invalid translation accepted: %v", translation)
		}
	}
	if err := validateTranslation(base, map[Key]string{"a": "{second} then {first}"}); err != nil {
		t.Fatal(err)
	}
}

func TestFormatting(t *testing.T) {
	for _, tc := range []struct {
		locale                          Locale
		progress, millis, version, step string
		available, updateTo             string
	}{
		{English, "63 MB of 134 MB", "312 ms", "Version 0.1.0", "Step 2 of 7", "Version 0.2.0 is available", "Update to 0.2.0"},
		{SimplifiedChinese, "63 MB / 134 MB", "312 毫秒", "版本 0.1.0", "第 2 步，共 7 步", "新版本 0.2.0 可用", "更新到 0.2.0"},
	} {
		t.Run(string(tc.locale), func(t *testing.T) {
			c := For(tc.locale)
			for _, pair := range [][2]string{
				{c.Progress(63<<20, 134<<20), tc.progress}, {c.Millis(312.4), tc.millis},
				{c.Version("0.1.0"), tc.version}, {c.Step(2, 7), tc.step},
				{c.UpdateAvailable("0.2.0"), tc.available}, {c.UpdateTo("0.2.0"), tc.updateTo},
				{c.Percent(0.041), "4.1%"}, {c.Bytes(1 << 30), "1.0 GB"},
				{c.Bytes(0), "0 MB"}, {c.Bytes(1 << 19), "1 MB"}, {c.Text(""), ""},
			} {
				if pair[0] != pair[1] {
					t.Errorf("got %q, want %q", pair[0], pair[1])
				}
			}
			if got := c.Version("{value}%s"); !strings.Contains(got, "{value}%s") {
				t.Fatalf("value reinterpreted as template: %q", got)
			}
		})
	}
}

func TestCatalogsAreIndependent(t *testing.T) {
	en, zh := For(English), For(SimplifiedChinese)
	if en.Text(Continue) != "Continue" || zh.Text(Continue) != "继续" || For("").Text(Continue) != "Continue" {
		t.Fatal("locale snapshot changed")
	}
	for _, action := range []func(){func() { For("fr") }, func() { en.Text("missing") }} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid lookup must fail explicitly")
				}
			}()
			action()
		}()
	}
}

// This gate covers direct literals in common copy-bearing APIs. User content,
// device/model names and physical key identifiers remain runtime data.
func TestNoHardcodedDesktopCopy(t *testing.T) {
	for _, root := range []string{"../..", "../../../bootstrap"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, path, nil, 0)
			if err != nil {
				return err
			}
			for _, position := range hardcodedCopy(file) {
				t.Errorf("%s: user-facing literals must use copy keys", set.Position(position))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func hardcodedCopy(file *ast.File) []token.Pos {
	var positions []token.Pos
	check := func(expression ast.Expr) {
		literal, ok := expression.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil && strings.TrimSpace(value) != "" {
			positions = append(positions, literal.Pos())
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok {
			if kind, ok := literal.Type.(*ast.SelectorExpr); ok {
				if owner, ok := kind.X.(*ast.Ident); ok && owner.Name == "mygo" {
					for _, element := range literal.Elts {
						if field, ok := element.(*ast.KeyValueExpr); ok {
							if name, ok := field.Key.(*ast.Ident); ok && (name.Name == "Title" || name.Name == "Label" || name.Name == "ToolTip") {
								check(field.Value)
							}
						}
					}
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if owner, ok := selector.X.(*ast.Ident); ok {
			indices := map[string]int{
				"ui.Text": 1, "ui.Button": 1, "ui.Radio": 3,
				"kit.Button": 3, "kit.IconButton": 3, "kit.Group": 2, "kit.Status": 4,
			}
			if index, ok := indices[owner.Name+"."+selector.Sel.Name]; ok && len(call.Args) > index {
				check(call.Args[index])
			}
			if owner.Name == "kit" && selector.Sel.Name == "Row" && len(call.Args) > 3 {
				check(call.Args[2])
				check(call.Args[3])
			}
		}
		switch selector.Sel.Name {
		case "Label", "Placeholder", "Tooltip", "ToolTip", "SetTitle", "SetToolTip":
			if len(call.Args) > 0 {
				check(call.Args[0])
			}
		}
		return true
	})
	return positions
}

func TestHardcodedCopyGateFindsLabels(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", `package fixture
	func render() {
	 ui.Text(c, "Direct text")
	 ui.Text(c, messages.Text(copy.Title))
	 ui.Text(c, transcript)
	 ui.TextInput(c, &value).Placeholder("Direct hint").Label("Direct label")
	 kit.Row(c, tokens, "Direct title", "", action)
	 c.Key("internal-key")
	 _ = mygo.WindowOptions{Title: "Direct native title"}
	}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(hardcodedCopy(file)); got != 5 {
		t.Fatalf("found %d literals, want 5", got)
	}
}

func TestNativePermissionMetadata(t *testing.T) {
	data, err := os.ReadFile("../../../mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MacOS struct {
			InfoPlist struct {
				Microphone string   `json:"NSMicrophoneUsageDescription"`
				Region     string   `json:"CFBundleDevelopmentRegion"`
				Languages  []string `json:"CFBundleLocalizations"`
			} `json:"infoPlist"`
		} `json:"macos"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.MacOS.InfoPlist.Microphone != For(English).Text(SystemMicrophoneUsage) {
		t.Fatal("bundle default permission copy differs from catalog")
	}
	if config.MacOS.InfoPlist.Region != "en" || !slices.Equal(config.MacOS.InfoPlist.Languages, []string{"en", "zh-Hans"}) {
		t.Fatal("bundle locale declarations differ from staged localizations")
	}
}
