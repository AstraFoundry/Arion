//! egui's app for the comparison of benchmarks/README.md, with the scenes the
//! other frameworks' apps have.

#[path = "../../common.rs"]
mod common;

use common::{Recorder, cell_color, cell_value, document_width, words};
use eframe::egui;

struct Counter {
    n: i32,
    frames: usize,
}

impl eframe::App for Counter {
    fn ui(&mut self, root: &mut egui::Ui, _frame: &mut eframe::Frame) {
        self.frames += 1;
        if self.frames == 1 {
            root.ctx().request_repaint();
        } else if self.frames == 2 {
            println!("ready");
        }
        egui::CentralPanel::default().show(root, |ui| {
            ui.vertical_centered(|ui| {
                ui.add_space(70.0);
                ui.label(egui::RichText::new(self.n.to_string()).size(56.0).strong());
                ui.add_space(16.0);
                ui.horizontal(|ui| {
                    ui.add_space(100.0);
                    if ui.button("−").clicked() {
                        self.n -= 1;
                    }
                    if ui.button("Reset").clicked() {
                        self.n = 0;
                    }
                    if ui.button("+").clicked() {
                        self.n += 1;
                    }
                });
            });
        });
    }
}

struct List {
    rec: Recorder,
    y: f32,
}

impl eframe::App for List {
    fn ui(&mut self, root: &mut egui::Ui, _frame: &mut eframe::Frame) {
        if self.rec.tick() {
            root.ctx().request_repaint();
            self.y += 8.0;
        }
        egui::CentralPanel::default()
            .frame(egui::Frame::central_panel(root.style()).inner_margin(0.0))
            .show(root, |ui| {
                egui::ScrollArea::vertical()
                    .auto_shrink(false)
                    .vertical_scroll_offset(self.y)
                    .show_rows(ui, 40.0, 100_000, |ui, rows| {
                        ui.spacing_mut().item_spacing.y = 0.0;
                        for i in rows {
                            ui.allocate_ui(egui::vec2(ui.available_width(), 40.0), |ui| {
                                ui.set_height(40.0);
                                ui.horizontal_centered(|ui| {
                                    ui.add_space(12.0);
                                    ui.vertical(|ui| {
                                        ui.label(
                                            egui::RichText::new(format!("Item {i}")).size(13.0),
                                        );
                                        ui.label(
                                            egui::RichText::new(words(i + 1, 6)).size(11.0).weak(),
                                        );
                                    });
                                    ui.with_layout(
                                        egui::Layout::right_to_left(egui::Align::Center),
                                        |ui| {
                                            ui.add_space(12.0);
                                            ui.label(
                                                egui::RichText::new(format!(
                                                    "{:.2}",
                                                    cell_value(i, 0)
                                                ))
                                                .size(13.0),
                                            );
                                        },
                                    );
                                });
                            });
                        }
                    });
            });
    }
}

struct Grid {
    rec: Recorder,
    frame: usize,
}

impl eframe::App for Grid {
    fn ui(&mut self, root: &mut egui::Ui, _frame: &mut eframe::Frame) {
        if self.rec.tick() {
            root.ctx().request_repaint();
            self.frame += 1;
        }
        egui::CentralPanel::default()
            .frame(egui::Frame::NONE)
            .show(root, |ui| {
                ui.spacing_mut().item_spacing = egui::Vec2::ZERO;
                for row in 0..30 {
                    ui.horizontal(|ui| {
                        for col in 0..20 {
                            let v = cell_value(row * 20 + col, self.frame);
                            let (r, g, b) = cell_color(v);
                            egui::Frame::new()
                                .fill(egui::Color32::from_rgb(r, g, b))
                                .show(ui, |ui| {
                                    ui.set_min_size(egui::vec2(40.0, 20.0));
                                    ui.set_max_size(egui::vec2(40.0, 20.0));
                                    ui.centered_and_justified(|ui| {
                                        ui.label(
                                            egui::RichText::new(format!("{v:.2}"))
                                                .size(11.0)
                                                .color(egui::Color32::from_rgb(0x10, 0x10, 0x10)),
                                        );
                                    });
                                });
                        }
                    });
                }
            });
    }
}

struct Document {
    rec: Recorder,
    paragraphs: Vec<String>,
    frame: usize,
}

impl eframe::App for Document {
    fn ui(&mut self, root: &mut egui::Ui, _frame: &mut eframe::Frame) {
        if self.rec.tick() {
            root.ctx().request_repaint();
            self.frame += 1;
        }
        let width = document_width(self.frame);
        egui::CentralPanel::default().show(root, |ui| {
            egui::ScrollArea::vertical().auto_shrink(false).show(ui, |ui| {
                ui.set_max_width(width);
                ui.spacing_mut().item_spacing.y = 8.0;
                for p in &self.paragraphs {
                    ui.add(egui::Label::new(egui::RichText::new(p).size(13.0)).wrap());
                }
            });
        });
    }
}

fn main() -> eframe::Result {
    let scene = std::env::args().nth(1).unwrap_or_else(|| "counter".into());
    let size = if scene == "counter" { [360.0, 260.0] } else { [800.0, 600.0] };
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_inner_size(size),
        ..Default::default()
    };
    eframe::run_native(
        "Benchmark",
        options,
        Box::new(move |_cc| -> Result<Box<dyn eframe::App>, _> {
            Ok(match scene.as_str() {
                "counter" => Box::new(Counter { n: 0, frames: 0 }),
                "list" => Box::new(List { rec: Recorder::new(), y: 0.0 }),
                "grid" => Box::new(Grid { rec: Recorder::new(), frame: 0 }),
                "document" => Box::new(Document {
                    rec: Recorder::new(),
                    paragraphs: (0..40).map(|i| words(i + 1, 70)).collect(),
                    frame: 0,
                }),
                other => panic!("no scene {other:?}"),
            })
        }),
    )
}
