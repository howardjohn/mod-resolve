package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func main() {
	var dir string
	if len(os.Args) == 1 {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		dir = string(b)
	} else if len(os.Args) == 2 {
		dir = os.Args[1]
	} else {
		log.Fatal("expected a single argument or stdin")
	}
	buf := bytes.Buffer{}
	cmd := exec.Command("git", "-c", "log.showsignature=false", "log", "-n1", "--format=format:%H %ct %D")
	cmd.Dir = dir
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	f := strings.Fields(buf.String())
	if len(f) < 2 {
		log.Fatal(fmt.Errorf("unexpected response from git log: %q", buf.String()))
	}
	hash := ShortenSHA1(f[0])
	t, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil {
		log.Fatal(fmt.Errorf("invalid time from git log: %q", buf.String()))
	}

	fmt.Println(PseudoVersion("", "", time.Unix(t, 0).UTC(), hash))
}

// ShortenSHA1 shortens a SHA1 hash (40 hex digits) to the canonical length
// used in pseudo-versions (12 hex digits).
func ShortenSHA1(rev string) string {
	if AllHex(rev) && len(rev) == 40 {
		return rev[:12]
	}
	return rev
}

// AllHex reports whether the revision rev is entirely lower-case hexadecimal digits.
func AllHex(rev string) bool {
	for i := 0; i < len(rev); i++ {
		c := rev[i]
		if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

const PseudoVersionTimestampFormat = "20060102150405"

// PseudoVersion returns a pseudo-version for the given major version ("v1")
// preexisting older tagged version ("" or "v1.2.3" or "v1.2.3-pre"), revision time,
// and revision identifier (usually a 12-byte commit hash prefix).
func PseudoVersion(major, older string, t time.Time, rev string) string {
	if major == "" {
		major = "v0"
	}
	segment := fmt.Sprintf("%s-%s", t.UTC().Format(PseudoVersionTimestampFormat), rev)
	build := semver.Build(older)
	older = semver.Canonical(older)
	if older == "" {
		return major + ".0.0-" + segment // form (1)
	}
	if semver.Prerelease(older) != "" {
		return older + ".0." + segment + build // form (4), (5)
	}

	// Form (2), (3).
	// Extract patch from vMAJOR.MINOR.PATCH
	i := strings.LastIndex(older, ".") + 1
	v, patch := older[:i], older[i:]

	// Reassemble.
	return v + incDecimal(patch) + "-0." + segment + build
}

// incDecimal returns the decimal string incremented by 1.
func incDecimal(decimal string) string {
	// Scan right to left turning 9s to 0s until you find a digit to increment.
	digits := []byte(decimal)
	i := len(digits) - 1
	for ; i >= 0 && digits[i] == '9'; i-- {
		digits[i] = '0'
	}
	if i >= 0 {
		digits[i]++
	} else {
		// digits is all zeros
		digits[0] = '1'
		digits = append(digits, '0')
	}
	return string(digits)
}
