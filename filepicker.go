package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FileInfo represents a file or directory in the file picker
type FileInfo struct {
	Name     string
	Path     string
	IsDir    bool
	IsParent bool
}

// listDirectory returns a list of files and directories in the given directory
func listDirectory(dirPath string) ([]FileInfo, error) {
	// If dirPath is empty, use the current directory
	if dirPath == "" {
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %v", err)
		}
	}

	// Get absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Read directory contents
	files, err := ioutil.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	// Create result slice with parent directory entry
	result := []FileInfo{
		{
			Name:     "..",
			Path:     filepath.Dir(absPath),
			IsDir:    true,
			IsParent: true,
		},
	}

	// Add files and directories to the result
	for _, file := range files {
		// Skip hidden files
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		fileInfo := FileInfo{
			Name:  file.Name(),
			Path:  filepath.Join(absPath, file.Name()),
			IsDir: file.IsDir(),
		}
		result = append(result, fileInfo)
	}

	// Sort the result: directories first, then files, both alphabetically
	sort.Slice(result, func(i, j int) bool {
		// Parent directory always comes first
		if result[i].IsParent {
			return true
		}
		if result[j].IsParent {
			return false
		}

		// Directories before files
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}

		// Alphabetical order
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// ShowFilePicker displays a file picker dialog and returns the selected file path
func ShowFilePicker(app *tview.Application, currentPath string, callback func(string)) {
	// Store the original root primitive and input capture function
	originalRoot := app.GetFocus()
	originalInputCapture := app.GetInputCapture()

	// Function to restore the original application state
	restoreOriginal := func() {
		app.SetRoot(originalRoot, true)
		app.SetInputCapture(originalInputCapture)
	}

	// Create a modal for the file picker
	modal := tview.NewModal()
	modal.SetBorder(true).SetTitle(" File Picker ")

	// Create a list for files and directories
	fileList := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)

	// Function to update the file list with the contents of the current directory
	var updateFileList func(string)

	// Function to handle file selection
	handleSelection := func(index int) {
		// Get the selected file info
		mainText, secondaryText := fileList.GetItemText(index)
		path := secondaryText
		isDir := strings.HasPrefix(mainText, "📁")

		if isDir {
			// If it's a directory, navigate into it
			updateFileList(path)
		} else {
			// If it's a file, return the path and close the modal
			callback(path)
			// Restore the original application state
			restoreOriginal()
		}
	}

	updateFileList = func(dirPath string) {
		// Clear the list
		fileList.Clear()

		// Get files and directories
		files, err := listDirectory(dirPath)
		if err != nil {
			modal.SetText(fmt.Sprintf("Error: %s", err.Error())).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					// Restore the original application state
					restoreOriginal()
				})
			return
		}

		// Update the current path
		currentPath = dirPath

		// Add files and directories to the list
		for i, file := range files {
			// Display icon based on type
			var prefix string
			if file.IsParent {
				prefix = "📁 "
			} else if file.IsDir {
				prefix = "📁 "
			} else {
				prefix = "📄 "
			}

			// Add item to the list
			fileList.AddItem(prefix+file.Name, file.Path, rune('a'+i%26), nil)
		}

		// Update the modal title to show current directory
		modal.SetTitle(fmt.Sprintf(" File Picker - %s ", currentPath))
	}

	// Set up the file list selection handler
	fileList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		handleSelection(fileList.GetCurrentItem())
	})

	// Create a flex layout for the file picker
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fileList, 0, 1, true).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("[yellow]↑↓[white]: Navigate  [yellow]Enter[white]: Select  [yellow]Esc[white]: Cancel"), 1, 0, false)

	// Set up the modal
	modal.
		SetText("Select a file").
		AddButtons([]string{"Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			// Restore the original application state
			restoreOriginal()
		})

	// Create a pages component to hold the flex layout
	pages := tview.NewPages().
		AddPage("picker", flex, true, true)

	// Set up key handling for the file list
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Restore the original application state
			restoreOriginal()
			return nil
		case tcell.KeyEnter:
			// Handle Enter key to select the current item
			handleSelection(fileList.GetCurrentItem())
			return nil
		}
		return event
	})

	// Initial update of the file list
	updateFileList(currentPath)

	// Show the file picker
	app.SetRoot(pages, true)
}
