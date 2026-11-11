# PermissionFlow

The permissionflow plugin checks and asks for the permissions the system
keeps for the user to give apps, such as Accessibility, Screen Recording
and the microphone, and guides the user to where they give them. On
macOS, its guide opens System Settings at the page and shows a panel
under its window, which follows it, with the app to drag into the list,
as [PermissionFlow](https://github.com/jaywcjlove/PermissionFlow) does for
Swift apps. It is all Go: no JavaScript package, and nothing to
`mygo.Use`. Pages reach it through the app's own bindings.

```go
import "github.com/egoist/mygo/plugins/permissionflow"

st, err := permissionflow.Check(permissionflow.Accessibility)
if err == nil && st != permissionflow.Granted {
	st, err = permissionflow.Guide(ctx, permissionflow.Accessibility, permissionflow.GuideOptions{})
}
```

[examples/permissionflow](../../examples/permissionflow) lists the
permissions of the system with their status, and asks for them both ways.

## The permissions

| Permission | macOS | Windows | Prompt (`Request`) | Drag (`Guide`) |
| --- | --- | --- | --- | --- |
| `Accessibility` | yes | — | yes | yes |
| `ScreenRecording` | yes | — | yes | yes |
| `InputMonitoring` | yes | — | yes | yes |
| `FullDiskAccess` | yes, by reading a protected file | — | — | yes |
| `AppManagement` | Check can't tell | — | — | yes |
| `DeveloperTools` | Check can't tell | — | — | yes |
| `Microphone` | yes | yes | macOS | — |
| `Camera` | yes | yes | macOS | — |
| `Calendars` | yes | — | yes | — |
| `Reminders` | yes | — | yes | — |
| `Bluetooth` | yes | — | yes | yes |
| `MediaAppleMusic` | Check can't tell | — | — | yes |

Where a system keeps no such permission, as Windows keeps nothing but the
microphone and the camera from desktop apps, and Linux nothing from apps
outside sandboxes, the functions return `errors.ErrUnsupported`.
`Allowed` folds that in: it reports whether the app may use what a
permission guards, because the app has it or because the system doesn't
ask, and is false in browsers.

macOS 27 names Accessibility Device Control and Data Access, which
`Title` follows.

## Checking and asking

`Check` reports a `Status` without asking the user: `Granted`, `Denied`,
`NotDetermined` (not asked yet), or `Unknown` when the system can't tell.
macOS doesn't tell whether it has asked for Accessibility, Screen
Recording and Input Monitoring, which are `Denied` until granted, and
reports a change of Screen Recording only after the app starts again.

`Request` asks with the system's prompt, which macOS shows once for each
app; after that it only reports, and the user changes their mind in
System Settings. The prompts of the microphone, the camera, calendars,
reminders and Bluetooth need the app's Info.plist to say why it asks
(`macos.infoPlist` in `mygo.config.ts`):

```ts
export default defineConfig({
  macos: {
    infoPlist: {
      NSMicrophoneUsageDescription: "Notes records the voice memos you start.",
    },
  },
});
```

macOS ends an app that asks without it, so `Request` returns
`ErrUsageDescription` instead. A program run outside an app bundle, as
with `go run`, asks for the app that started it, such as the terminal,
which gets the permission.

`Wait` waits until the app has a permission, or the context is done.

## The guide

`Guide` opens the page of the system's settings (`OpenSettings` only
opens it) and waits for the user to give the permission. On macOS, on
the pages whose list takes apps dragged into it, it also shows a panel
under the window of System Settings:

- It says "Drag **MyApp** to the list above to allow **Accessibility**.",
  over a card of the app, its icon and its name, to drag into the list.
- It lines up with the page, right of the sidebar, follows the window as
  it moves and resizes, and stays on the screen.
- It never takes the focus, so System Settings stays the app in front,
  and hides while another app is.
- The app is the bundle the program runs from (`GuideOptions.App`
  changes it, and `Name` its name).

`Guide` returns the status when the app has the permission, when the
user closes the panel or System Settings, or when the context is done.
One panel shows at a time: another `Guide` closes the one before. On
Windows, it opens Settings at Privacy & security and returns when the
app has the permission or the context is done.

Call `Request`, `Wait` and `Guide` from a goroutine, not from the main
thread, which they would block: bound methods and event handlers of pages
run on goroutines of their own. `OpenSettings` and `Guide` need `App.Run`
started.

## Notice

The plugin follows PermissionFlow (MIT, © 小弟调调), whose license is in
[NOTICE.md](../../plugins/permissionflow/NOTICE.md): what its guide does,
its panes, the panel's geometry and text. Its code is MyGo's own.
