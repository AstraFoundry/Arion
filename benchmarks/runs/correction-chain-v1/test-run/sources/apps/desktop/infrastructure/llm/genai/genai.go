// Package genai binds the ONNX Runtime GenAI C API through purego, so the
// LLM worker loads a provisioned runtime at run time without cgo and never
// shares a process with the speech runtime's ONNX Runtime.
package genai

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// LibraryName is the runtime file name inside a provisioned runtime directory.
func LibraryName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime-genai.dylib"
	case "windows":
		return "onnxruntime-genai.dll"
	}
	return "libonnxruntime-genai.so"
}

type api struct {
	resultGetError  func(uintptr) string
	destroyResult   func(uintptr)
	destroyString   func(unsafe.Pointer)
	shutdown        func()
	setTelemetry    func(bool)
	createConfig    func(string, *uintptr) uintptr
	configClear     func(uintptr) uintptr
	configOverlay   func(uintptr, string) uintptr
	destroyConfig   func(uintptr)
	createModel     func(uintptr, *uintptr) uintptr
	destroyModel    func(uintptr)
	createTokenizer func(uintptr, *uintptr) uintptr
	destroyTok      func(uintptr)
	encode          func(uintptr, string, uintptr) uintptr
	decode          func(uintptr, unsafe.Pointer, uintptr, *unsafe.Pointer) uintptr
	eosIDs          func(uintptr, *unsafe.Pointer, *uintptr) uintptr
	createSeqs      func(*uintptr) uintptr
	destroySeqs     func(uintptr)
	seqCount        func(uintptr, uintptr) uintptr
	seqData         func(uintptr, uintptr) unsafe.Pointer
	createParams    func(uintptr, *uintptr) uintptr
	destroyParams   func(uintptr)
	setNumber       func(uintptr, string, float64) uintptr
	setBool         func(uintptr, string, bool) uintptr
	createGen       func(uintptr, uintptr, *uintptr) uintptr
	destroyGen      func(uintptr)
	appendTokens    func(uintptr, unsafe.Pointer, uintptr) uintptr
	generateNext    func(uintptr) uintptr
	nextTokens      func(uintptr, *unsafe.Pointer, *uintptr) uintptr
	isDone          func(uintptr) bool
	rewindTo        func(uintptr, uintptr) uintptr
	tokenCount      func(uintptr) uintptr
}

var (
	loadOnce sync.Once
	loaded   *api
	loadErr  error
)

// Load opens the runtime in dir once per process. GenAI resolves its ONNX
// Runtime from the same directory; telemetry is disabled before any use.
func Load(dir string) error {
	loadOnce.Do(func() {
		handle, err := open(filepath.Join(dir, LibraryName()))
		if err != nil {
			loadErr = fmt.Errorf("load genai runtime: %w", err)
			return
		}
		a := &api{}
		bind := []struct {
			fn   any
			name string
		}{
			{&a.resultGetError, "OgaResultGetError"}, {&a.destroyResult, "OgaDestroyResult"}, {&a.destroyString, "OgaDestroyString"},
			{&a.shutdown, "OgaShutdown"}, {&a.setTelemetry, "OgaSetTelemetryEnabled"},
			{&a.createConfig, "OgaCreateConfig"}, {&a.configClear, "OgaConfigClearProviders"}, {&a.configOverlay, "OgaConfigOverlay"},
			{&a.destroyConfig, "OgaDestroyConfig"}, {&a.createModel, "OgaCreateModelFromConfig"}, {&a.destroyModel, "OgaDestroyModel"},
			{&a.createTokenizer, "OgaCreateTokenizer"}, {&a.destroyTok, "OgaDestroyTokenizer"}, {&a.encode, "OgaTokenizerEncode"},
			{&a.decode, "OgaTokenizerDecode"}, {&a.eosIDs, "OgaTokenizerGetEosTokenIds"},
			{&a.createSeqs, "OgaCreateSequences"}, {&a.destroySeqs, "OgaDestroySequences"},
			{&a.seqCount, "OgaSequencesGetSequenceCount"}, {&a.seqData, "OgaSequencesGetSequenceData"},
			{&a.createParams, "OgaCreateGeneratorParams"}, {&a.destroyParams, "OgaDestroyGeneratorParams"},
			{&a.setNumber, "OgaGeneratorParamsSetSearchNumber"}, {&a.setBool, "OgaGeneratorParamsSetSearchBool"},
			{&a.createGen, "OgaCreateGenerator"}, {&a.destroyGen, "OgaDestroyGenerator"},
			{&a.appendTokens, "OgaGenerator_AppendTokens"}, {&a.generateNext, "OgaGenerator_GenerateNextToken"},
			{&a.nextTokens, "OgaGenerator_GetNextTokens"}, {&a.isDone, "OgaGenerator_IsDone"},
			{&a.rewindTo, "OgaGenerator_RewindTo"}, {&a.tokenCount, "OgaGenerator_TokenCount"},
		}
		for _, b := range bind {
			symbol, err := lookup(handle, b.name)
			if err != nil {
				loadErr = fmt.Errorf("genai symbol %s: %w", b.name, err)
				return
			}
			purego.RegisterFunc(b.fn, symbol)
		}
		a.setTelemetry(false)
		loaded = a
	})
	return loadErr
}

// Shutdown tears the runtime down; every model must be closed first.
func Shutdown() {
	if loaded != nil {
		loaded.shutdown()
	}
}

func check(result uintptr) error {
	if result == 0 {
		return nil
	}
	message := loaded.resultGetError(result)
	loaded.destroyResult(result)
	return errors.New(message)
}

// Model owns resident weights and their tokenizer.
type Model struct {
	model, tokenizer uintptr
	eos              map[int32]bool
}

// Options fix the CPU execution policy of one model.
type Options struct {
	IntraOpThreads int
	InterOpThreads int
}

// OpenModel loads a GenAI model directory on the CPU provider.
func OpenModel(dir string, options Options) (*Model, error) {
	if loaded == nil {
		return nil, errors.New("genai runtime not loaded")
	}
	var config uintptr
	if err := check(loaded.createConfig(dir, &config)); err != nil {
		return nil, err
	}
	defer loaded.destroyConfig(config)
	if err := check(loaded.configClear(config)); err != nil {
		return nil, err
	}
	overlay := fmt.Sprintf(`{"model":{"decoder":{"session_options":{"intra_op_num_threads":%d,"inter_op_num_threads":%d}}}}`,
		options.IntraOpThreads, options.InterOpThreads)
	if err := check(loaded.configOverlay(config, overlay)); err != nil {
		return nil, err
	}
	m := &Model{eos: map[int32]bool{}}
	if err := check(loaded.createModel(config, &m.model)); err != nil {
		return nil, err
	}
	if err := check(loaded.createTokenizer(m.model, &m.tokenizer)); err != nil {
		loaded.destroyModel(m.model)
		return nil, err
	}
	var ids unsafe.Pointer
	var count uintptr
	if err := check(loaded.eosIDs(m.tokenizer, &ids, &count)); err != nil {
		m.Close()
		return nil, err
	}
	for _, id := range unsafe.Slice((*int32)(ids), count) {
		m.eos[id] = true
	}
	return m, nil
}

// Close releases the tokenizer and weights; generators must be closed first.
func (m *Model) Close() {
	if m.tokenizer != 0 {
		loaded.destroyTok(m.tokenizer)
		m.tokenizer = 0
	}
	if m.model != 0 {
		loaded.destroyModel(m.model)
		m.model = 0
	}
}

// IsEOS reports a control token that ends the answer.
func (m *Model) IsEOS(token int32) bool { return m.eos[token] }

// Encode tokenizes text exactly, with no special tokens added.
func (m *Model) Encode(text string) ([]int32, error) {
	var seqs uintptr
	if err := check(loaded.createSeqs(&seqs)); err != nil {
		return nil, err
	}
	defer loaded.destroySeqs(seqs)
	if err := check(loaded.encode(m.tokenizer, text, seqs)); err != nil {
		return nil, err
	}
	n := loaded.seqCount(seqs, 0)
	data := loaded.seqData(seqs, 0)
	if n == 0 {
		return nil, nil
	}
	return append([]int32(nil), unsafe.Slice((*int32)(data), n)...), nil
}

// Decode converts tokens to text.
func (m *Model) Decode(tokens []int32) (string, error) {
	if len(tokens) == 0 {
		return "", nil
	}
	var out unsafe.Pointer
	if err := check(loaded.decode(m.tokenizer, unsafe.Pointer(&tokens[0]), uintptr(len(tokens)), &out)); err != nil {
		return "", err
	}
	defer loaded.destroyString(out)
	return goString(out), nil
}

func goString(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(p, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(p), n))
}

// Generator is one greedy KV state. It is not safe for concurrent use.
type Generator struct {
	model     *Model
	params    uintptr
	generator uintptr
}

// NewGenerator allocates a greedy generator bounded to maxLength tokens.
func (m *Model) NewGenerator(maxLength int) (*Generator, error) {
	g := &Generator{model: m}
	if err := check(loaded.createParams(m.model, &g.params)); err != nil {
		return nil, err
	}
	if err := check(loaded.setNumber(g.params, "max_length", float64(maxLength))); err != nil {
		g.Close()
		return nil, err
	}
	if err := check(loaded.setBool(g.params, "do_sample", false)); err != nil {
		g.Close()
		return nil, err
	}
	if err := check(loaded.createGen(m.model, g.params, &g.generator)); err != nil {
		g.Close()
		return nil, err
	}
	return g, nil
}

func (g *Generator) Close() {
	if g.generator != 0 {
		loaded.destroyGen(g.generator)
		g.generator = 0
	}
	if g.params != 0 {
		loaded.destroyParams(g.params)
		g.params = 0
	}
}

// Append prefills tokens into the KV state.
func (g *Generator) Append(tokens []int32) error {
	if len(tokens) == 0 {
		return nil
	}
	err := check(loaded.appendTokens(g.generator, unsafe.Pointer(&tokens[0]), uintptr(len(tokens))))
	runtime.KeepAlive(tokens)
	return err
}

// Len is the number of tokens held, prompt and generated.
func (g *Generator) Len() int { return int(loaded.tokenCount(g.generator)) }

// RewindTo drops KV state after length tokens.
func (g *Generator) RewindTo(length int) error {
	return check(loaded.rewindTo(g.generator, uintptr(length)))
}

// Done reports EOS or the max-length bound.
func (g *Generator) Done() bool { return loaded.isDone(g.generator) }

// Next decodes one greedy token.
func (g *Generator) Next() (int32, error) {
	if err := check(loaded.generateNext(g.generator)); err != nil {
		return 0, err
	}
	var out unsafe.Pointer
	var count uintptr
	if err := check(loaded.nextTokens(g.generator, &out, &count)); err != nil {
		return 0, err
	}
	if count != 1 {
		return 0, fmt.Errorf("expected one next token, got %d", count)
	}
	return *(*int32)(out), nil
}
