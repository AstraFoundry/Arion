//go:build darwin

package darwin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"

	"github.com/egoist/mygo/internal/platform"
)

// Notifications use UNUserNotificationCenter, which only works for apps
// running from a bundle with an identifier.

var (
	mainQueueMu sync.Mutex
	mainQueue   []func()
)

// runOnMain runs fn on the main thread; used by callbacks the system
// delivers on background queues.
func (b *Backend) runOnMain(fn func()) {
	if b.IsMainThread() {
		fn()
		return
	}
	b.post(fn)
}

// post runs fn on the main thread after the current event, even when
// called there.
func (b *Backend) post(fn func()) {
	mainQueueMu.Lock()
	mainQueue = append(mainQueue, fn)
	mainQueueMu.Unlock()
	b.Signal()
}

func drainMainQueue() {
	mainQueueMu.Lock()
	q := mainQueue
	mainQueue = nil
	mainQueueMu.Unlock()
	for _, fn := range q {
		fn()
	}
}

func registerNotificationDelegate() {
	if !hasClass("UNUserNotificationCenter") {
		return
	}
	classDef("MyGoNotificationDelegate", "NSObject", []string{"UNUserNotificationCenterDelegate"}, []objc.MethodDef{
		method("userNotificationCenter:willPresentNotification:withCompletionHandler:", func(self id, _ objc.SEL, center, n id, handler uintptr) {
			// Show banners even while the app is in the foreground.
			callBlock(handler, 1<<3|1<<4) // list | banner; content owns the sound
		}),
		method("userNotificationCenter:didReceiveNotificationResponse:withCompletionHandler:", func(self id, _ objc.SEL, center, resp id, handler uintptr) {
			ident := goString(send(send(send(send(resp, "notification"), "request"), "identifier"), "self"))
			action := goString(send(resp, "actionIdentifier"))
			if action == "com.apple.UNNotificationDefaultActionIdentifier" {
				action = ""
			}
			if action == "com.apple.UNNotificationDismissActionIdentifier" {
				callBlock(handler)
				return
			}
			// The OS may end a background activation after completion. Retain its
			// block until the response has reached application listeners on main.
			completion := blockCopy(handler)
			theBackend.runOnMain(func() {
				defer blockRelease(completion)
				defer callBlock(completion)
				theBackend.h.NotificationAction(ident, action)
			})
		}),
	})
}

func (b *Backend) NotificationsSupported() bool {
	_, packaged := appController{b}.Package()
	return packaged && hasClass("UNUserNotificationCenter")
}

// notificationCenter returns the app's notification center. It throws
// when the app does not run from a bundle: check NotificationsSupported
// first.
func notificationCenter() id {
	return send(class("UNUserNotificationCenter"), "currentNotificationCenter")
}

// attachNotificationDelegate gives the notification center the delegate
// that receives clicks, at launch: the system delivers the click that
// launched the app before the app has finished launching, and drops it
// when the center has no delegate yet.
func (b *Backend) attachNotificationDelegate() {
	if !b.NotificationsSupported() {
		return
	}
	b.notifyDelegate = alloc("MyGoNotificationDelegate")
	withPool(func() {
		send(notificationCenter(), "setDelegate:", uintptr(b.notifyDelegate))
	})
}

// NotificationPermission always reads the OS state, including changes made in
// System Settings while the application was running.
func (b *Backend) NotificationPermission(done func(platform.NotificationAuthorization, error)) {
	if !b.NotificationsSupported() {
		done(platform.NotificationAuthorizationUnsupported, platform.ErrUnsupported)
		return
	}
	withPool(func() {
		blk := newBlock(func(_ objc.Block, settings id) {
			status := platform.NotificationAuthorizationUnknown
			var err error
			if settings == 0 {
				err = fmt.Errorf("mygo: notification settings unavailable")
			} else {
				switch sendInt(settings, "authorizationStatus") {
				case 0:
					status = platform.NotificationAuthorizationUnknown
				case 1:
					status = platform.NotificationAuthorizationDenied
				case 2, 3, 4:
					status = platform.NotificationAuthorizationAllowed
				default:
					err = fmt.Errorf("mygo: unknown notification authorization status")
				}
			}
			b.runOnMain(func() { done(status, err) })
		})
		send(notificationCenter(), "getNotificationSettingsWithCompletionHandler:", uintptr(blk))
		blk.Release()
	})
}

func (b *Backend) RequestNotificationPermission(done func(platform.NotificationAuthorization, error)) {
	if !b.NotificationsSupported() {
		done(platform.NotificationAuthorizationUnsupported, platform.ErrUnsupported)
		return
	}
	withPool(func() {
		blk := newBlock(func(_ objc.Block, granted bool, nsErr id) {
			var err error
			withPool(func() { err = notificationError(nsErr) })
			b.runOnMain(func() {
				if err != nil {
					done(platform.NotificationAuthorizationUnknown, err)
					return
				}
				if !granted {
					done(platform.NotificationAuthorizationDenied, nil)
					return
				}
				b.NotificationPermission(done)
			})
		})
		// Update reminders need alert presentation only, without badges or sound.
		send(notificationCenter(), "requestAuthorizationWithOptions:completionHandler:", 1<<2, uintptr(blk))
		blk.Release()
	})
}

func (b *Backend) ShowNotification(n *platform.Notification, done func(error)) {
	if b.notifyPending == nil {
		b.notifyPending = map[string]uint64{}
	}
	b.notifyGeneration++
	generation := b.notifyGeneration
	b.notifyPending[n.ID] = generation
	b.NotificationPermission(func(status platform.NotificationAuthorization, err error) {
		if b.notifyPending[n.ID] != generation {
			done(context.Canceled)
			return
		}
		if err != nil {
			done(err)
			return
		}
		if status != platform.NotificationAuthorizationAllowed {
			done(platform.ErrNotificationsDenied)
			return
		}
		b.addNotification(n, func(err error) {
			current, exists := b.notifyPending[n.ID]
			if !exists {
				b.removeNativeNotification(n.ID)
				done(context.Canceled)
				return
			}
			if current != generation {
				done(context.Canceled)
				return
			}
			done(err)
		})
	})
}

// addNotification gives the system a request to show a notification. Its
// trigger is nil, which shows it at once.
func (b *Backend) addNotification(n *platform.Notification, done func(error)) {
	withPool(func() {
		content := autorelease(alloc("UNMutableNotificationContent"))
		send(content, "setTitle:", uintptr(nsString(n.Title)))
		if n.Subtitle != "" {
			send(content, "setSubtitle:", uintptr(nsString(n.Subtitle)))
		}
		send(content, "setBody:", uintptr(nsString(n.Body)))
		if len(n.Actions) > 0 {
			encoded, _ := json.Marshal(n.Actions)
			categoryID := fmt.Sprintf("mygo-%x", sha256.Sum256(encoded))
			actions := make([]id, 0, len(n.Actions))
			for _, a := range n.Actions {
				var options uintptr
				if a.Foreground {
					options = 1 << 2
				}
				actions = append(actions, send(class("UNNotificationAction"), "actionWithIdentifier:title:options:", uintptr(nsString(a.ID)), uintptr(nsString(a.Title)), options))
			}
			category := send(class("UNNotificationCategory"), "categoryWithIdentifier:actions:intentIdentifiers:options:", uintptr(nsString(categoryID)), uintptr(nsArray(actions...)), uintptr(nsArray()), 0)
			if b.notifyCategories == 0 {
				b.notifyCategories = alloc("NSMutableSet")
			}
			send(b.notifyCategories, "addObject:", uintptr(category))
			send(notificationCenter(), "setNotificationCategories:", uintptr(b.notifyCategories))
			send(content, "setCategoryIdentifier:", uintptr(nsString(categoryID)))
		}
		if !n.Silent {
			send(content, "setSound:", uintptr(send(class("UNNotificationSound"), "defaultSound")))
		}
		if n.Group != "" {
			send(content, "setThreadIdentifier:", uintptr(nsString(n.Group)))
		}
		req := send(class("UNNotificationRequest"), "requestWithIdentifier:content:trigger:",
			uintptr(nsString(n.ID)), uintptr(content), 0)
		blk := newBlock(func(_ objc.Block, nsErr id) {
			// The error is the caller's only while the block runs.
			var err error
			withPool(func() { err = notificationError(nsErr) })
			b.runOnMain(func() { done(err) })
		})
		send(notificationCenter(), "addNotificationRequest:withCompletionHandler:", uintptr(req), uintptr(blk))
		blk.Release()
	})
}

// unErrorNotificationsNotAllowed is UNErrorCodeNotificationsNotAllowed.
const unErrorNotificationsNotAllowed = 1

// notificationError describes an error of the notification center.
func notificationError(err id) error {
	if err == 0 {
		return nil
	}
	if goString(send(err, "domain")) == "UNErrorDomain" && sendInt(err, "code") == unErrorNotificationsNotAllowed {
		return platform.ErrNotificationsDenied
	}
	return nsError(err)
}

func (b *Backend) RemoveNotification(ident string) {
	delete(b.notifyPending, ident)
	b.removeNativeNotification(ident)
}

func (b *Backend) removeNativeNotification(ident string) {
	if !b.NotificationsSupported() {
		return
	}
	withPool(func() {
		center := notificationCenter()
		ids := nsArray(nsString(ident))
		send(center, "removeDeliveredNotificationsWithIdentifiers:", uintptr(ids))
		send(center, "removePendingNotificationRequestsWithIdentifiers:", uintptr(ids))
	})
}

func (b *Backend) RemoveAllNotifications() {
	clear(b.notifyPending)
	if !b.NotificationsSupported() {
		return
	}
	withPool(func() {
		center := notificationCenter()
		send(center, "removeAllDeliveredNotifications")
		send(center, "removeAllPendingNotificationRequests")
	})
}
