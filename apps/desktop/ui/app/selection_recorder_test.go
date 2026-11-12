package app

func (r *recorder) CopySelection() error {
	r.call("selection-copy")
	return r.selectionError
}

func (r *recorder) AddSelectionVocabulary() error {
	r.call("selection-vocabulary")
	return r.selectionError
}

func (r *recorder) OpenSelection(actionID, clientID, question string) error {
	call := "selection-open:" + clientID + ":" + actionID
	if question != "" {
		call += ":" + question
	}
	r.call(call)
	return r.selectionError
}
