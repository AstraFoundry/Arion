# Game

The game plugin (`plugins/game`) has what games need beyond what apps do:
MyGo is an app framework, and these are opt-in, in a package of their
own. It is all Go. Native UI draws a game as it draws the rest: package
`game` runs its loop in an element and looks at its world through a
camera, package `game/gamepad` reads its controllers, and core packages
draw its pixel art ([Pixelated bitmaps](../ui/image.md) and
[parts of them](../ui/drawing.md#parts-of-bitmaps)) and play its sound
([`audio`](../audio.md)).

[examples/mayoi](../../examples/mayoi), a little survivor game after the
anime MyGO!!!!!, which MyGo is named after, is made of all of them, and
plays here, compiled to WebAssembly. Click it, then walk with WASD or the
arrows: Tomori sings on her own at stage fright, picks up stones, and at
each level a bandmate may join her, and the music.

<!-- app mayoi 480 -->

[examples/farm](../../examples/farm) is a calmer one, a farm to till and
water.

## The game loop

A `game.Game` has two methods: `Update` advances its world one step with
the player's input, and `Draw` draws it. `game.View` runs it in an
element:

```go
import "github.com/egoist/mygo/plugins/game"

type farm struct {
	x, y   float32 // the player, in the world
	px, py float32 // where they were a step before
}

func (f *farm) Update(in *game.Input) {
	f.px, f.py = f.x, f.y
	if in.Down(ui.KeyD) {
		f.x += 2
	}
	if in.Pressed(ui.KeySpace) {
		f.jump()
	}
}

func (f *farm) Draw(p *ui.Painter, r ui.Rect, alpha float64) {
	// Between the last two steps, as far as the frame is into the next.
	x := f.px + (f.x-f.px)*float32(alpha)
	y := f.py + (f.y-f.py)*float32(alpha)
	p.ImageRegion(sheet, hero, ui.Rect{X: r.X + x, Y: r.Y + y, W: 48, H: 48}, ui.ImageOptions{})
}

mygo.NewWindow(mygo.WindowOptions{
	Title:   "Farm",
	Content: ui.View(func(c *ui.Context) { game.View(c, f).Fill() }),
})
```

Steps are fixed, `game.Rate` (60) a second, or what a `Rate() int` method
of the game says, whatever the display's rate, as Ebitengine and MonoGame
run games: a game plays alike on a display of 60 Hz and one of 120, and
its physics stay stable. `Draw` runs every frame; its `alpha`, from 0 to 1,
is how far the frame is into the next step, for drawing what moves
between its last two places. After a pause, as a window out of sight,
steps catch up on a quarter of a second at most.

### Input

`game.Input` is the player's input as a step sees it:

| method | reports |
|---|---|
| `Down(k)` | key `k` held |
| `Pressed(k)`, `Released(k)` | `k` pressed or released since the step before, even between two steps |
| `Pointer()` | where the pointer is, from the view's corner, and whether it is over it |
| `PointerDown(b)`, `PointerPressed(b)`, `PointerReleased(b)` | pointer button `b` (0 the primary) |
| `Wheel()` | how far the wheel scrolled since the step before |
| `Size()` | the view's size |
| `Focused()` | the view has the keyboard: a game pauses without it |

Keys are by position: `KeyW` is the key in W's place on any layout, as
WASD wants. The view takes the keyboard focus as it appears and when
clicked, leaves keys with ⌘, Ctrl or Alt to the menus' shortcuts, and lets
go of everything held when the window loses the keyboard, whose releases
never come. Outside the view, as for a cheat sheet of the game's keys,
[`c.KeyDown`](../ui/input.md#keys-held) tells the keys held in the window
whichever element has the focus.

### Testing

A game's `Update` is plain Go: test its world without a window. The view
runs in a `ui.Tester`, as any element: `KeyDown` and `KeyUp` hold a key
between frames, and a step runs once a frame comes a step's time after
the last.

## The camera

A `game.Camera` looks at the world's point `X, Y`, which shows at the
middle of the view, `Zoom` times as large, turned `Rotation` degrees, as
raylib's `Camera2D`. `Draw` draws a world in its own coordinates through
it, clipped to the view:

```go
cam := game.Camera{X: player.x, Y: player.y, Zoom: 3}
cam.Draw(p, r, func() {
	for _, t := range tiles {
		p.ImageRegion(sheet, t.src, t.box, ui.ImageOptions{}) // culled where the camera does not look
	}
})
wx, wy := cam.ToWorld(r, px, py) // the pointer, in the world
```

Unturned, the world moves by whole pixels of the screen, so that tiles
drawn at whole units keep their edges on them, while parts of bitmaps,
placed where they are under a transform, glide. `Visible` returns the
world's part in view, for a game that iterates its tiles itself, and
`Transform` the camera's transform, for `Painter.Transform`.

Zooms that are whole numbers give each of a `Pixelated` bitmap's pixels
the same whole number of the screen's; others keep them square, with
edges blended across one screen pixel. For a whole number at any window
size, draw into a canvas (below).

## Pixel-perfect scaling

For pixel art that is pixel-perfect at any window size, as Godot's
viewport stretch and Stardew Valley draw it, a game draws its world into a
[`ui.Canvas`](../ui/drawing.md#canvases) of a logical size and shows it at
the largest whole multiple of that size that fits, letterboxed
(`game.IntegerFit`):

```go
world := ui.NewCanvas(480, 270)
world.Bitmap().Pixelated()

func (f *farm) Draw(p *ui.Painter, r ui.Rect, alpha float64) {
	world.Draw(p, func(cp *ui.Painter) {
		cam := game.Camera{X: f.x, Y: f.y}
		cam.Draw(cp, ui.Rect{W: 480, H: 270}, f.drawWorld)
	})
	dst, _ := game.IntegerFit(p, 480, 270, r)
	p.Image(world.Bitmap(), dst, ui.FillBox)
}
```

## Gamepads

Package `plugins/game/gamepad` reads game controllers, in apps of native
UI or web pages, on every platform MyGo runs on, browsers included. Every
pad reads in one layout, the standard one of the W3C's Gamepad API and
SDL's game controller database, whatever pad it is: buttons by place, two
sticks, analog triggers and a d-pad.

```go
import "github.com/egoist/mygo/plugins/game/gamepad"

var last gamepad.State

func update() {
	pads := gamepad.Pads()
	if len(pads) == 0 {
		return
	}
	s := pads[0].State()
	x, y := s.LeftStick() // -1 to 1, right and down, with a dead zone
	player.move(x, y)
	if s.JustPressed(last, gamepad.South) {
		player.jump()
	}
	last = s
}
```

### Pads

`Pads` returns the pads connected, in the order they connected, and looks
for new ones at most once a second; a game calls it once a frame. A
`Pad` lasts from when it connects until it disconnects (`Connected` turns
false, and it leaves `Pads`); plugged in again, it is a new one. `Name` is
what its maker calls it.

`State` reads a pad's controls now, as a value: keep the last one to
tell a button just pressed (`JustPressed`) or released (`JustReleased`)
from one held.

### The layout

| button | Xbox | PlayStation | Nintendo |
|---|---|---|---|
| `South` | A | Cross | B |
| `East` | B | Circle | A |
| `West` | X | Square | Y |
| `North` | Y | Triangle | X |
| `LeftShoulder`, `RightShoulder` | LB, RB | L1, R1 | L, R |
| `LeftTrigger`, `RightTrigger` | LT, RT | L2, R2 | ZL, ZR |
| `Back`, `Start` | View, Menu | Create, Options | −, + |
| `LeftStickButton`, `RightStickButton` | LS, RS | L3, R3 | |
| `Up`, `Down`, `Left`, `Right` | d-pad | d-pad | d-pad |
| `Guide` | Xbox | PS | Home |

Buttons read from 0 to 1 (`Value`), the triggers by how far they are
pulled; `Pressed` is a quarter of the way. Axes (`Axis`) read from -1 to 1,
right and down positive, as the pad reports them; `LeftStick` and
`RightStick` apply a radial dead zone (`Deadzone`, 0.15), so that a stick
at rest reads nothing and leans as far in every direction.

### Platforms

| platform | from | pads |
|---|---|---|
| macOS | GameController.framework | Xbox, PlayStation, Switch Pro and MFi controllers |
| Windows | XInput | Xbox controllers and those that act as one (Steam Input makes others so); Guide through `XInputGetStateEx` |
| Linux | evdev, `/dev/input/event*` | what the kernel's drivers report as gamepads, readable by the user at the seat; Xbox controllers' X and Y, reported as BTN_X and BTN_Y, read in place |
| browsers | the Gamepad API | those with the standard mapping; others in the order the browser gives |

On macOS, a pad's input reaches the app in front only: in the background
it reads nothing pressed, as games expect. Browsers report a pad only
once the page saw one of its buttons pressed.
Windows sees no PlayStation or Nintendo controller without Steam Input or
a driver that makes it an XInput one.

[examples/farm](../../examples/farm) plays with a gamepad as with the
keyboard.
