package assetcap

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"unicode"
)

func isWeekend(t time.Time) bool {
	day := t.Weekday()
	return day == time.Saturday || day == time.Sunday
}

func CalculateWorkingHours(issueKey string, manualAdjustments map[string]float64, startTime, endTime time.Time) float64 {
	if ok := manualAdjustments[issueKey]; ok != 0 {
		return manualAdjustments[issueKey]
	}

	workHours := 0.0
	current := startTime
	for current.Before(endTime) {
		if !isWeekend(current) {
			if current.Hour() >= 9 && current.Hour() < 17 {
				workHours += 1.0
			}
		}
		current = current.Add(time.Hour)
	}
	return workHours
}

// JiraIssuesGetter is a function type for getting Jira issues
type JiraIssuesGetter func(url, authHeader string) ([]JiraIssue, error)

// GetJiraIssues is the default implementation for getting Jira issues
var GetJiraIssues JiraIssuesGetter = func(url, authHeader string) ([]JiraIssue, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jiraResponse JiraResponse
	err = json.Unmarshal(body, &jiraResponse)
	if err != nil {
		return nil, err
	}

	return jiraResponse.Issues, nil
}

func StructArrayToCSVOrdered(data []map[string]interface{}, headers []string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	for _, header := range headers {
		if data[0][header] == nil {
			return "", fmt.Errorf("header %s not found in data", header)
		}
	}

	if err := writer.Write(headers); err != nil {
		return "", err
	}

	for _, record := range data {
		var row []string
		for _, header := range headers {
			value, ok := record[header]
			if !ok {
				row = append(row, "")
				continue
			}
			row = append(row, fmt.Sprintf("%v", value))
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func Contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func CheckAndWrap(sprint string) string {
	for _, r := range sprint {
		if !unicode.IsDigit(r) {
			return "\"" + sprint + "\""
		}
	}
	return sprint
}
