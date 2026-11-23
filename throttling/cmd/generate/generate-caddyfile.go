package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	bucketPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	hostPattern   = regexp.MustCompile(
		`^(?:\*\.)?[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`,
	)
	ratePattern = regexp.MustCompile(`^([1-9][0-9]*)r/([smh])$`)
)

type rule struct {
	bucket string
	events int
	window string
	hosts  []string
}

func main() {
	rulesPath := flag.String("rules", "", "Path to the gateway policy file")
	outputPath := flag.String("out", "Caddyfile", "Generated Caddyfile path")
	flag.Parse()

	if strings.TrimSpace(*rulesPath) == "" {
		exitError(errors.New("-rules is required"))
	}

	input, err := os.Open(*rulesPath)
	if err != nil {
		exitError(fmt.Errorf("open rules: %w", err))
	}
	defer input.Close()

	rules, err := parseRules(input)
	if err != nil {
		exitError(err)
	}

	contents := generateCaddyfile(rules)
	if err := os.WriteFile(*outputPath, []byte(contents), 0o644); err != nil {
		exitError(fmt.Errorf("write Caddyfile: %w", err))
	}

	fmt.Printf("generated %s with %d rate bucket(s)\n", *outputPath,
		len(rules))
}

func parseRules(reader io.Reader) ([]rule, error) {
	var rules []rule

	buckets := make(map[string]struct{})
	hosts := make(map[string]string)
	scanner := bufio.NewScanner(reader)

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("line %d: expected bucket, rate, and host",
				lineNumber)
		}

		bucket := fields[0]
		if !bucketPattern.MatchString(bucket) {
			return nil, fmt.Errorf("line %d: invalid bucket %q", lineNumber,
				bucket)
		}
		if _, exists := buckets[bucket]; exists {
			return nil, fmt.Errorf("line %d: duplicate bucket %q", lineNumber,
				bucket)
		}

		events, window, err := parseRate(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}

		ruleHosts := make([]string, 0, len(fields)-2)
		for _, candidate := range fields[2:] {
			host := strings.ToLower(candidate)
			if !validHost(host) {
				return nil, fmt.Errorf("line %d: invalid host %q", lineNumber,
					candidate)
			}
			if owner, exists := hosts[host]; exists {
				return nil, fmt.Errorf(
					"line %d: host %q already belongs to bucket %q",
					lineNumber, host, owner,
				)
			}

			hosts[host] = bucket
			ruleHosts = append(ruleHosts, host)
		}

		buckets[bucket] = struct{}{}
		rules = append(rules, rule{
			bucket: bucket, events: events, window: window, hosts: ruleHosts,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}
	if len(rules) == 0 {
		return nil, errors.New("policy must contain at least one rule")
	}

	return rules, nil
}

func parseRate(value string) (int, string, error) {
	matches := ratePattern.FindStringSubmatch(value)
	if matches == nil {
		return 0, "", fmt.Errorf(
			"invalid rate %q; use values such as 8r/s, 30r/m, or 100r/h",
			value,
		)
	}

	events, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", fmt.Errorf("invalid rate %q: %w", value, err)
	}

	return events, "1" + matches[2], nil
}

func validHost(host string) bool {
	if len(host) > 253 || !hostPattern.MatchString(host) {
		return false
	}

	name := strings.TrimPrefix(host, "*.")
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 ||
			strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return strings.Contains(name, ".")
}

func generateCaddyfile(rules []rule) string {
	var output strings.Builder

	output.WriteString("{\n\tadmin off\n}\n\n:8080 {\n\troute {\n")
	output.WriteString("\t\t@healthz path /healthz\n")
	output.WriteString("\t\tlog_skip @healthz\n")
	output.WriteString("\t\trespond @healthz 200\n\n")
	output.WriteString("\t\tbasic_auth {\n")
	output.WriteString("\t\t\t{$PROXY_USER:test-proxy-user} {$PROXY_HASH:$2a$14$D8kc7DkE2otVllA4EM0s0eQ2SvXy46nRCdkbjbCRnbDxxrQrT1nE2}\n")
	output.WriteString("\t\t}\n\n")
	output.WriteString("\t\trequest_header -Authorization\n\n")
	output.WriteString("\t\trate_limit {\n")
	for _, rule := range rules {
		fmt.Fprintf(&output, "\t\t\tzone %s {\n", rule.bucket)
		output.WriteString("\t\t\t\tmatch {\n")
		fmt.Fprintf(&output, "\t\t\t\t\thost %s\n",
			strings.Join(rule.hosts, " "))
		output.WriteString("\t\t\t\t}\n")
		fmt.Fprintf(&output, "\t\t\t\tkey %s\n", rule.bucket)
		fmt.Fprintf(&output, "\t\t\t\tevents %d\n", rule.events)
		fmt.Fprintf(&output, "\t\t\t\twindow %s\n", rule.window)
		output.WriteString("\t\t\t}\n")
	}
	output.WriteString("\t\t}\n\n")

	var allHosts []string
	for _, rule := range rules {
		allHosts = append(allHosts, rule.hosts...)
	}
	output.WriteString("\t\t@allowed host ")
	output.WriteString(strings.Join(allHosts, " "))
	output.WriteString("\n")
	output.WriteString("\t\treverse_proxy @allowed {host}:443 {\n")
	output.WriteString("\t\t\theader_up Host {host}\n")
	output.WriteString("\t\t\ttransport http {\n")
	output.WriteString("\t\t\t\tnetwork_proxy none\n")
	output.WriteString("\t\t\t\ttls\n")
	output.WriteString("\t\t\t\ttls_server_name {host}\n")
	output.WriteString("\t\t\t}\n")
	output.WriteString("\t\t}\n\n")
	output.WriteString("\t\trespond 403\n\t}\n\n")
	output.WriteString(
		"\tlog {\n\t\toutput stdout\n\t\tformat json\n\t}\n}\n",
	)

	return output.String()
}

func exitError(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
