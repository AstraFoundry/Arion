package main

var samples = []struct{ name, text string }{
	{"server.go", `// Package server serves greetings over HTTP.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Server greets the names it is sent, and counts them.
type Server struct {
	mu     sync.Mutex
	counts map[string]int
	prefix string
}

// New returns a server greeting with prefix.
func New(prefix string) *Server {
	return &Server{counts: map[string]int{}, prefix: prefix}
}

// ServeHTTP greets the name in the query, as JSON.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.counts[name]++
	n := s.counts[name]
	s.mu.Unlock()
	// TODO: rate limit greetings per address.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"greeting": fmt.Sprintf("%s, %s!", s.prefix, name),
		"count":    n,
	})
}
`},
	{"App.tsx", `import { useEffect, useState } from "react";

type Todo = { id: number; title: string; done: boolean };

// TODO: persist todos in local storage.
export function App({ initial }: { initial: Todo[] }) {
  const [todos, setTodos] = useState<Todo[]>(initial);
  const [filter, setFilter] = useState<"all" | "open">("all");

  useEffect(() => {
    document.title = ` + "`${todos.filter((t) => !t.done).length} open`" + `;
  }, [todos]);

  const shown = filter === "all" ? todos : todos.filter((t) => !t.done);
  return (
    <ul className="todos">
      {shown.map((t) => (
        <li key={t.id} onClick={() => setTodos(toggle(todos, t.id))}>
          {t.title}
        </li>
      ))}
    </ul>
  );
}

function toggle(todos: Todo[], id: number): Todo[] {
  return todos.map((t) => (t.id === id ? { ...t, done: !t.done } : t));
}
`},
	{"style.css", `:root {
  --accent: #2563eb;
  --radius: 6px;
}

.todos {
  display: grid;
  gap: 8px;
  padding: 0;
}

.todos li:hover {
  background: color-mix(in oklab, var(--accent) 12%, white);
  border-radius: var(--radius);
}
`},
	{"fib.py", `"""Fibonacci numbers, three ways."""
from functools import cache


@cache
def fib(n: int) -> int:
    """The nth Fibonacci number, recursively."""
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)


def fib_iter(n):
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a


if __name__ == "__main__":
    # TODO: take n from the command line.
    print(f"fib(30) = {fib(30)}, {fib_iter(30)!r}")
`},
	{"README.md", "# Code editor\n\nAn example of **MyGo**'s `ui.CodeEditor` and `ui.DiffViewer`.\n\n- Edit a file, then open *Changes*.\n- Press `Cmd+D` to select the next occurrence.\n\n```go\nst := &ui.CodeEditorState{Buffer: ui.NewCodeBuffer(src)}\nui.CodeEditor(c, st).Fill()\n```\n"},
}
