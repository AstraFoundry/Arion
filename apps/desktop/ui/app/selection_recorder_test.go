package app

func (r *recorder) CopySelection() error {
	r.call("selection-copy")
	return r.selectionError
}

func (r *recorder) AddSelectionVocabulary() error {
	r.call("selection-vocabulary")
	return r.selectionError
}

func (r *recorder) OpenSelection(actionID, question string) error {
	if question != "" {
		r.call("selection-open:" + actionID + ":" + question)
		return r.selectionError
	}
	r.call("selection-open:" + actionID)
	return r.selectionError
}
