package buildtest

import (
	"encoding/json"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func JSONDiff(l, r interface{}) string {
	lb, err := json.MarshalIndent(l, "", " ")
	if err != nil {
		panic(err.Error())
	}
	rb, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		panic(err.Error())
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(lb), string(rb), true)
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			return dmp.DiffPrettyText(diffs)
		}
	}
	return ""
}
