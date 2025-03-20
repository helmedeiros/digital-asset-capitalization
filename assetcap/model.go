package assetcap

type Team struct {
	Team []string `json:"team"`
}

type T map[string]Team

type JiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary  string `json:"summary"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		StoryPoints *float64 `json:"customfield_13192"`
	} `json:"fields"`
	Changelog struct {
		Histories []struct {
			Created string `json:"created"`
			Items   []struct {
				Field      string `json:"field"`
				FromString string `json:"fromString"`
				ToString   string `json:"toString"`
			} `json:"items"`
		} `json:"histories"`
	} `json:"changelog"`
}

type JiraResponse struct {
	Issues []JiraIssue `json:"issues"`
}
