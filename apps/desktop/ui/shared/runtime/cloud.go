package runtime

// CloudPhase is the cloud polishing connection's lifecycle.
type CloudPhase string

const (
	CloudDisconnected CloudPhase = "disconnected"
	CloudSigningIn    CloudPhase = "signing_in"
	CloudVerifying    CloudPhase = "verifying"
	CloudConnected    CloudPhase = "connected"
)

// Cloud providers. ChatGPT signs in with a subscription; the others take
// an API key. Their names are in config.
const (
	CloudChatGPT    = "chatgpt"
	CloudOpenAI     = "openai"
	CloudAnthropic  = "anthropic"
	CloudGoogle     = "google"
	CloudDeepSeek   = "deepseek"
	CloudOpenRouter = "openrouter"
)

// CloudKeyProviders are the API key providers in picker order.
var CloudKeyProviders = []string{CloudOpenAI, CloudAnthropic, CloudGoogle, CloudDeepSeek, CloudOpenRouter}

// Cloud is the connected provider, its models and the last failure.
type Cloud struct {
	Phase CloudPhase
	// Provider is the connected provider, or the one being connected.
	Provider string
	// Email names the ChatGPT account.
	Email   string
	Models  []string
	Listing bool
	// Saved lists providers whose credentials are still stored. The active
	// provider is Provider while Phase is connected.
	Saved []string
	// Failure is one of CloudCodes; empty when the last attempt worked.
	Failure string
}

// Busy reports a sign-in or key check in flight.
func (c Cloud) Busy() bool { return c.Phase == CloudSigningIn || c.Phase == CloudVerifying }

// Cloud.Failure values views map to copy. They equal the cloud polishing
// service's codes.
const (
	CloudCodeSignIn         = "SIGN_IN"
	CloudCodeCancelled      = "CANCELLED"
	CloudCodeTimeout        = "TIMEOUT"
	CloudCodePortInUse      = "PORT_IN_USE"
	CloudCodeNoSubscription = "NO_SUBSCRIPTION"
	CloudCodeRejected       = "REJECTED"
	CloudCodeNetwork        = "NETWORK"
	CloudCodeStore          = "SECURE_STORE"
	CloudCodeModels         = "MODELS"
)

// CloudCodes lists every Failure the service reports.
var CloudCodes = []string{CloudCodeSignIn, CloudCodeCancelled, CloudCodeTimeout, CloudCodePortInUse, CloudCodeNoSubscription,
	CloudCodeRejected, CloudCodeNetwork, CloudCodeStore, CloudCodeModels}

// PolishAvailable reports whether polishing can run with the chosen
// engine: the cloud everywhere, on device only where the runtime ships.
func PolishAvailable(s *State) bool {
	return s.Preferences.PolishEngine == PolishCloud || s.Polish.State != PolishUnsupported
}

// PolishOn reports polishing chosen and available.
func PolishOn(s *State) bool { return s.Preferences.Polish && PolishAvailable(s) }
