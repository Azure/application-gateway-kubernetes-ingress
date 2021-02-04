// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"gopkg.in/yaml.v2"
)

// Snapshot represents a rendered Helm chart.
type Snapshot struct {
	files map[string][]map[interface{}]interface{}
}

// Diff takes an expected snapshot, and compares it to itself.
func (s *Snapshot) Diff(t *testing.T, expected *Snapshot) {
	for file, parsedFile := range s.files {
		parsedExpFile, ok := expected.files[file]
		if !ok {
			t.Errorf("missing manifest: %s", file)
			continue
		}

		diff := pretty.Compare(parsedFile, parsedExpFile)
		if len(diff) == 0 {
			continue
		}

		t.Errorf("file %q doesn't match expectation\n%s", file, diff)
	}

	for expFile := range expected.files {
		if _, ok := s.files[expFile]; !ok {
			t.Errorf("expected snapshot has file missing in actual: %s", expFile)
		}
	}
}

// RenderChart renderss a Helm chart to a given directory.
func RenderChart(chart, values, dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("unable to create output dir %q: %v", dir, err)
	}

	helmCmdFile := os.Getenv("HELM_CMD")
	if helmCmdFile == "" {
		helmCmdFile = "helm"
	}
	out, err := exec.Command(helmCmdFile, "template", "..", "--output-dir", dir, "--values", values).CombinedOutput()
	if err != nil {
		return &HelmError{
			RawOutput:    out,
			CommandError: err,
		}
	}

	return StripNonDeterministic(dir)
}

// CaptureSnapshot renders a new snapshot from a given Helm chart and values file.
func CaptureSnapshot(chart, values string) (*Snapshot, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("creating tempdir: %v", err)
	}

	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			panic(fmt.Errorf("unable to clean up tempdir for chart %q with values %q: %v", chart, values, err))
		}
	}()

	if err := RenderChart(chart, values, dir); err != nil {
		return nil, err
	}

	return LoadSnapshot(dir)
}

// LoadSnapshot loads a snapshot from disk. Expects the directory to be a chart rendered with
// `helm template --output-dir`.
func LoadSnapshot(dir string) (*Snapshot, error) {
	s := &Snapshot{
		files: map[string][]map[interface{}]interface{}{},
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %q: %v", path, err)
		}
		defer file.Close()
		shortPath := strings.TrimPrefix(path, dir+"/")

		dec := yaml.NewDecoder(file)
		for {
			obj := map[interface{}]interface{}{}
			err = dec.Decode(&obj)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("parsing file %q: %v", shortPath, err)
			}
			s.files[shortPath] = append(s.files[shortPath], obj)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking rendered chart: %v", err)
	}

	return s, nil
}

// StripNonDeterministic removes properties from all of the manifests in a given directory.
func StripNonDeterministic(path string) error {
	return filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			lines := []string{}
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				text := scanner.Text()
				if strings.Contains(text, "checksum/") ||
					strings.Contains(text, "aks.microsoft.com/release-time") {
					continue
				}
				lines = append(lines, text)
			}

			return ioutil.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		})
}

// HelmError represents an error returned by the Helm CLI.
type HelmError struct {
	RawOutput    []byte
	CommandError error
}

// Error satisfies the error interface.
func (h *HelmError) Error() string {
	return fmt.Sprintf("Helm CLI error: %q - raw output:\n%s", h.CommandError.Error(), h.RawOutput)
}
