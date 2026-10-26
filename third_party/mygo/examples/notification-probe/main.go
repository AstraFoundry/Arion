// Notification-probe is an isolated, packaged native notification harness.
// A sidecar config survives an OS cold activation without environment variables.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	probeName  = "Arion Notification Probe"
	probeTitle = "Arion notification test"
	probeBody  = "An isolated native notification transport test."
	probeView  = "View update"
	probeLater = "Later"
)

type config struct {
	Report  string `json:"report"`
	Control string `json:"control"`
}
type command struct {
	Token string `json:"token"`
	Op    string `json:"op"`
	ID    string `json:"id"`
}
type reporter struct {
	sync.Mutex
	path string
}

func (r *reporter) write(event string, fields map[string]any) {
	r.Lock()
	defer r.Unlock()
	if fields == nil {
		fields = map[string]any{}
	}
	fields["event"] = event
	fields["pid"] = os.Getpid()
	fields["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(fields); err != nil {
		log.Fatal(err)
	}
}
func main() {
	if handled, err := mygo.HandleNotificationRegistration(); handled {
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	// A sidecar outside the executable is test-owned. The harness never reads or
	// writes production notification preferences or application profile data.
	path := filepath.Join(filepath.Dir(exe), "notification-probe.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var cfg config
	if err = json.Unmarshal(raw, &cfg); err != nil || !filepath.IsAbs(cfg.Report) || !filepath.IsAbs(cfg.Control) {
		log.Fatal("invalid notification probe config")
	}
	r := &reporter{path: cfg.Report}
	r.write("launch", map[string]any{"args": os.Args[1:]})
	mygo.App.SetName(probeName)
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	var view *mygo.Window
	var ready bool
	var queued []struct{ id, action string }
	handle := func(id, action string) {
		r.write("activation", map[string]any{"id": id, "action": action, "ready": ready})
		if action == "" || action == "view" {
			if view == nil {
				view = mygo.NewWindow(mygo.WindowOptions{Title: probeTitle, Width: 420, Height: 180, Content: ui.View(func(c *ui.Context) {
					ui.Column(c).Fill().Center().Children(func() { ui.Text(c, probeTitle); ui.Text(c, id) })
				})})
			}
			view.Show()
			view.Focus()
		}
	}
	mygo.App.OnNotificationAction(func(id, action string) {
		if !ready {
			queued = append(queued, struct{ id, action string }{id, action})
			return
		}
		handle(id, action)
	})
	mygo.App.OnDidBecomeActive(func() { r.write("active", nil) })
	mygo.App.OnDidResignActive(func() { r.write("inactive", nil) })
	mygo.App.WhenReady(func() {
		ready = true
		r.write("ready", map[string]any{"supported": mygo.NotificationsSupported()})
		for _, a := range queued {
			handle(a.id, a.action)
		}
		queued = nil
		go func() {
			seen := map[string]bool{}
			if raw, err := os.ReadFile(cfg.Report); err == nil {
				scan := bufio.NewScanner(strings.NewReader(string(raw)))
				for scan.Scan() {
					var event struct {
						Event string `json:"event"`
						Token string `json:"token"`
					}
					if json.Unmarshal(scan.Bytes(), &event) == nil && event.Event == "command" {
						seen[event.Token] = true
					}
				}
			}
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				raw, err := os.ReadFile(cfg.Control)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					r.write("control-error", map[string]any{"error": err.Error()})
					return
				}
				scanner := bufio.NewScanner(strings.NewReader(string(raw)))
				for scanner.Scan() {
					var cmd command
					if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
						continue
					}
					if cmd.Token == "" || seen[cmd.Token] {
						continue
					}
					seen[cmd.Token] = true
					fields := map[string]any{"token": cmd.Token, "op": cmd.Op, "id": cmd.ID}
					switch cmd.Op {
					case "query", "request":
						var status mygo.NotificationAuthorization
						var err error
						if cmd.Op == "request" {
							status, err = mygo.RequestNotificationPermission()
						} else {
							status, err = mygo.NotificationPermission()
						}
						fields["status"] = status
						if err != nil {
							fields["error"] = err.Error()
						}
					case "show":
						n := mygo.NewNotification(mygo.NotificationOptions{ID: cmd.ID, Title: probeTitle, Body: probeBody, Silent: true, Group: "notification-probe", Actions: []mygo.NotificationAction{{ID: "view", Title: probeView, Foreground: true}, {ID: "later", Title: probeLater}}})
						if err := n.Show(); err != nil {
							fields["error"] = err.Error()
						}
					case "remove":
						mygo.RemoveNotification(cmd.ID)
					case "clear":
						mygo.ClearNotifications()
					case "quit":
						r.write("command", fields)
						mygo.App.Quit()
						return
					default:
						fields["error"] = fmt.Sprintf("unknown probe operation %q", cmd.Op)
					}
					r.write("command", fields)
				}
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		r.write("failure", map[string]any{"error": err.Error()})
		log.Fatal(err)
	}
	r.write("exit", nil)
}
