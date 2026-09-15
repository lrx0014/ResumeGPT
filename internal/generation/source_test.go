package generation

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestFallbackLatexBuildsCompleteEscapedDocument(t *testing.T) {
	source := fallbackLatex("# Jane Doe\n\n## Experience\n- Built R&D systems with 99% uptime.\n- Used Go_1 and $metrics.")
	for _, expected := range []string{"\\begin{document}", "\\end{document}", `R\&D`, `99\%`, `Go\_1`, `\$metrics`} {
		if !strings.Contains(source, expected) {
			t.Fatalf("fallback source does not contain %q:\n%s", expected, source)
		}
	}
}

func TestReplaceArchiveEntrySelectsMainAndPreservesAssets(t *testing.T) {
	var input bytes.Buffer
	writer := zip.NewWriter(&input)
	mainFile, _ := writer.Create("main.tex")
	_, _ = io.WriteString(mainFile, "old")
	asset, _ := writer.Create("images/photo.png")
	_, _ = asset.Write([]byte("image"))
	_ = writer.Close()

	result, entry, err := replaceArchiveEntry(input.Bytes(), "", "new source")
	if err != nil || entry != "main.tex" {
		t.Fatalf("replace entry: %v, %q", err, entry)
	}
	archive, _ := zip.NewReader(bytes.NewReader(result), int64(len(result)))
	values := map[string]string{}
	for _, file := range archive.File {
		body, _ := file.Open()
		data, _ := io.ReadAll(body)
		_ = body.Close()
		values[file.Name] = string(data)
	}
	if values["main.tex"] != "new source" || values["images/photo.png"] != "image" {
		t.Fatalf("unexpected archive: %#v", values)
	}
}

func TestReplaceArchiveEntryRejectsAmbiguousSources(t *testing.T) {
	var input bytes.Buffer
	writer := zip.NewWriter(&input)
	_, _ = writer.Create("one.tex")
	_, _ = writer.Create("two.tex")
	_ = writer.Close()
	if _, _, err := replaceArchiveEntry(input.Bytes(), "", "new"); err == nil {
		t.Fatal("expected ambiguous entry error")
	}
}
