package app

func (r *recorder) CopySelection() error {
	r.call("selection-copy")
	return nil
}

func (r *recorder) AddSelectionVocabulary() error {
	r.call("selection-vocabulary")
	return nil
}

func (r *recorder) OpenSelection(actionID, _ string) error {
	r.call("selection-open:" + actionID)
	return nil
}
