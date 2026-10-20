package domain

import "strings"

// Scene is where a dictation will be typed, which sets the layout of its
// rewrite (009). A scene changes layout only; every scene keeps the
// dictated meaning.
type Scene string

const (
	// SceneChat is general typing and personal messages: one plain paragraph.
	SceneChat Scene = "chat"
	// SceneEmail lays out greeting, body and closing.
	SceneEmail Scene = "email"
	// SceneWork is a workplace message; enumerated points become numbered
	// lines.
	SceneWork Scene = "work"
	// ScenePrompt is an instruction for a coding agent or AI assistant;
	// enumerated requirements become numbered lines.
	ScenePrompt Scene = "prompt"
	// SceneSearch is a query without sentence-final punctuation.
	SceneSearch Scene = "search"
	// SceneList formats the dictation as numbered lines.
	SceneList Scene = "list"
)

// Scenes lists every scene.
var Scenes = []Scene{SceneChat, SceneEmail, SceneWork, ScenePrompt, SceneSearch, SceneList}

// ChoosableScenes are the scenes a user can pick for a target; list is a
// layout the coordinator derives, not a place.
var ChoosableScenes = []Scene{SceneChat, SceneEmail, SceneWork, ScenePrompt, SceneSearch}

// ModelScenes are the layouts the rewrite model is taught. Other scenes
// are expressed through them by ModelScene and finished by FinishScene, so
// adding a scene never changes the model's prompt.
var ModelScenes = []Scene{SceneChat, SceneEmail, SceneList}

// Known reports whether s is a scene.
func (s Scene) Known() bool {
	for _, known := range Scenes {
		if s == known {
			return true
		}
	}
	return false
}

// ModelScene is the layout the rewrite model is asked for. Work messages
// and coding prompts become numbered lines only when the speaker
// enumerates; search queries are written as plain text.
func ModelScene(s Scene, source string) Scene {
	switch s {
	case SceneEmail, SceneList:
		return s
	case SceneWork, ScenePrompt:
		if Enumerated(source) {
			return SceneList
		}
	}
	return SceneChat
}

// FinishScene applies the mechanical part of a scene's layout to the text
// about to be delivered: a search query is one line without sentence-final
// punctuation. Other scenes are returned unchanged.
func FinishScene(s Scene, text string) string {
	if s != SceneSearch {
		return text
	}
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	return strings.TrimRightFunc(text, func(r rune) bool { return strings.ContainsRune(sentenceEnds, r) })
}

// ordinalPairs are spoken enumerations: a first marker followed later by
// its second.
var ordinalPairs = [][2]string{{"第一", "第二"}, {"一是", "二是"}, {"一个是", "另一个是"}, {"首先", "其次"}, {"first", "second"}}

// Enumerated reports whether the speaker enumerated points.
func Enumerated(source string) bool {
	lower := strings.ToLower(source)
	for _, pair := range ordinalPairs {
		if i := strings.Index(lower, pair[0]); i >= 0 && strings.Contains(lower[i+len(pair[0]):], pair[1]) {
			return true
		}
	}
	return false
}

// FieldKind is the focused field's type, derived from platform roles
// without reading the field's value.
type FieldKind string

const (
	FieldUnknown FieldKind = ""
	FieldText    FieldKind = "text"
	FieldSearch  FieldKind = "search"
)

// SceneHints are metadata of the focused target. They never contain field
// text, page titles or paths, and they are never logged.
type SceneHints struct {
	// App is the platform application identity (a macOS bundle ID).
	App string
	// Field is the focused field's type.
	Field FieldKind
	// Site is the lowercase host of the web page that owns the focused
	// field, or empty outside web content.
	Site string
}

// SceneRoute names the signal that decided a scene.
type SceneRoute string

const (
	// RouteUser is a scene the user chose for this site or app.
	RouteUser    SceneRoute = "user"
	RouteField   SceneRoute = "field"
	RouteSite    SceneRoute = "site"
	RouteApp     SceneRoute = "app"
	RouteDefault SceneRoute = "default"
)

// SceneDecision is a scene and the rule that chose it. Rule names a table
// entry, which can be a host; it stays in memory with the session.
type SceneDecision struct {
	Scene Scene
	Route SceneRoute
	Rule  string
}

// SceneOverrides are scenes the user chose, by SceneKey.
type SceneOverrides map[string]Scene

// SceneKey names the place a user's choice applies to: the web site
// without "www." when there is one, else the app; a search field of that
// place is a separate place. It is empty when nothing identifies the target.
func SceneKey(h SceneHints) string {
	key := strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(h.Site, ".")), "www.")
	if key == "" {
		key = h.App
	}
	if key == "" {
		return ""
	}
	if h.Field == FieldSearch {
		return "search@" + key
	}
	return key
}

// WithScene returns a copy of o where key uses scene, or follows the
// built-in rules again when scene is empty.
func (o SceneOverrides) WithScene(key string, scene Scene) SceneOverrides {
	out := SceneOverrides{}
	for k, v := range o {
		out[k] = v
	}
	if scene == "" {
		delete(out, key)
	} else {
		out[key] = scene
	}
	return out
}

// DecideScene picks the scene for a target. A scene the user chose for the
// place wins. Otherwise the most specific signal wins: a search field, then
// the web site (an exact listed host, else a listed workspace domain), then
// the application. Anything unrecognized is chat, the scene that changes
// layout least.
func DecideScene(h SceneHints, overrides SceneOverrides) SceneDecision {
	if key := SceneKey(h); key != "" {
		if scene, ok := overrides[key]; ok && slicesHas(ChoosableScenes, scene) {
			return SceneDecision{Scene: scene, Route: RouteUser, Rule: key}
		}
	}
	if h.Field == FieldSearch {
		return SceneDecision{Scene: SceneSearch, Route: RouteField, Rule: string(FieldSearch)}
	}
	if site := strings.ToLower(strings.TrimSuffix(h.Site, ".")); site != "" {
		host := strings.TrimPrefix(site, "www.")
		if scene, ok := hostScenes[host]; ok {
			return SceneDecision{Scene: scene, Route: RouteSite, Rule: host}
		}
		for tenant := host; tenant != ""; tenant = parentDomain(tenant) {
			if scene, ok := tenantScenes[tenant]; ok {
				return SceneDecision{Scene: scene, Route: RouteSite, Rule: tenant}
			}
		}
	}
	if scene, ok := appScenes[h.App]; ok {
		return SceneDecision{Scene: scene, Route: RouteApp, Rule: h.App}
	}
	for _, p := range appPrefixScenes {
		if strings.HasPrefix(h.App, p.prefix) {
			return SceneDecision{Scene: p.scene, Route: RouteApp, Rule: p.prefix}
		}
	}
	return SceneDecision{Scene: SceneChat, Route: RouteDefault}
}

// parentDomain drops the leftmost label: mail.google.com → google.com.
// A single label has no parent.
func parentDomain(host string) string {
	_, parent, ok := strings.Cut(host, ".")
	if !ok || !strings.Contains(parent, ".") {
		return ""
	}
	return parent
}

// hostScenes maps exact hosts, compared without a leading "www.". Other
// subdomains are different products: docs.google.com is not a search page.
var hostScenes = map[string]Scene{
	// Web mail.
	"mail.google.com": SceneEmail, "outlook.live.com": SceneEmail, "outlook.office.com": SceneEmail,
	"outlook.office365.com": SceneEmail, "mail.yahoo.com": SceneEmail, "mail.proton.me": SceneEmail,
	"app.fastmail.com": SceneEmail, "mail.zoho.com": SceneEmail, "mail.qq.com": SceneEmail,
	"exmail.qq.com": SceneEmail, "mail.163.com": SceneEmail, "mail.126.com": SceneEmail,
	"mail.aliyun.com": SceneEmail, "qiye.aliyun.com": SceneEmail,
	// Workplace messaging and trackers.
	"app.slack.com": SceneWork, "teams.microsoft.com": SceneWork, "teams.live.com": SceneWork,
	"work.weixin.qq.com": SceneWork, "linear.app": SceneWork,
	// Coding agents, AI assistants and code hosting.
	"chatgpt.com": ScenePrompt, "chat.openai.com": ScenePrompt, "claude.ai": ScenePrompt,
	"gemini.google.com": ScenePrompt, "aistudio.google.com": ScenePrompt, "chat.deepseek.com": ScenePrompt,
	"kimi.com": ScenePrompt, "kimi.moonshot.cn": ScenePrompt, "doubao.com": ScenePrompt,
	"yuanbao.tencent.com": ScenePrompt, "tongyi.com": ScenePrompt, "perplexity.ai": ScenePrompt,
	"poe.com": ScenePrompt, "grok.com": ScenePrompt, "copilot.microsoft.com": ScenePrompt,
	"chat.mistral.ai": ScenePrompt, "v0.dev": ScenePrompt, "v0.app": ScenePrompt, "bolt.new": ScenePrompt,
	"lovable.dev": ScenePrompt, "replit.com": ScenePrompt, "github.com": ScenePrompt, "gitlab.com": ScenePrompt,
	// Web search.
	"google.com": SceneSearch, "google.com.hk": SceneSearch, "google.com.tw": SceneSearch, "google.co.jp": SceneSearch,
	"google.co.uk": SceneSearch, "bing.com": SceneSearch, "cn.bing.com": SceneSearch, "baidu.com": SceneSearch,
	"duckduckgo.com": SceneSearch, "search.brave.com": SceneSearch, "sogou.com": SceneSearch, "kagi.com": SceneSearch,
	"so.com": SceneSearch,
}

// tenantScenes map a domain and all its subdomains: workspaces live on
// per-organization hosts such as acme.feishu.cn.
var tenantScenes = map[string]Scene{
	"feishu.cn": SceneWork, "larksuite.com": SceneWork, "dingtalk.com": SceneWork, "atlassian.net": SceneWork,
}

// appScenes maps macOS bundle identifiers to scenes. Messengers map to chat
// explicitly so the decision records that the app was recognized.
var appScenes = map[string]Scene{
	// Mail clients.
	"com.apple.mail": SceneEmail, "com.microsoft.Outlook": SceneEmail, "com.readdle.SparkDesktop": SceneEmail,
	"com.readdle.smartemail-Mac": SceneEmail, "com.mimestream.Mimestream": SceneEmail,
	"org.mozilla.thunderbird": SceneEmail, "com.tencent.Foxmail": SceneEmail, "it.bloop.airmail2": SceneEmail,
	// Workplace messaging.
	"com.tinyspeck.slackmacgap": SceneWork, "com.microsoft.teams2": SceneWork, "com.microsoft.teams": SceneWork,
	"com.electron.lark": SceneWork, "com.alibaba.DingTalkMac": SceneWork, "com.tencent.WeWorkMac": SceneWork,
	"us.zoom.xos": SceneWork,
	// Personal messaging.
	"com.tencent.xinWeChat": SceneChat, "com.tencent.qq": SceneChat, "com.tdesktop.Telegram": SceneChat,
	"ru.keepcoder.Telegram": SceneChat, "net.whatsapp.WhatsApp": SceneChat, "com.hnc.Discord": SceneChat,
	"com.apple.MobileSMS": SceneChat, "org.whispersystems.signal-desktop": SceneChat, "jp.naver.line.mac": SceneChat,
	// Editors, terminals and AI assistants.
	"com.microsoft.VSCode": ScenePrompt, "com.microsoft.VSCodeInsiders": ScenePrompt,
	"com.todesktop.230313mzl4w4u92": ScenePrompt, "com.exafunction.windsurf": ScenePrompt, "dev.zed.Zed": ScenePrompt,
	"com.apple.dt.Xcode": ScenePrompt, "com.google.android.studio": ScenePrompt, "com.sublimetext.4": ScenePrompt, "com.google.antigravity": ScenePrompt, "com.apple.Terminal": ScenePrompt,
	"com.googlecode.iterm2": ScenePrompt, "dev.warp.Warp-Stable": ScenePrompt, "com.mitchellh.ghostty": ScenePrompt,
	"com.termius.mac": ScenePrompt, "com.anthropic.claudefordesktop": ScenePrompt, "com.openai.chat": ScenePrompt,
	"com.openai.codex": ScenePrompt, "ai.opencode.desktop": ScenePrompt, "app.chatwise": ScenePrompt,
	"tech.baye.OpenCat": ScenePrompt, "ai.elementlabs.lmstudio": ScenePrompt,
}

// appPrefixScenes cover vendors whose bundle IDs share a prefix.
var appPrefixScenes = []struct {
	prefix string
	scene  Scene
}{{"com.jetbrains.", ScenePrompt}}

func slicesHas(list []Scene, s Scene) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
