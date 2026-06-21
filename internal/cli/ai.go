package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mnhkahn/cyeam-cli/internal/ai"
	"github.com/mnhkahn/cyeam-cli/internal/output"
	"github.com/spf13/cobra"
)

type modelsFilter struct {
	platform  string
	modelType string
	search    string
}

func newAIModelsCommand() *cobra.Command {
	var f modelsFilter
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List free AI models from cyeam.com leaderboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := ai.FetchModels()
			if err != nil {
				return fmt.Errorf("fetch models: %w", err)
			}

			models := result.Models
			if f.platform != "" {
				var filtered []ai.Model
				for _, m := range models {
					if strings.EqualFold(m.Platform, f.platform) {
						filtered = append(filtered, m)
					}
				}
				models = filtered
			}
			if f.modelType != "" {
				var filtered []ai.Model
				for _, m := range models {
					if strings.EqualFold(m.ModelType, f.modelType) {
						filtered = append(filtered, m)
					}
				}
				models = filtered
			}
			if f.search != "" {
				q := strings.ToLower(f.search)
				var filtered []ai.Model
				for _, m := range models {
					if strings.Contains(strings.ToLower(m.ModelName), q) ||
						strings.Contains(strings.ToLower(m.Provider), q) {
						filtered = append(filtered, m)
					}
				}
				models = filtered
			}

			body, _ := json.Marshal(map[string]interface{}{
				"date":   result.Date,
				"count":  len(models),
				"models": models,
			})
			return output.WriteJSON(cmd.OutOrStdout(), body)
		},
	}
	cmd.Flags().StringVar(&f.platform, "platform", "", "filter by platform: openrouter, siliconflow, nvidia, huggingface")
	cmd.Flags().StringVar(&f.modelType, "type", "", "filter by model type: text, multimodal, reasoning")
	cmd.Flags().StringVar(&f.search, "search", "", "search models by name or provider")
	return cmd
}

func newAICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI tools - free models, benchmarks",
	}
	cmd.AddCommand(newAIModelsCommand())
	return cmd
}
