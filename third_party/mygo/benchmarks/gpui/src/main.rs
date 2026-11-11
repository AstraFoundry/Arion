//! GPUI's app for the comparison of benchmarks/README.md, with the scenes the
//! other frameworks' apps have.

#[path = "../../common.rs"]
mod common;

use common::{Recorder, cell_color, cell_value, document_width, words};
use gpui::{
    App, Application, Bounds, Context, Menu, MenuItem, Rgba, SharedString, TitlebarOptions, UniformListScrollHandle, Window,
    WindowBounds, WindowOptions, actions, div, point, prelude::*, px, rgb, size, uniform_list,
};

actions!(benchmark, [Quit]);

fn color(c: (u8, u8, u8)) -> Rgba {
    rgb((c.0 as u32) << 16 | (c.1 as u32) << 8 | c.2 as u32)
}

struct Counter {
    n: i32,
    frames: usize,
}

fn button(id: &'static str, label: &'static str) -> gpui::Stateful<gpui::Div> {
    div()
        .id(id)
        .px_3()
        .py_1()
        .rounded_md()
        .bg(rgb(0xe8e8e8))
        .text_size(px(13.0))
        .child(label)
}

impl Render for Counter {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        self.frames += 1;
        if self.frames == 1 {
            window.on_next_frame(|_, _| println!("ready"));
        }
        div()
            .size_full()
            .flex()
            .flex_col()
            .items_center()
            .justify_center()
            .gap_4()
            .bg(rgb(0xffffff))
            .text_color(rgb(0x000000))
            .child(div().text_size(px(56.0)).font_weight(gpui::FontWeight::BOLD).child(self.n.to_string()))
            .child(
                div()
                    .flex()
                    .gap_2()
                    .child(button("decrement", "−").on_click(cx.listener(|s, _, _, cx| {
                        s.n -= 1;
                        cx.notify();
                    })))
                    .child(button("reset", "Reset").on_click(cx.listener(|s, _, _, cx| {
                        s.n = 0;
                        cx.notify();
                    })))
                    .child(button("increment", "+").on_click(cx.listener(|s, _, _, cx| {
                        s.n += 1;
                        cx.notify();
                    }))),
            )
    }
}

struct List {
    rec: Recorder,
    scroll: UniformListScrollHandle,
    y: f32,
}

impl Render for List {
    fn render(&mut self, window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        if self.rec.tick() {
            window.request_animation_frame();
            self.y += 8.0;
        }
        self.scroll
            .0
            .borrow()
            .base_handle
            .set_offset(point(px(0.0), px(-self.y)));
        div()
            .size_full()
            .bg(rgb(0xffffff))
            .text_color(rgb(0x000000))
            .child(
                uniform_list(
                    "rows",
                    100_000,
                    cx.processor(|_this, range: std::ops::Range<usize>, _, _| {
                        range
                            .map(|i| {
                                div()
                                    .id(i)
                                    .w_full()
                                    .h(px(40.0))
                                    .px_3()
                                    .gap_3()
                                    .flex()
                                    .items_center()
                                    .child(
                                        div()
                                            .flex_1()
                                            .flex()
                                            .flex_col()
                                            .child(
                                                div()
                                                    .text_size(px(13.0))
                                                    .whitespace_nowrap()
                                                    .child(format!("Item {i}")),
                                            )
                                            .child(
                                                div()
                                                    .text_size(px(11.0))
                                                    .text_color(rgb(0x707070))
                                                    .whitespace_nowrap()
                                                    .child(words(i + 1, 6)),
                                            ),
                                    )
                                    .child(
                                        div()
                                            .text_size(px(13.0))
                                            .whitespace_nowrap()
                                            .child(format!("{:.2}", cell_value(i, 0))),
                                    )
                            })
                            .collect::<Vec<_>>()
                    }),
                )
                .size_full()
                .track_scroll(self.scroll.clone()),
            )
    }
}

struct Grid {
    rec: Recorder,
    frame: usize,
}

impl Render for Grid {
    fn render(&mut self, window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        if self.rec.tick() {
            window.request_animation_frame();
            self.frame += 1;
        }
        let frame = self.frame;
        div()
            .size_full()
            .flex()
            .flex_col()
            .bg(rgb(0xffffff))
            .children((0..30).map(move |row| {
                div().flex().children((0..20).map(move |col| {
                    let v = cell_value(row * 20 + col, frame);
                    div()
                        .w(px(40.0))
                        .h(px(20.0))
                        .flex()
                        .items_center()
                        .justify_center()
                        .bg(color(cell_color(v)))
                        .text_size(px(11.0))
                        .text_color(rgb(0x101010))
                        .whitespace_nowrap()
                        .child(format!("{v:.2}"))
                }))
            }))
    }
}

struct Document {
    rec: Recorder,
    paragraphs: Vec<SharedString>,
    frame: usize,
}

impl Render for Document {
    fn render(&mut self, window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        if self.rec.tick() {
            window.request_animation_frame();
            self.frame += 1;
        }
        let width = document_width(self.frame);
        div()
            .id("document")
            .size_full()
            .overflow_y_scroll()
            .bg(rgb(0xffffff))
            .text_color(rgb(0x000000))
            .child(
                div()
                    .w(px(width))
                    .p_3()
                    .flex()
                    .flex_col()
                    .gap_2()
                    .text_size(px(13.0))
                    .children(self.paragraphs.iter().map(|p| div().child(p.clone()))),
            )
    }
}

fn main() {
    let scene = std::env::args().nth(1).unwrap_or_else(|| "counter".into());
    Application::new().run(move |cx: &mut App| {
        let (w, h) = if scene == "counter" { (360.0, 260.0) } else { (800.0, 600.0) };
        let options = WindowOptions {
            focus: true,
            titlebar: Some(TitlebarOptions {
                title: Some("Benchmark".into()),
                ..Default::default()
            }),
            window_bounds: Some(WindowBounds::Windowed(Bounds::centered(None, size(px(w), px(h)), cx))),
            ..Default::default()
        };
        // A menu, as the other frameworks give an app by default.
        cx.set_menus(vec![
            Menu {
                name: "Benchmark".into(),
                items: vec![MenuItem::action("Quit", Quit)],
            },
            Menu {
                name: "Window".into(),
                items: vec![],
            },
        ]);
        match scene.as_str() {
            "counter" => {
                cx.open_window(options, |_, cx| cx.new(|_| Counter { n: 0, frames: 0 })).unwrap();
            }
            "list" => {
                cx.open_window(options, |_, cx| {
                    cx.new(|_| List {
                        rec: Recorder::new(),
                        scroll: UniformListScrollHandle::new(),
                        y: 0.0,
                    })
                })
                .unwrap();
            }
            "grid" => {
                cx.open_window(options, |_, cx| cx.new(|_| Grid { rec: Recorder::new(), frame: 0 })).unwrap();
            }
            "document" => {
                cx.open_window(options, |_, cx| {
                    cx.new(|_| Document {
                        rec: Recorder::new(),
                        paragraphs: (0..40).map(|i| words(i + 1, 70).into()).collect(),
                        frame: 0,
                    })
                })
                .unwrap();
            }
            other => panic!("no scene {other:?}"),
        }
        cx.activate(true);
    });
}
