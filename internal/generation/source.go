package generation

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"path"
	"regexp"
	"strings"
)

var markdownLink = regexp.MustCompile(`\[([^]]+)]\(([^)]+)\)`)

func cleanModelSource(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "```") {
		first := strings.IndexByte(value, '\n')
		last := strings.LastIndex(value, "```")
		if first >= 0 && last > first {
			value = value[first+1 : last]
		}
	}
	return strings.TrimSpace(value)
}

func fallbackLatex(draft string) string {
	var body strings.Builder
	inList := false
	closeList := func() {
		if inList {
			body.WriteString("\\end{itemize}\n")
			inList = false
		}
	}
	for _, raw := range strings.Split(draft, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			closeList()
			continue
		}
		switch {
		case strings.HasPrefix(line, "### "):
			closeList()
			body.WriteString("\\subsection*{" + latexText(strings.TrimPrefix(line, "### ")) + "}\n")
		case strings.HasPrefix(line, "## "):
			closeList()
			body.WriteString("\\section*{" + latexText(strings.TrimPrefix(line, "## ")) + "}\n")
		case strings.HasPrefix(line, "# "):
			closeList()
			body.WriteString("{\\centering\\LARGE\\bfseries " + latexText(strings.TrimPrefix(line, "# ")) + "\\par}\n\\vspace{4pt}\n")
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			if !inList {
				body.WriteString("\\begin{itemize}\n")
				inList = true
			}
			body.WriteString("\\item " + latexText(line[2:]) + "\n")
		default:
			closeList()
			body.WriteString(latexText(line) + "\\par\n")
		}
	}
	closeList()
	return `\documentclass[10pt]{article}
\usepackage[margin=0.55in]{geometry}
\usepackage[T1]{fontenc}
\usepackage{enumitem}
\usepackage[hidelinks]{hyperref}
\pagestyle{empty}
\setlength{\parindent}{0pt}
\setlength{\parskip}{2pt}
\setlist[itemize]{leftmargin=1.2em,itemsep=0pt,topsep=2pt,parsep=0pt}
\usepackage{titlesec}
\titlespacing*{\section}{0pt}{6pt}{2pt}
\titlespacing*{\subsection}{0pt}{4pt}{1pt}
\titleformat{\section}{\large\bfseries}{}{0pt}{}
\titleformat{\subsection}{\normalsize\bfseries}{}{0pt}{}
\begin{document}
\small
\sloppy
\emergencystretch=2em
` + body.String() + `\end{document}
`
}

func latexText(value string) string {
	value = markdownLink.ReplaceAllString(value, "$1 ($2)")
	value = strings.NewReplacer("**", "", "__", "", "`", "").Replace(value)
	var result strings.Builder
	for _, char := range value {
		switch char {
		case '\\':
			result.WriteString(`\textbackslash{}`)
		case '{':
			result.WriteString(`\{`)
		case '}':
			result.WriteString(`\}`)
		case '$', '&', '#', '_', '%':
			result.WriteByte('\\')
			result.WriteRune(char)
		case '~':
			result.WriteString(`\textasciitilde{}`)
		case '^':
			result.WriteString(`\textasciicircum{}`)
		default:
			result.WriteRune(char)
		}
	}
	return result.String()
}

func replaceArchiveEntry(data []byte, requested, source string) ([]byte, string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, "", err
	}
	entry := strings.TrimSpace(requested)
	tex := make([]string, 0)
	for _, file := range reader.File {
		name := path.Clean(file.Name)
		if !file.FileInfo().IsDir() && strings.HasSuffix(strings.ToLower(name), ".tex") {
			tex = append(tex, name)
		}
	}
	if entry == "" {
		for _, name := range tex {
			if name == "main.tex" {
				entry = name
				break
			}
		}
		if entry == "" {
			nested := []string{}
			for _, name := range tex {
				if path.Base(strings.ToLower(name)) == "main.tex" {
					nested = append(nested, name)
				}
			}
			if len(nested) == 1 {
				entry = nested[0]
			}
		}
		if entry == "" && len(tex) == 1 {
			entry = tex[0]
		}
	}
	if entry == "" {
		return nil, "", errors.New("LaTeX entry file is ambiguous")
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	found := false
	for _, file := range reader.File {
		header := file.FileHeader
		target, err := writer.CreateHeader(&header)
		if err != nil {
			return nil, "", err
		}
		if path.Clean(file.Name) == entry {
			_, err = io.WriteString(target, source)
			found = true
		} else {
			input, openErr := file.Open()
			if openErr != nil {
				return nil, "", openErr
			}
			_, err = io.Copy(target, input)
			input.Close()
		}
		if err != nil {
			return nil, "", err
		}
	}
	if !found {
		return nil, "", errors.New("LaTeX entry file was not found")
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return output.Bytes(), entry, nil
}

func latexArchive(source, assetName string, asset []byte) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	entry, err := writer.Create("main.tex")
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(entry, source); err != nil {
		return nil, err
	}
	assetEntry, err := writer.Create(assetName)
	if err != nil {
		return nil, err
	}
	if _, err := assetEntry.Write(asset); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func archiveAssetPath(entry, assetName string) string {
	directory := path.Dir(path.Clean(entry))
	if directory == "." {
		return assetName
	}
	return path.Join(directory, assetName)
}

func addArchiveFile(data []byte, name string, content []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		if path.Clean(file.Name) == path.Clean(name) {
			continue
		}
		header := file.FileHeader
		target, err := writer.CreateHeader(&header)
		if err != nil {
			return nil, err
		}
		input, err := file.Open()
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(target, input)
		closeErr := input.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	asset, err := writer.Create(name)
	if err != nil {
		return nil, err
	}
	if _, err := asset.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
