// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
CL prints a list of open Go code reviews (also known as change lists, or CLs).

Usage:

	cl [-closed] [-json] [-r] [-url] [query ...]

CL searches Gerrit for CLs matching the query and then
prints a line for each CL that is waiting for review
(as opposed to waiting for revisions by the author).

The output line looks like:

	CL 9225    0/ 2d  go   rsc   austin*   cmd/internal/gc: emit write barrier

From left to right, the columns show the CL number,
the number of days the CL has been in the current waiting state
(waiting for author or waiting for review),
the number of days since the CL was created,
the project name ("go" or the name of a subrepository),
the author, the reviewer, and the subject.
If the query restricts results to a single project
(for example, "cl project:go"), the project column is elided.
If the CL is waiting for revisions by the author,
the author column has an asterisk.
If the CL is waiting for a reviewer, the reviewer column
has an asterisk.
If the CL has been reviewed by the reviewer,
the reviewer column shows the current score.

By default, CL omits closed reviews, those with an R=close reply
and no subsequent upload of a new patch set.
If the -closed flag is specified, CL adds closed reviews to the output.

If the -r flag is specified, CL shows only CLs that need review,
not those waiting for the author. In this mode, the
redundant ``waiting for reviewer'' asterisk is elided.

If the -url flag is specified, CL replaces "CL 1234" at the beginning
of each output line with a full URL, "https://golang.org/cl/1234".

By default, CL sorts the output first by the combination of
project name and change subject.
The -sort flag changes the sort order. The choices are
"delay", to sort by the time the change has been in the current
waiting state, and "age", to sort by creation time.
When sorting, ties are broken by CL number.

If the -json flag is specified, CL does not print the usual listing.
Instead, it prints a JSON array holding CL objects, one for each
matched CL. Each of the CL objects is generated by this Go struct:

	type CL struct {
		Number             int            // CL number
		Subject            string         // subject (first line of commit message)
		Project            string         // "go" or a subrepository name
		Author             string         // author, short form or else full email
		AuthorEmail        string         // author, full email
		Reviewer           string         // expected reviewer, short form or else full email
		ReviewerEmail      string         // expected reviewer, full email
		Start              time.Time      // time CL was first uploaded
		NeedsReview        bool           // CL is waiting for reviewer (otherwise author)
		NeedsReviewChanged time.Time      // time NeedsReview last changed
		Closed             bool           // CL closed with R=close
		Issues             []int          // issues referenced by commit message
		Scores             map[string]int // current review scores
		Files              []string       // files changed in CL
	}

*/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/build/gerrit"
)

var (
	flagClosed      = flag.Bool("closed", false, "print closed CLs")
	flagNeedsReview = flag.Bool("r", false, "print only CLs in need of review")
	flagJSON        = flag.Bool("json", false, "print CLs in JSON format")
	flagURL         = flag.Bool("url", false, "print full URLs for CLs")
	flagSort        = flag.String("sort", "", "sort by `order` (age or delay) instead of project+subject")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: cl [query]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var now = time.Now() // so time stays the same during computations

func main() {
	log.SetFlags(0)
	log.SetPrefix("cl: ")
	flag.Usage = usage
	flag.Parse()

	switch *flagSort {
	case "", "age", "delay":
		// ok
	default:
		log.Fatal("unknown sort order")
	}

	c := gerrit.NewClient("https://go-review.googlesource.com", gerrit.NoAuth)
	query := strings.Join(flag.Args(), " ")
	open := "is:open"
	if strings.Contains(query, " is:") || strings.HasPrefix(query, "is:") {
		open = ""
	}
	cis, err := c.QueryChanges(open+" -project:scratch -message:do-not-review "+query, gerrit.QueryChangesOpt{
		N: 5000,
		Fields: []string{
			"LABELS",
			"CURRENT_FILES",
			"CURRENT_REVISION",
			"CURRENT_COMMIT",
			"MESSAGES",
			"DETAILED_ACCOUNTS", // fill out Owner.AuthorInfo, etc
			"DETAILED_LABELS",
		},
	})
	if err != nil {
		log.Fatalf("error querying changes: %v", err)
	}

	cls := []*CL{} // non-nil for json
	for _, ci := range cis {
		cl := parseCL(ci)
		if *flagNeedsReview && !cl.NeedsReview || !*flagClosed && cl.Closed {
			continue
		}
		cls = append(cls, cl)
	}

	switch *flagSort {
	case "":
		sort.Sort(byRepoAndSubject(cls))
	case "age":
		sort.Sort(byAge(cls))
	case "delay":
		sort.Sort(byDelay(cls))
	}

	if *flagJSON {
		data, err := json.MarshalIndent(cls, "", "\t")
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, '\n')
		os.Stdout.Write(data)
		return
	}

	clPrefix := "CL "
	if *flagURL {
		clPrefix = "https://golang.org/cl/"
	}

	var projectLen, authorLen, reviewerLen int
	for _, cl := range cls {
		projectLen = max(projectLen, len(cl.Project))
		authorLen = max(authorLen, len(cl.Author))
		reviewerLen = max(reviewerLen, len(cl.Reviewer))
	}
	if authorLen > 12 {
		authorLen = 12
	}
	if reviewerLen > 12 {
		reviewerLen = 12
	}
	authorLen += 1   // for *
	reviewerLen += 3 // for +2*

	var buf bytes.Buffer
	for _, cl := range cls {
		fmt.Fprintf(&buf, "%s%-5d %3.0f/%3.0fd %-*s  %-*s %-*s %s%s\n",
			clPrefix, cl.Number,
			cl.Delay().Seconds()/86400, cl.Age().Seconds()/86400,
			projectLen, cl.Project,
			authorLen, authorString(cl, authorLen),
			reviewerLen, reviewerString(cl, reviewerLen),
			cl.Subject, issuesString(cl))
	}
	os.Stdout.Write(buf.Bytes())
}

func max(i, j int) int {
	if i < j {
		return j
	}
	return i
}

// CL records information about a single CL.
type CL struct {
	Number             int            // CL number
	Subject            string         // subject (first line of commit message)
	Project            string         // "go" or a subrepository name
	Author             string         // author, short form or else full email
	AuthorEmail        string         // author, full email
	Reviewer           string         // expected reviewer, short form or else full email
	ReviewerEmail      string         // expected reviewer, full email
	Start              time.Time      // time CL was first uploaded
	NeedsReview        bool           // CL is waiting for reviewer (otherwise author)
	NeedsReviewChanged time.Time      // time NeedsReview last changed
	Closed             bool           // CL closed with R=close
	Issues             []int          // issues referenced by commit message
	Scores             map[string]int // current review scores
	Files              []string       // files changed in CL
	Status             string         // "new", "submitted", "merged", ...
}

func (cl *CL) Age() time.Duration {
	return now.Sub(cl.Start)
}

func (cl *CL) Delay() time.Duration {
	return now.Sub(cl.NeedsReviewChanged)
}

// authorString returns the author column, limited to n bytes.
func authorString(cl *CL, n int) string {
	suffix := ""
	if !cl.NeedsReview {
		suffix = "*"
	}
	return truncate(cl.Author, n-len(suffix)) + suffix
}

// reviewerString returns the reviewer column, limited to n bytes.
func reviewerString(cl *CL, n int) string {
	suffix := ""
	if cl.NeedsReview && !*flagNeedsReview {
		suffix = "*"
	}
	if score := cl.Scores[cl.ReviewerEmail]; score != 0 {
		suffix = fmt.Sprintf("%+d", score) + suffix
	}
	return truncate(cl.Reviewer, n-len(suffix)) + suffix
}

// truncate returns the name truncated to n bytes.
func truncate(text string, n int) string {
	if len(text) <= n {
		return text
	}
	return text[:n-3] + "..."
}

func issuesString(cl *CL) string {
	s := ""
	for _, id := range cl.Issues {
		s += fmt.Sprintf(" #%d", id)
	}
	return s
}

// Parsing of Gerrit information and review messages to produce CL structure.

var (
	reviewerRE     = regexp.MustCompile(`(?m)^R=([\w\-.@]+)\b`)
	scoreRE        = regexp.MustCompile(`\APatch Set \d+: Code-Review([+-][0-9]+)`)
	removedScoreRE = regexp.MustCompile(`\ARemoved the following votes:\n\n\* Code-Review([+-][0-9]+) by [^\n]* <([^\n]*)>`)
	issueRefRE     = regexp.MustCompile(`#\d+\b`)
	goIssueRefRE   = regexp.MustCompile(`\bgolang/go#\d+\b`)
)

const goReleaseCycle = 6 // working on Go 1.x

func parseCL(ci *gerrit.ChangeInfo) *CL {
	loadReviewers()

	// Gather information.
	var (
		scores           = make(map[string]int)
		initialReviewer  = ""
		firstResponder   = ""
		explicitReviewer = ""
		closeReason      = ""
	)
	for _, msg := range ci.Messages {
		if msg.Author == nil { // happens for Gerrit-generated messages
			continue
		}
		if strings.HasPrefix(msg.Message, "Uploaded patch set ") {
			if explicitReviewer == "close" && !strings.HasPrefix(closeReason, "Go") {
				explicitReviewer = ""
				closeReason = ""
			}
			for who, score := range scores {
				if score == +1 || score == -1 {
					delete(scores, who)
				}
			}
		}
		if m := reviewerRE.FindStringSubmatch(msg.Message); m != nil {
			if m[1] == "close" {
				explicitReviewer = "close"
				closeReason = "Closed"
			} else if strings.HasPrefix(m[1], "go1.") {
				n, _ := strconv.Atoi(m[1][len("go1."):])
				if n > goReleaseCycle {
					explicitReviewer = "close"
					closeReason = "Go" + m[1][2:]
				}
			} else if m[1] == "golang-dev" || m[1] == "golang-codereviews" {
				explicitReviewer = "golang-dev"
			} else if x := mailLookup[m[1]]; x != "" {
				explicitReviewer = x
			}
		}
		if m := scoreRE.FindStringSubmatch(msg.Message); m != nil {
			n, _ := strconv.Atoi(m[1])
			scores[msg.Author.Email] = n
		}
		if m := removedScoreRE.FindStringSubmatch(msg.Message); m != nil {
			delete(scores, m[1])
		}
		if firstResponder == "" && isReviewer[msg.Author.Email] && msg.Author.Email != ci.Owner.Email {
			firstResponder = msg.Author.Email
		}
	}

	cl := &CL{
		Number:      ci.ChangeNumber,
		Subject:     ci.Subject,
		Project:     ci.Project,
		Author:      shorten(ci.Owner.Email),
		AuthorEmail: ci.Owner.Email,
		Scores:      scores,
		Status:      strings.ToLower(ci.Status),
	}

	// Determine reviewer, in priorty order.
	// When breaking ties, give preference to R= setting.
	// Otherwise compare by email address.
	maybe := func(who string) {
		if cl.ReviewerEmail == "" || who == explicitReviewer || cl.ReviewerEmail != explicitReviewer && cl.ReviewerEmail > who {
			cl.ReviewerEmail = who
		}
	}

	// 1. Anyone who -2'ed the CL.
	if cl.ReviewerEmail == "" {
		for who, score := range scores {
			if score == -2 {
				maybe(who)
			}
		}
	}

	// 2. Anyone who +2'ed the CL.
	if cl.ReviewerEmail == "" {
		for who, score := range scores {
			if score == +2 {
				maybe(who)
			}
		}
	}

	// 2½. Even if a CL is +2 or -2, R=closed wins,
	// so that it doesn't appear in listings by default.
	if explicitReviewer == "close" {
		cl.ReviewerEmail = "close"
	}

	// 3. Last explicit R= in review message.
	if cl.ReviewerEmail == "" {
		cl.ReviewerEmail = explicitReviewer
	}
	// 4. Initial target of review requecl.
	// TODO: If there's some way to figure this out, do so.
	_ = initialReviewer
	// 5. Whoever responds first and looks like a reviewer.
	if cl.ReviewerEmail == "" {
		cl.ReviewerEmail = firstResponder
	}

	// Allow R=golang-dev in #2 as "unassign".
	if cl.ReviewerEmail == "golang-dev" {
		cl.ReviewerEmail = ""
	}

	cl.Reviewer = shorten(cl.ReviewerEmail)

	// Now that we know who the reviewer is,
	// figure out whether the CL is in need of review
	// (or else is waiting for the author to do more work).
	for _, msg := range ci.Messages {
		if msg.Author == nil { // happens for Gerrit-generated messages
			continue
		}
		if cl.Start.IsZero() {
			cl.Start = msg.Time.Time()
		}
		if strings.HasPrefix(msg.Message, "Uploaded patch set ") {
			cl.NeedsReview = true
			cl.NeedsReviewChanged = msg.Time.Time()
		}
		if msg.Author.Email == cl.ReviewerEmail {
			cl.NeedsReview = false
			cl.NeedsReviewChanged = msg.Time.Time()
		}
	}

	if cl.ReviewerEmail == "close" {
		cl.Reviewer = closeReason
		cl.ReviewerEmail = ""
		cl.Closed = true
		cl.NeedsReview = false
	}

	// DO NOT REVIEW overrides anything in the CL state.
	// We shouldn't see these, because the query always
	// contains -message:do-not-review, but check anyway.
	if _, ok := ci.Labels["Do-Not-Review"]; ok {
		cl.NeedsReview = false
	}

	// Find issue numbers.
	cl.Issues = []int{} // non-nil for json
	refRE := issueRefRE
	if cl.Project != "go" {
		refRE = goIssueRefRE
	}

	rev := ci.Revisions[ci.CurrentRevision]
	for file := range rev.Files {
		cl.Files = append(cl.Files, file)
	}
	sort.Strings(cl.Files)
	if rev.Commit != nil {
		for _, ref := range refRE.FindAllString(rev.Commit.Message, -1) {
			n, _ := strconv.Atoi(ref[strings.Index(ref, "#")+1:])
			if n != 0 {
				cl.Issues = append(cl.Issues, n)
			}
		}
	}
	cl.Issues = uniq(cl.Issues)

	return cl
}

func uniq(x []int) []int {
	sort.Ints(x)
	out := x[:0]
	for _, v := range x {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

func shorten(email string) string {
	if i := strings.Index(email, "@"); i >= 0 {
		if mailLookup[email[:i]] == email {
			return email[:i]
		}
	}
	return email
}

// Sort interfaces.

type byRepoAndSubject []*CL

func (x byRepoAndSubject) Len() int      { return len(x) }
func (x byRepoAndSubject) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byRepoAndSubject) Less(i, j int) bool {
	if x[i].Project != x[j].Project {
		return projectOrder(x[i].Project) < projectOrder(x[j].Project)
	}
	if x[i].Subject != x[j].Subject {
		return x[i].Subject < x[j].Subject
	}
	return x[i].Number < x[j].Number
}

type byAge []*CL

func (x byAge) Len() int      { return len(x) }
func (x byAge) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byAge) Less(i, j int) bool {
	if !x[i].Start.Equal(x[j].Start) {
		return x[i].Start.Before(x[j].Start)
	}
	return x[i].Number > x[j].Number
}

type byDelay []*CL

func (x byDelay) Len() int      { return len(x) }
func (x byDelay) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byDelay) Less(i, j int) bool {
	if !x[i].NeedsReviewChanged.Equal(x[j].NeedsReviewChanged) {
		return x[i].NeedsReviewChanged.Before(x[j].NeedsReviewChanged)
	}
	return x[i].Number < x[j].Number
}

func projectOrder(name string) string {
	if name == "go" {
		return "\x00" // sort before everything except empty string
	}
	return name
}
