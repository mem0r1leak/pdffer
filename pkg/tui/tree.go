package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileNode struct {
	Path     string
	Name     string
	IsDir    bool
	Selected bool
	Open     bool
	Depth    int
	Parent   *FileNode
	Children []*FileNode
}

type TreeModel struct {
	Root         *FileNode
	Cursor       int
	VisibleNodes []*FileNode
	Width        int
	Height       int
	Offset       int

	SelectedStyle lipgloss.Style
	CursorStyle   lipgloss.Style
	DirStyle      lipgloss.Style
	FileStyle     lipgloss.Style
}

func NewTreeModel(rootPath string, width, height int) (*TreeModel, error) {
	rootNode, err := buildNode(rootPath, nil, 0)
	if err != nil {
		return nil, err
	}
	rootNode.Open = true
	// Load initial children
	loadChildren(rootNode)

	m := &TreeModel{
		Root:          rootNode,
		Width:         width,
		Height:        height,
		SelectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		CursorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		DirStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		FileStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
	}
	m.UpdateVisibleNodes()
	return m, nil
}

func buildNode(path string, parent *FileNode, depth int) (*FileNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &FileNode{
		Path:   path,
		Name:   filepath.Base(path),
		IsDir:  info.IsDir(),
		Depth:  depth,
		Parent: parent,
	}, nil
}

func loadChildren(node *FileNode) error {
	if !node.IsDir || len(node.Children) > 0 {
		return nil
	}

	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		childPath := filepath.Join(node.Path, entry.Name())
		child, err := buildNode(childPath, node, node.Depth+1)
		if err == nil {
			node.Children = append(node.Children, child)
		}
	}
	return nil
}

func (m *TreeModel) ChangeRoot(newRoot string) error {
	rootNode, err := buildNode(newRoot, nil, 0)
	if err != nil {
		return err
	}
	rootNode.Open = true
	loadChildren(rootNode)
	m.Root = rootNode
	m.Cursor = 0
	m.Offset = 0
	m.UpdateVisibleNodes()
	return nil
}

func (m *TreeModel) UpdateVisibleNodes() {
	m.VisibleNodes = []*FileNode{}

	// Add ".." entry if not at root
	if m.Root.Path != "/" {
		parentPath := filepath.Dir(m.Root.Path)
		// Ensure we don't get stuck at root if path is something like "/home" -> "/"
		// filepath.Dir("/home") = "/"

		dotDot := &FileNode{
			Path:  parentPath,
			Name:  "..",
			IsDir: true,
			Depth: 0,
		}
		m.VisibleNodes = append(m.VisibleNodes, dotDot)
	}

	var traverse func(*FileNode)
	traverse = func(n *FileNode) {
		m.VisibleNodes = append(m.VisibleNodes, n)
		if n.IsDir && n.Open {
			for _, child := range n.Children {
				traverse(child)
			}
		}
	}
	if m.Root != nil {
		traverse(m.Root)
	}

	if m.Cursor >= len(m.VisibleNodes) {
		m.Cursor = len(m.VisibleNodes) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

func (m *TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				if m.Cursor < m.Offset {
					m.Offset = m.Cursor
				}
			}
		case "down", "j":
			if m.Cursor < len(m.VisibleNodes)-1 {
				m.Cursor++
				if m.Cursor >= m.Offset+m.Height {
					m.Offset = m.Cursor - m.Height + 1
				}
			}
		case "right", "l":
			node := m.VisibleNodes[m.Cursor]
			if node.Name == ".." {
				m.ChangeRoot(node.Path)
				return *m, nil
			}
			if node.IsDir {
				if !node.Open {
					node.Open = true
					loadChildren(node) // Lazy load
					m.UpdateVisibleNodes()
				}
			}
		case "left", "h":
			node := m.VisibleNodes[m.Cursor]
			if node.Name == ".." {
				// Already at top, maybe do nothing or go up again?
				return *m, nil
			}
			if node.IsDir && node.Open {
				node.Open = false
				m.UpdateVisibleNodes()
			} else if node.Parent != nil {
				// Jump to parent
				for i, n := range m.VisibleNodes {
					if n == node.Parent {
						m.Cursor = i
						if m.Cursor < m.Offset {
							m.Offset = m.Cursor
						}
						break
					}
				}
			} else {
				// At root of current view?
				// If checking parent of root node, we might want to go up
				parent := filepath.Dir(m.Root.Path)
				if parent != m.Root.Path {
					m.ChangeRoot(parent)
				}
			}
		case "enter":
			node := m.VisibleNodes[m.Cursor]
			if node.Name == ".." {
				m.ChangeRoot(node.Path)
				return *m, nil
			}
			if node.IsDir {
				node.Open = !node.Open
				if node.Open {
					loadChildren(node)
				}
				m.UpdateVisibleNodes()
			}
		case " ": // Space: Toggle selection
			node := m.VisibleNodes[m.Cursor]
			if node.Name == ".." {
				return *m, nil
			} // Ignore selection on ..
			newState := !node.Selected
			m.toggleSelection(node, newState)

		// Navigation Keys
		case "backspace":
			parent := filepath.Dir(m.Root.Path)
			if parent != m.Root.Path {
				m.ChangeRoot(parent)
			}
		case "/":
			m.ChangeRoot("/")
		case "~":
			home, _ := os.UserHomeDir()
			if home != "" {
				m.ChangeRoot(home)
			}
		}
	}
	return *m, nil
}

func (m *TreeModel) toggleSelection(node *FileNode, state bool) {
	node.Selected = state
	// Propagate down
	if node.IsDir {
		for _, child := range node.Children {
			m.toggleSelection(child, state)
		}
	}
}

func (m *TreeModel) GetSelectedFiles() []string {
	var files []string
	var traverse func(*FileNode)
	traverse = func(n *FileNode) {
		if !n.IsDir && n.Selected {
			files = append(files, n.Path)
		}
		for _, child := range n.Children {
			traverse(child)
		}
	}
	if m.Root != nil {
		traverse(m.Root)
	}
	return files
}

func (m *TreeModel) View() string {
	s := fmt.Sprintf("Current Directory: %s\n", m.Root.Path)
	s += "(Navigation: Arrows, Enter, Backspace=Up, /=Root, ~=Home)\n"
	s += "(Selection: Space=Toggle, c=Confirm)\n\n"

	if len(m.VisibleNodes) == 0 {
		return s + "Empty directory or loading..."
	}

	lines := []string{}
	end := m.Offset + m.Height
	if end > len(m.VisibleNodes) {
		end = len(m.VisibleNodes)
	}

	for i := m.Offset; i < end; i++ {
		node := m.VisibleNodes[i]

		// Indent
		indent := strings.Repeat("  ", node.Depth)

		// Checkbox
		check := "[ ]"
		if node.Selected {
			check = "[x]"
		}

		// Icon

		icon := "📄"

		if node.Name == ".." {

			icon = "⤴ " // Up arrow

		} else if node.IsDir {

			if node.Open {

				icon = "📂"

			} else {

				icon = "📁"

			}

		}

		// Line construction

		cursor := " "

		if i == m.Cursor {

			cursor = ">"

		}

		line := ""

		if node.Name == ".." {

			line = fmt.Sprintf("%s %s%s %s", cursor, indent, icon, "[Go Up]")

		} else {

			line = fmt.Sprintf("%s %s%s %s %s", cursor, indent, check, icon, node.Name)

		}

		// Style

		if i == m.Cursor {

			line = m.CursorStyle.Render(line)

		} else {

			if node.Name == ".." {

				// Different style for ..

				line = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(line) // Grey

			} else if node.Selected {

				line = m.SelectedStyle.Render(line)

			} else if node.IsDir {

				line = m.DirStyle.Render(line)

			} else {

				line = m.FileStyle.Render(line)

			}

		}

		lines = append(lines, line)

	}

	return strings.Join(lines, "\n")

}
