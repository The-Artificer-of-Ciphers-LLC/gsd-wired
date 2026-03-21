package mcp

import (
	"context"
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// toolError returns a CallToolResult with IsError=true and a descriptive message.
func toolError(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
	}
}

// toolResult marshals data as JSON and returns it in TextContent.
func toolResult(data any) (*mcpsdk.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return toolError("marshal error: " + err.Error()), nil
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(b)}},
	}, nil
}

// closeResult is the response for close_plan: the closed bead plus newly unblocked beads.
type closeResult struct {
	Closed    *graph.Bead  `json:"closed"`
	Unblocked []graph.Bead `json:"unblocked"`
}

// registerTools registers all 17 GSD MCP tools on the server.
// Each handler calls state.init(ctx) before any graph operation (D-06, D-07).
// Tool errors use IsError=true — Go errors are only for protocol failures (D-09).
func registerTools(server *mcpsdk.Server, state *serverState) {
	// create_phase — Creates a GSD phase as an epic bead.
	server.AddTool(&mcpsdk.Tool{
		Name:        "create_phase",
		Description: "Creates a GSD phase as an epic bead in the beads graph.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number (1-99)"},"title":{"type":"string","description":"Phase title"},"goal":{"type":"string","description":"Phase goal description"},"acceptance":{"type":"string","description":"Acceptance criteria"},"req_ids":{"type":"array","items":{"type":"string"},"description":"Requirement IDs (e.g. INFRA-01)"}},"required":["phase_num","title","goal","acceptance"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			PhaseNum   int      `json:"phase_num"`
			Title      string   `json:"title"`
			Goal       string   `json:"goal"`
			Acceptance string   `json:"acceptance"`
			ReqIDs     []string `json:"req_ids"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		bead, err := state.client.CreatePhase(ctx, args.PhaseNum, args.Title, args.Goal, args.Acceptance, args.ReqIDs)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(bead)
	})

	// create_plan — Creates a GSD plan as a task bead under a phase epic.
	server.AddTool(&mcpsdk.Tool{
		Name:        "create_plan",
		Description: "Creates a GSD plan as a task bead under a phase epic bead.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"plan_id":{"type":"string","description":"Plan ID (e.g. 03-01)"},"phase_num":{"type":"integer","description":"Phase number"},"parent_bead_id":{"type":"string","description":"Parent phase epic bead ID"},"title":{"type":"string","description":"Plan title"},"acceptance":{"type":"string","description":"Acceptance criteria"},"context":{"type":"string","description":"Plan context/description"},"req_ids":{"type":"array","items":{"type":"string"},"description":"Requirement IDs"},"dep_bead_ids":{"type":"array","items":{"type":"string"},"description":"Dependency bead IDs"}},"required":["plan_id","phase_num","parent_bead_id","title","acceptance","context"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			PlanID       string   `json:"plan_id"`
			PhaseNum     int      `json:"phase_num"`
			ParentBeadID string   `json:"parent_bead_id"`
			Title        string   `json:"title"`
			Acceptance   string   `json:"acceptance"`
			Context      string   `json:"context"`
			ReqIDs       []string `json:"req_ids"`
			DepBeadIDs   []string `json:"dep_bead_ids"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		bead, err := state.client.CreatePlan(ctx, args.PlanID, args.PhaseNum, args.ParentBeadID, args.Title, args.Acceptance, args.Context, args.ReqIDs, args.DepBeadIDs)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(bead)
	})

	// get_bead — Retrieves a single bead by ID.
	server.AddTool(&mcpsdk.Tool{
		Name:        "get_bead",
		Description: "Retrieves a single bead by ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Bead ID"}},"required":["id"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		bead, err := state.client.GetBead(ctx, args.ID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(bead)
	})

	// list_ready — Lists all unblocked (ready) beads.
	server.AddTool(&mcpsdk.Tool{
		Name:        "list_ready",
		Description: "Lists all unblocked (ready) beads with no limit.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		beads, err := state.client.ListReady(ctx)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(beads)
	})

	// query_by_label — Queries beads matching a label.
	server.AddTool(&mcpsdk.Tool{
		Name:        "query_by_label",
		Description: "Queries beads matching a label (e.g. gsd:phase, INFRA-02).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"label":{"type":"string","description":"Label to query (e.g. gsd:phase, INFRA-02)"}},"required":["label"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			Label string `json:"label"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		beads, err := state.client.QueryByLabel(ctx, args.Label)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(beads)
	})

	// claim_bead — Atomically claims a bead for the current agent.
	server.AddTool(&mcpsdk.Tool{
		Name:        "claim_bead",
		Description: "Atomically claims a bead for the current agent (fails if already claimed).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Bead ID to claim"}},"required":["id"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		bead, err := state.client.ClaimBead(ctx, args.ID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(bead)
	})

	// close_plan — Closes a plan bead with reason, returns newly unblocked beads.
	server.AddTool(&mcpsdk.Tool{
		Name:        "close_plan",
		Description: "Closes a plan bead with a reason and returns newly unblocked beads.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Bead ID to close"},"reason":{"type":"string","description":"Close reason"}},"required":["id"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		var args struct {
			ID     string `json:"id"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		closed, unblocked, err := state.client.ClosePlan(ctx, args.ID, args.Reason)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(&closeResult{Closed: closed, Unblocked: unblocked})
	})

	// flush_writes — Flushes accumulated batch writes to Dolt (INFRA-10).
	server.AddTool(&mcpsdk.Tool{
		Name:        "flush_writes",
		Description: "Flushes accumulated batch writes to Dolt. Call after a series of create/update operations.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"Optional commit message"}},"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		if err := state.init(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		if err := state.client.FlushWrites(ctx); err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(map[string]string{"status": "flushed"})
	})

	// init_project — Initializes a new gsd-wired project with bead creation and file writing.
	server.AddTool(&mcpsdk.Tool{
		Name:        "init_project",
		Description: "Initialize a new gsd-wired project: runs bd init if needed, creates project epic bead + context beads, writes PROJECT.md and .gsdw/config.json.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"project_name":{"type":"string","description":"Project name"},"what":{"type":"string","description":"What the project builds"},"why":{"type":"string","description":"Why it exists"},"who":{"type":"string","description":"Target users"},"done_criteria":{"type":"string","description":"What done looks like"},"tech_stack":{"type":"string","description":"Technology stack"},"constraints":{"type":"string","description":"Project constraints"},"risks":{"type":"string","description":"Known risks"},"mode":{"type":"string","enum":["full","quick","pr"],"description":"Init mode: full (all questions), quick (essentials), pr (import from PR/issue)"},"pr_url":{"type":"string","description":"PR or issue URL for pr mode"}},"required":["project_name","what","why","done_criteria","mode"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args initProjectArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleInitProject(ctx, state, args)
	})

	// get_status — Returns current project status from the beads graph.
	server.AddTool(&mcpsdk.Tool{
		Name:        "get_status",
		Description: "Returns current project status from beads graph: project name, current phase, ready tasks, phase counts.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return handleGetStatus(ctx, state)
	})

	// run_research — Creates a research epic bead + 4 child beads for parallel research agents.
	server.AddTool(&mcpsdk.Tool{
		Name:        "run_research",
		Description: "Creates a research epic bead plus 4 child beads (stack, features, architecture, pitfalls) for parallel research agents. Returns epic_bead_id and child_bead_ids map.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number for this research"},"title":{"type":"string","description":"Research title (e.g. project name)"},"req_ids":{"type":"array","items":{"type":"string"},"description":"Requirement IDs to attach to the research epic"}},"required":["phase_num","title"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args runResearchArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleRunResearch(ctx, state, args)
	})

	// synthesize_research — Queries completed research child beads and creates a summary bead.
	server.AddTool(&mcpsdk.Tool{
		Name:        "synthesize_research",
		Description: "Queries the research epic for the given phase and creates a summary bead combining all findings.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number of the research to synthesize"},"summary":{"type":"string","description":"Combined summary of all research findings"}},"required":["phase_num","summary"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args synthesizeResearchArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleSynthesizeResearch(ctx, state, args)
	})

	// create_plan_beads — Batch-creates task beads from a structured JSON plan with dependency wiring.
	server.AddTool(&mcpsdk.Tool{
		Name:        "create_plan_beads",
		Description: "Batch-creates task beads from a structured plan, resolving local task IDs to bead IDs in topological order for dependency wiring.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number for these task beads"},"epic_bead_id":{"type":"string","description":"Parent epic bead ID for the phase"},"tasks":{"type":"array","description":"Ordered list of task definitions","items":{"type":"object","properties":{"id":{"type":"string","description":"Local task ID (e.g. 06-01)"},"title":{"type":"string","description":"Task title"},"acceptance":{"type":"string","description":"Acceptance criteria"},"context":{"type":"string","description":"Task context/description"},"req_ids":{"type":"array","items":{"type":"string"},"description":"Requirement IDs covered by this task"},"depends_on":{"type":"array","items":{"type":"string"},"description":"Local task IDs this task depends on"},"complexity":{"type":"string","description":"Estimated complexity: S, M, or L"},"files":{"type":"array","items":{"type":"string"},"description":"Files to be created or modified"}},"required":["id","title","acceptance","context","complexity","files"],"additionalProperties":false}}},"required":["phase_num","epic_bead_id","tasks"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args createPlanBeadsArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleCreatePlanBeads(ctx, state, args)
	})

	// execute_wave — Returns full context chains for all ready tasks in a phase (tool 14).
	server.AddTool(&mcpsdk.Tool{
		Name:        "execute_wave",
		Description: "Returns pre-computed context chains for all ready tasks in a phase. Each context includes bead_id, plan_id, title, acceptance_criteria, parent_summary, and dep_summaries.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to execute"}},"required":["phase_num"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args executeWaveArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleExecuteWave(ctx, state, args)
	})

	// verify_phase — Checks phase success criteria against codebase state (tool 15).
	server.AddTool(&mcpsdk.Tool{
		Name:        "verify_phase",
		Description: "Checks phase success criteria (acceptance criteria) against codebase state. Returns structured pass/fail per criterion with method and detail.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to verify"},"project_dir":{"type":"string","description":"Absolute path to project root for file checks"}},"required":["phase_num"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args verifyPhaseArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleVerifyPhase(ctx, state, args)
	})

	// create_pr_summary — Creates a bead-sourced PR summary for a phase (tool 16).
	server.AddTool(&mcpsdk.Tool{
		Name:        "create_pr_summary",
		Description: "Creates a bead-sourced PR summary for a phase: title, markdown body with requirements and phase checklist, and branch name.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to create PR summary for"}},"required":["phase_num"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args createPrSummaryArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleCreatePrSummary(ctx, state, args)
	})

	// advance_phase — Closes a phase epic bead and returns newly unblocked beads (tool 17).
	server.AddTool(&mcpsdk.Tool{
		Name:        "advance_phase",
		Description: "Closes a phase epic bead with a reason and returns newly unblocked beads and next phase info.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to advance"},"reason":{"type":"string","description":"Completion reason"}},"required":["phase_num","reason"],"additionalProperties":false}`),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		var args advancePhaseArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return toolError("invalid arguments: " + err.Error()), nil
		}
		return handleAdvancePhase(ctx, state, args)
	})
}
