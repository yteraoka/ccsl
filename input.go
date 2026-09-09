package main

// Input is the JSON payload Claude Code writes to the status line command's
// stdin. Only the fields this tool renders are declared; pointers and pointer
// fields mark everything the docs list as optional or nullable.
//
// Schema: https://code.claude.com/docs/en/statusline.md
type Input struct {
	CWD            string `json:"cwd"`
	SessionID      string `json:"session_id"`
	SessionName    string `json:"session_name"`
	TranscriptPath string `json:"transcript_path"`
	Version        string `json:"version"`

	Model         Model         `json:"model"`
	Workspace     Workspace     `json:"workspace"`
	Cost          Cost          `json:"cost"`
	ContextWindow ContextWindow `json:"context_window"`

	Exceeds200kTokens bool `json:"exceeds_200k_tokens"`

	RateLimits *RateLimits `json:"rate_limits"`
	PR         *PR         `json:"pr"`
	Worktree   *Worktree   `json:"worktree"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir  string   `json:"current_dir"`
	ProjectDir  string   `json:"project_dir"`
	AddedDirs   []string `json:"added_dirs"`
	GitWorktree string   `json:"git_worktree"`
	Repo        *Repo    `json:"repo"`
}

type Repo struct {
	Host  string `json:"host"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

type Cost struct {
	TotalCostUSD       float64 `json:"total_cost_usd"`
	TotalDurationMS    int64   `json:"total_duration_ms"`
	TotalAPIDurationMS int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

type ContextWindow struct {
	TotalInputTokens    int      `json:"total_input_tokens"`
	TotalOutputTokens   int      `json:"total_output_tokens"`
	ContextWindowSize   int      `json:"context_window_size"`
	UsedPercentage      *float64 `json:"used_percentage"`
	RemainingPercentage *float64 `json:"remaining_percentage"`
	CurrentUsage        *Usage   `json:"current_usage"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type RateLimits struct {
	FiveHour   *RateLimit `json:"five_hour"`
	SevenDay   *RateLimit `json:"seven_day"`
	SpendLimit *RateLimit `json:"spend_limit"`
}

type RateLimit struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

type PR struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	ReviewState string `json:"review_state"`
	Kind        string `json:"kind"`
}

type Worktree struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Branch         string `json:"branch"`
	OriginalCWD    string `json:"original_cwd"`
	OriginalBranch string `json:"original_branch"`
}

// dir returns the directory the session is working in, preferring the
// workspace field the docs recommend over the legacy top-level cwd.
func (in *Input) dir() string {
	if in.Workspace.CurrentDir != "" {
		return in.Workspace.CurrentDir
	}
	return in.CWD
}
