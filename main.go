package main

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
)

// Constants
const (
	// ProvidersFilePath is the path to the file where providers are stored
	ProvidersFilePath = "providers.json"
)

// Provider represents an OTP provider/account
type Provider struct {
	Name   string
	Secret string
}

// generateTOTP creates a time-based one-time password
func generateTOTP(secret string) (string, error) {
	// Generate a TOTP code using the provided secret
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", err
	}
	return code, nil
}

// readSecretFromFile reads a secret from a file
// It can handle both plain text files and QR code images
func readSecretFromFile(filePath string) (string, error) {
	// Get the file extension
	ext := strings.ToLower(filepath.Ext(filePath))

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// If it's a text file, assume it contains the secret directly
	if ext == ".txt" || ext == ".key" {
		// Trim any whitespace
		secret := strings.TrimSpace(string(data))

		// Validate that the secret is a valid base32 string
		_, err := base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", fmt.Errorf("invalid base32 string in file: %v", err)
		}

		return secret, nil
	}

	// For other file types, try to parse as a QR code
	// In a real implementation, you would use a QR code parsing library
	// For now, we'll just return an error message
	return "", fmt.Errorf("QR code parsing is not supported in this version")
}

// saveProviders saves the providers to a JSON file
func saveProviders(providers []Provider, filePath string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Marshal the providers to JSON
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal providers: %v", err)
	}

	// Write the JSON to the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// loadProviders loads the providers from a JSON file
func loadProviders(filePath string) ([]Provider, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, return an empty slice
		return []Provider{}, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal the JSON
	var providers []Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal providers: %v", err)
	}

	return providers, nil
}

func main() {
	// Create a new application
	app := tview.NewApplication()

	// Load providers from file
	providers, err := loadProviders(ProvidersFilePath)
	if err != nil {
		// If there's an error loading providers, log it and start with an empty slice
		fmt.Printf("Error loading providers: %v\n", err)
		providers = []Provider{}
	}

	// Track the currently selected provider
	selectedProviderIndex := -1 // -1 indicates no provider is selected

	// Create a text view to display the OTP
	otpView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]Welcome to OTP Auth![white]\n\n" +
			"[red]No providers available[white]\n\n" +
			"[green]To get started:[white]\n" +
			"1. Enter a provider name in the form on the left\n" +
			"2. Enter a secret or load one from a file\n" +
			"3. Click 'Add Provider' to create your first provider\n\n" +
			"Your providers will be saved automatically and loaded when you restart the app.")

	// Variables to store current OTP information
	var currentOTP string
	var currentProvider string

	// Function to update only the countdown display
	updateCountdown := func() {
		if currentOTP == "" || currentProvider == "" {
			return
		}

		// Calculate remaining seconds
		remainingSeconds := 30 - time.Now().Second()%30
		// Escape any square brackets in the provider name and OTP
		escapedProvider := strings.ReplaceAll(strings.ReplaceAll(currentProvider, "[", "[["), "]", "]]")
		escapedOTP := strings.ReplaceAll(strings.ReplaceAll(currentOTP, "[", "[["), "]", "]]")
		otpView.SetText(fmt.Sprintf("[green]Provider: [yellow]%s[white]\n[green]Your OTP: [yellow]%s[white]\n\nValid for [red]%d[white] seconds", escapedProvider, escapedOTP, remainingSeconds))
	}

	// Function to generate and display OTP for the selected provider
	generateAndDisplayOTP := func() {
		if len(providers) == 0 || selectedProviderIndex == -1 {
			otpView.SetText("[yellow]Welcome to OTP Auth![white]\n\n" +
				"[red]No providers available or no provider selected[white]\n\n" +
				"[green]To get started:[white]\n" +
				"1. Enter a provider name in the form on the left\n" +
				"2. Enter a secret or load one from a file\n" +
				"3. Click 'Add Provider' to create your first provider\n\n" +
				"Your providers will be saved automatically and loaded when you restart the app.")
			currentOTP = ""
			currentProvider = ""
			return
		}

		// Get the selected provider
		provider := providers[selectedProviderIndex]

		// Generate a new OTP
		otp, err := generateTOTP(provider.Secret)
		if err != nil {
			// Escape any square brackets in the error message
			escapedError := strings.ReplaceAll(strings.ReplaceAll(err.Error(), "[", "[["), "]", "]]")
			otpView.SetText(fmt.Sprintf("[red]Error: %s[white]", escapedError))
			currentOTP = ""
			currentProvider = ""
			return
		}

		// Store current OTP information
		currentOTP = otp
		currentProvider = provider.Name

		// Update the display with countdown
		updateCountdown()
	}

	// Create a list to display providers
	providerList := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			selectedProviderIndex = index
			generateAndDisplayOTP()
		})

	// Add initial providers to the list
	for i, provider := range providers {
		providerList.AddItem(provider.Name, "", rune('1'+i), nil)
	}

	// Create a form to add new providers
	newProviderForm := tview.NewForm()
	newProviderForm.
		AddInputField("Provider Name", "", 20, nil, nil).
		AddInputField("Secret", "", 40, nil, nil).
		AddInputField("File Path", "", 40, nil, nil).
		AddButton("Browse...", func() {
			// Get the current file path from the form
			currentPath := newProviderForm.GetFormItem(2).(*tview.InputField).GetText()

			// If the path is empty, use the current directory
			if currentPath == "" {
				var err error
				currentPath, err = os.Getwd()
				if err != nil {
					// Escape any square brackets in the error message
					escapedError := strings.ReplaceAll(strings.ReplaceAll(err.Error(), "[", "[["), "]", "]]")
					otpView.SetText(fmt.Sprintf("[red]Error: %s[white]", escapedError))
					return
				}
			}

			// Show the file picker
			ShowFilePicker(app, currentPath, func(selectedPath string) {
				// Set the selected path in the file path input field
				newProviderForm.GetFormItem(2).(*tview.InputField).SetText(selectedPath)

				// Escape any square brackets in the file path
				escapedFilePath := strings.ReplaceAll(strings.ReplaceAll(selectedPath, "[", "[["), "]", "]]")
				otpView.SetText(fmt.Sprintf("[green]Selected file: [yellow]%s[white]", escapedFilePath))
			})
		}).
		AddButton("Read Secret from File", func() {
			// Get the file path from the form
			filePath := newProviderForm.GetFormItem(2).(*tview.InputField).GetText()
			if filePath == "" {
				otpView.SetText("[red]Error: File path cannot be empty[white]")
				return
			}

			// Read the secret from the file
			secret, err := readSecretFromFile(filePath)
			if err != nil {
				// Escape any square brackets in the error message
				escapedError := strings.ReplaceAll(strings.ReplaceAll(err.Error(), "[", "[["), "]", "]]")
				otpView.SetText(fmt.Sprintf("[red]Error: %s[white]", escapedError))
				return
			}

			// Set the secret in the form
			newProviderForm.GetFormItem(1).(*tview.InputField).SetText(secret)
			// Escape any square brackets in the file path
			escapedFilePath := strings.ReplaceAll(strings.ReplaceAll(filePath, "[", "[["), "]", "]]")
			otpView.SetText(fmt.Sprintf("[green]Secret read successfully from file: [yellow]%s[white]", escapedFilePath))
		}).
		AddButton("Add Provider", func() {
			// Get the provider name from the form
			providerName := newProviderForm.GetFormItem(0).(*tview.InputField).GetText()
			if providerName == "" {
				otpView.SetText("[red]Error: Provider name cannot be empty[white]")
				return
			}

			// Get the secret from the form
			secret := newProviderForm.GetFormItem(1).(*tview.InputField).GetText()
			if secret == "" {
				otpView.SetText("[red]Error: Secret cannot be empty[white]")
				return
			}

			// Validate that the secret is a valid base32 string
			_, err := base32.StdEncoding.DecodeString(secret)
			if err != nil {
				otpView.SetText("[red]Error: Secret must be a valid base32 string[white]")
				return
			}

			// Create a new provider with the provided secret
			newProvider := Provider{
				Name:   providerName,
				Secret: secret,
			}

			// Add the new provider to the list
			providers = append(providers, newProvider)
			providerList.AddItem(newProvider.Name, "", rune('1'+len(providers)-1), nil)

			// Save providers to file
			if err := saveProviders(providers, ProvidersFilePath); err != nil {
				// Escape any square brackets in the error message
				escapedError := strings.ReplaceAll(strings.ReplaceAll(err.Error(), "[", "[["), "]", "]]")
				otpView.SetText(fmt.Sprintf("[red]Error saving providers: %s[white]", escapedError))
				return
			}

			// Set the newly added provider as the selected one
			selectedProviderIndex = len(providers) - 1

			// Clear the form
			newProviderForm.GetFormItem(0).(*tview.InputField).SetText("")
			newProviderForm.GetFormItem(1).(*tview.InputField).SetText("")

			// Update the view and reset current OTP information
			currentOTP = ""
			currentProvider = ""
			// Escape any square brackets in the provider name and file path
			escapedProviderName := strings.ReplaceAll(strings.ReplaceAll(providerName, "[", "[["), "]", "]]")
			escapedFilePath := strings.ReplaceAll(strings.ReplaceAll(ProvidersFilePath, "[", "[["), "]", "]]")
			otpView.SetText(fmt.Sprintf("[green]Success![white]\n\n"+
				"Added new provider: [yellow]%s[white]\n\n"+
				"Your provider has been saved to [blue]%s[white]\n"+
				"It will be automatically loaded when you restart the app.",
				escapedProviderName, escapedFilePath))

			// Generate OTP for the newly added provider
			generateAndDisplayOTP()
		})

	// Create a button to quit the application
	quitButton := tview.NewButton("Quit").
		SetSelectedFunc(func() {
			app.Stop()
		})

	// Create a flex layout for buttons
	buttonFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(quitButton, 0, 2, true).
		AddItem(nil, 0, 1, false)

	// Create a flex layout for the provider management
	providerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("[yellow]Providers[white]"), 1, 0, false).
		AddItem(providerList, 0, 3, true).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("[yellow]Add New Provider[white]"), 1, 0, false).
		AddItem(newProviderForm, 0, 1, true)

	// Create a flex layout for the OTP display and generation
	otpFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("[yellow]OTP[white]"), 1, 0, false).
		AddItem(otpView, 0, 2, false).
		AddItem(buttonFlex, 3, 0, true)

	// Create a flex layout for the entire UI
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(providerFlex, 0, 1, true).
		AddItem(otpFlex, 0, 2, false)

	// Set up a timer to refresh the OTP every 15 seconds
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					if len(providers) > 0 && selectedProviderIndex >= 0 {
						generateAndDisplayOTP()
					}
				})
			}
		}
	}()

	// Set up a timer to update the countdown every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					updateCountdown()
				})
			}
		}
	}()

	// No need to generate OTP initially as there are no providers

	// Set the flex as the root of the application and start it
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
