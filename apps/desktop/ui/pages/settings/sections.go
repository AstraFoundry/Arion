package settings

import (
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/features/keyboard"
	"arion.local/desktop/ui/features/permissions"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// sectionInfo describes one section of Settings.
type sectionInfo struct {
	id    string
	icon  *ui.SVG
	title copy.Key
	// shown hides the section where it does not apply; nil always shows it.
	shown func(runtime.Store) bool
	// attention reports something in the section waiting for the user,
	// marked with a dot beside its sidebar item. nil means the section has
	// nothing that can wait for the user.
	attention func(runtime.Store) bool
	// card draws the section as the account card at the sidebar's foot
	// instead of as an item.
	card bool
}

// sections lists every section of Settings in sidebar order. The sidebar,
// the window title and the attention dots all read it: a new section, or a
// new state that waits for the user, belongs here.
var sections = []sectionInfo{
	{id: Overview, icon: icons.ChartLine, title: copy.NavOverview},
	{id: General, icon: icons.Settings, title: copy.NavGeneral, attention: permissions.Missing},
	{id: Dictation, icon: icons.AudioLines, title: copy.NavDictation, attention: holdkey.Unavailable},
	{id: Keyboard, icon: icons.Keyboard, title: copy.NavKeyboard, shown: keyboard.Supported},
	{id: Microphone, icon: icons.Mic, title: copy.NavMicrophone},
	{id: Model, icon: icons.Cpu, title: copy.NavModel, attention: func(s runtime.Store) bool { return model.NeedsAction(s.State.Model) }},
	{id: Vocabulary, icon: icons.BookOpen, title: copy.NavVocabulary},
	{id: About, icon: icons.Info, title: copy.NavAbout, attention: func(s runtime.Store) bool { return s.State.Update.Pending() }},
	{id: Account, title: copy.NavAccount, shown: func(s runtime.Store) bool { return !s.State.Account.Disabled }, card: true},
}

func sectionOf(id string) (sectionInfo, bool) {
	for _, s := range sections {
		if s.id == id {
			return s, true
		}
	}
	return sectionInfo{}, false
}

func (s sectionInfo) visible(store runtime.Store) bool { return s.shown == nil || s.shown(store) }

func (s sectionInfo) waiting(store runtime.Store) bool {
	return s.attention != nil && s.attention(store)
}
