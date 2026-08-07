package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// replayIsolated re-runs the critic over connections that were logged as
// having no matching predicate, asserting the predicate they should have had.
//
// The case this exists for: 596492b added StructurallyAnalogousTo to the trip
// emit menu, and before that every cross-cluster analogy was forced to NONE
// and stranded in .metabolism-trip-isolated.jsonl. Those rows were never
// critiqued — they failed at emit, upstream of the gate — so nothing on record
// says whether they were any good. This asks, without regenerating them.
//
// The answer that matters is the survival rate, not the individual verdicts:
// near-zero means the stranded batch is junk and can be deleted, while a rate
// near the live cycle's means real claims are sitting in a file unread.
func replayIsolated(dir, logPath, predicate string, limit int) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("replay-isolated needs ANTHROPIC_API_KEY")
	}
	f, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	exemplars := sampleHighQualityClaims(dir, 5, 200)
	priors := priorClaimsOfPredicate(dir, predicate)

	accepted, total := 0, 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for sc.Scan() && (limit <= 0 || total < limit) {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var conn TripConnection
		if err := json.Unmarshal(sc.Bytes(), &conn); err != nil {
			return fmt.Errorf("line %d: %w", total+1, err)
		}
		// The predicate is the whole question: these rows carry NONE because
		// the emit menu had no route, not because the critic saw them and
		// disagreed. Assert the predicate they were denied and ask the gate.
		conn.Predicate = predicate
		total++

		verdict := critiqueTripConnection(client, conn, exemplars, priors)
		mark := "x"
		if verdict.Accept {
			mark = "+"
			accepted++
		}
		fmt.Printf("  %s %s ~ %s (score %d)", mark, conn.EntityA, conn.EntityB, conn.Score)
		if !verdict.Accept {
			fmt.Printf("  - %s", verdict.RawReason)
		}
		fmt.Println()
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if total == 0 {
		fmt.Println("[replay] no rows read")
		return nil
	}
	fmt.Printf("\n[replay] %d/%d accepted (%d%%) as %s\n", accepted, total, accepted*100/total, predicate)
	return nil
}
