package application

// ClipboardCapture tracks OS revisions rather than content. Configure consumes
// the clipboard at every collection boundary so stopped copies are never
// imported on resume. Native adapters identify internal/sensitive revisions.
type ClipboardCapture struct {
	revision uint64
	active   bool
}

func (s *ClipboardCapture) Configure(active bool, revision uint64) {
	if active != s.active {
		s.revision = revision
		s.active = active
	}
}
func (s *ClipboardCapture) Pending(revision uint64, excluded bool) bool {
	if !s.active {
		return false
	}
	if excluded {
		s.revision = revision
		return false
	}
	return revision != s.revision
}
func (s *ClipboardCapture) Accept(revision uint64) { s.revision = revision }
