package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Rule struct {
	Domain    string
	RateLimit string
}

type ruleFlag []Rule

func (r *ruleFlag) String() string {
	return "domain=rate pairs"
}

func (r *ruleFlag) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use domain=rate (e.g., yahoo.com=2r/s)")
	}
	*r = append(*r, Rule{Domain: parts[0], RateLimit: parts[1]})
	return nil
}

func main() {
	var rules ruleFlag
	flag.Var(&rules, "rule", "Domain and rate limit pair (e.g., -rule yahoo.com=2r/s)")
	outputPath := flag.String("out", "Caddyfile", "Path to output the generated Caddyfile")
	flag.Parse()

	if len(rules) == 0 {
		fmt.Println("Error: at least one rule is required.")
		fmt.Println("Usage: go run generate_caddyfile.go -rule yahoo.com=2r/s -rule example.com=10r/s")
		os.Exit(1)
	}

	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString("\torder forward_proxy before route\n")
	sb.WriteString("}\n\n")
	sb.WriteString(":8080 {\n")
	sb.WriteString("\tforward_proxy {\n")
	sb.WriteString("\t\tbasic_auth {$PROXY_USER} {$PROXY_HASH}\n")
	sb.WriteString("\t\thide_ip\n")
	sb.WriteString("\t\thide_via\n")
	sb.WriteString("\t}\n\n")

	for _, rule := range rules {
		sb.WriteString(fmt.Sprintf("\trate_limit /%s %s\n", rule.Domain, rule.RateLimit))
	}

	sb.WriteString("}\n")

	err := os.WriteFile(*outputPath, []byte(sb.String()), 0644)
	if err != nil {
		fmt.Printf("Failed to write Caddyfile: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %s with %d rule(s).\n", *outputPath, len(rules))
}
