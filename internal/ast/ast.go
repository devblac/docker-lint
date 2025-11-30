// Package ast defines the Abstract Syntax Tree data structures for Dockerfile representation.
package ast

// InstructionType represents the type of a Dockerfile instruction.
type InstructionType string

const (
	InstrFROM        InstructionType = "FROM"
	InstrRUN         InstructionType = "RUN"
	InstrCOPY        InstructionType = "COPY"
	InstrADD         InstructionType = "ADD"
	InstrENV         InstructionType = "ENV"
	InstrARG         InstructionType = "ARG"
	InstrEXPOSE      InstructionType = "EXPOSE"
	InstrWORKDIR     InstructionType = "WORKDIR"
	InstrUSER        InstructionType = "USER"
	InstrLABEL       InstructionType = "LABEL"
	InstrVOLUME      InstructionType = "VOLUME"
	InstrCMD         InstructionType = "CMD"
	InstrENTRYPOINT  InstructionType = "ENTRYPOINT"
	InstrHEALTHCHECK InstructionType = "HEALTHCHECK"
	InstrSHELL       InstructionType = "SHELL"
	InstrSTOPSIGNAL  InstructionType = "STOPSIGNAL"
	InstrONBUILD     InstructionType = "ONBUILD"
)

// Severity represents the severity level of a lint finding.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

// String returns the string representation of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}


// Finding represents a lint finding from rule analysis.
type Finding struct {
	RuleID     string
	Severity   Severity
	Line       int
	Column     int
	Message    string
	Suggestion string
}

// Instruction is the interface that all Dockerfile instructions implement.
type Instruction interface {
	Line() int
	Raw() string
	Type() InstructionType
}

// Comment represents a comment in a Dockerfile.
type Comment struct {
	LineNum int
	Text    string
}

// Stage represents a build stage in a multi-stage Dockerfile.
type Stage struct {
	Name         string
	FromInstr    *FromInstruction
	Instructions []Instruction
	Index        int
}

// Dockerfile represents a parsed Dockerfile.
type Dockerfile struct {
	Stages       []Stage
	Instructions []Instruction
	Comments     []Comment
	InlineIgnores map[int][]string // line -> rule IDs to ignore
}

// FromInstruction represents a FROM instruction.
type FromInstruction struct {
	LineNum  int
	RawText  string
	Image    string
	Tag      string
	Digest   string
	Alias    string // AS name
	Platform string // --platform flag
}

func (f *FromInstruction) Line() int             { return f.LineNum }
func (f *FromInstruction) Raw() string           { return f.RawText }
func (f *FromInstruction) Type() InstructionType { return InstrFROM }


// RunInstruction represents a RUN instruction.
type RunInstruction struct {
	LineNum int
	RawText string
	Command string
	Shell   bool // shell form vs exec form
}

func (r *RunInstruction) Line() int             { return r.LineNum }
func (r *RunInstruction) Raw() string           { return r.RawText }
func (r *RunInstruction) Type() InstructionType { return InstrRUN }

// CopyInstruction represents a COPY instruction.
type CopyInstruction struct {
	LineNum int
	RawText string
	Sources []string
	Dest    string
	From    string // --from flag for multi-stage
	Chown   string // --chown flag
}

func (c *CopyInstruction) Line() int             { return c.LineNum }
func (c *CopyInstruction) Raw() string           { return c.RawText }
func (c *CopyInstruction) Type() InstructionType { return InstrCOPY }

// AddInstruction represents an ADD instruction.
type AddInstruction struct {
	LineNum int
	RawText string
	Sources []string
	Dest    string
	Chown   string // --chown flag
}

func (a *AddInstruction) Line() int             { return a.LineNum }
func (a *AddInstruction) Raw() string           { return a.RawText }
func (a *AddInstruction) Type() InstructionType { return InstrADD }

// EnvInstruction represents an ENV instruction.
type EnvInstruction struct {
	LineNum int
	RawText string
	Key     string
	Value   string
}

func (e *EnvInstruction) Line() int             { return e.LineNum }
func (e *EnvInstruction) Raw() string           { return e.RawText }
func (e *EnvInstruction) Type() InstructionType { return InstrENV }

// ArgInstruction represents an ARG instruction.
type ArgInstruction struct {
	LineNum int
	RawText string
	Name    string
	Default string
}

func (a *ArgInstruction) Line() int             { return a.LineNum }
func (a *ArgInstruction) Raw() string           { return a.RawText }
func (a *ArgInstruction) Type() InstructionType { return InstrARG }


// ExposeInstruction represents an EXPOSE instruction.
type ExposeInstruction struct {
	LineNum int
	RawText string
	Ports   []string
}

func (e *ExposeInstruction) Line() int             { return e.LineNum }
func (e *ExposeInstruction) Raw() string           { return e.RawText }
func (e *ExposeInstruction) Type() InstructionType { return InstrEXPOSE }

// WorkdirInstruction represents a WORKDIR instruction.
type WorkdirInstruction struct {
	LineNum int
	RawText string
	Path    string
}

func (w *WorkdirInstruction) Line() int             { return w.LineNum }
func (w *WorkdirInstruction) Raw() string           { return w.RawText }
func (w *WorkdirInstruction) Type() InstructionType { return InstrWORKDIR }

// UserInstruction represents a USER instruction.
type UserInstruction struct {
	LineNum int
	RawText string
	User    string
	Group   string
}

func (u *UserInstruction) Line() int             { return u.LineNum }
func (u *UserInstruction) Raw() string           { return u.RawText }
func (u *UserInstruction) Type() InstructionType { return InstrUSER }

// LabelInstruction represents a LABEL instruction.
type LabelInstruction struct {
	LineNum int
	RawText string
	Labels  map[string]string
}

func (l *LabelInstruction) Line() int             { return l.LineNum }
func (l *LabelInstruction) Raw() string           { return l.RawText }
func (l *LabelInstruction) Type() InstructionType { return InstrLABEL }

// VolumeInstruction represents a VOLUME instruction.
type VolumeInstruction struct {
	LineNum int
	RawText string
	Paths   []string
}

func (v *VolumeInstruction) Line() int             { return v.LineNum }
func (v *VolumeInstruction) Raw() string           { return v.RawText }
func (v *VolumeInstruction) Type() InstructionType { return InstrVOLUME }


// CmdInstruction represents a CMD instruction.
type CmdInstruction struct {
	LineNum int
	RawText string
	Command []string
	Shell   bool // shell form vs exec form
}

func (c *CmdInstruction) Line() int             { return c.LineNum }
func (c *CmdInstruction) Raw() string           { return c.RawText }
func (c *CmdInstruction) Type() InstructionType { return InstrCMD }

// EntrypointInstruction represents an ENTRYPOINT instruction.
type EntrypointInstruction struct {
	LineNum int
	RawText string
	Command []string
	Shell   bool // shell form vs exec form
}

func (e *EntrypointInstruction) Line() int             { return e.LineNum }
func (e *EntrypointInstruction) Raw() string           { return e.RawText }
func (e *EntrypointInstruction) Type() InstructionType { return InstrENTRYPOINT }

// HealthcheckInstruction represents a HEALTHCHECK instruction.
type HealthcheckInstruction struct {
	LineNum  int
	RawText  string
	None     bool // HEALTHCHECK NONE
	Interval string
	Timeout  string
	Retries  string
	Start    string
	Command  []string
}

func (h *HealthcheckInstruction) Line() int             { return h.LineNum }
func (h *HealthcheckInstruction) Raw() string           { return h.RawText }
func (h *HealthcheckInstruction) Type() InstructionType { return InstrHEALTHCHECK }

// ShellInstruction represents a SHELL instruction.
type ShellInstruction struct {
	LineNum int
	RawText string
	Shell   []string
}

func (s *ShellInstruction) Line() int             { return s.LineNum }
func (s *ShellInstruction) Raw() string           { return s.RawText }
func (s *ShellInstruction) Type() InstructionType { return InstrSHELL }

// StopsignalInstruction represents a STOPSIGNAL instruction.
type StopsignalInstruction struct {
	LineNum int
	RawText string
	Signal  string
}

func (s *StopsignalInstruction) Line() int             { return s.LineNum }
func (s *StopsignalInstruction) Raw() string           { return s.RawText }
func (s *StopsignalInstruction) Type() InstructionType { return InstrSTOPSIGNAL }

// OnbuildInstruction represents an ONBUILD instruction.
type OnbuildInstruction struct {
	LineNum     int
	RawText     string
	Instruction Instruction // The wrapped instruction
}

func (o *OnbuildInstruction) Line() int             { return o.LineNum }
func (o *OnbuildInstruction) Raw() string           { return o.RawText }
func (o *OnbuildInstruction) Type() InstructionType { return InstrONBUILD }
