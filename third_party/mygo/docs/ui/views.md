# Views

The view is a function from your app's state to its interface. MyGo calls
it on the main thread to build frames after input, model changes and while
something animates. A frame can contain several build passes, so it can
show the outcome of an action before it is painted.

The window has one stable `*ui.Context`. Child builders share it and
switch its current parent for the duration of the callback. Pass it to
helpers that build part of the view.

Each `ui.Element` value belongs to one build pass. Its owner, slot and
generation are checked before accessing the control. Use `Valid` for an
optional element and `ui.Element{}` for absence. Stale use is diagnosed in
development and `Tester`; production methods return empty results or
ignore the operation.

Keep build values out of app state and background work. Store `ui.Handle`
for persistent control identity and `ui.Services` for callbacks between
builds. Model data and widget state such as `ListState` belong in your app.

```go
// todoList is the app's state, which lasts.
type todoList struct {
	todos []Todo
	draft string
}

// view builds the interface from it.
func (app *todoList) view(c *ui.Context) {
	ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
		for i := range app.todos {
			ui.Checkbox(c, &app.todos[i].Done, app.todos[i].Title)
		}
		if ui.TextInput(c, &app.draft).Placeholder("New to-do").Submitted() {
			app.todos = append(app.todos, Todo{Title: app.draft})
			app.draft = ""
		}
	})
}
```

Make the state once, as the app starts, and give its view to the window:

```go
app := &todoList{}
mygo.NewWindow(mygo.WindowOptions{Title: "To-dos", Content: ui.View(app.view)})
```

`ui.View(app.view)` is the window's content (`mygo.WindowOptions.Content`);
a `Content` can serve several windows, each with its own element state.

The examples in these guides are parts of such a view: `c` is the view's
`*ui.Context`, and `app` its receiver, the value of your own type that
holds the state, as `todoList` here. A field such as `app.volume` or a
method such as `app.save()` is one you declare on that type. Values that
last, such as a [router](navigation.md), are fields made with the rest of
the state, never in the view, which runs for every frame.

## Events and actions

`Clicked` reports a click since the last frame and keeps ordinary control
flow beside the control:

```go
if ui.Button(c, "Save").Clicked() {
    app.save()
}
```

A control takes the user's input as its constructor returns, so an inline
save sees the latest edits of the controls built before it.

`OnClick` schedules the action after the view finishes building and bound
input has been applied:

```go
ui.Button(c, "Save").Disabled(app.saving).OnClick(app.save)
```

Use polling when inline `return`, `break` or `continue` is useful. Use
callbacks for actions that modify the collection being built, so the loop
finishes before its model changes:

```go
for _, item := range app.items {
    ui.Row(c.Key(item.ID)).Children(func() {
        ui.Text(c, item.Title).Grow(1)
        ui.Button(c, "Delete").OnClick(func() {
            app.removeItem(item.ID)
        })
    })
}
```

Handled input is consumed before rebuilding, so an action runs once for
that input. MyGo rebuilds changed state before painting the frame.

Widgets that change a value take a pointer to it, so they need no handler:
`ui.Checkbox(c, &app.settings.Sync, "Sync")` binds the field to the control.
As in Dear ImGui and egui, the control writes the user's edit through the
pointer before its constructor returns, so the next statement reads the new
value. `Changed` reports that it changed in this build pass, and `Submitted`
that Enter was pressed in a single-line text input.

```go
if ui.TextInput(c, &app.query).Placeholder("Search").Changed() {
	app.results = search(app.query)
}
```

Local copies work too, including values read from a map, compared with the
value they came from or through `Changed`:

```go
viewed := app.viewed[file.ID]
if ui.Checkbox(c, &viewed, "Viewed").Changed() {
    app.viewed[file.ID] = viewed
}

chosen := app.language
ui.RadioGroup(c, func() {
    for _, tag := range app.languages {
        ui.Radio(c, &chosen, tag, i18n.Autonym(tag))
    }
})
if chosen != app.language {
    app.setLanguage(chosen)
}
```

A container takes its input with its children, as the call that built it
returns: a radio group's arrow keys move among all of its radios. Input
follows the options of the frame the user acted on: an input shown
read-only takes no typing, and `Disabled`, `ReadOnly` or `Step` set in
this build govern the input of the frames after it, as they show.

`OnChange` and `OnSubmit` run after construction and bound input, before the
rebuild. Use callbacks when an action should wait until the view has finished
building, such as modifying the collection being rendered. They can also
commit local copies:

```go
viewed := app.viewed[file.ID]
ui.Checkbox(c, &viewed, "Viewed").OnChange(func() {
    app.viewed[file.ID] = viewed
})
```

## Keys

Give a control its key before construction, so its state follows the item
when the surrounding collection changes:

```go
ui.Checkbox(c.Key(todo.ID), &todo.Done, todo.Title)
ui.TextInput(c.Key("search"), &app.query)
```

Keys are unique among siblings, and compare as Go compares a map's keys: a
pointer keys the object it points to, whatever the object holds, so an
item's pointer serves as its key as well as an ID does. A key Go cannot
compare, as a slice, or not equal to itself, as a NaN, compares by its
value printed. `Element.Key` can
key a container before its children or local state are built. Use
`Context.Key` for stateful controls and custom base widgets.

`Context.Key` sets one pending key for the next control; calling it again
replaces that key. `c.Key("account").Key("secret")` uses only `"secret"`.
Create a keyed parent to give its children a separate identity scope:

```go
ui.Column(c.Key(app.accountID)).Children(func() {
	ui.TextInput(c.Key("secret"), &app.secret).Password()
})
```

Changing the account ID gives the input a new identity, even though its
own key stays `"secret"`. Within the same account, that input keeps its
state from build to build. The new editor starts with `app.secret`; clear
or load that model value separately when switching accounts.

## Change the state from other goroutines

The view reads your state on the main thread. Change it from other
goroutines with `Window.Update`, which runs a function on the main thread
and then draws a new frame:

```go
win := mygo.NewWindow(mygo.WindowOptions{Content: ui.View(app.view)})
go func() {
	items, err := fetchItems()
	win.Update(func() { app.items, app.err = items, err })
}()
```

`Window.Invalidate` only draws a new frame, for state you guard yourself.
Within the view, `c.Invalidate()` asks for another frame and `c.After(d)` for
one after a delay, such as a clock's next second.

`Update` builds the view again after every call, whether the function
changed the state or not. A goroutine that polls data that rarely
changes, as a connection count or a status, compares it on the main thread
with `mygo.RunOnMain`, which draws nothing, and calls `Invalidate` only
when it changed:

```go
for n := range counts {
	mygo.RunOnMain(func() {
		if n != app.connections {
			app.connections = n
			win.Invalidate()
		}
	})
}
```

## State of an element

An element keeps state of its own from frame to frame with `ui.Local`, for
widgets you build yourself; see [custom widgets](custom-widgets.md). A
value of it that is an `io.Closer` closes when the element is no longer
built, which stops the work the widget started
([stopping what a widget started](custom-widgets.md#stopping-what-a-widget-started)).

## Check build lifetimes

Run `mygo vet` from your app directory to find build values stored in
structs, package variables or goroutines. See [Migration](migration.md)
for the source migration command and [Performance](performance.md) for
measured frame costs.
