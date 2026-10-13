# Application use cases

`Coordinator` orchestrates one dictation session at a time over domain-owned [ports](../domain/ports/ports.go): target capture, recognizer stream, bounded frame queue, optional acoustic finalizer, correction policy and the single commit. Every public method is safe for concurrent use; a new generation invalidates late callbacks. `Preferences` holds the user's persisted choices. Concrete adapters are wired by the desktop composition root.
