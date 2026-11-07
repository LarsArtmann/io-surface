// Suppression fixture (superb plan T5): exercises the
// //io-surface:ignore grammar — valid above-line and trailing forms suppress
// their finding; a wrong-rule or malformed directive must NOT suppress and
// the malformed ones add their own ignore-invalid finding. The last two
// functions pin the T17 decisions (2026-11-07): accepted-with-suppression
// forms — os.Remove best-effort cleanup and a ctx force-kill are correct as
// written; each carries a reason instead of forcing a rule change. Like the
// rest of the tree this file is never compiled; the extractor only parses it.
package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
)

func suppressedAbove(path string) {
	//io-surface:ignore err-dropped deliberate fire-and-forget cleanup
	os.Remove(path)
}

func suppressedTrailing(path string) {
	data, _ := os.ReadFile(path) //io-surface:ignore err-dropped contents are re-read on the error path
	_ = data
}

func wrongRuleDoesNotSuppress(path string) {
	//io-surface:ignore closer-missing not a closer; the err-dropped finding must still fire
	os.Remove(path)
}

func reasonlessDirectiveDoesNotSuppress(path string) {
	//io-surface:ignore err-dropped
	os.Remove(path)
}

func unknownRuleDirectiveDoesNotSuppress(path string) {
	//io-surface:ignore no-such-rule whatever
	os.Remove(path)
}

func suppressedCleanupUnlink(dir string) {
	//io-surface:ignore err-dropped best-effort teardown; leftover temp data is removed by the next sweep
	os.Remove(dir)
}

func suppressedForceKill(_ context.Context, pid int) {
	//io-surface:ignore ctx-dropped force-kill must run even when the context is cancelled; that is the point of the kill
	exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
}
