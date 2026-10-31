package analyzer

import (
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/parser"
	"github.com/devblac/docker-lint/internal/rules"
)

// detectionCases are Dockerfiles each rule must report, written the way the
// problem shows up in real projects. The corpus test guards against false
// positives; these guard against a rule that quietly stops firing.
var detectionCases = map[string][]string{
	rules.RuleNoFrom: {
		"RUN echo hi\nFROM alpine:3.20\n",
		"# no FROM at all\nRUN echo hi\n",
	},
	rules.RuleMissingTag: {
		"FROM node\n",
		"FROM registry.example.com:5000/team/app\n",
	},
	rules.RuleLatestTag: {
		"FROM node:latest\n",
	},
	rules.RuleLargeBaseImage: {
		"FROM ubuntu:24.04\n",
		"FROM golang:1.23 AS build\nFROM build\n",
	},
	rules.RuleCacheNotCleaned: {
		"FROM debian:12-slim\nRUN apt-get update && apt-get install -y curl\n",
		"FROM alpine:3.20\nRUN apk add curl\n",
		"FROM python:3.12-slim\nRUN pip install flask\n",
	},
	rules.RuleConsecutiveRun: {
		"FROM alpine:3.20\nRUN echo a\nRUN echo b\n",
	},
	rules.RuleSuboptimalOrdering: {
		"FROM node:20-slim\nWORKDIR /app\nCOPY . .\nRUN npm ci\nRUN npm run build\n",
		"FROM python:3.12-slim\nWORKDIR /app\nCOPY . .\nRUN pip install --no-cache-dir -r requirements.txt\n",
		"FROM golang:1.23\nWORKDIR /src\nCOPY . .\nRUN go mod download\n",
	},
	rules.RuleUpdateWithoutInstall: {
		"FROM debian:12-slim\nRUN apt-get update\nRUN apt-get install -y curl && rm -rf /var/lib/apt/lists/*\n",
	},
	rules.RuleSecretInEnv: {
		"FROM alpine:3.20\nENV DB_PASSWORD=hunter2\n",
		"FROM alpine:3.20\nENV AWS_SECRET_ACCESS_KEY=abc GITHUB_TOKEN=ghp_x\n",
	},
	rules.RuleSecretInArg: {
		"FROM alpine:3.20\nARG NPM_TOKEN\n",
		"FROM alpine:3.20\nARG API_KEY=abc\n",
	},
	rules.RuleNoUser: {
		"FROM alpine:3.20\nRUN echo hi\n",
		"FROM alpine:3.20\nUSER app\nUSER root\n",
		"FROM alpine:3.20\nUSER 0:0\n",
	},
	rules.RuleAddWithURL: {
		"FROM alpine:3.20\nADD https://example.com/install.sh /install.sh\n",
	},
	rules.RuleAddOverCopy: {
		"FROM alpine:3.20\nADD app.py /app/app.py\n",
	},
	rules.RuleMultipleCMD: {
		"FROM alpine:3.20\nCMD [\"a\"]\nCMD [\"b\"]\n",
	},
	rules.RuleMultipleEntrypoint: {
		"FROM alpine:3.20\nENTRYPOINT [\"a\"]\nENTRYPOINT [\"b\"]\n",
	},
	rules.RuleRelativeWorkdir: {
		"FROM alpine:3.20\nWORKDIR app\n",
	},
	rules.RuleMissingHealthcheck: {
		"FROM alpine:3.20\nCMD [\"app\"]\n",
	},
	rules.RuleWildcardCopy: {
		"FROM alpine:3.20\nCOPY *.py /app/\n",
	},
}

func TestEveryRuleDetectsItsCase(t *testing.T) {
	analyzer := NewWithDefaults(Config{})

	for _, rule := range rules.DefaultRegistry.All() {
		id := rule.ID()
		cases, ok := detectionCases[id]
		if !ok || len(cases) == 0 {
			t.Errorf("%s has no detection case; add one to detectionCases", id)
			continue
		}
		for _, src := range cases {
			df, err := parser.ParseString(src)
			if err != nil {
				t.Errorf("%s: case does not parse: %v\n%s", id, err, src)
				continue
			}
			found := false
			for _, f := range analyzer.Analyze(df) {
				if f.RuleID == id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s did not report:\n%s", id, strings.TrimSpace(src))
			}
		}
	}

	for id := range detectionCases {
		if rules.DefaultRegistry.Get(id) == nil {
			t.Errorf("detectionCases lists %s, which is not a registered rule", id)
		}
	}
}
