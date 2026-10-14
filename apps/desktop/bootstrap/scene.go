package bootstrap

import (
	"context"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// peekScene describes the frontmost app's focused field for the panel's
// scene card. The owner is read now, before the panel takes the focus; the
// accessibility queries run off the main thread.
func (a *App) peekScene() {
	a.sceneSeq++
	seq := a.sceneSeq
	a.state.SceneTarget = arion.SceneTarget{}
	owner := a.targets.FocusOwner()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		snap, err := a.targets.Peek(ctx, owner)
		mygo.RunOnMain(func() {
			if seq != a.sceneSeq {
				return
			}
			// Secure fields never get a scene: dictation is blocked there.
			if err == nil && !snap.Sensitive {
				a.state.SceneTarget = arion.SceneTarget{Known: true, AppName: snap.ApplicationName, Hints: snap.SceneHints()}
			}
			a.invalidate()
		})
	}()
}
