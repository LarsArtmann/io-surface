// Suppression fixture (superb plan T5): exercises the
// //io-surface:ignore grammar — valid above-line and trailing forms suppress
// their finding; a wrong-rule or malformed directive must NOT suppress and
// the malformed ones add their own ignore-invalid finding. Like the rest of
// the tree this file is never compiled; the extractor only parses it.
package main

import (
	"os"
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
