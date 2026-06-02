package logx

import "regexp"

var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s"'\\]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|password|secret|auth[_-]?token))(\s*[:=]\s*)[^\s"'\\]+`),
	regexp.MustCompile(`(?i)((?:--(?:api-key|token|password|secret|auth-token))(?:=|\s+))[^\s"'\\]+`),
}

// redactionReplacements maps the substitution applied to each pattern in
// redactPatterns. Patterns that should normalize the separator to '=' use
// `${1}=<redacted>`; patterns that preserve the original separator (e.g.
// "Authorization: Bearer") use `${1}<redacted>`.
var redactionReplacements = []string{
	`${1}<redacted>`,
	`${1}=<redacted>`,
	`${1}<redacted>`,
}

func Redact(s string) string {
	if s == "" {
		return ""
	}
	out := s
	for i, re := range redactPatterns {
		out = re.ReplaceAllString(out, redactionReplacements[i])
	}
	return out
}
