package ast

import "testing"

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.severity.String()
		if got != tt.expected {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.severity, got, tt.expected)
		}
	}
}

func TestFindingCreation(t *testing.T) {
	finding := Finding{
		RuleID:     "DL3006",
		Severity:   SeverityWarning,
		Line:       10,
		Column:     1,
		Message:    "Missing explicit image tag",
		Suggestion: "Use 'FROM alpine:3.18' instead of 'FROM alpine'",
	}

	if finding.RuleID != "DL3006" {
		t.Errorf("Finding.RuleID = %q, want %q", finding.RuleID, "DL3006")
	}
	if finding.Severity != SeverityWarning {
		t.Errorf("Finding.Severity = %v, want %v", finding.Severity, SeverityWarning)
	}
	if finding.Line != 10 {
		t.Errorf("Finding.Line = %d, want %d", finding.Line, 10)
	}
}

func TestFromInstructionMethods(t *testing.T) {
	instr := &FromInstruction{
		LineNum:  1,
		RawText:  "FROM alpine:3.18 AS builder",
		Image:    "alpine",
		Tag:      "3.18",
		Alias:    "builder",
	}

	if instr.Line() != 1 {
		t.Errorf("FromInstruction.Line() = %d, want %d", instr.Line(), 1)
	}
	if instr.Raw() != "FROM alpine:3.18 AS builder" {
		t.Errorf("FromInstruction.Raw() = %q, want %q", instr.Raw(), "FROM alpine:3.18 AS builder")
	}
	if instr.Type() != InstrFROM {
		t.Errorf("FromInstruction.Type() = %v, want %v", instr.Type(), InstrFROM)
	}
}


func TestRunInstructionMethods(t *testing.T) {
	instr := &RunInstruction{
		LineNum: 5,
		RawText: "RUN apt-get update",
		Command: "apt-get update",
		Shell:   true,
	}

	if instr.Line() != 5 {
		t.Errorf("RunInstruction.Line() = %d, want %d", instr.Line(), 5)
	}
	if instr.Raw() != "RUN apt-get update" {
		t.Errorf("RunInstruction.Raw() = %q, want %q", instr.Raw(), "RUN apt-get update")
	}
	if instr.Type() != InstrRUN {
		t.Errorf("RunInstruction.Type() = %v, want %v", instr.Type(), InstrRUN)
	}
}

func TestCopyInstructionMethods(t *testing.T) {
	instr := &CopyInstruction{
		LineNum: 10,
		RawText: "COPY --from=builder /app /app",
		Sources: []string{"/app"},
		Dest:    "/app",
		From:    "builder",
	}

	if instr.Line() != 10 {
		t.Errorf("CopyInstruction.Line() = %d, want %d", instr.Line(), 10)
	}
	if instr.Type() != InstrCOPY {
		t.Errorf("CopyInstruction.Type() = %v, want %v", instr.Type(), InstrCOPY)
	}
}

func TestAddInstructionMethods(t *testing.T) {
	instr := &AddInstruction{
		LineNum: 12,
		RawText: "ADD https://example.com/file.tar.gz /app/",
		Sources: []string{"https://example.com/file.tar.gz"},
		Dest:    "/app/",
	}

	if instr.Line() != 12 {
		t.Errorf("AddInstruction.Line() = %d, want %d", instr.Line(), 12)
	}
	if instr.Type() != InstrADD {
		t.Errorf("AddInstruction.Type() = %v, want %v", instr.Type(), InstrADD)
	}
}

func TestEnvInstructionMethods(t *testing.T) {
	instr := &EnvInstruction{
		LineNum: 3,
		RawText: "ENV NODE_ENV=production",
		Key:     "NODE_ENV",
		Value:   "production",
	}

	if instr.Line() != 3 {
		t.Errorf("EnvInstruction.Line() = %d, want %d", instr.Line(), 3)
	}
	if instr.Type() != InstrENV {
		t.Errorf("EnvInstruction.Type() = %v, want %v", instr.Type(), InstrENV)
	}
}

func TestArgInstructionMethods(t *testing.T) {
	instr := &ArgInstruction{
		LineNum: 2,
		RawText: "ARG VERSION=1.0",
		Name:    "VERSION",
		Default: "1.0",
	}

	if instr.Line() != 2 {
		t.Errorf("ArgInstruction.Line() = %d, want %d", instr.Line(), 2)
	}
	if instr.Type() != InstrARG {
		t.Errorf("ArgInstruction.Type() = %v, want %v", instr.Type(), InstrARG)
	}
}


func TestExposeInstructionMethods(t *testing.T) {
	instr := &ExposeInstruction{
		LineNum: 8,
		RawText: "EXPOSE 8080 443",
		Ports:   []string{"8080", "443"},
	}

	if instr.Line() != 8 {
		t.Errorf("ExposeInstruction.Line() = %d, want %d", instr.Line(), 8)
	}
	if instr.Type() != InstrEXPOSE {
		t.Errorf("ExposeInstruction.Type() = %v, want %v", instr.Type(), InstrEXPOSE)
	}
}

func TestWorkdirInstructionMethods(t *testing.T) {
	instr := &WorkdirInstruction{
		LineNum: 4,
		RawText: "WORKDIR /app",
		Path:    "/app",
	}

	if instr.Line() != 4 {
		t.Errorf("WorkdirInstruction.Line() = %d, want %d", instr.Line(), 4)
	}
	if instr.Type() != InstrWORKDIR {
		t.Errorf("WorkdirInstruction.Type() = %v, want %v", instr.Type(), InstrWORKDIR)
	}
}

func TestUserInstructionMethods(t *testing.T) {
	instr := &UserInstruction{
		LineNum: 15,
		RawText: "USER appuser:appgroup",
		User:    "appuser",
		Group:   "appgroup",
	}

	if instr.Line() != 15 {
		t.Errorf("UserInstruction.Line() = %d, want %d", instr.Line(), 15)
	}
	if instr.Type() != InstrUSER {
		t.Errorf("UserInstruction.Type() = %v, want %v", instr.Type(), InstrUSER)
	}
}

func TestLabelInstructionMethods(t *testing.T) {
	instr := &LabelInstruction{
		LineNum: 6,
		RawText: `LABEL maintainer="test@example.com"`,
		Labels:  map[string]string{"maintainer": "test@example.com"},
	}

	if instr.Line() != 6 {
		t.Errorf("LabelInstruction.Line() = %d, want %d", instr.Line(), 6)
	}
	if instr.Type() != InstrLABEL {
		t.Errorf("LabelInstruction.Type() = %v, want %v", instr.Type(), InstrLABEL)
	}
}

func TestVolumeInstructionMethods(t *testing.T) {
	instr := &VolumeInstruction{
		LineNum: 9,
		RawText: "VOLUME /data /logs",
		Paths:   []string{"/data", "/logs"},
	}

	if instr.Line() != 9 {
		t.Errorf("VolumeInstruction.Line() = %d, want %d", instr.Line(), 9)
	}
	if instr.Type() != InstrVOLUME {
		t.Errorf("VolumeInstruction.Type() = %v, want %v", instr.Type(), InstrVOLUME)
	}
}

func TestCmdInstructionMethods(t *testing.T) {
	instr := &CmdInstruction{
		LineNum: 20,
		RawText: `CMD ["node", "server.js"]`,
		Command: []string{"node", "server.js"},
		Shell:   false,
	}

	if instr.Line() != 20 {
		t.Errorf("CmdInstruction.Line() = %d, want %d", instr.Line(), 20)
	}
	if instr.Type() != InstrCMD {
		t.Errorf("CmdInstruction.Type() = %v, want %v", instr.Type(), InstrCMD)
	}
}

func TestEntrypointInstructionMethods(t *testing.T) {
	instr := &EntrypointInstruction{
		LineNum: 19,
		RawText: `ENTRYPOINT ["docker-entrypoint.sh"]`,
		Command: []string{"docker-entrypoint.sh"},
		Shell:   false,
	}

	if instr.Line() != 19 {
		t.Errorf("EntrypointInstruction.Line() = %d, want %d", instr.Line(), 19)
	}
	if instr.Type() != InstrENTRYPOINT {
		t.Errorf("EntrypointInstruction.Type() = %v, want %v", instr.Type(), InstrENTRYPOINT)
	}
}


func TestHealthcheckInstructionMethods(t *testing.T) {
	instr := &HealthcheckInstruction{
		LineNum:  18,
		RawText:  `HEALTHCHECK --interval=30s CMD curl -f http://localhost/`,
		Interval: "30s",
		Command:  []string{"curl", "-f", "http://localhost/"},
	}

	if instr.Line() != 18 {
		t.Errorf("HealthcheckInstruction.Line() = %d, want %d", instr.Line(), 18)
	}
	if instr.Type() != InstrHEALTHCHECK {
		t.Errorf("HealthcheckInstruction.Type() = %v, want %v", instr.Type(), InstrHEALTHCHECK)
	}
}

func TestShellInstructionMethods(t *testing.T) {
	instr := &ShellInstruction{
		LineNum: 7,
		RawText: `SHELL ["/bin/bash", "-c"]`,
		Shell:   []string{"/bin/bash", "-c"},
	}

	if instr.Line() != 7 {
		t.Errorf("ShellInstruction.Line() = %d, want %d", instr.Line(), 7)
	}
	if instr.Type() != InstrSHELL {
		t.Errorf("ShellInstruction.Type() = %v, want %v", instr.Type(), InstrSHELL)
	}
}

func TestStopsignalInstructionMethods(t *testing.T) {
	instr := &StopsignalInstruction{
		LineNum: 16,
		RawText: "STOPSIGNAL SIGTERM",
		Signal:  "SIGTERM",
	}

	if instr.Line() != 16 {
		t.Errorf("StopsignalInstruction.Line() = %d, want %d", instr.Line(), 16)
	}
	if instr.Type() != InstrSTOPSIGNAL {
		t.Errorf("StopsignalInstruction.Type() = %v, want %v", instr.Type(), InstrSTOPSIGNAL)
	}
}

func TestOnbuildInstructionMethods(t *testing.T) {
	wrapped := &RunInstruction{
		LineNum: 11,
		RawText: "RUN npm install",
		Command: "npm install",
		Shell:   true,
	}
	instr := &OnbuildInstruction{
		LineNum:     11,
		RawText:     "ONBUILD RUN npm install",
		Instruction: wrapped,
	}

	if instr.Line() != 11 {
		t.Errorf("OnbuildInstruction.Line() = %d, want %d", instr.Line(), 11)
	}
	if instr.Type() != InstrONBUILD {
		t.Errorf("OnbuildInstruction.Type() = %v, want %v", instr.Type(), InstrONBUILD)
	}
}

// TestInstructionInterface verifies all instruction types implement the Instruction interface.
func TestInstructionInterface(t *testing.T) {
	instructions := []Instruction{
		&FromInstruction{LineNum: 1, RawText: "FROM alpine"},
		&RunInstruction{LineNum: 2, RawText: "RUN echo hello"},
		&CopyInstruction{LineNum: 3, RawText: "COPY . ."},
		&AddInstruction{LineNum: 4, RawText: "ADD file.tar.gz /"},
		&EnvInstruction{LineNum: 5, RawText: "ENV FOO=bar"},
		&ArgInstruction{LineNum: 6, RawText: "ARG VERSION"},
		&ExposeInstruction{LineNum: 7, RawText: "EXPOSE 8080"},
		&WorkdirInstruction{LineNum: 8, RawText: "WORKDIR /app"},
		&UserInstruction{LineNum: 9, RawText: "USER app"},
		&LabelInstruction{LineNum: 10, RawText: "LABEL version=1.0"},
		&VolumeInstruction{LineNum: 11, RawText: "VOLUME /data"},
		&CmdInstruction{LineNum: 12, RawText: "CMD echo"},
		&EntrypointInstruction{LineNum: 13, RawText: "ENTRYPOINT /start"},
		&HealthcheckInstruction{LineNum: 14, RawText: "HEALTHCHECK CMD curl"},
		&ShellInstruction{LineNum: 15, RawText: "SHELL /bin/sh"},
		&StopsignalInstruction{LineNum: 16, RawText: "STOPSIGNAL SIGTERM"},
		&OnbuildInstruction{LineNum: 17, RawText: "ONBUILD RUN echo"},
	}

	for i, instr := range instructions {
		if instr.Line() != i+1 {
			t.Errorf("Instruction %d: Line() = %d, want %d", i, instr.Line(), i+1)
		}
		if instr.Raw() == "" {
			t.Errorf("Instruction %d: Raw() returned empty string", i)
		}
		if instr.Type() == "" {
			t.Errorf("Instruction %d: Type() returned empty string", i)
		}
	}
}
