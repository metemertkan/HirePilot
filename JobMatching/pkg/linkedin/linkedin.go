package linkedin

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

func LoginAndSearch(ctx context.Context, email, password, keywords, location string) ([]string, error) {
	// Use Chromium explicitly for chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/snap/bin/chromium"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	var jobLinks []string

	tasks := chromedp.Tasks{
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
		chromedp.SendKeys(`#username`, email, chromedp.ByID),
		chromedp.SendKeys(`#password`, password, chromedp.ByID),
		chromedp.Click(`[type=submit]`, chromedp.ByQuery),
		chromedp.Sleep(3 * time.Second), // Wait for login
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	// Navigate to job search page
	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=%s", keywords, location)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a.job-card-list__title')).map(a => a.href)`, &jobLinks),
	); err != nil {
		return nil, fmt.Errorf("job search failed: %w", err)
	}

	return jobLinks, nil
}
