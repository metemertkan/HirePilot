package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	sharedNats "github.com/hirepilot/shared/nats"
)

func main() {
	email := os.Getenv("LINKEDIN_EMAIL")
	password := os.Getenv("LINKEDIN_PASSWORD")

	// Launch browser with better stealth configuration
	browser := launchBrowser()
	log.Println("Browser created")
	defer browser.MustClose()

	page := browser.MustPage("").MustWaitLoad()
	log.Println("Page created")
	// Configure browser to look more human-like
	setHumanLikeBehavior(page)
	log.Println("Human like behavior set")
	// Login with precise element targeting
	if err := loginToLinkedIn(page, email, password); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	log.Println("Login successful")
	// Navigate to saved jobs
	if err := navigateToSavedJobs(page); err != nil {
		log.Fatalf("Failed to navigate to saved jobs: %v", err)
	}
	log.Println("Navigated to saved jobs")
	// Process saved jobs
	if err := processSavedJobs(page); err != nil {
		log.Fatalf("Failed to process saved jobs: %v", err)
	}
	log.Println("Processed saved jobs")
	log.Println("Automation completed successfully")
}

func launchBrowser() *rod.Browser {
	// For debugging (visible browser):
	path, _ := launcher.LookPath()
	url := launcher.New().Bin(path).Headless(false).MustLaunch()
	return rod.New().ControlURL(url).MustConnect()

	// For production (headless with better stealth):
	// browser := rod.New().
	// 	ControlURL(launcher.New().
	// 		Headless(true).
	// 		Set("disable-blink-features", "AutomationControlled").
	// 		MustLaunch()).
	// 	MustConnect()

	// // // Mimic human behavior better
	// browser.MustIgnoreCertErrors(true)
	// return browser
}

func setHumanLikeBehavior(page *rod.Page) {
	// Override navigator properties to look less like automation
	page.MustEval(`() => {
		Object.defineProperty(navigator, 'webdriver', { get: () => false });
		Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3] });
		Object.defineProperty(navigator, 'languages', { get: () => ['en-US', 'en'] });
	}`)
}

func loginToLinkedIn(page *rod.Page, email, password string) error {
	// Navigate to login page with timeout
	page.Timeout(30 * time.Second).MustNavigate("https://www.linkedin.com/login").MustWaitLoad()
	log.Println("Navigated to login page")

	// Wait for page to fully load
	time.Sleep(2 * time.Second)

	// Fill email using rod.Try for better error handling
	err := rod.Try(func() {
		emailElement := page.MustElement("#username")
		emailElement.MustWaitVisible()
		emailElement.MustClick() // Focus the element first
		emailElement.MustSelectAllText().MustInput(email)
	})
	if err != nil {
		return fmt.Errorf("email input failed: %w", err)
	}
	log.Println("Email input successful")

	// Fill password using rod.Try for better error handling
	err = rod.Try(func() {
		passwordElement := page.MustElement("#password")
		passwordElement.MustWaitVisible()
		passwordElement.MustClick() // Focus the element first
		passwordElement.MustSelectAllText().MustInput(password)
	})
	if err != nil {
		return fmt.Errorf("password input failed: %w", err)
	}
	log.Println("Password input successful")

	// Uncheck "Keep me logged in" checkbox for security - click the label
	err = rod.Try(func() {
		rememberMeLabel := page.MustElement("label[for='rememberMeOptIn-checkbox']")
		rememberMeLabel.MustWaitVisible()
		rememberMeLabel.MustClick()
		log.Println("Clicked 'Keep me logged in' label to uncheck checkbox")
	})
	if err != nil {
		log.Printf("Warning: Could not click remember me label: %v", err)
		// Continue anyway - this is not critical
	}

	// Add a small delay before clicking login
	time.Sleep(1 * time.Second)
	// Click the correct login submit button (not Apple Sign In)
	err = rod.Try(func() {
		// Try multiple selectors to find the correct login button
		var loginButton *rod.Element

		// Method 1: Try by specific classes and attributes
		loginButton, err := page.Element(`button[data-litms-control-urn="login-submit"]`)
		if err == nil {
			loginButton.MustClick()
			return
		}

		// Method 2: Try by form submit button
		loginButton, err = page.Element(`button[type="submit"]`)
		if err == nil {
			loginButton.MustClick()
			return
		}

		// Method 3: Try by class combination
		loginButton, err = page.Element(`.btn__primary--large.from__button--floating`)
		if err == nil {
			loginButton.MustClick()
			return
		}

		// Method 4: Try by aria-label
		loginButton, err = page.Element(`button[aria-label="Sign in"]`)
		if err == nil {
			loginButton.MustClick()
			return
		}

		// If all methods fail, throw error
		panic("Could not find login submit button")
	})
	if err != nil {
		return fmt.Errorf("login button click failed: %w", err)
	}
	log.Println("Login button clicked")

	// Wait for navigation to complete
	time.Sleep(5 * time.Second)

	// Try multiple verification methods as LinkedIn UI can vary
	var loginSuccess bool

	// Method 1: Check for profile menu
	err = rod.Try(func() {
		page.Timeout(15 * time.Second).MustElement(".global-nav__me").MustWaitVisible()
		loginSuccess = true
	})

	if !loginSuccess {
		// Method 2: Check for feed page elements
		err = rod.Try(func() {
			page.Timeout(10 * time.Second).MustElement(".feed-identity-module").MustWaitVisible()
			loginSuccess = true
		})
	}

	if !loginSuccess {
		// Method 3: Check URL change (should redirect from login page)
		currentURL := page.MustInfo().URL
		if currentURL != "https://www.linkedin.com/login" && currentURL != "https://www.linkedin.com/login/" {
			loginSuccess = true
		}
	}

	if !loginSuccess {
		// Method 4: Check for any LinkedIn navigation elements
		err = rod.Try(func() {
			page.Timeout(10 * time.Second).MustElement(".global-nav").MustWaitVisible()
			loginSuccess = true
		})
	}

	if !loginSuccess {
		// Get current page info for debugging
		currentURL := page.MustInfo().URL
		pageTitle := page.MustInfo().Title
		log.Printf("Login verification failed. Current URL: %s, Title: %s", currentURL, pageTitle)

		// Check if there's a CAPTCHA or additional verification
		captchaExists := rod.Try(func() {
			page.MustElement("#captcha-internal").MustWaitVisible()
		}) == nil

		if captchaExists {
			return fmt.Errorf("CAPTCHA detected - manual intervention required")
		}

		return fmt.Errorf("login verification failed - unable to confirm successful login")
	}

	log.Println("Login verification successful")
	return nil
}

func navigateToSavedJobs(page *rod.Page) error {
	// Direct navigation is more reliable than clicking through UI
	err := rod.Try(func() {
		page.Timeout(30 * time.Second).MustNavigate("https://www.linkedin.com/my-items/saved-jobs/").MustWaitLoad()
	})
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	log.Println("Navigated to saved jobs page")

	// Add debugging information
	currentURL := page.MustInfo().URL
	pageTitle := page.MustInfo().Title
	log.Printf("Debug - Current URL: %s, Title: %s", currentURL, pageTitle)

	// Check if we're redirected back to login - this means we need to handle the redirect
	if strings.Contains(currentURL, "/uas/login") || strings.Contains(pageTitle, "Login") {
		log.Println("Detected redirect to login page, waiting for automatic redirect...")
		
		// Wait for automatic redirect to complete
		for i := 0; i < 10; i++ {
			time.Sleep(3 * time.Second)
			currentURL = page.MustInfo().URL
			pageTitle = page.MustInfo().Title
			log.Printf("Redirect attempt %d - URL: %s, Title: %s", i+1, currentURL, pageTitle)
			
			// Check if we've been redirected to the saved jobs page
			if strings.Contains(currentURL, "/my-items/saved-jobs") || strings.Contains(pageTitle, "My Jobs") {
				log.Println("Successfully redirected to saved jobs page")
				break
			}
			
			// If still on login page after several attempts, try to navigate again
			if i == 4 {
				log.Println("Still on login page, trying direct navigation again...")
				err = rod.Try(func() {
					page.Timeout(15 * time.Second).MustNavigate("https://www.linkedin.com/my-items/saved-jobs/").MustWaitLoad()
				})
				if err != nil {
					log.Printf("Second navigation attempt failed: %v", err)
				}
			}
		}
		
		// Final check
		currentURL = page.MustInfo().URL
		pageTitle = page.MustInfo().Title
		if strings.Contains(currentURL, "/uas/login") || strings.Contains(pageTitle, "Login") {
			return fmt.Errorf("unable to navigate away from login page after multiple attempts")
		}
	}

	// Wait a bit for the page to fully load
	time.Sleep(3 * time.Second)

	// Try to find any recognizable LinkedIn elements first
	var pageLoaded bool
	
	// Check if we're still on LinkedIn
	err = rod.Try(func() {
		page.Timeout(10 * time.Second).MustElement("*[class*='linkedin']").MustWaitVisible()
		pageLoaded = true
	})
	if !pageLoaded {
		// Try alternative LinkedIn indicators
		err = rod.Try(func() {
			page.Timeout(5 * time.Second).MustElement("nav").MustWaitVisible()
			pageLoaded = true
		})
	}
	
	if !pageLoaded {
		log.Println("Warning: Page doesn't seem to be fully loaded or might not be LinkedIn")
	}

	// Wait for the saved jobs structure using more stable selectors
	err = rod.Try(func() {
		page.Timeout(20 * time.Second).MustElement(".workflow-results-container").MustWaitVisible()
	})
	if err != nil {
		// Try alternative containers that might indicate saved jobs page
		log.Printf("workflow-results-container not found, trying alternatives: %v", err)
		
		// Try to find any container that might hold job results
		err = rod.Try(func() {
			page.Timeout(10 * time.Second).MustElement("*[class*='workflow']").MustWaitVisible()
		})
		if err != nil {
			// Try to find any job-related container
			err = rod.Try(func() {
				page.Timeout(10 * time.Second).MustElement("*[class*='job']").MustWaitVisible()
			})
			if err != nil {
				// Log page source for debugging (first 1000 characters)
				pageHTML := page.MustHTML()
				if len(pageHTML) > 1000 {
					pageHTML = pageHTML[:1000] + "..."
				}
				log.Printf("Page HTML snippet: %s", pageHTML)
				return fmt.Errorf("no recognizable saved jobs container found: %w", err)
			}
		}
	}
	log.Println("Some form of jobs container found")

	// Wait for "My Jobs" header to ensure page is loaded
	err = rod.Try(func() {
		page.Timeout(15 * time.Second).MustElement("h1:contains('My Jobs')").MustWaitVisible()
	})
	if err != nil {
		log.Printf("Warning: 'My Jobs' header not found: %v", err)
	} else {
		log.Println("'My Jobs' header found")
	}

	// Add extra wait time for dynamic content to load
	log.Println("Waiting for dynamic content to load...")
	time.Sleep(5 * time.Second)

	// Try to find the job list with extended timeout and multiple attempts
	var jobListFound bool
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("Attempt %d to find job list", attempt)
		
		err = rod.Try(func() {
			page.Timeout(15 * time.Second).MustElement("ul[role='list']").MustWaitVisible()
			jobListFound = true
		})
		
		if jobListFound {
			log.Println("Job list found successfully")
			break
		}
		
		log.Printf("Attempt %d failed: %v", attempt, err)
		
		// Try scrolling to trigger content loading
		if attempt < 3 {
			log.Println("Scrolling to trigger content loading...")
			rod.Try(func() {
				page.Mouse.Scroll(0, 500, 1)
			})
			time.Sleep(3 * time.Second)
		}
	}

	if !jobListFound {
		// Final fallback: check if there are any saved jobs at all
		log.Println("Checking if there are any saved jobs...")
		
		// Look for "No saved jobs" message or similar
		noJobsFound := rod.Try(func() {
			page.Timeout(5 * time.Second).MustElement("*:contains('No saved jobs')")
		}) == nil
		
		if noJobsFound {
			return fmt.Errorf("no saved jobs found on the page")
		}
		
		// Log current page content for debugging
		currentURL := page.MustInfo().URL
		pageTitle := page.MustInfo().Title
		log.Printf("Debug info - Current URL: %s, Title: %s", currentURL, pageTitle)
		
		return fmt.Errorf("job list (ul[role='list']) not found after multiple attempts")
	}

	log.Println("Successfully reached saved jobs page")
	return nil
}

func processSavedJobs(page *rod.Page) error {
	// Scroll to load all jobs (simulates human behavior)
	scrollToLoadAllJobs(page)
	log.Println("Scrolled to load all jobs")

	// Get initial count of job cards using more robust selector
	var jobCards []*rod.Element
	err := rod.Try(func() {
		// Find the job list container and get all job items
		jobList := page.MustElement("ul[role='list']")
		jobCards = jobList.MustElements("li")
	})
	if err != nil {
		return fmt.Errorf("failed to find job cards: %w", err)
	}

	totalJobs := len(jobCards)
	log.Printf("Found %d saved jobs to process\n", totalJobs)
	log.Println("Processing saved jobs")

	// Process jobs one by one, refetching the list after each job
	for i := 0; i < totalJobs; i++ {
		log.Printf("Processing job %d of %d", i+1, totalJobs)

		// Refetch job cards to get fresh elements
		var currentJobCards []*rod.Element
		err := rod.Try(func() {
			jobList := page.MustElement("ul[role='list']")
			currentJobCards = jobList.MustElements("li")
		})
		if err != nil {
			log.Printf("Error refetching job cards: %v", err)
			break
		}

		// Check if we still have jobs to process
		if len(currentJobCards) == 0 {
			log.Println("No more job cards found, stopping processing")
			break
		}

		// Check if we have enough jobs left to process
		if i >= len(currentJobCards) {
			log.Printf("Reached end of available jobs (trying to process job %d but only %d jobs available)", i+1, len(currentJobCards))
			break
		}

		// Process the job at index i (this way we process different jobs each time)
		if err := processSingleJob(page, currentJobCards[i], i); err != nil {
			log.Printf("Error processing job %d: %v\n", i+1, err)
			continue
		}

		// Small delay between jobs
		time.Sleep(1 * time.Second)
	}
	log.Println("Processed saved jobs")
	return nil
}

func scrollToLoadAllJobs(page *rod.Page) {
	// Simulate human-like scrolling using Rod's built-in methods
	log.Printf("Scrolling down")
	for i := 0; i < 3; i++ {
		// Use Rod's scroll method instead of JavaScript eval
		err := rod.Try(func() {
			page.Mouse.Scroll(0, 800, 1)
			//page.MustScrollDown(800) // Scroll down 800 pixels
		})
		if err != nil {
			log.Printf("Scroll attempt %d failed: %v", i+1, err)
			// Fallback: try alternative scrolling method
			rod.Try(func() {
				page.MustEval(`() => { window.scrollBy(0, 800); }`)
			})
		}
		time.Sleep(time.Duration(1.5+float64(i)*0.5) * time.Second)
	}
}

func processSingleJob(page *rod.Page, job *rod.Element, index int) error {
	// Scroll to job with human-like delay
	job.MustScrollIntoView()
	time.Sleep(500 * time.Millisecond)

	// Extract job details using more robust selectors
	var title, company string
	var titleLink *rod.Element

	// Find the job title link - look for the actual job title, not company logo
	err := rod.Try(func() {
		// Look for links that contain job title text (not just company logos)
		jobLinks := job.MustElements("a[data-test-app-aware-link]")
		
		for _, link := range jobLinks {
			linkText := strings.TrimSpace(link.MustText())
			linkHref := link.MustAttribute("href")
			
			log.Printf("Debug - Found link with text: '%s', href: %s", linkText, *linkHref)
			
			// Skip company logo links (they typically have no text or just company name)
			// Job title links should have substantial text and point to /jobs/view/
			if linkText != "" && len(linkText) > 10 && strings.Contains(*linkHref, "/jobs/view/") {
				titleLink = link
				
				// Clean up the title text
				titleText := strings.TrimSpace(linkText)
				titleText = strings.ReplaceAll(titleText, ", Verified", "")
				titleText = strings.ReplaceAll(titleText, "Verified", "")
				titleText = strings.ReplaceAll(titleText, "<!---->", "")
				
				// Remove any trailing whitespace and newlines
				lines := strings.Split(titleText, "\n")
				if len(lines) > 0 {
					titleText = strings.TrimSpace(lines[0]) // Take first line only
				}
				
				title = strings.TrimSpace(titleText)
				log.Printf("Debug - Selected job title: '%s'", title)
				return
			}
		}
		
		// If no suitable link found, try to find title in the job card structure
		// Look for the job title in span elements within the job card
		titleSpans := job.MustElements("span")
		for _, span := range titleSpans {
			spanText := strings.TrimSpace(span.MustText())
			// Job titles are usually longer than company names and don't contain "Posted" or location info
			if len(spanText) > 15 && !strings.Contains(spanText, "Posted") && !strings.Contains(spanText, "Remote") && !strings.Contains(spanText, "ago") {
				title = spanText
				log.Printf("Debug - Found title in span: '%s'", title)
				
				// Still try to find the corresponding link for clicking
				for _, link := range jobLinks {
					if strings.Contains(*link.MustAttribute("href"), "/jobs/view/") {
						titleLink = link
						break
					}
				}
				return
			}
		}
		
		panic("No suitable job title found")
	})
	
	if err != nil {
		// Final fallback: try to find any job link and extract title later
		err = rod.Try(func() {
			titleLink = job.MustElement("a[href*='/jobs/view/']")
			title = "Title to be extracted from job page"
			log.Printf("Debug - Using fallback link, will extract title from job page")
		})
		if err != nil {
			title = "Title not found"
			log.Printf("Warning: Could not find job title or link: %v", err)
		}
	}
	log.Printf("Title found: %s", title)

	// Get company name using more stable approach
	err = rod.Try(func() {
		// Look for elements with t-black and t-normal classes (company name styling)
		companyElements := job.MustElements(".t-black.t-normal")
		// The company name is typically the first or second element with these classes
		for _, element := range companyElements {
			text := element.MustText()
			// Skip empty text and job titles (which might also have these classes)
			if text != "" && text != title {
				company = text
				return
			}
		}
		panic("No suitable company element found")
	})
	if err != nil {
		// Fallback: try to find company name in a more general way
		err = rod.Try(func() {
			// Look for div elements and find one that looks like a company name
			divElements := job.MustElements("div")
			for _, div := range divElements {
				text := div.MustText()
				// Company names are usually short, not empty, and not the job title
				if text != "" && text != title && len(text) < 100 && !strings.Contains(text, "Posted") {
					company = text
					return
				}
			}
			panic("Could not find company name in divs")
		})
		if err != nil {
			company = "Company name not found"
			log.Printf("Warning: Could not extract company name: %v", err)
		}
	}
	log.Printf("Company found: %s", company)

	log.Printf("\nProcessing job %d: %s at %s\n", index+1, title, company)

	// Log current URL for debugging
	currentURL := page.MustInfo().URL
	log.Printf("Current URL: %s", currentURL)

	// Open job details by clicking the job title link
	if titleLink != nil {
		err = rod.Try(func() {
			titleLink.MustClick()
		})
	} else {
		// Fallback: try to find and click any job link
		err = rod.Try(func() {
			jobLink := job.MustElement("a[data-test-app-aware-link]")
			jobLink.MustClick()
		})
	}
	if err != nil {
		return fmt.Errorf("failed to click job: %w", err)
	}

	// Wait for job details page to load with better detection
	log.Println("Waiting for job details page to load...")
	
	// Wait for URL change or job details elements to appear
	var jobDetailsLoaded bool
	for attempt := 1; attempt <= 10; attempt++ {
		time.Sleep(1 * time.Second)
		currentURL := page.MustInfo().URL
		
		log.Printf("Attempt %d - Current URL: %s", attempt, currentURL)
		
		// Check if we're on a job details page
		if strings.Contains(currentURL, "/jobs/view/") {
			log.Println("Successfully navigated to job details page")
			jobDetailsLoaded = true
			break
		}
		
		// Also check for job details elements
		err = rod.Try(func() {
			page.Timeout(2 * time.Second).MustElement("h1.t-24.t-bold.inline")
			jobDetailsLoaded = true
		})
		if jobDetailsLoaded {
			log.Println("Job details elements found")
			break
		}
		
		// If still on saved jobs page after 5 attempts, try clicking again
		if attempt == 5 && strings.Contains(currentURL, "/my-items/saved-jobs/") {
			log.Println("Still on saved jobs page, trying to click job link again...")
			if titleLink != nil {
				rod.Try(func() {
					titleLink.MustClick()
				})
			}
		}
	}
	
	if !jobDetailsLoaded {
		log.Println("Warning: Could not confirm job details page loaded, continuing anyway...")
	} else {
		log.Println("Job details page loaded successfully")
	}

	// If we couldn't extract the title from the job card, try to get it from the job details page
	if title == "Title to be extracted from job page" || title == "Title not found" || strings.Contains(title, "<div") {
		log.Println("Attempting to extract job title from job details page...")
		err = rod.Try(func() {
			// Look for the job title in the h1 element
			titleElement := page.MustElement("h1.t-24.t-bold.inline")
			extractedTitle := strings.TrimSpace(titleElement.MustText())
			if extractedTitle != "" {
				title = extractedTitle
				log.Printf("Successfully extracted title from job page: '%s'", title)
				return
			}
			panic("Title element found but empty")
		})
		if err != nil {
			// Fallback: try other possible title selectors
			err = rod.Try(func() {
				titleElement := page.MustElement("h1")
				extractedTitle := strings.TrimSpace(titleElement.MustText())
				if extractedTitle != "" && len(extractedTitle) > 5 {
					title = extractedTitle
					log.Printf("Successfully extracted title from fallback h1: '%s'", title)
					return
				}
				panic("No suitable h1 title found")
			})
			if err != nil {
				log.Printf("Warning: Could not extract title from job details page: %v", err)
				if title == "Title to be extracted from job page" {
					title = "Unknown Job Title"
				}
			}
		}
	}

	// Try to get job description from "About the job" section with timeout
	var jobDescription string
	log.Println("Extracting job description...")
	
	err = rod.Try(func() {
		// Look for the specific job description container with timeout
		descriptionElement := page.Timeout(5 * time.Second).MustElement(".jobs-box__html-content")
		jobDescription = descriptionElement.MustText()
	})
	if err == nil && jobDescription != "" {
		log.Printf("Job Description found (method 1)")
	} else {
		// Fallback 1: try the jobs description content class
		err = rod.Try(func() {
			descriptionElement := page.Timeout(5 * time.Second).MustElement(".jobs-description-content__text")
			jobDescription = descriptionElement.MustText()
		})
		if err == nil && jobDescription != "" {
			log.Printf("Job Description found (method 2)")
		} else {
			// Fallback 2: try to find the div after "About the job" header
			err = rod.Try(func() {
				aboutJobHeader := page.Timeout(3 * time.Second).MustElement("h2:contains('About the job')")
				if aboutJobHeader != nil {
					// Get the parent container and find the description div
					parentContainer := aboutJobHeader.MustParent()
					descriptionDiv := parentContainer.MustElement("div.mt4")
					jobDescription = descriptionDiv.MustText()
				}
			})
			if err == nil && jobDescription != "" {
				log.Printf("Job Description found (method 3)")
			} else {
				// Fallback 3: try the jobs description container
				err = rod.Try(func() {
					descriptionElement := page.Timeout(3 * time.Second).MustElement(".jobs-description__container")
					jobDescription = descriptionElement.MustText()
				})
				if err == nil && jobDescription != "" {
					log.Printf("Job Description found (method 4)")
				} else {
					log.Println("No job description found after trying all methods")
					jobDescription = "Description not available"
				}
			}
		}
	}
	
	// Truncate description if too long for logging
	if len(jobDescription) > 200 {
		log.Printf("Job Description: %s... (truncated)", jobDescription[:200])
	} else if jobDescription != "" {
		log.Printf("Job Description: %s", jobDescription)
	}

	// Return to saved jobs list
	log.Println("Navigating back to saved jobs list...")
	
	// Try navigation back first
	var backToSavedJobs bool
	err = rod.Try(func() {
		page.MustNavigateBack().MustWaitLoad()
		time.Sleep(2 * time.Second)
		currentURL := page.MustInfo().URL
		if strings.Contains(currentURL, "/my-items/saved-jobs/") {
			backToSavedJobs = true
		}
	})
	
	if !backToSavedJobs {
		log.Printf("Navigation back didn't work, trying direct navigation...")
		// Alternative: navigate directly to saved jobs URL
		err = rod.Try(func() {
			page.Timeout(30 * time.Second).MustNavigate("https://www.linkedin.com/my-items/saved-jobs/").MustWaitLoad()
			time.Sleep(2 * time.Second)
			currentURL := page.MustInfo().URL
			if strings.Contains(currentURL, "/my-items/saved-jobs/") {
				backToSavedJobs = true
			}
		})
		if err != nil {
			return fmt.Errorf("failed to navigate back to saved jobs: %w", err)
		}
	}

	// Wait for job list to be visible again with timeout
	log.Println("Waiting for job list to reload...")
	err = rod.Try(func() {
		page.Timeout(15 * time.Second).MustElement("ul[role='list']").MustWaitVisible()
	})
	if err != nil {
		log.Printf("Warning: Could not verify job list is visible after navigation back: %v", err)
		// Try scrolling to trigger content loading
		rod.Try(func() {
			page.Mouse.Scroll(0, 300, 1)
		})
		time.Sleep(2 * time.Second)
	} else {
		log.Println("Successfully returned to saved jobs list")
	}

	// Save job to database
	if err := sendJobCreationRequest(title, company, currentURL, jobDescription); err != nil {
		log.Printf("Warning: Failed to save job to database: %v", err)
	}

	return nil
}

func sendJobCreationRequest(title, company, link, description string) error {
	// Publish job creation request to NATS JetStream (JobService will handle DB insertion)
	err := sharedNats.PublishJobCreationRequest(title, company, link, description)
	if err != nil {
		return fmt.Errorf("failed to publish job creation request: %w", err)
	}

	log.Printf("Job creation request published for scraped job: %s at %s", title, company)
	return nil
}
