package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pdffer/pkg/pdf"
	"pdffer/pkg/utils"
)

// Application States
const (
	StateMenu = iota
	StateCreate_InputDir
	StateCreate_Output
	StateMerge_InputDir
	StateMerge_Output
	StateAppend_BasePDF
	StateAppend_ImageDir
	StateAppend_Output
	StateTreeSelection
	StateProcessing
	StateDone
)

type tickMsg time.Time
type successMsg string
type errMsg error

type model struct {
	state          int
	menuIndex      int
	textInput      textinput.Model
	filePicker     filepicker.Model
	treeModel      *TreeModel
	spinner        spinner.Model
	
	// Data storage
	inputDir       string
	outputFile     string
	selectedFiles  []string
	basePDF        string
	
	err            error
	statusMsg      string
}

func InitialModel() model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	fp := filepicker.New()
	fp.AllowedTypes = []string{".jpg", ".jpeg", ".png", ".pdf"}
	fp.CurrentDirectory, _ = os.Getwd()
	fp.Height = 10

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		state:     StateMenu,
		menuIndex: 0,
		textInput: ti,
		filePicker: fp,
		spinner:   s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.filePicker.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		if m.state == StateMenu {
			switch msg.Type {
			case tea.KeyUp:
				if m.menuIndex > 0 { m.menuIndex-- }
			case tea.KeyDown:
				if m.menuIndex < 2 { m.menuIndex++ }
			case tea.KeyEnter:
				m.textInput.Reset()
				m.filePicker.CurrentDirectory, _ = os.Getwd()
				m.filePicker.FileSelected = ""

				// Initialize Tree for directory selection immediately
				cwd, _ := os.Getwd()
				tm, err := NewTreeModel(cwd, 80, 20)
				if err != nil {
					m.err = err
					return m, nil
				}

				switch m.menuIndex {
				case 0: // Create from Images
					m.state = StateTreeSelection
					m.treeModel = tm
				case 1: // Merge PDFs
					m.state = StateTreeSelection
					m.treeModel = tm
				case 2: // Append Images
					// Still need to pick Base PDF first
					m.state = StateAppend_BasePDF
					m.filePicker.AllowedTypes = []string{".pdf"}
					return m, m.filePicker.Init()
				}
				return m, nil
			}
			return m, nil
		}

	case successMsg:
		m.state = StateDone
		m.statusMsg = string(msg)
		return m, tea.Quit
	
	case errMsg:
		m.err = msg
		m.state = StateDone 
		return m, tea.Quit
	
	case spinner.TickMsg:
		if m.state == StateProcessing {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.state == StateTreeSelection && m.treeModel != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "c" {
				// Confirm selection
				m.selectedFiles = m.treeModel.GetSelectedFiles()
				if len(m.selectedFiles) == 0 {
					// Maybe show error or just stay? Let's just stay for now or flash error
					return m, nil 
				}
				
				// Determine next state based on current flow
				// We don't store which flow explicitly, but we can infer or store it.
				// Wait, state machine logic:
				// If we came from Create_InputDir -> Go to Create_Output
				// If from Merge_InputDir -> Merge_Output
				// If from Append_ImageDir -> Append_Output
				
				// Problem: I overwrote m.state to StateTreeSelection. I lost where I came from.
				// Solution: I can store `nextState` in model or infer from menuIndex.
				
				switch m.menuIndex {
				case 0:
					m.state = StateCreate_Output
					m.textInput.SetValue("output.pdf")
				case 1:
					m.state = StateMerge_Output
					m.textInput.SetValue("merged_output.pdf")
				case 2:
					m.state = StateAppend_Output
					m.textInput.SetValue("final_output.pdf")
				}
				return m, textinput.Blink
			}
		}
		
		newTree, cmd := m.treeModel.Update(msg)
		m.treeModel = &newTree
		return m, cmd
	}

	switch m.state {
	case StateAppend_BasePDF:
		m.filePicker, cmd = m.filePicker.Update(msg)
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			return m.handleSelection(path)
		}
		return m, cmd
		
	case StateCreate_Output, StateMerge_Output, StateAppend_Output:
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEnter {
			return m.handleOutputInput()
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleSelection(path string) (tea.Model, tea.Cmd) {
	info, err := os.Stat(path)
	
	switch m.state {
	case StateAppend_BasePDF:
		// Selecting base PDF (single file)
		if err != nil || info.IsDir() {
			m.err = fmt.Errorf("please select a PDF file, not a directory")
			return m, nil
		}
		m.basePDF = path
		
		// Go to Tree Selection for images to append
		cwd, _ := os.Getwd()
		tm, err := NewTreeModel(cwd, 80, 20)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.treeModel = tm
		m.state = StateTreeSelection
		return m, nil
	}
	return m, nil
}

func (m model) handleOutputInput() (tea.Model, tea.Cmd) {
	val := m.textInput.Value()
	if val == "" {
		val = m.textInput.Placeholder
	}
	
	// Expand the output path!
	expanded, err := utils.ExpandPath(val)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.outputFile = expanded
	m.state = StateProcessing
	
	switch m.menuIndex {
	case 0:
		return m, tea.Batch(m.spinner.Tick, generatePDFCmd(m.selectedFiles, m.outputFile))
	case 1:
		return m, tea.Batch(m.spinner.Tick, mergePDFsCmd(m.selectedFiles, m.outputFile))
	case 2:
		return m, tea.Batch(m.spinner.Tick, appendImagesCmd(m.basePDF, m.selectedFiles, m.outputFile))
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress any key to exit.", m.err)
	}

	var s string
	switch m.state {
	case StateMenu:
		s = "Welcome to Pdffer! Choose an option:\n\n"
		choices := []string{"Create PDF from Images", "Merge PDFs from Directory", "Append Images to PDF"}
		for i, choice := range choices {
			cursor := " "
			if m.menuIndex == i { cursor = ">" }
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
		s += "\n(Arrows to navigate, Enter to select)"

	case StateAppend_BasePDF:
		s = "Select the Base PDF file:\n\n" + m.filePicker.View()

	case StateTreeSelection:
		if m.treeModel != nil {
			s = "Select files (Space to toggle, 'c' to confirm):\n\n" + m.treeModel.View()
		}

	case StateCreate_Output, StateMerge_Output, StateAppend_Output:
		s = "Enter output file path (supports ~):\n\n" + m.textInput.View() + "\n\n(Enter to confirm)"

	case StateProcessing:
		s = fmt.Sprintf("\n %s Processing... %s\n", m.spinner.View(), m.outputFile)
	case StateDone:
		s = fmt.Sprintf("\nDone! %s\n", m.statusMsg)
	}
	return s
}

func generatePDFCmd(files []string, output string) tea.Cmd {
	return func() tea.Msg {
		if err := pdf.GeneratePDF(files, output); err != nil { return errMsg(err) }
		return successMsg(fmt.Sprintf("Saved to %s", output))
	}
}

func mergePDFsCmd(files []string, output string) tea.Cmd {
	return func() tea.Msg {
		if err := pdf.MergePDFs(files, output); err != nil { return errMsg(err) }
		return successMsg(fmt.Sprintf("Merged to %s", output))
	}
}

func appendImagesCmd(basePDF string, images []string, output string) tea.Cmd {
	return func() tea.Msg {
		if err := pdf.AppendImagesToPDF(basePDF, images, output); err != nil { return errMsg(err) }
		return successMsg(fmt.Sprintf("Appended to %s", output))
	}
}
