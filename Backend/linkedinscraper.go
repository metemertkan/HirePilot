package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

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

	// // Mimic human behavior better
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

	// Wait for the new LinkedIn saved jobs structure
	err = rod.Try(func() {
		page.Timeout(20 * time.Second).MustElement(".workflow-results-container").MustWaitVisible()
	})
	if err != nil {
		// Try alternative selectors for saved jobs
		err = rod.Try(func() {
			page.Timeout(10 * time.Second).MustElement("ul.cnDmzjuEppehumrakqhTozbFKDNlvJvTAmiEQ").MustWaitVisible()
		})
		if err != nil {
			err = rod.Try(func() {
				page.Timeout(10 * time.Second).MustElement("h1:contains('My Jobs')").MustWaitVisible()
			})
			if err != nil {
				return fmt.Errorf("saved jobs container not found: %w", err)
			}
		}
	}
	log.Println("Saved jobs container found")
	log.Println("Successfully reached saved jobs page")
	return nil
}

func processSavedJobs(page *rod.Page) error {
	// Scroll to load all jobs (simulates human behavior)
	scrollToLoadAllJobs(page)
	log.Println("Scrolled to load all jobs")

	// Get initial count of job cards
	jobCards := page.MustElements("li.OIwPNvoxrJMIabHpwqHTqaJnAoMtmmlKmk")
	totalJobs := len(jobCards)
	log.Printf("Found %d saved jobs to process\n", totalJobs)
	log.Println("Processing saved jobs")

	// Process jobs one by one, refetching the list after each job
	for i := 0; i < totalJobs; i++ {
		log.Printf("Processing job %d of %d", i+1, totalJobs)

		// Refetch job cards to get fresh elements
		currentJobCards := page.MustElements("li.OIwPNvoxrJMIabHpwqHTqaJnAoMtmmlKmk")

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

	// Extract job details using the new LinkedIn structure
	var title, company string

	// Get job title from the new structure
	err := rod.Try(func() {
		titleElement := job.MustElement("a.uZVJLMgibRwTOKwARSdRvZWHssCpjVhkbllURjpR")
		title = titleElement.MustText()
	})
	if err != nil {
		// Try alternative title selector
		err = rod.Try(func() {
			titleElement := job.MustElement("a[data-test-app-aware-link]")
			title = titleElement.MustText()
		})
		if err != nil {
			title = "Title not found"
			log.Printf("Warning: Could not extract job title: %v", err)
		}
	}
	log.Printf("Title found: %s", title)

	// Get company name from the new structure - try multiple selectors
	err = rod.Try(func() {
		// Try the first div with company info (should be the company name)
		companyElements := job.MustElements(".IxdoWhWcQbSijYabWdEPSEvEYgHpeXPIfEt-14.t-black.t-normal")
		if len(companyElements) > 0 {
			company = companyElements[0].MustText()
			return
		}
		panic("No company elements found")
	})
	if err != nil {
		// Fallback: try a more general selector
		err = rod.Try(func() {
			companyElement := job.MustElement(".t-black.t-normal")
			company = companyElement.MustText()
		})
		if err != nil {
			// Last fallback: try to find any text that looks like a company
			err = rod.Try(func() {
				// Look for the div that contains company info (usually after job title)
				jobInfoDiv := job.MustElement(".hJaDnwBIZFOsokjePrmkFJdgbWftEKbLYoFNnmmw")
				companyElements := jobInfoDiv.MustElements("div")
				if len(companyElements) >= 2 {
					// Usually the second div contains company name
					company = companyElements[1].MustText()
					return
				}
				panic("Could not find company name")
			})
			if err != nil {
				company = "Company name not found"
				log.Printf("Warning: Could not extract company name: %v", err)
			}
		}
	}
	log.Println("Company found")

	log.Printf("\nProcessing job %d: %s at %s\n", index+1, title, company)

	// Log current URL for debugging
	currentURL := page.MustInfo().URL
	log.Printf("Current URL: %s", currentURL)

	// Open job details by clicking the job title link
	err = rod.Try(func() {
		titleLink := job.MustElement("a.uZVJLMgibRwTOKwARSdRvZWHssCpjVhkbllURjpR")
		titleLink.MustClick()
	})
	if err != nil {
		return fmt.Errorf("failed to click job: %w", err)
	}

	// Wait for job details page to load
	time.Sleep(3 * time.Second)
	log.Println("Job details page loaded")

	// Try to get job description from "About the job" section
	var jobDescription string
	err = rod.Try(func() {
		// Look for the specific job description container
		descriptionElement := page.MustElement(".jobs-box__html-content")
		jobDescription = descriptionElement.MustText()
	})
	if err == nil && jobDescription != "" {
		log.Printf("Job Description:\n%s\n", jobDescription)
	} else {
		// Fallback 1: try the jobs description content class
		err = rod.Try(func() {
			descriptionElement := page.MustElement(".jobs-description-content__text")
			jobDescription = descriptionElement.MustText()
		})
		if err == nil && jobDescription != "" {
			log.Printf("Job Description (fallback 1):\n%s\n", jobDescription)
		} else {
			// Fallback 2: try to find the div after "About the job" header
			err = rod.Try(func() {
				aboutJobHeader := page.MustElement("h2:contains('About the job')")
				if aboutJobHeader != nil {
					// Get the parent container and find the description div
					parentContainer := aboutJobHeader.MustParent()
					descriptionDiv := parentContainer.MustElement("div.mt4")
					jobDescription = descriptionDiv.MustText()
				}
			})
			if err == nil && jobDescription != "" {
				log.Printf("Job Description (fallback 2):\n%s\n", jobDescription)
			} else {
				// Fallback 3: try the jobs description container
				err = rod.Try(func() {
					descriptionElement := page.MustElement(".jobs-description__container")
					jobDescription = descriptionElement.MustText()
				})
				if err == nil && jobDescription != "" {
					log.Printf("Job Description (fallback 3):\n%s\n", jobDescription)
				} else {
					log.Println("No job description found")
				}
			}
		}
	}

	// Return to saved jobs list
	log.Println("Navigating back to saved jobs list...")
	err = rod.Try(func() {
		page.MustNavigateBack().MustWaitLoad()
	})
	if err != nil {
		log.Printf("Navigation back failed, trying alternative method: %v", err)
		// Alternative: navigate directly to saved jobs URL
		err = rod.Try(func() {
			page.Timeout(30 * time.Second).MustNavigate("https://www.linkedin.com/my-items/saved-jobs/").MustWaitLoad()
		})
		if err != nil {
			return fmt.Errorf("failed to navigate back to saved jobs: %w", err)
		}
	}

	// Wait for list to reload and verify we're back on the saved jobs page
	time.Sleep(3 * time.Second)

	// Wait for job list to be visible again
	err = rod.Try(func() {
		page.Timeout(10 * time.Second).MustElement("ul.cnDmzjuEppehumrakqhTozbFKDNlvJvTAmiEQ").MustWaitVisible()
	})
	if err != nil {
		log.Printf("Warning: Could not verify job list is visible after navigation back: %v", err)
	} else {
		log.Println("Successfully returned to saved jobs list")
	}

	// Save job to database
	if err := saveJobToDatabase(title, company, currentURL, jobDescription); err != nil {
		log.Printf("Warning: Failed to save job to database: %v", err)
	}

	return nil
}

// saveJobToDatabase saves a scraped job to the database and publishes a message
func saveJobToDatabase(title, company, link, description string) error {
	// Create job struct
	job := Job{
		Title:       title,
		Company:     company,
		Link:        link,
		Status:      "open", // Default status for scraped jobs
		CvGenerated: false,
		Description: description,
	}

	// Insert job into database
	result, err := db.Exec(
		"INSERT INTO jobs (title, company, link, status, cvGenerated, cv, description) VALUES (?, ?, ?, ?, ?, ?, ?)",
		job.Title, job.Company, job.Link, job.Status, job.CvGenerated, job.Cv, job.Description,
	)
	if err != nil {
		return fmt.Errorf("DB insert error: %w", err)
	}

	// Get the inserted job ID
	id, err := result.LastInsertId()
	if err == nil {
		job.Id = int(id)
		log.Printf("Job saved to database with ID: %d", job.Id)
	}

	// Check if CV generation feature is enabled
	var cvGenerationEnabled bool
	err = db.QueryRow("SELECT value FROM features WHERE name = 'cvGeneration'").
		Scan(&cvGenerationEnabled)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Warning: Could not check CV generation feature: %v", err)
		return nil // Don't fail the whole process for this
	}

	// Publish job message if CV generation is enabled
	if cvGenerationEnabled {
		if err := publishJobMessage(job); err != nil {
			log.Printf("Warning: Failed to publish job message: %v", err)
		} else {
			log.Printf("Job message published for CV generation")
		}
	}

	return nil
}
