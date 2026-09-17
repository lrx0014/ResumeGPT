package hunter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type staticWebSearch struct{ results []SearchResult }

func (s staticWebSearch) Search(context.Context, string, int) ([]SearchResult, error) {
	return s.results, nil
}

func TestParseSearchResultAndUnwrapRedirect(t *testing.T) {
	document, err := html.Parse(strings.NewReader(`<div class="result"><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fcareers.example.com%2Fjobs%2F123%3Futm_source%3Dsearch">Backend Engineer</a><a class="result__snippet">Build distributed systems.</a></div>`))
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	var found bool
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found {
			return
		}
		if hasClass(node, "result") {
			result, found = parseResult(node)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	if !found || result.Title != "Backend Engineer" || result.URL != "https://careers.example.com/jobs/123" || result.Snippet != "Build distributed systems." {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFinishHuntDeduplicatesAndRejectsUnsafeURLs(t *testing.T) {
	tool := &finishHuntTool{limit: 1}
	response, err := tool.Call(context.Background(), `{"urls":["https://example.com/jobs/1?utm_source=a","https://example.com/jobs/1?utm_source=b","https://example.com/jobs/2","http://localhost/job"]}`)
	if err != nil || !strings.Contains(response, "accepted") || len(tool.urls) != 1 || tool.urls[0] != "https://example.com/jobs/1" {
		t.Fatalf("unexpected finish result: response=%s urls=%#v error=%v", response, tool.urls, err)
	}
}

func TestSearchToolRetainsBoundedFallbackCandidates(t *testing.T) {
	tool := &webSearchTool{search: staticWebSearch{results: []SearchResult{
		{URL: "https://careers.example.com/jobs/1?utm_source=search"},
		{URL: "https://careers.example.com/jobs/1"},
		{URL: "https://careers.example.com/jobs/2"},
		{URL: "https://careers.example.com/jobs/search"},
	}}}
	if _, err := tool.Call(context.Background(), `{"query":"software engineer"}`); err != nil {
		t.Fatal(err)
	}
	urls := tool.candidateURLs(2)
	if len(urls) != 2 || urls[0] != "https://careers.example.com/jobs/1" || urls[1] != "https://careers.example.com/jobs/2" {
		t.Fatalf("unexpected fallback candidates: %#v", urls)
	}
	if !agentIterationsExhausted(errors.New("agent not finished before max iterations")) {
		t.Fatal("max-iterations error was not recognized")
	}
}

func TestIndividualJobURLFilterRejectsSearchAndCategoryPages(t *testing.T) {
	tests := map[string]bool{
		"https://www.linkedin.com/jobs/view/4451527183/":                                    true,
		"https://www.linkedin.com/jobs/software-engineer-jobs-luxemburg":                    false,
		"https://www.glassdoor.sg/job-listing/backend-engineer-example-JV_IC123.htm":        true,
		"https://www.glassdoor.sg/Job/jobs.htm?locId=217&sc.occupationParam=backend":        false,
		"https://sg.jobstreet.com/full-stack-engineer-jobs":                                 false,
		"https://www.fastjobs.sg/singapore-jobs/all-categories-jobs/developer-jobs-search/": false,
		"https://jobs.example.com/openings/software-engineer-123":                           true,
	}
	for raw, expected := range tests {
		if actual := looksLikeIndividualJobURL(raw); actual != expected {
			t.Errorf("looksLikeIndividualJobURL(%q) = %v, want %v", raw, actual, expected)
		}
	}
}
