package diagnostics

// CloudModelsDiagnostic excludes keys, URLs, model IDs and response content.
type CloudModelsDiagnostic struct {
	Failure    string `json:"failure"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
}

func (d CloudModelsDiagnostic) valid() bool {
	switch d.Failure {
	case "MODELS", "REJECTED", "NETWORK", "CANCELLED":
	default:
		return false
	}
	return d.HTTPStatus == 0 || (d.HTTPStatus >= 100 && d.HTTPStatus <= 599)
}
