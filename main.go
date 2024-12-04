package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//go:embed default.json
var camoData []byte

const listHeight = 16

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type CamoItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Checked  bool   `json:"checked"`
}

type model struct {
	list      list.Model
	nested    map[string][]list.Item
	camoItems []CamoItem
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "enter":
			// Handle selection based on the current list type
			selected, ok := m.list.SelectedItem().(item)
			if ok {
				selection := string(selected)

				// If we're in the main categories list
				if items, exists := m.nested[selection]; exists {
					// Switch to the nested list of items for this category
					m.list.SetItems(items)
					m.list.Title = selection
				} else {
					// Otherwise, toggle the checked state for a specific item
					itemName := strings.TrimSuffix(selection, " ✅")

					for index, camo := range m.camoItems {
						if camo.Name == itemName {
							// Toggle the checked state
							m.camoItems[index].Checked = !camo.Checked
							break
						}
					}

					// Save the updated camo items to the writable JSON file
					if err := saveCamoItems("camos.json", m.camoItems); err != nil {
						fmt.Println("Error saving JSON:", err)
					}

					// Refresh the nested map and update the current category's list
					m.nested = groupItemsByCategory(m.camoItems)

					// Refresh the current category's list (if we're in a nested list)
					if items, exists := m.nested[m.list.Title]; exists {
						m.list.SetItems(items)
					}
				}
			}
			return m, nil

		case "backspace":
			// Go back to the main categories list
			categories := []list.Item{}
			for category := range m.nested {
				categories = append(categories, item(category))
			}

			m.list.SetItems(categories)
			m.list.Title = "BO6 Camo Tracker"
			return m, nil

		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}




func (m model) View() string {
	return "\n" + m.list.View()
}

func loadEmbeddedCamoItems() []CamoItem {
	var camoItems []CamoItem
	if err := json.Unmarshal(camoData, &camoItems); err != nil {
		fmt.Println("Error loading embedded JSON:", err)
		os.Exit(1)
	}
	return camoItems
}
func loadWritableCamoItems(filePath string) ([]CamoItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var camoItems []CamoItem
	if err := json.NewDecoder(file).Decode(&camoItems); err != nil {
		return nil, err
	}
	return camoItems, nil
}



func saveCamoItems(filePath string, camoItems []CamoItem) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(camoItems)
}

func groupItemsByCategory(camoItems []CamoItem) map[string][]list.Item {
	nested := make(map[string][]list.Item)
	for _, camo := range camoItems {
		displayName := camo.Name
		if camo.Checked {
			displayName += " ✅"
		}
		nested[camo.Category] = append(nested[camo.Category], item(displayName))
	}
	return nested
}

func main() {
	// Path to the writable JSON file
	writablePath := "camos.json"

	// Check if the writable JSON file exists
	if _, err := os.Stat(writablePath); os.IsNotExist(err) {
		// If it doesn't exist, create it with the embedded data
		if err := saveCamoItems(writablePath, loadEmbeddedCamoItems()); err != nil {
			fmt.Println("Error saving writable JSON:", err)
			os.Exit(1)
		}
	}

	// Load camo items from the writable JSON file
	camoItems, err := loadWritableCamoItems(writablePath)
	if err != nil {
		fmt.Println("Error loading writable JSON:", err)
		os.Exit(1)
	}

	// Group items by category
	nestedItems := groupItemsByCategory(camoItems)

	// Prepare category list
	categories := []list.Item{}
	for category := range nestedItems {
		categories = append(categories, item(category))
	}

	// Initialize list model
	const defaultWidth = 20
	l := list.New(categories, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "BO6 Camo Tracker"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Initialize program model
	m := model{list: l, nested: nestedItems, camoItems: camoItems}

	// Run the program
	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

