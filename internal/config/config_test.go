package config_test

import (
	"testing"

	"github.com/ja-he/dayplan/internal/config"
)

func TestParseConfigAugmentDefaults_ValidConfig(t *testing.T) {
	yamlData := []byte(`
stylesheet:

categories:
  - name: work::notrack
    color: '#dfbcbc'

  - name: work::customer-a
    color: '#ffabbc'
    priority: 25

  - name: work::customer-a::seminar
    color: '#ffabbc'
    priority: 25

  - name: work::customer-a::sow-draft
    color: '#ffabbc'
    priority: 25

  - name: work::customer-b
    color: '#ffabbc'
    priority: 25

  - name: work
    color: '#ffcccc'
    priority: 20
    goal:
      workweek:
        monday:    3h
        tuesday:   4h
        thursday:  5h
        friday:    8h
      except:
        - 2024-03-21
        - 2024-03-22
        - 2024-03-23
        - 2024-03-24
        - 2024-03-25
        - 2024-03-26
        - 2024-03-27
        - 2024-03-28
        - 2024-03-29
        - 2024-03-30
        - 2024-03-31
        - 2024-04-01
        - 2024-04-02
        - 2024-04-03
        - 2024-04-04
        - 2024-04-05
        - 2024-04-06
        - 2024-04-07
        - 2024-04-08
        - 2024-04-09
        - 2024-04-10
        - 2024-04-11
        - 2024-10-23
        - 2024-10-24
        - 2024-10-31

  - name: uni
    color: '#eedccc'
    priority: 10
`)

	defaultTheme := config.Dark
	configParsed, err := config.ParseConfigAugmentDefaults(defaultTheme, yamlData)
	if err != nil {
		t.Fatalf("ParseConfigAugmentDefaults() error = %v", err)
	}

	if len(configParsed.Categories) != 7 {
		t.Errorf("expected 7 categories, got %d", len(configParsed.Categories))
	}

	// Test work category
	workCat := findCategoryByName(configParsed.Categories, "work")
	if workCat == nil {
		t.Fatal("work category not found")
	}

	if workCat.Color != "#ffcccc" {
		t.Errorf("work color: expected #ffcccc, got %s", workCat.Color)
	}
	if workCat.Priority != 20 {
		t.Errorf("work priority: expected 20, got %d", workCat.Priority)
	}

	// Test work category's goal
	workGoal := workCat.Goal
	if workGoal.Workweek == nil {
		t.Fatal("work goal's workweek is nil")
	}
	ww := workGoal.Workweek
	if ww.Monday != "3h" {
		t.Errorf("work week Monday: expected 3h, got %s", ww.Monday)
	}
	if ww.Tuesday != "4h" {
		t.Errorf("work week Tuesday: expected 4h, got %s", ww.Tuesday)
	}
	if ww.Thursday != "5h" {
		t.Errorf("work week Thursday: expected 5h, got %s", ww.Thursday)
	}
	if ww.Friday != "8h" {
		t.Errorf("work week Friday: expected 8h, got %s", ww.Friday)
	}
	if ww.Wednesday != "" {
		t.Errorf("work week Wednesday: expected empty, got %s", ww.Wednesday)
	}
	if ww.Saturday != "" {
		t.Errorf("work week Saturday: expected empty, got %s", ww.Saturday)
	}
	if ww.Sunday != "" {
		t.Errorf("work week Sunday: expected empty, got %s", ww.Sunday)
	}

	// Check except dates
	except := workGoal.Except
	expectedExceptLen := 25
	if len(except) != expectedExceptLen {
		t.Errorf("expected %d except dates, got %d", expectedExceptLen, len(except))
	} else {
		if except[0] != "2024-03-21" {
			t.Errorf("except[0]: expected 2024-03-21, got %s", except[0])
		}
		if except[len(except)-1] != "2024-10-31" {
			t.Errorf("except[last]: expected 2024-10-31, got %s", except[len(except)-1])
		}
	}

	// Test uni category
	uniCat := findCategoryByName(configParsed.Categories, "uni")
	if uniCat == nil {
		t.Fatal("uni category not found")
	}
	if uniCat.Color != "#eedccc" {
		t.Errorf("uni color: expected #eedccc, got %s", uniCat.Color)
	}
	if uniCat.Priority != 10 {
		t.Errorf("uni priority: expected 10, got %d", uniCat.Priority)
	}
	if uniCat.Goal.Workweek != nil || len(uniCat.Goal.Except) != 0 {
		t.Error("uni category should have no goal")
	}
}

func findCategoryByName(cats []config.Category, name string) *config.Category {
	for i, cat := range cats {
		if cat.Name == name {
			return &cats[i]
		}
	}
	return nil
}
