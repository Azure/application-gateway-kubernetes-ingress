// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const valuesDir = "fixtures"

// TestChart is a simple snapshot-style regression test for the CCP chart.
//
// Each overlaymgr Helm value generation test fixture is used to render the chart, and the output
// is compared to a previous snapshot. Any changes to the chart will require changes to one or more
// associated snapshots.
//
// This approach proves that the chart can be rendered successfully given various inputs, and that
// the resulting manifests haven't changed unexpectedly since a known good state.
//
// When making a change to the chart, the test snapshots can be generated by running this test case
// with RENDER_SNAPSHOTS=true. Then, `git diff` the new snapshots to see if the changes are expected.
//
// NOTE(jordan): Using the overlaymgr fixtures might not result in adequate test coverage. We should
//               put more thought into this in the future.
func TestChart(t *testing.T) {
	snapshots := []string{}
	err := filepath.Walk(valuesDir, func(path string, f os.FileInfo, err error) error {
		if f != nil && !f.IsDir() {
			snapshots = append(snapshots, path)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unable to list snapshots: %v", err)
	}

	for _, snapshot := range snapshots {
		snapshotName, _ := filepath.Rel(valuesDir, snapshot)
		name := strings.TrimRight(snapshotName, ".json")

		t.Run(name, func(t *testing.T) {
			snapshotDir := fmt.Sprintf("snapshots/%s", name)

			if os.Getenv("RENDER_SNAPSHOTS") != "" {
				err := RenderChart("..", snapshot, snapshotDir)
				if err != nil {
					t.Fatalf("unable to render chart: %v", err)
				}

				return
			}

			actual, err := CaptureSnapshot("..", snapshot)
			if err != nil {
				t.Fatalf("unable to capture snapshot: %v", err)
			}

			expected, err := LoadSnapshot(snapshotDir)
			if err != nil {
				t.Fatalf("unable to load snapshot: %v", err)
			}

			actual.Diff(t, expected)
		})
	}
}