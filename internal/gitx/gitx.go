package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Repo struct {
	Dir string
}

func (r Repo) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

func (r Repo) StatusPorcelain() (string, error) { return r.run("status", "--porcelain") }
func (r Repo) Branch() (string, error)          { return r.run("branch", "--show-current") }
func (r Repo) DiffStat() (string, error) {
	unstaged, _ := r.run("diff", "--stat")
	staged, _ := r.run("diff", "--cached", "--stat")
	both := strings.TrimSpace(unstaged + "\n" + staged)
	return both, nil
}
func (r Repo) Log(n int) (string, error) {
	return r.run("log", fmt.Sprintf("-%d", n), "--oneline")
}
func (r Repo) CheckoutNew(branch string) error {
	_, err := r.run("checkout", "-B", branch)
	return err
}
func (r Repo) AddAll() error {
	_, err := r.run("add", "-A")
	return err
}
func (r Repo) Commit(message string) error {
	_, err := r.run("commit", "-m", message)
	return err
}
func (r Repo) Push(remote, branch string) error {
	if remote == "" {
		remote = "origin"
	}
	args := []string{"push", "-u", remote}
	if branch != "" {
		args = append(args, branch)
	}
	_, err := r.run(args...)
	return err
}
func (r Repo) RevParseHead() (string, error) { return r.run("rev-parse", "HEAD") }

// ChangedPaths is the changeset Frontier examines: files different from
// main/master (if present) plus unstaged and staged work. Deduped.
func (r Repo) ChangedPaths() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(blob string) {
		for _, line := range strings.Split(blob, "\n") {
			line = strings.TrimSpace(strings.Trim(line, "\""))
			if line == "" {
				continue
			}
			if _, ok := seen[line]; ok {
				continue
			}
			seen[line] = struct{}{}
			out = append(out, line)
		}
	}
	base := "main"
	if _, err := r.run("rev-parse", "--verify", "--quiet", "refs/heads/main"); err != nil {
		if _, err2 := r.run("rev-parse", "--verify", "--quiet", "refs/heads/master"); err2 == nil {
			base = "master"
		} else {
			base = ""
		}
	}
	if base != "" {
		names, _ := r.run("diff", "--name-only", "--diff-filter=ACMR", base+"...HEAD")
		add(names)
	}
	unstaged, _ := r.run("diff", "--name-only", "--diff-filter=ACMR")
	add(unstaged)
	staged, _ := r.run("diff", "--cached", "--name-only", "--diff-filter=ACMR")
	add(staged)
	return out
}
