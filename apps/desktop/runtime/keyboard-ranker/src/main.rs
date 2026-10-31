use chinese_ime_lm::{CandidateFacts, Reranker, SentenceModel};
use serde::Deserialize;
use std::{
    io::{self, BufRead, Write},
    sync::Arc,
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Candidate {
    id: String,
    text: String,
    source: String,
    start: usize,
    end: usize,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Request {
    session: String,
    input: String,
    before: String,
    candidates: Vec<Candidate>,
}
fn valid(r: &Request) -> bool {
    !r.session.is_empty()
        && r.session.len() <= 128
        && !r.input.is_empty()
        && r.input.len() <= 256
        && r.before.len() <= 2048
        && !r.candidates.is_empty()
        && r.candidates.len() <= 27
        && r.candidates.iter().all(|c| {
            !c.id.is_empty()
                && c.id.len() <= 64
                && !c.text.is_empty()
                && c.text.len() <= 256
                && c.source.len() <= 64
                && c.start < c.end
                && c.end <= r.input.len()
                && r.input.is_char_boundary(c.start)
                && r.input.is_char_boundary(c.end)
        })
}
fn run() -> Result<(), Box<dyn std::error::Error>> {
    let path = std::env::args().nth(1).ok_or("model path required")?;
    let bytes = std::fs::read(path)?;
    let model = Arc::new(SentenceModel::load(&bytes)?);
    if model.license() != "Apache-2.0" || model.attribution().is_empty() {
        return Err("missing model provenance".into());
    }
    let mut ranker = Reranker::new(model.clone());
    let mut session = String::new();
    let mut input = io::stdin().lock();
    let mut output = io::stdout().lock();
    writeln!(output, "{{\"ready\":true}}")?;
    output.flush()?;
    loop {
        let mut line = Vec::new();
        // Refuse overlong messages before allocating their full payload.
        while line.len() <= 32768 {
            let bytes = input.fill_buf()?;
            if bytes.is_empty() {
                return Ok(());
            }
            let n = bytes
                .iter()
                .position(|b| *b == b'\n')
                .map(|p| p + 1)
                .unwrap_or(bytes.len())
                .min(32769 - line.len());
            let done = bytes[n - 1] == b'\n';
            line.extend_from_slice(&bytes[..n]);
            input.consume(n);
            if done {
                break;
            }
        }
        if line.len() > 32768 {
            return Err("request exceeds bounds".into());
        }
        let request = serde_json::from_slice::<Request>(&line)?;
        if !valid(&request) {
            return Err("invalid request".into());
        }
        if request.session != session {
            ranker = Reranker::new(model.clone());
            session = request.session.clone();
        }
        let texts: Vec<&str> = request.candidates.iter().map(|c| c.text.as_str()).collect();
        let chosen = ranker
            .best_where(&request.before, &texts, |i| {
                let c = &request.candidates[i];
                CandidateFacts {
                    answers_key: c.start == 0 && c.end == request.input.len(),
                    trusted_dictionary_hit: matches!(c.source.as_str(), "phrase" | "user_phrase"),
                }
            })
            .unwrap_or(0);
        serde_json::to_writer(
            &mut output,
            &serde_json::json!({"pick":request.candidates[chosen].id}),
        )?;
        writeln!(output)?;
        output.flush()?;
    }
}
fn main() {
    // Error details can contain supplied text or paths; never print them.
    if run().is_err() {
        eprintln!("KEYBOARD_WORKER_FAILED");
        std::process::exit(1);
    }
}
