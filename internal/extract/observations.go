package extract

// Detector observations: the parallel channel of per-call context (assignment
// shape, loop ancestry, enclosing signature) that Site cannot carry. Site's
// JSON IS the manifest facts contract (D1), so observations never touch it —
// they exist only in memory, produced by ExtractWithObservations, consumed by
// the lint face and check. Detectors run on resolved-confidence captures only
// (D3): the weak token tier structurally never reaches a detector, because
// only the resolved branch of scanFunc hands a capture over.

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Observation is one detector fact about a call site.
type Observation struct {
	SiteID   string `json:"siteId"`   // enclosing symbol anchor (Site.ID space)
	Rule     string `json:"rule"`     // "err-dropped" — no io-surface/ prefix
	Call     string `json:"call"`     // watched call, e.g. "os.ReadFile"
	Location string `json:"location"` // repo-relative "path:line" of the call
	Evidence string `json:"evidence"` // human clause, e.g. `x, _ := os.ReadFile(path)`
}

// Rule names for the detector vocabulary (D7: additive and stable; the lint
// face prefixes io-surface/ to form the finding rule ID). ignore-invalid is
// metadata-only: emitted by the suppression scanner, never by a detector
// capture (see suppress.go).
const (
	RuleErrDropped      = "err-dropped"
	RuleCloserMissing   = "closer-missing"
	RuleCtxDropped      = "ctx-dropped"
	RuleDeferInLoop     = "defer-in-loop"
	RuleClientNoTimeout = "client-no-timeout"
	RuleStatThenOpen    = "stat-then-open"
	RuleIOInLoop        = "io-in-loop"
	RuleWriteNonAtomic  = "write-non-atomic"
	RuleTinyBuffer      = "tiny-buffer"
	RuleIgnoreInvalid   = "ignore-invalid"
)

// Assign shapes: how a call's results are consumed at the statement level.
const (
	shapeBare       = "bare"       // f() as its own statement — every result dropped
	shapeAssign     = "assign"     // a, b := f() — per-position blanks recorded
	shapeReturned   = "returned"   // return f() — results propagate
	shapeDeferred   = "deferred"   // defer/go f() — deliberate fire-and-forget
	shapeExpression = "expression" // consumed inside a larger expression
)

// capture is the local context of one call inside a function body.
type capture struct {
	id       string // enclosing symbol ID (set by the tracker)
	node     *ast.CallExpr
	parent   ast.Node // the statement directly containing the call
	fn       *ast.FuncDecl
	imports  map[string]string
	sel      string // resolved key, e.g. "os.ReadFile" (set at detect time)
	family   string
	location string
	shape    string
	slotIdx  int   // assign shape: which RHS expression this call is
	slots    int   // assign shape: len(Lhs)
	blanks   []int // assign shape: LHS positions bound to _
	inLoop   bool  // inside a for/range BODY subtree (loop heads stay false)
	ctxParam bool  // enclosing func has a context.Context parameter
	// io-in-loop same-path refinement (superb T16): the innermost enclosing
	// loop, the call's printed first-argument key, and how often that key
	// has occurred in the SAME loop so far. Together they separate true N+1
	// (fixed or repeated target) from per-item I/O (target bound to the
	// range variables), the fleet's dominant non-actionable class.
	loopRef   ast.Node
	targetKey string
	loopSeen  int
	perItem   bool
}

// ExtractWithObservations walks root exactly like Extract and additionally
// returns the detector observations. Sites are identical between the two
// entrypoints — pinned by test — because the capture machinery writes no site
// state (D1).
func ExtractWithObservations(root string, opts Options) ([]*Site, []Observation, error) {
	sites, obs, err := runExtract(root, opts, true)

	return sites, obs, err
}

// captureTracker annotates one function body with parent, loop depth, and
// per-call capture contexts. ast.Walk guarantees pre-order Visit and
// post-order Visit(nil), so the stack discipline below is exact.
type captureTracker struct {
	e        *extractor
	fn       *ast.FuncDecl
	id       string
	imports  map[string]string
	ctxParam bool
	parents  map[ast.Node]ast.Node
	inLoop   map[ast.Node]bool
	stack    []ast.Node
	loops    int
	// loopTargets counts, per loop node, the printed first-arg key of every
	// watched call; loopRangeIdents caches each range statement's iteration
	// identifiers (key/value) so per-item targets can be recognized.
	loopTargets     map[ast.Node]map[string]int
	loopRangeIdents map[ast.Node]map[string]bool
}

func (t *captureTracker) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		popped := t.stack[len(t.stack)-1]
		t.stack = t.stack[:len(t.stack)-1]
		if isLoopNode(popped) {
			t.loops--
		}

		return nil
	}
	var parent ast.Node
	if len(t.stack) > 0 {
		parent = t.stack[len(t.stack)-1]
	}
	t.parents[n] = parent
	t.inLoop[n] = t.loops > 0
	if rng, ok := n.(*ast.RangeStmt); ok {
		if t.loopRangeIdents == nil {
			t.loopRangeIdents = map[ast.Node]map[string]bool{}
		}
		t.loopRangeIdents[n] = rangeIdents(rng)
	}
	if isLoopNode(n) {
		t.loops++
	}
	t.stack = append(t.stack, n)
	if call, ok := n.(*ast.CallExpr); ok {
		t.record(call)
	}

	return t
}

// rangeIdents names the iteration identifiers of a range statement (key and
// value); nil when neither is bound to an identifier.
func rangeIdents(rng *ast.RangeStmt) map[string]bool {
	out := map[string]bool{}
	for _, expr := range []ast.Expr{rng.Key, rng.Value} {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
			out[ident.Name] = true
		}
	}

	return out
}

func isLoopNode(n ast.Node) bool {
	switch n.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return true
	}

	return false
}

// innermostLoop walks the parent chain to the nearest enclosing loop node,
// nil when the node sits outside any loop body subtree.
func innermostLoop(parents map[ast.Node]ast.Node, n ast.Node) ast.Node {
	for cur := n; cur != nil; cur = parents[cur] {
		if cur != n && isLoopNode(cur) {
			return cur
		}
	}

	return nil
}

// record computes the statement-level shape of one call and stores its
// capture. Blanks are recorded positionally so err-dropped can name the exact
// dropped result.
func (t *captureTracker) record(call *ast.CallExpr) {
	parent := t.parents[call]
	c := &capture{
		id:       t.id,
		node:     call,
		parent:   parent,
		fn:       t.fn,
		imports:  t.imports,
		inLoop:   t.inLoop[call],
		ctxParam: t.ctxParam,
		location: t.e.locOf(call),
	}
	if c.inLoop {
		c.loopRef = innermostLoop(t.parents, call)
		if len(call.Args) > 0 {
			c.targetKey = evidenceOf(&t.e.fset, call.Args[0])
		}
		if c.loopRef != nil {
			if t.loopTargets == nil {
				t.loopTargets = map[ast.Node]map[string]int{}
			}
			counts := t.loopTargets[c.loopRef]
			if counts == nil {
				counts = map[string]int{}
				t.loopTargets[c.loopRef] = counts
			}
			counts[c.targetKey]++
			c.loopSeen = counts[c.targetKey]
			c.perItem = t.loopRangeIdents[c.loopRef][c.targetKey]
		}
	}
	switch st := parent.(type) {
	case *ast.ExprStmt:
		c.shape = shapeBare
	case *ast.AssignStmt:
		idx := nodeIndex(st.Rhs, call)
		if idx >= 0 && (len(st.Rhs) == 1 || len(st.Lhs) == len(st.Rhs)) {
			c.shape = shapeAssign
			c.slotIdx = idx
			c.slots = len(st.Lhs)
			for i, lhs := range st.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "_" {
					c.blanks = append(c.blanks, i)
				}
			}
		} else {
			c.shape = shapeExpression
		}
	case *ast.ReturnStmt:
		c.shape = shapeReturned
	case *ast.DeferStmt, *ast.GoStmt:
		c.shape = shapeDeferred
	default:
		c.shape = shapeExpression
	}
	t.e.captures[call] = c
	t.e.captureList = append(t.e.captureList, c)
}

func nodeIndex(list []ast.Expr, n ast.Node) int {
	for i, e := range list {
		if e == n {
			return i
		}
	}

	return -1
}

// captureContexts fills the capture map for one function body.
func (e *extractor) captureContexts(fn *ast.FuncDecl, imports map[string]string, id string) {
	t := &captureTracker{
		e:        e,
		fn:       fn,
		id:       id,
		imports:  imports,
		ctxParam: hasContextParam(fn),
		parents:  map[ast.Node]ast.Node{},
		inLoop:   map[ast.Node]bool{},
	}
	ast.Walk(t, fn.Body)
}

// hasContextParam reports whether any parameter's type is context.Context
// (selector form; dot-imported Context is not recognized — deliberate, it is
// vanishingly rare and the wrong scope for a hermetic heuristic).
func hasContextParam(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
				return true
			}
		}
	}

	return false
}

// locOf formats a node position as the repo-relative "path:line" the manifest
// contract keys on. Same formatting as the suppression scanner's relPath();
// factored so observations and sites can never drift apart.
func (e *extractor) locOf(n ast.Node) string {
	pos := e.fset.Position(n.Pos())

	return e.relPath(pos.Filename) + ":" + strconv.Itoa(pos.Line)
}

// evidence renders an AST node as the single-line source form used in
// Observation.Evidence (deterministic printer output, whitespace flattened).
func evidenceOf(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, n); err != nil {
		return ""
	}

	return strings.Join(strings.Fields(buf.String()), " ")
}

// runDetectors executes the capture-shaped and declaration-shaped detector
// registries after the walk: captureList preserves deterministic walk order,
// and sortedObservations normalizes the global ordering.
func (e *extractor) runDetectors(fnID string, fn *ast.FuncDecl, imports map[string]string, captures []*capture) {
	for _, c := range captures {
		for _, d := range detectors {
			e.observations = append(e.observations, d.run(e, c)...)
		}
	}
	for _, d := range fnDetectors {
		e.observations = append(e.observations, d.run(e, fn, imports, fnID)...)
	}
}

// emit builds one observation from a capture.
func (e *extractor) emit(c *capture, rule, ev string) Observation {
	return Observation{SiteID: c.id, Rule: rule, Call: c.sel, Location: c.location, Evidence: ev}
}

// sortedObservations orders observations deterministically: by location, then
// rule, then site — so goldens and fleet runs are byte-stable.
func sortedObservations(obs []Observation) []Observation {
	sort.SliceStable(obs, func(i, j int) bool {
		a, b := obs[i], obs[j]
		if a.Location != b.Location {
			return a.Location < b.Location
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}

		return a.SiteID < b.SiteID
	})

	return obs
}
