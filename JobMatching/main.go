package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mete/HirePilot-JobMatching/internal/config"
	"github.com/mete/HirePilot-JobMatching/pkg/linkedin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Config error:", err)
		os.Exit(1)
	}
	ctx := context.Background()
	jobs, err := linkedin.LoginAndSearch(ctx, cfg.LinkedInEmail, cfg.LinkedInPassword, cfg.JobKeywords, cfg.Location)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	for _, job := range jobs {
		fmt.Println("Job:", job)
	}
}
