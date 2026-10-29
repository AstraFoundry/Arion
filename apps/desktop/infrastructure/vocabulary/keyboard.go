package vocabulary

import "arion.local/core/domain"

// KeyboardTerms decrypts one current snapshot and projects approved readings.
// Deleted records disappear on the next snapshot; no separate term copy exists.
func (s *Store) KeyboardTerms() ([]domain.KeyboardTerm, error) {
	entries, err := s.List()
	if err != nil {
		return nil, err
	}
	return domain.KeyboardTerms(Memory(entries))
}

// SetKeyboardPinyin binds an explicit reading to the exact approved spelling.
// The spelling guard prevents a stale editor from assigning an old reading to
// a learned record that has since been re-pointed. Empty removes the binding.
func (s *Store) SetKeyboardPinyin(id, spelling, reading string) (Entry, error) {
	if err := domain.ValidateKeyboardPinyin(spelling, reading); err != nil {
		return Entry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.read()
	if err != nil {
		return Entry{}, err
	}
	for i, entry := range entries {
		if entry.ID != id {
			continue
		}
		if !entry.Approved || entry.To != spelling {
			return Entry{}, ErrConflict
		}
		if entry.KeyboardPinyin == reading {
			return entry, nil
		}
		entries[i].KeyboardPinyin = reading
		if err := s.write(entries); err != nil {
			return Entry{}, err
		}
		return entries[i], nil
	}
	return Entry{}, ErrConflict
}
