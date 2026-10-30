// Package config holds the fixed facts about the project the UI links to:
// where it lives, who builds it and under which license.
package config

// Proper names and handles, shown as they are in every language.
const (
	Team       = "AstraFoundry"
	Repository = "AstraFoundry/Arion"
	XHandle    = "@AstraFoundry"
	License    = "AGPL-3.0"
)

// Destinations the About section opens in the browser.
const (
	SiteURL       = "https://arion.astrafoundry.si"
	TermsPath     = "/terms/"
	PrivacyPath   = "/privacy/"
	TeamURL       = "https://github.com/AstraFoundry"
	RepositoryURL = "https://github.com/AstraFoundry/Arion"
	XURL          = "https://x.com/AstraFoundry"
	// DiscordURL and FeishuURL invite to the community chats.
	DiscordURL = "https://discord.com/invite/72kh465mMm"
	FeishuURL  = "https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae"
	IssuesURL  = RepositoryURL + "/issues/new"
	LicenseURL = RepositoryURL + "/blob/main/LICENSE"
	// ReleaseURL is the page of a release, before its version; releases are
	// tagged v1.2.0 and so on.
	ReleaseURL = RepositoryURL + "/releases/tag/v"
)

// CloudProviders names the cloud polishing providers by their runtime IDs.
var CloudProviders = map[string]string{
	"chatgpt":    "ChatGPT",
	"openai":     "OpenAI",
	"anthropic":  "Anthropic",
	"google":     "Google Gemini",
	"deepseek":   "DeepSeek",
	"openrouter": "OpenRouter",
}
