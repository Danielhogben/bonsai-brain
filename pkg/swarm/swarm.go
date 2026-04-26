package swarm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/tools"
)

// Task is a unit of work sent to the swarm.
type Task struct {
	ID      string
	Prompt  string
	System  string
	MaxIter int
}

// TaskResult is the output from a single swarm agent.
type TaskResult struct {
	TaskID   string
	AgentID  string
	Model    string
	Provider ProviderType
	Output   string
	Latency  time.Duration
	Error    error
}

// Swarm aggregates multiple agents and distributes tasks.
type Swarm struct {
	Registry        *ProviderRegistry
	Agents          map[string]*SwarmAgent
	mu              sync.RWMutex
	maxAgents       int
	maxConcurrency  int
	rateLimiters    map[ProviderType]*rateLimiter
}

// SwarmAgent wraps an agent.Agent with swarm metadata.
type SwarmAgent struct {
	ID       string
	Model    string
	Provider ProviderType
	Agent    *agent.Agent
}

// NewSwarm creates a swarm from a provider registry.
func NewSwarm(registry *ProviderRegistry) *Swarm {
	s := &Swarm{
		Registry:       registry,
		Agents:         make(map[string]*SwarmAgent),
		maxAgents:      50,
		maxConcurrency: 10,
		rateLimiters:   make(map[ProviderType]*rateLimiter),
	}
	// Set up rate limiters based on provider configs.
	if registry != nil {
		registry.mu.RLock()
		for t, c := range registry.providers {
			if c.RateLimit > 0 {
				s.rateLimiters[t] = newRateLimiter(c.RateLimit)
			}
		}
		registry.mu.RUnlock()
	}
	return s
}

// Spawn creates a new swarm agent for the given model.
func (s *Swarm) Spawn(modelID string) (*SwarmAgent, error) {
	client, err := s.Registry.ModelClient(modelID)
	if err != nil {
		return nil, err
	}

	prov := s.Registry.ProviderFor(modelID)
	agentID := fmt.Sprintf("%s-%s", prov, sanitizeModelID(modelID))

	eng := &engine.QueryEngine{
		Model: client,
		Tools: make(map[string]engine.ToolEntry),
	}

	// Register tools inspired by OpenRouter top apps (OpenClaw, Hermes, Claude Code)
	eng.RegisterTool(
		engine.ToolSchema{Name: "web_search", Description: "Search the web for information", Parameters: map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}, "required": []string{"query"}}},
		func(ctx context.Context, args map[string]any) (string, error) {
			return tools.TavilySearch(ctx, args["query"].(string), 3)
		},
	)
	eng.RegisterTool(
		engine.ToolSchema{Name: "fetch_url", Description: "Fetch a web page", Parameters: map[string]any{"type": "object", "properties": map[string]any{"url": map[string]any{"type": "string"}}, "required": []string{"url"}}},
		func(ctx context.Context, args map[string]any) (string, error) {
			html, err := tools.BrowserFetch(ctx, args["url"].(string))
			if err != nil {
				return "", err
			}
			if len(html) > 2000 {
				html = html[:2000] + "..."
			}
			return html, nil
		},
	)
	eng.RegisterTool(
		engine.ToolSchema{Name: "read_file", Description: "Read a file", Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}}},
		func(ctx context.Context, args map[string]any) (string, error) {
			return tools.FileRead(ctx, args["path"].(string))
		},
	)
	eng.RegisterTool(
		engine.ToolSchema{Name: "write_file", Description: "Write to a file", Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"path", "content"}}},
		func(ctx context.Context, args map[string]any) (string, error) {
			return tools.FileWrite(ctx, args["path"].(string), args["content"].(string))
		},
	)
	eng.RegisterTool(
		engine.ToolSchema{Name: "grep_files", Description: "Search for text in files", Parameters: map[string]any{"type": "object", "properties": map[string]any{"pattern": map[string]any{"type": "string"}, "glob": map[string]any{"type": "string"}}, "required": []string{"pattern", "glob"}}},
		func(ctx context.Context, args map[string]any) (string, error) {
			return tools.FileGrep(ctx, args["pattern"].(string), args["glob"].(string))
		},
	)

	cfg := agent.DefaultConfig(agentID)
	cfg.SystemPrompt = "You are a helpful assistant with access to web search, file tools, and a browser. Be concise."
	a := agent.New(cfg, eng)

	sa := &SwarmAgent{
		ID:       agentID,
		Model:    modelID,
		Provider: prov,
		Agent:    a,
	}

	s.mu.Lock()
	if len(s.Agents) >= s.maxAgents {
		s.mu.Unlock()
		return nil, fmt.Errorf("swarm agent limit reached (%d)", s.maxAgents)
	}
	s.Agents[agentID] = sa
	s.mu.Unlock()
	return sa, nil
}

// SpawnAll creates one agent per available model.
func (s *Swarm) SpawnAll() ([]*SwarmAgent, error) {
	models := s.Registry.AllModels()
	var spawned []*SwarmAgent
	for _, m := range models {
		sa, err := s.Spawn(m)
		if err != nil {
			// Skip models that fail to initialize (e.g. no key).
			continue
		}
		spawned = append(spawned, sa)
	}
	return spawned, nil
}

// Distribute sends the task to all agents concurrently (with a semaphore) and returns results.
func (s *Swarm) Distribute(ctx context.Context, task Task) []TaskResult {
	s.mu.RLock()
	agents := make([]*SwarmAgent, 0, len(s.Agents))
	for _, a := range s.Agents {
		agents = append(agents, a)
	}
	s.mu.RUnlock()

	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	results := make([]TaskResult, len(agents))
	// Pre-fill with "not started" so any early-returned slots are visible.
	for i, a := range agents {
		results[i] = TaskResult{
			TaskID:   task.ID,
			AgentID:  a.ID,
			Model:    a.Model,
			Provider: a.Provider,
			Error:    fmt.Errorf("task not started or cancelled"),
		}
	}
	for i, sa := range agents {
		wg.Add(1)
		go func(idx int, a *SwarmAgent) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			// Per-provider rate limit.
			if lim, ok := s.rateLimiters[a.Provider]; ok {
				if err := lim.Acquire(ctx); err != nil {
					results[idx] = TaskResult{
						TaskID:   task.ID,
						AgentID:  a.ID,
						Model:    a.Model,
						Provider: a.Provider,
						Error:    err,
					}
					return
				}
			}
			start := time.Now()
			resp, err := a.Agent.GenerateText(ctx, task.Prompt)
			results[idx] = TaskResult{
				TaskID:   task.ID,
				AgentID:  a.ID,
				Model:    a.Model,
				Provider: a.Provider,
				Output:   resp,
				Latency:  time.Since(start),
				Error:    err,
			}
		}(i, sa)
	}

	wg.Wait()
	return results
}

// DistributeWithStrategy runs the task and applies a result strategy.
func (s *Swarm) DistributeWithStrategy(ctx context.Context, task Task, strategy ResultStrategy) (Result, error) {
	results := s.Distribute(ctx, task)
	return strategy(results)
}

// ResultStrategy picks a winner from a set of task results.
type ResultStrategy func([]TaskResult) (Result, error)

// Result is the aggregated output of a swarm run.
type Result struct {
	Winner      TaskResult
	AllResults  []TaskResult
	Strategy    string
	Description string
}

// FirstWinner returns the first successful result.
func FirstWinner(results []TaskResult) (Result, error) {
	for _, r := range results {
		if r.Error == nil && r.Output != "" {
			return Result{Winner: r, AllResults: results, Strategy: "first", Description: "First successful response"}, nil
		}
	}
	return Result{}, fmt.Errorf("no successful results")
}

// FastestWinner returns the result with the lowest latency and non-empty output.
func FastestWinner(results []TaskResult) (Result, error) {
	var best *TaskResult
	for i := range results {
		if results[i].Error != nil || results[i].Output == "" {
			continue
		}
		if best == nil || results[i].Latency < best.Latency {
			best = &results[i]
		}
	}
	if best == nil {
		return Result{}, fmt.Errorf("no successful results")
	}
	return Result{Winner: *best, AllResults: results, Strategy: "fastest", Description: "Lowest latency response"}, nil
}

// ConsensusWinner returns the most common answer (simple string match).
func ConsensusWinner(results []TaskResult) (Result, error) {
	freq := make(map[string]int)
	var best string
	var bestCount int
	for _, r := range results {
		if r.Error != nil {
			continue
		}
		normalized := strings.TrimSpace(strings.ToLower(r.Output))
		if len(normalized) > 100 {
			normalized = normalized[:100]
		}
		freq[normalized]++
		if freq[normalized] > bestCount {
			bestCount = freq[normalized]
			best = r.Output
		}
	}
	if best == "" {
		return Result{}, fmt.Errorf("no successful results for consensus")
	}
	return Result{
		Winner:      TaskResult{Output: best},
		AllResults:  results,
		Strategy:    "consensus",
		Description: fmt.Sprintf("Most common answer (%d/%d agree)", bestCount, len(results)),
	}, nil
}

// BestQualityWinner returns the result with the longest output as a heuristic.
func BestQualityWinner(results []TaskResult) (Result, error) {
	var best *TaskResult
	for i := range results {
		if results[i].Error != nil {
			continue
		}
		if best == nil || len(results[i].Output) > len(best.Output) {
			best = &results[i]
		}
	}
	if best == nil {
		return Result{}, fmt.Errorf("no successful results")
	}
	return Result{Winner: *best, AllResults: results, Strategy: "best_quality", Description: "Longest response (heuristic)"}, nil
}

// JudgeWinner uses an LLM judge to score results and pick the best.
// It falls back to BestQualityWinner if the judge fails.
func JudgeWinner(judge *Judge, task string) ResultStrategy {
	return func(results []TaskResult) (Result, error) {
		if judge == nil || judge.Client == nil {
			return BestQualityWinner(results)
		}
		idx, err := judge.PickBest(context.Background(), task, results)
		if err != nil || idx < 0 || idx >= len(results) {
			return BestQualityWinner(results)
		}
		return Result{
			Winner:      results[idx],
			AllResults:  results,
			Strategy:    "judge",
			Description: "LLM judge scored and selected best response",
		}, nil
	}
}

// HealthCheckAll pings every agent in the swarm.
func (s *Swarm) HealthCheckAll(ctx context.Context) map[string]time.Duration {
	s.mu.RLock()
	agents := make([]*SwarmAgent, 0, len(s.Agents))
	for _, a := range s.Agents {
		agents = append(agents, a)
	}
	s.mu.RUnlock()

	var mu sync.Mutex
	health := make(map[string]time.Duration)
	var wg sync.WaitGroup
	for _, sa := range agents {
		wg.Add(1)
		go func(a *SwarmAgent) {
			defer wg.Done()
			lat, err := s.Registry.HealthCheck(ctx, a.Model)
			mu.Lock()
			if err != nil {
				health[a.ID] = -1
			} else {
				health[a.ID] = lat
			}
			mu.Unlock()
		}(sa)
	}
	wg.Wait()
	return health
}

// AgentCount returns the number of live agents.
func (s *Swarm) AgentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Agents)
}

func sanitizeModelID(id string) string {
	return strings.ReplaceAll(strings.ReplaceAll(id, "/", "-"), ":", "-")
}

// ------------------------------------------------------------------
// Rate limiter
// ------------------------------------------------------------------

type rateLimiter struct {
	ticker *time.Ticker
	sem    chan struct{}
}

func newRateLimiter(rpm int) *rateLimiter {
	interval := time.Minute / time.Duration(rpm)
	r := &rateLimiter{
		ticker: time.NewTicker(interval),
		sem:    make(chan struct{}, 1),
	}
	r.sem <- struct{}{} // start with one token
	go func() {
		for range r.ticker.C {
			select {
			case r.sem <- struct{}{}:
			default:
			}
		}
	}()
	return r
}

func (r *rateLimiter) Acquire(ctx context.Context) error {
	select {
	case <-r.sem:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *rateLimiter) Stop() {
	r.ticker.Stop()
}
