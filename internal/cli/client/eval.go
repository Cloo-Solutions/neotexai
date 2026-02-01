package client

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

type EvalCase struct {
	Query       string   `json:"query"`
	ExpectedIDs []string `json:"expected_ids"`
	ProjectID   string   `json:"project_id,omitempty"`
	Type        string   `json:"type,omitempty"`
}

type EvalSuite struct {
	Cases []EvalCase `json:"cases"`
	Limit int        `json:"limit,omitempty"`
}

type EvalSummary struct {
	Total      int     `json:"total"`
	K          int     `json:"k"`
	Limit      int     `json:"limit"`
	RecallAtK  float64 `json:"recall_at_k"`
	MRR        float64 `json:"mrr"`
	HitRateAtK float64 `json:"hit_rate_at_k"`
}

type EvalCaseResult struct {
	Query       string   `json:"query"`
	ExpectedIDs []string `json:"expected_ids"`
	FoundIDs    []string `json:"found_ids"`
	Rank        int      `json:"rank"`
	RecallAtK   float64  `json:"recall_at_k"`
	RR          float64  `json:"rr"`
}

type EvalOutput struct {
	Summary EvalSummary      `json:"summary"`
	Cases   []EvalCaseResult `json:"cases,omitempty"`
}

// EvalCmd creates the eval command.
func EvalCmd() *cobra.Command {
	var (
		file    string
		limit   int
		k       int
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "eval --file <eval.json>",
		Short: "Evaluate search quality",
		Long: `Evaluate search quality against a set of queries and expected IDs.

The input file can be either:
  - { "cases": [ { "query": "...", "expected_ids": [...] } ], "limit": 20 }
  - [ { "query": "...", "expected_ids": [...] } ]`,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, _ := cmd.Flags().GetBool("output")
			return runEval(file, limit, k, verbose, outputJSON)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Evaluation JSON file (required)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Override search limit for evaluation")
	cmd.Flags().IntVar(&k, "k", 10, "Compute recall@k and hit@k")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Print per-case results")
	cmd.MarkFlagRequired("file")

	return cmd
}

func runEval(file string, limit, k int, verbose, outputJSON bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read eval file: %w", err)
	}

	var suite EvalSuite
	if err := json.Unmarshal(data, &suite); err != nil || len(suite.Cases) == 0 {
		var cases []EvalCase
		if err := json.Unmarshal(data, &cases); err != nil {
			return fmt.Errorf("failed to parse eval file: %w", err)
		}
		suite.Cases = cases
	}

	if len(suite.Cases) == 0 {
		return fmt.Errorf("no eval cases provided")
	}

	if limit <= 0 {
		limit = suite.Limit
	}
	if limit <= 0 {
		limit = 20
	}
	if k <= 0 {
		k = 10
	}
	if k > limit {
		k = limit
	}

	defaultProjectID := ""
	if cfg, err := LoadConfig(); err == nil {
		defaultProjectID = cfg.ProjectID
	}

	api, err := NewAPIClient()
	if err != nil {
		return err
	}

	var (
		sumRecall   float64
		sumRR       float64
		hitCount    int
		caseResults []EvalCaseResult
	)

	for _, c := range suite.Cases {
		if c.Query == "" {
			return fmt.Errorf("eval case query is required")
		}
		if len(c.ExpectedIDs) == 0 {
			return fmt.Errorf("eval case expected_ids is required")
		}

		projectID := c.ProjectID
		if projectID == "" {
			projectID = defaultProjectID
		}

		req := SearchRequest{
			Query:     c.Query,
			ProjectID: projectID,
			Type:      c.Type,
			Limit:     limit,
		}

		resp, err := api.Post("/search", req)
		if err != nil {
			return fmt.Errorf("search failed for query %q: %w", c.Query, err)
		}

		var searchResp SearchResponse
		if err := json.Unmarshal(resp.Data, &searchResp); err != nil {
			return fmt.Errorf("failed to parse search response: %w", err)
		}

		expectedSet := make(map[string]struct{}, len(c.ExpectedIDs))
		for _, id := range c.ExpectedIDs {
			expectedSet[id] = struct{}{}
		}

		foundIDs := make([]string, 0, len(searchResp.Results))
		hits := 0
		rank := 0
		for i, result := range searchResp.Results {
			foundIDs = append(foundIDs, result.ID)
			if i < k {
				if _, ok := expectedSet[result.ID]; ok {
					hits++
					if rank == 0 {
						rank = i + 1
					}
				}
			}
		}

		recall := float64(hits) / float64(len(expectedSet))
		sumRecall += recall
		rr := 0.0
		if rank > 0 {
			rr = 1.0 / float64(rank)
			sumRR += rr
			hitCount++
		}

		if verbose || outputJSON {
			caseResults = append(caseResults, EvalCaseResult{
				Query:       c.Query,
				ExpectedIDs: c.ExpectedIDs,
				FoundIDs:    foundIDs,
				Rank:        rank,
				RecallAtK:   recall,
				RR:          rr,
			})
		}
	}

	summary := EvalSummary{
		Total:      len(suite.Cases),
		K:          k,
		Limit:      limit,
		RecallAtK:  sumRecall / float64(len(suite.Cases)),
		MRR:        sumRR / float64(len(suite.Cases)),
		HitRateAtK: float64(hitCount) / float64(len(suite.Cases)),
	}

	if outputJSON {
		out := EvalOutput{Summary: summary}
		if verbose {
			sort.Slice(caseResults, func(i, j int) bool {
				return caseResults[i].Query < caseResults[j].Query
			})
			out.Cases = caseResults
		}
		encoded, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}

	fmt.Printf("Eval results (k=%d, limit=%d)\n", summary.K, summary.Limit)
	fmt.Printf("Recall@%d: %.4f\n", summary.K, summary.RecallAtK)
	fmt.Printf("MRR: %.4f\n", summary.MRR)
	fmt.Printf("Hit@%d: %.4f\n", summary.K, summary.HitRateAtK)

	if verbose {
		for _, r := range caseResults {
			fmt.Printf("\nQuery: %s\n", r.Query)
			fmt.Printf("Rank: %d  Recall@%d: %.4f  RR: %.4f\n", r.Rank, summary.K, r.RecallAtK, r.RR)
			fmt.Printf("Expected: %v\n", r.ExpectedIDs)
			fmt.Printf("Found: %v\n", r.FoundIDs)
		}
	}

	return nil
}
