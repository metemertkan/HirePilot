package db

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hirepilot/shared/models"
)

// Custom error types
var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("record already exists")
)

// Using shared models package for Job, Prompt, and Feature types

var (
	db   *sql.DB
	once sync.Once
)

// InitDB initializes the database connection (singleton)
func InitDB() *sql.DB {
	once.Do(func() {
		dsn := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") +
			"@tcp(" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + ")/" + os.Getenv("DB_NAME")

		var err error
		for i := 0; i < 10; i++ { // Try 10 times
			db, err = sql.Open("mysql", dsn)
			if err == nil {
				err = db.Ping()
				if err == nil {
					break
				}
			}
			log.Printf("Waiting for DB to be ready (%d/10): %v", i+1, err)
			time.Sleep(3 * time.Second)
		}

		if err != nil {
			log.Fatalf("DB ping error: %v", err)
		}

		// Create all necessary tables
		createTables()

		log.Println("Database initialized successfully")
	})

	return db
}

// GetDB returns the existing database instance
func GetDB() *sql.DB {
	if db == nil {
		return InitDB()
	}
	return db
}

// Close closes the database connection
func Close() {
	if db != nil {
		db.Close()
	}
}

// createTables creates all necessary database tables
func createTables() {
	// Create jobs table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255),
			company VARCHAR(255),
			link VARCHAR(512),
			status ENUM('open','applied','closed') NOT NULL DEFAULT 'open',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			applied_at TIMESTAMP NULL,
			cvGenerated BOOLEAN DEFAULT FALSE,
			cv TEXT,
			description TEXT,
			score FLOAT NULL,
			cover_letter TEXT
		)
	`)
	if err != nil {
		log.Fatalf("Jobs table creation error: %v", err)
	}

	// Create prompts table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS prompts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			prompt TEXT,
			cvGenerationDefault BOOLEAN DEFAULT FALSE,
			scoreGenerationDefault BOOLEAN DEFAULT FALSE,
			coverGenerationDefault BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		log.Fatalf("Prompts table creation error: %v", err)
	}

	// Create features table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS features (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			value BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		log.Fatalf("Features table creation error: %v", err)
	}

	// Insert default features if not exists
	_, err = db.Exec(`
		INSERT IGNORE INTO features (id, name, value) VALUES
			(1,'cvGeneration', true),
			(2,'scoreGeneration', true)
	`)

	// Insert default prompts
	cvPrompt := `You are an expert Resume Builder Agent specialising in creating ATS-optimized, professionally written resumes. I will provide you with my full work history and a target job description. Based on this, your task is to generate tailored, two-page resume that is impactful, human-readable, and ATS-friendly.
Resume Goals & Guidelines:
1. ATS Compatibility: 
	* Use consistent section headings, spacing, and simple formatting (no tables or columns).
	* Align keywords naturally with the job description.
	* Maintain a clean layout for ATS parsing (e.g., use plain text with bullet points, avoid graphics or non-standard fonts).
2. Content Personalization:
	* Write in a natural, confident, and professional tone.
	* Avoid robotic or generic AI-generated phrases.
	* Ensure bullet points are achievement-focused, not task-oriented.
	* Replace vague language with clear, precise, and quantified statements.
	* Maintain a human voice, as if written by a thoughtful, experienced professional.
3. Bullet Point Formula (Mandatory)
	* Each bullet must follow this structure:
		Action Verb + Noun + Metric + [Strategy or Tool, optional] + Outcome
	* If a metric is missing, infer or suggest one where appropriate to strengthen the bullet's impact.
	Example:
	Improved CI/CD pipeline efficiency by 35% by integrating GitHub Actions and Terraform, enabling faster deployments with rollback safety.
4. Adaptation:
	* Align previous job titles to better reflect the target role (without misrepresentation).
	* Extract and use relevant keywords, tools, and skills from the job description.
	* Adjust project highlights, tech stack, and accomplishments to match the responsibilities and expectations of the target job.
5. Resume Structure:
	* Length: Two full pages, well-balanced.
	* Sections to Include (in order):
		1- Personal Details
		2- Professional Summary (1 paragraph, tailored to the job)
		3- Experience (reverse chronological, use all roles provided)
		4- Education
		5- Skills & Tools (aligned with job description)
		6- Certificates
6. Formatting Instructions
	* Use bold for section headers and job titles.
	* Section names should be in ALL CAPS or Title Case.
	* Use simple bullet points (*) for experience items.
	* Avoid any styling that could interfere with ATS parsing (e.g., tables, text boxes, images).
	* Ensure consistent spacing and visual balance across sections.

My Background:
I have 10+ years of professional experience since 2015, having worked with various software development methodologies such as Waterfall, Scrum, and Kanban. I have held several roles including Software Engineer, Senior Software Engineer, Lead Software Engineer, Solutions Architect, Senior Solutions Architect, Lead Solutions Architect, and Team Lead. I will provide my job history and projects under each role.
You will use this background along with the job description I provide to tailor the resume.

Personal Details (Use exactly as below);
Name: Mete Dosemeci
Location: Hoofddorp, Netherlands
Phone: +31618579439
Email: metemertkan@gmail.com

Education;
	* Master's Degree in Computer Engineering – Izmir Institute of Technology (2016 – 2019)
	* Bachelor's Degree in Computer Engineering – Pamukkale University (2010 – 2015)
		* Exchange: Kasetsart University, Thailand (2014 – 2015)
		* Exchange: Politechnika Slaska, Poland (2012 – 2013)

Certificates;
	* AWS Solutions Architect Professional
	* AWS Solutions Architect Associate

My previous work experience in reverse chronological order;

Company: Devoteam
Location: Amsterdam
Title: Lead Solutions Architect / Technical Team Lead / Lead Software Engineer (Choose title aligns well with the job description)
Time: Dec 2023 - Apr 2025
Projects and Works:
	I created a serverless platform for generating insights for a fintech company to where to invest in upcoming months. Company had all their documents stored in a dropbox. I used AWS powered LLMs and fine-tuned and tailored prompts to answer financial investing related questions. This platform helped client to find facts and important information inside a 1TB of documents. Improved their performance and speed. With reference based points they were able to shorten their analysis of the documents. Tech stack; ReactJS, python, AWS Bedrock, AWS RDS Aurora for MYSQL, lambda functions, step functions.
	I created a serverless event based platform for HR department for entering manual google forms. The form was for writing new job vacancies. The form had a lot of fields to fill according to companie's standard. HR person read out all the information for the job and the platform converts that into structured google forms input. Tech stack; ReactJS, Golang, AWS transcribe, S3, Lambda, step functions , api gateway, cloud front.
	Led and mentored Solutions architects, software engineers, including 1-1 meetings, performance reviews, future plannings.
	Stakeholders management with sales, senior management, developers, architects, clients and customers.
	Worked on company's AI vision with my AWS experties.
	Facilitated cross functional collaboration between product, engineering and business teams.
	Controlled projects timelines, budgets, team resources to drive timely, high quality results
	Aligned business goals to technical priorities by closely governing and prioritizing the backlog with stakeholders.  
       Generated a landing zone template for clients that can adapt and customise according to their need
       Worked on aws cdk in golang, javascript, python to modernise and refactor infra creating. I have deployed production scale projects with Kubernetes, terraform, 

Company: Grandvision
Location: Amsterdam
Title: Senior Solutions Architect / Technical Team Lead / Senior Software Engineer (Choose title aligns well with the job description)
Time: Jan 2023 - Dec 2023
Projects and Works:
	Worked as architect for glasses recommendation engine. The company had optical stores and in that stores there was a machine to make measurements for the eye and the face. After making the measurements we saved those data in or SAP CRM and started recommending glasses that fits for customers. There was augmented reality that we could put glasses into person's face virtually and show how that looked
	Worked in ecommerce project that has integration of POS system, SAP CRM. Pilot store opened in Portugal and fully integrated our enterprise system
	Stakeholder management, Enterprise architect, developer team, project managers, senior managers.
	Worked on cloud migration and modernization using AWS and Azure. Company was relying on old logging/monitoring system. Converted whole system into elastic stack cloud. Couple projects was working on on-premises virtual machines, lift and shift them into aws eks and azure aks
       Created modernized CI/CD pipeline with github actions and environment to deploy code to production with ease. That allowed project to adapt test driven development and rollback anytime.
       Working with enterprise architect and project managers to spreadhead the cloud journey of the company
       Mentored multiple developer teams, work on their career goals and grow as team

Company: Stablr
Location: Amsterdam
Title: Senior Solutions Architect / Technical Team Lead / Senior Software Engineer (Choose title aligns well with the job description)
Time: Feb 2021 - Dec 2022
Projects and Works:
       Designed, created and maintained cloud architecture for blockchain based deposit platform. With the help of the platform you can buy stable coin in eur and use it as deposit to earn interest. We were able to do it via bank's traditional deposit products. It was highly regulated by Mica, worked and created governance, security and observability policies. 3rd party integrations like revolut, for keeping track of fiat transactions. Guided technical and non technical stakeholders to work on company's vision. Established automatic deployment via CI/CD pipelines using github actions terraform cloud. Techstack; C#, golang, javascript, lambda, dynamodb, stepfunctions, eks, sqs, sns, eventbridge, apigateway, cloudfront, terraform.
      Mentored developers and analyst on their journey of the company
      Created data analysis platform for tracking user's actions through the website. Used bigdata tools like aws redshift to generate and maintain insights of behaviour. This resulted marketing team to generate personalised notifications and at the end increased customer retention %15.

Company: Valtech
Location: Utrecht
Title: Senior Software Engineer
Time: Mar 2020 - Feb 2021
Projects and Works:
     Worked on migrating globalfarmers.com of ABN Amro project to cloud using Azure. Converted from monolithic to Microservices for having better scaling capabilities
     Worked with traditional Sitecore in the beginning after converted that to headless cms.
Techstack; C#, javascript, azure

Company: EMAKINA
Location: Izmir
Title: Software Engineer
Time: Jun 2016 - Mar 2020
Projects and Works:
     Worked with Sitecore and Sitefinity to develop customer requirements
     Integrated multiple solutions together such as Salesforce Marketing Cloud and sitecore
     Extended CMS systems to have dynamic item management. Sitefinity has fixed types of input fields. I created a plugin for managing multiple list items with dynamic fields. That plugin has been used company wide and made content management way more dynamic
Techstack; C#, javascript, azure pipelines, octopus deploy, 

Company: Vargonen
Location: Izmir
Title: Software Engineer
Time: Jul 2015 - Jun 2016
Projects and Works:
    Created a solution to automise virtual machine creation. The company is a hosting company and when a new customer need a virtual machine they used vmware esx to create the vm, use 3rdparty system to arrange ip address, run the necessary scripts according to OS. I created event driven solution to shorten vm generation from 20mins to 4mins. Techstack; C#, Vmware API, mysql, docker 
    Integrated company's invoice creation system to government's e-invoice system. Used well known accounting app's api to manage company's financial systems.

Output Format:
Only output the resume do not include any other text. Imagine that generated text will be directly sent to recruiter. Avoid using page information as well like 'Page 1 of 2'`

	coverPrompt := `You are an expert Cover Letter Writing Agent skilled in creating engaging, confident, and ATS-optimized cover letters for technical leadership and engineering roles. I will provide you with my background and a target job description.

Your task is to generate a one-page, personalized cover letter that aligns closely with the job description, emphasizes my achievements, and reflects a tone that is professional, authentic, and human—not generic or overly formal.

Cover Letter Goals
1. Tone & Voice

Use a confident, warm, and professional tone — avoid stiff, overly formal, or robotic phrasing.
Write in a way that reflects a human applicant who understands the job and is excited about the opportunity.
Demonstrate personality while maintaining professionalism.
2. Structure & Formatting

Length: One page (3–5 paragraphs max).
Sections:
Introduction: Brief intro with the role applied for and a hook about why I'm a strong fit.
Body (1–2 paragraphs): Describe relevant qualifications, leadership experience, and major achievements that align with the job requirements. Use keywords and phrasing from the job description naturally.
Closing: Express interest in a conversation/interview. Reiterate value to the company. End with a friendly, professional sign-off.
3. Content Strategy

Customize the letter for the specific role and company.
Reference the company's mission, culture, or industry if relevant.
Emphasize value delivered in prior roles (leadership, impact, tech expertise, cross-functional collaboration).
Mention a few key technologies or methodologies from the job posting that match my background.
4. Adaptation & Alignment

Reflect elements from the job description in my own words.
Avoid regurgitating the resume — instead, highlight select experiences and why they matter.
Keep transitions smooth and paragraphs logically connected.
5. Formatting Rules

Avoid bullet points or tables — use standard paragraph form.
Do not include headers, tables, or excessive spacing.
Write in first-person but do not overuse "I" at the start of every sentence.
Use a professional sign-off like:
Kind regards,
Mete Dosemeci

Output Instructions
Generate a one-page cover letter using the above structure. Only output the cover letter. It should:

Be tailored to the specific job description provided
Showcase leadership, impact, and technical depth
Include specific technologies or outcomes where possible
Read like it was written by a real person with intent and confidence`

	scorePrompt := `Score the following CV based on the provided job description. The CV will start below '**Resume**'. Only return a numerical score between 0 and 100, where 0 is a poor match and 100 is a perfect match. Do not include any explanations, notes, or additional text.`

	_, err = db.Exec(`
		INSERT IGNORE INTO prompts (id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault) VALUES
			(1,'DefaultCvGenerator', ?, true, false, false),
			(2,'DefaultCoverGenerator', ?, false, false, true),
			(3,'DefaultScoreGenerator', ?, false, true, false)
	`, cvPrompt, coverPrompt, scorePrompt)
	if err != nil {
		log.Fatalf("Default prompts insertion error: %v", err)
	}

	log.Println("All database tables created successfully")
}

// Job-related database operations
func InsertJob(title, company, link, description string) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO jobs (title, company, link, status, cvGenerated, cv, description, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		title, company, link, "open", false, "", description, time.Now(),
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func GetJobByID(id int) (*models.Job, error) {
	var job models.Job
	var createdAtStr string
	var appliedAtStr sql.NullString
	var titleStr sql.NullString
	var companyStr sql.NullString
	var linkStr sql.NullString
	var cvStr sql.NullString
	var descriptionStr sql.NullString
	var coverLetterStr sql.NullString

	err := db.QueryRow(
		"SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at, cover_letter FROM jobs WHERE id = ?",
		id,
	).Scan(&job.Id, &titleStr, &companyStr, &linkStr, &job.Status, &job.CvGenerated, &cvStr, &descriptionStr, &job.Score, &createdAtStr, &appliedAtStr, &coverLetterStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Handle potentially NULL string fields
	if titleStr.Valid {
		job.Title = titleStr.String
	}
	if companyStr.Valid {
		job.Company = companyStr.String
	}
	if linkStr.Valid {
		job.Link = linkStr.String
	}
	if cvStr.Valid {
		job.Cv = cvStr.String
	}
	if descriptionStr.Valid {
		job.Description = descriptionStr.String
	}

	// Parse created_at
	if createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		job.CreatedAt = createdAt
	}

	// Parse applied_at if not null
	if appliedAtStr.Valid {
		if appliedAt, err := time.Parse("2006-01-02 15:04:05", appliedAtStr.String); err == nil {
			job.AppliedAt = &appliedAt
		}
	}

	// Handle cover_letter if not null
	if coverLetterStr.Valid {
		job.CoverLetter = coverLetterStr.String
	}

	return &job, nil
}

func GetJobsByStatus(status string) ([]models.Job, error) {
	var query string
	var rows *sql.Rows
	var err error

	if status != "" {
		query = "SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at, cover_letter FROM jobs WHERE status = ?"
		rows, err = db.Query(query, status)
	} else {
		query = "SELECT id, title, company, link, status, cvGenerated, cv, description, score, created_at, applied_at, cover_letter FROM jobs"
		rows, err = db.Query(query)
	}

	if err != nil {
		log.Printf("Database query error in GetJobsByStatus: %v", err)
		return nil, err
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var job models.Job
		var createdAtStr string
		var appliedAtStr sql.NullString
		var titleStr sql.NullString
		var companyStr sql.NullString
		var linkStr sql.NullString
		var cvStr sql.NullString
		var descriptionStr sql.NullString
		var coverLetterStr sql.NullString

		if err := rows.Scan(&job.Id, &titleStr, &companyStr, &linkStr, &job.Status, &job.CvGenerated, &cvStr, &descriptionStr, &job.Score, &createdAtStr, &appliedAtStr, &coverLetterStr); err != nil {
			log.Printf("Database scan error in GetJobsByStatus: %v", err)
			return nil, err
		}

		// Handle potentially NULL string fields
		if titleStr.Valid {
			job.Title = titleStr.String
		}
		if companyStr.Valid {
			job.Company = companyStr.String
		}
		if linkStr.Valid {
			job.Link = linkStr.String
		}
		if cvStr.Valid {
			job.Cv = cvStr.String
		}
		if descriptionStr.Valid {
			job.Description = descriptionStr.String
		}

		// Parse created_at
		if createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			job.CreatedAt = createdAt
		}

		// Parse applied_at if not null
		if appliedAtStr.Valid {
			if appliedAt, err := time.Parse("2006-01-02 15:04:05", appliedAtStr.String); err == nil {
				job.AppliedAt = &appliedAt
			}
		}

		// Handle cover_letter if not null
		if coverLetterStr.Valid {
			job.CoverLetter = coverLetterStr.String
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func UpdateJobStatus(id int, status string) error {
	if status == "applied" {
		_, err := db.Exec("UPDATE jobs SET applied_at = CURRENT_TIMESTAMP() WHERE id = ?", id)
		if err != nil {
			return err
		}
	}

	_, err := db.Exec("UPDATE jobs SET status = ? WHERE id = ?", status, id)
	return err
}

func GetAppliedJobsToday() (int, error) {
	var count int
	err := db.QueryRow("SELECT count(*) FROM jobs WHERE status = 'applied' AND DATE(applied_at) = CURDATE()").Scan(&count)
	return count, err
}

// Feature-related database operations
func GetFeatureValue(name string) (bool, error) {
	var value bool
	err := db.QueryRow("SELECT value FROM features WHERE name = ?", name).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrNotFound
		}
		return false, err
	}
	return value, nil
}

func UpdateFeatureValue(name string, value bool) error {
	_, err := db.Exec("UPDATE features SET value = ? WHERE name = ?", value, name)
	return err
}

func GetAllFeatures() ([]models.Feature, error) {
	rows, err := db.Query("SELECT id, name, value FROM features")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []models.Feature
	for rows.Next() {
		var feature models.Feature
		if err := rows.Scan(&feature.Id, &feature.Name, &feature.Value); err != nil {
			return nil, err
		}
		features = append(features, feature)
	}

	return features, nil
}

// Prompt-related database operations
func InsertPrompt(name, prompt string, cvDefault, scoreDefault bool) (int64, error) {
	return InsertPromptWithCover(name, prompt, cvDefault, scoreDefault, false)
}

func InsertPromptWithCover(name, prompt string, cvDefault, scoreDefault, coverDefault bool) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO prompts (name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault) VALUES (?, ?, ?, ?, ?)",
		name, prompt, cvDefault, scoreDefault, coverDefault,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func GetPromptByID(id int) (*models.Prompt, error) {
	var prompt models.Prompt
	err := db.QueryRow(
		"SELECT id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault FROM prompts WHERE id = ?",
		id,
	).Scan(&prompt.Id, &prompt.Name, &prompt.Prompt, &prompt.CvGenerationDefault, &prompt.ScoreGenerationDefault, &prompt.CoverGenerationDefault)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &prompt, nil
}

func GetAllPrompts() ([]models.Prompt, error) {
	rows, err := db.Query("SELECT id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault FROM prompts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		if err := rows.Scan(&prompt.Id, &prompt.Name, &prompt.Prompt, &prompt.CvGenerationDefault, &prompt.ScoreGenerationDefault, &prompt.CoverGenerationDefault); err != nil {
			return nil, err
		}
		prompts = append(prompts, prompt)
	}

	return prompts, nil
}

func UpdatePrompt(id int, name, promptText string, cvDefault, scoreDefault bool) error {
	_, err := db.Exec(
		"UPDATE prompts SET name = ?, prompt = ?, cvGenerationDefault = ?, scoreGenerationDefault = ? WHERE id = ?",
		name, promptText, cvDefault, scoreDefault, id,
	)
	return err
}

// UpdatePromptWithCover updates a prompt including cover generation default
func UpdatePromptWithCover(id int, name, promptText string, cvDefault, scoreDefault, coverDefault bool) error {
	_, err := db.Exec(
		"UPDATE prompts SET name = ?, prompt = ?, cvGenerationDefault = ?, scoreGenerationDefault = ?, coverGenerationDefault = ? WHERE id = ?",
		name, promptText, cvDefault, scoreDefault, coverDefault, id,
	)
	return err
}

// UpdateJobCV updates the CV and cvGenerated status for a job
func UpdateJobCV(jobID int, cv string) error {
	_, err := db.Exec("UPDATE jobs SET cv = ?, cvGenerated = TRUE WHERE id = ?", cv, jobID)
	return err
}

// UpdateJobCoverLetter updates the cover letter for a job
func UpdateJobCoverLetter(jobID int, coverLetter string) error {
	_, err := db.Exec("UPDATE jobs SET cover_letter = ? WHERE id = ?", coverLetter, jobID)
	return err
}

// GetDefaultCVPrompt gets the default CV generation prompt
func GetDefaultCVPrompt() (*models.Prompt, error) {
	var prompt models.Prompt
	err := db.QueryRow(
		"SELECT id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault FROM prompts WHERE cvGenerationDefault = TRUE LIMIT 1",
	).Scan(&prompt.Id, &prompt.Name, &prompt.Prompt, &prompt.CvGenerationDefault, &prompt.ScoreGenerationDefault, &prompt.CoverGenerationDefault)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &prompt, nil
}

// GetDefaultScorePrompt gets the default score generation prompt
func GetDefaultScorePrompt() (*models.Prompt, error) {
	var prompt models.Prompt
	err := db.QueryRow(
		"SELECT id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault FROM prompts WHERE scoreGenerationDefault = TRUE LIMIT 1",
	).Scan(&prompt.Id, &prompt.Name, &prompt.Prompt, &prompt.CvGenerationDefault, &prompt.ScoreGenerationDefault, &prompt.CoverGenerationDefault)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &prompt, nil
}

// GetDefaultCoverPrompt gets the default cover letter generation prompt
func GetDefaultCoverPrompt() (*models.Prompt, error) {
	var prompt models.Prompt
	err := db.QueryRow(
		"SELECT id, name, prompt, cvGenerationDefault, scoreGenerationDefault, coverGenerationDefault FROM prompts WHERE coverGenerationDefault = TRUE LIMIT 1",
	).Scan(&prompt.Id, &prompt.Name, &prompt.Prompt, &prompt.CvGenerationDefault, &prompt.ScoreGenerationDefault, &prompt.CoverGenerationDefault)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &prompt, nil
}

// UpdateJobScore updates the score for a job
func UpdateJobScore(jobID int, score string) error {
	_, err := db.Exec("UPDATE jobs SET score = ? WHERE id = ?", score, jobID)
	return err
}
