package cmd

import (
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// program defines the executable name.
const program = "sprint-update"

// sprintUpdateTemplate is a Discourse Markdown template used for generating
// the mid- and end of sprint updates.
const sprintUpdateTemplate string = `
**{{ .Title  }}**

**Worked on**

{{- range $status, $updates := .Issues }}

[details="{{ $status }}"]
{{- range $i, $item := $updates }}
* [{{ $item.Key }}]({{ $item.URL }}) - {{ $item.Summary }}:
{{- end }}
[/details]
{{- end }}

**Spillovers**

No spillovers in this sprint.

**Kudos**

* TODO

**Time off**

I did not plan any time off.
`

// jiraSearchQuery represents the JQL query used to search tickets of the
// assignee within the given sprint.
const jiraSearchQuery string = `assignee = currentUser() AND Sprint = "%s" AND status != Recurring`

var (
	configFile string
	version    string
	commit     string
	date       string
	rootCmd    = &cobra.Command{
		Use:     program,
		Short:   "Generate a sprint update.",
		Long:    "Generate a sprint update in Discourse-compatible Markdown format.",
		Example: fmt.Sprintf("%s --sprint SE.253 -e", program),
		Run:     runRootCmd,
	}
)

// jiraIssue represents an item in the sprint update.
type jiraIssue struct {
	Key     string
	Summary string
	URL     string
	Status  string
}

// newJiraIssue returns a new jiraIssue from the given jira.Issue.
func newJiraIssue(serverURL string, issue *jira.Issue) jiraIssue {
	summary := issue.Fields.Summary
	if len(summary) > 55 {
		summary = summary[:52] + "..."
	}

	return jiraIssue{
		Key:     issue.Key,
		Summary: summary,
		URL:     fmt.Sprintf("%s/browse/%s", serverURL, issue.Key),
		Status:  issue.Fields.Status.Name,
	}
}

// jiraIssues is the grouping of multiple jiraIssue by their status.
type jiraIssues map[string][]jiraIssue

// newJiraIssues returns jiraIssues grouped by issue status.
func newJiraIssues(serverURL string, issues []jira.Issue) jiraIssues {
	groupedIssues := make(jiraIssues)

	for _, issue := range issues {
		transformedIssue := newJiraIssue(serverURL, &issue)
		groupedIssues[issue.Fields.Status.Name] = append(groupedIssues[issue.Fields.Status.Name], transformedIssue)
	}

	return groupedIssues
}

// sprintUpdate is the actual sprint update used as the input for the sprint
// update template.
type sprintUpdate struct {
	Title  string
	Issues jiraIssues
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%s.yaml)", program))

	rootCmd.Flags().StringP("sprint", "s", "", "sprint name (ex: SE.253)")
	rootCmd.Flags().BoolP("end-of-sprint", "e", false, "indicate end of sprint update")

	rootCmd.Flags().StringP("jira-url", "", "", "jira server URL")
	rootCmd.Flags().StringP("jira-username", "", "", "jira user username")
	rootCmd.Flags().StringP("jira-password", "", "", "jira user password")

	rootCmd.Flags().BoolP("version", "", false, "show command version")
}

// initConfig initializes Cobra and Viper configuration.
func initConfig() {
	envPrefix := strings.ToUpper(program)

	if configFile != "" {
		viper.SetConfigName(configFile)
	} else {
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configDir, err := os.UserConfigDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(homeDir)
		viper.AddConfigPath(configDir)
		viper.SetConfigName("." + program)
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			cobra.CheckErr(err)
		}
	} else {
		fmt.Println("Using config file:", viper.ConfigFileUsed(), configFile)
	}

	// Bind flags to config value
	cobra.CheckErr(viper.BindPFlags(rootCmd.Flags()))
}

// printVersion prints the version number to stdout.
func printVersion() {
	if version == "" || len(commit) < 7 || date == "" {
		fmt.Println("dirty build")
	} else {
		fmt.Printf("%s version %s, commit %s (%s)\n", program, version, commit[:7], date)
	}
}

// newJiraClient returns creates a transport and returns a new jira.Client.
func newJiraClient(serverURL string, username string, password string) (*jira.Client, error) {
	transport := jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}

	return jira.NewClient(transport.Client(), serverURL)
}

// fetchIssues fetches issues from Jira returned as a result of the given JQL.
// The maximum number of issues returned by a search is limited to 1000 entries;
// to fetch every issue regardless the limit, we must do a basic pagination.
//
// Note: It is not realistic that anyone would hit the 1000 items limit, but be
// on the safe side.
func fetchIssues(client *jira.Client, jql string) ([]jira.Issue, error) {
	var issues []jira.Issue
	startAt := 0

	for {
		searchOpts := &jira.SearchOptions{
			StartAt:    startAt,
			MaxResults: 1000,
		}

		chunk, resp, err := client.Issue.Search(jql, searchOpts)
		if err != nil {
			return nil, err
		}

		total := resp.Total

		if total == 0 {
			break
		}

		// If no items were set yet, resize the slice since we know the number
		// of total issues at this point.
		if issues == nil {
			issues = make([]jira.Issue, 0, total)
		}

		issues = append(issues, chunk...)
		startAt = resp.StartAt + len(chunk)

		if startAt >= total {
			break
		}
	}

	return issues, nil
}

// runRootCmd is the root command run at command execution by Cobra.
func runRootCmd(_ *cobra.Command, _ []string) {
	var err error

	if viper.GetBool("version") {
		printVersion()
		os.Exit(0)
	}

	jiraServerURL := viper.GetString("jira-url")
	jiraUsername := viper.GetString("jira-username")
	jiraPassword := viper.GetString("jira-password")

	jiraClient, err := newJiraClient(jiraServerURL, jiraUsername, jiraPassword)
	cobra.CheckErr(err)

	sprintName := viper.GetString("sprint")
	rawIssues, err := fetchIssues(jiraClient, fmt.Sprintf(jiraSearchQuery, sprintName))
	cobra.CheckErr(err)

	issues := newJiraIssues(jiraServerURL, rawIssues)

	sprintUpdateType := "Mid-sprint"
	if viper.GetBool("end-of-sprint") {
		sprintUpdateType = "End of sprint"
	}

	descriptionTemplate := template.Must(template.New("description").Parse(sprintUpdateTemplate))
	err = descriptionTemplate.Execute(os.Stdout, &sprintUpdate{
		Title:  fmt.Sprintf("%s - %s", sprintName, sprintUpdateType),
		Issues: issues,
	})

	cobra.CheckErr(err)
}

func Execute(buildVersion string, buildCommit string, buildDate string) {
	version = buildVersion
	commit = buildCommit
	date = buildDate

	cobra.CheckErr(rootCmd.Execute())
}
