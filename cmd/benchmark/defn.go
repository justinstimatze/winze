package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func defnRetrieve(q Question) []string {
	if q.DefnSQL == "" {
		return nil
	}

	out, err := exec.Command("defn", "query", q.DefnSQL).Output()
	if err != nil {
		return nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil
	}

	var results []string
	for _, row := range rows {
		for _, v := range row {
			s := fmt.Sprintf("%v", v)
			s = strings.TrimSpace(s)
			if s != "" && s != "<nil>" {
				results = append(results, s)
			}
		}
	}
	return results
}
