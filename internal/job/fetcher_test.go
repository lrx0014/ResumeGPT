package job

import "testing"

func TestParseJobPostingJSONLD(t *testing.T) {
	source := []byte(`<!doctype html><html><head><script type="application/ld+json">{
  "@context":"https://schema.org","@type":["Thing","JobPosting"],"title":"Senior Backend Engineer",
  "description":"<p>Build reliable APIs.</p>","employmentType":["FULL_TIME","PERMANENT"],
  "jobLocationType":"TELECOMMUTE","hiringOrganization":{"@type":"Organization","name":"Example GmbH"},
  "jobLocation":{"@type":"Place","address":{"addressLocality":"Berlin","addressRegion":"Berlin","addressCountry":{"name":"Germany"}}}
}</script></head></html>`)
	parsed, err := parseJobPage(source)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Senior Backend Engineer" || parsed.Company != "Example GmbH" || parsed.City != "Berlin" ||
		parsed.Country != "Germany" || parsed.WorkMode != "Remote" || parsed.EmploymentType != "FULL_TIME, PERMANENT" ||
		parsed.Description != "Build reliable APIs." {
		t.Fatalf("unexpected parsed job: %#v", parsed)
	}
}

func TestParseLinkedInMetadataFallback(t *testing.T) {
	source := []byte(`<!doctype html><html><head>
<meta property="og:title" content="Platform Engineer | Example AG | LinkedIn">
<meta property="og:description" content="Join the platform team.">
</head></html>`)
	parsed, err := parseJobPage(source)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Platform Engineer" || parsed.Company != "Example AG" || parsed.Description != "Join the platform team." {
		t.Fatalf("unexpected parsed job: %#v", parsed)
	}
}

func TestPublicIPValidation(t *testing.T) {
	if publicIP([]byte{127, 0, 0, 1}) || publicIP([]byte{10, 0, 0, 1}) ||
		publicIP([]byte{100, 64, 0, 1}) || publicIP([]byte{192, 0, 2, 1}) || !publicIP([]byte{8, 8, 8, 8}) {
		t.Fatal("unexpected public IP classification")
	}
}
