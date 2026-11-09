// Package config holds the fixed facts about the project the UI links to:
// where it lives, who builds it and under which license.
package config

// Proper names and handles, shown as they are in every language.
const (
	Team       = "AstraFoundry"
	Repository = "AstraFoundry/Arion"
	XHandle    = "@AstraFoundry"
	License    = "AGPL-3.0"
	// Agent is the coding agent the feedback illustration shows.
	Agent = "Codex"
)

// Destinations the About section opens in the browser.
const (
	SiteURL       = "https://arion.astrafoundry.si"
	TermsPath     = "/terms/"
	PrivacyPath   = "/privacy/"
	UsageDataPath = "/usage-data/"
	TeamURL       = "https://github.com/AstraFoundry"
	RepositoryURL = "https://github.com/AstraFoundry/Arion"
	XURL          = "https://x.com/AstraFoundry"
	// DiscordURL and FeishuURL invite to the community chats.
	DiscordURL = "https://discord.com/invite/72kh465mMm"
	FeishuURL  = "https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=e05vef49-cce2-404f-9740-2fb99d17d444"
	IssuesURL  = RepositoryURL + "/issues/new"
	// FeedbackSkillURL is the agent feedback skill on the site; the feedback
	// window's prompt asks a coding agent to fetch and follow it.
	FeedbackSkillURL = SiteURL + "/feedback.md"
	LicenseURL       = RepositoryURL + "/blob/main/LICENSE"
	// ReleaseURL is the page of a release, before its version; releases are
	// tagged v1.2.0 and so on.
	ReleaseURL = RepositoryURL + "/releases/tag/v"
)

// CloudProviders names the cloud polishing providers by their runtime IDs.
var CloudProviders = map[string]string{
	"chatgpt":              "ChatGPT",
	"openai":               "OpenAI",
	"anthropic":            "Anthropic",
	"google":               "Google Gemini",
	"deepseek":             "DeepSeek",
	"openrouter":           "OpenRouter",
	"openai-compatible":    "OpenAI compatible",
	"anthropic-compatible": "Anthropic compatible",
}
