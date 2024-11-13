package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 16

var (
    titleStyle        = lipgloss.NewStyle().MarginLeft(2)
    itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
    selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
    paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
    helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
    quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
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

// CamoItem represents the structure of each item in the JSON file
type CamoItem struct {
    Name     string `json:"name"`
    Category string `json:"category"`
}

type model struct {
    list     list.Model
    choice   string
    quitting bool
    depth    int                    // Track current depth in nested list
    history  []list.Model           // Store previous list states for back navigation
    nested   map[string][]list.Item // Define nested items for each main item
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.list.SetWidth(msg.Width)
        return m, nil

    case tea.KeyMsg:
        switch keypress := msg.String(); keypress {
        case "q", "ctrl+c":
            m.quitting = true
            return m, tea.Quit

        case "enter", "right":
            i, ok := m.list.SelectedItem().(item)
            if ok {
                m.choice = string(i)

                // Check if there are nested items for the selected choice
                if nestedItems, exists := m.nested[m.choice]; exists {
                    // Save current list state to history for back navigation
                    m.history = append(m.history, m.list)
                    m.list = list.New(nestedItems, itemDelegate{}, m.list.Width(), listHeight)
                    m.depth++
                    m.list.Title = m.choice // Update title to show current selection context
                    return m, nil
                } else {
                    // No nested items, so quit
                    return m, tea.Quit
                }
            }

        case "backspace", "left":
            // Go back to the previous list level
            if m.depth > 0 {
                m.list = m.history[m.depth-1]
                m.history = m.history[:m.depth-1]
                m.depth--
            }
            return m, nil
        }
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m model) View() string {
    // if m.choice != "" && m.depth == 0 {
    //     return quitTextStyle.Render(fmt.Sprintf("Selected: %s", m.choice))
    // }
    if m.quitting {
        return quitTextStyle.Render("Exiting App... Goodbye!")
    }
    return "\n" + m.list.View()
}

// loadNestedItems loads data from camos.json and organizes it into a nested items map
func loadNestedItems(filePath string) (map[string][]list.Item, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var camoItems []CamoItem
    if err := json.NewDecoder(file).Decode(&camoItems); err != nil {
        return nil, err
    }

    nestedItems := make(map[string][]list.Item)
    for _, camo := range camoItems {
        nestedItems[camo.Category] = append(nestedItems[camo.Category], item(camo.Name))
    }

    return nestedItems, nil
}

func main() {
    // Load nested items from JSON
    nestedItems, err := loadNestedItems("camos.json")
    if err != nil {
        fmt.Println("Error loading camos.json:", err)
        os.Exit(1)
    }

    categories := []list.Item{}
    for category := range nestedItems {
        categories = append(categories, item(category))
    }

    const defaultWidth = 20

    l := list.New(categories, itemDelegate{}, defaultWidth, listHeight)
    l.Title = "BO6 Camo Tracker"
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(false)
    l.Styles.Title = titleStyle
    l.Styles.PaginationStyle = paginationStyle
    l.Styles.HelpStyle = helpStyle

    m := model{list: l, nested: nestedItems}

    if _, err := tea.NewProgram(&m).Run(); err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
}
