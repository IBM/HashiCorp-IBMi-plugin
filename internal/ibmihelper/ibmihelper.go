package ibmihelper

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"strings"

	"github.com/hashicorp/vault/sdk/helper/template"
)

const (
	// IBM i valid password characters: A-Z, a-z, 0-9, and special chars: @ # $ _
	ibmiPasswordChars       = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789@#$_"
	defaultUserNameTemplate = `{{ printf "V%s%s" (.RoleName | truncate 4 | uppercase) (random 4) | truncate 10 | uppercase }}`
)

// PythonResult represents the result from Python script execution
type PythonResult struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Command string      `json:"command,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

// GenerateUsername generates a username based on the role name and template
func GenerateUsername(roleName, usernameTemplate string) (string, error) {
	if usernameTemplate == "" {
		usernameTemplate = defaultUserNameTemplate
	}

	up, err := template.NewTemplate(template.Template(usernameTemplate))
	if err != nil {
		return "", fmt.Errorf("unable to initialize username template: %w", err)
	}

	// Create metadata for template generation
	metadata := struct {
		RoleName string
	}{
		RoleName: roleName,
	}

	username, err := up.Generate(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate username: %w", err)
	}

	// Ensure username is max 10 characters and uppercase (IBM i requirement)
	username = strings.ToUpper(username)
	if len(username) > 10 {
		username = username[:10]
	}

	return username, nil
}

// GenerateIBMiPassword generates a random password using only IBM i-compatible characters
// IBM i valid characters: A-Z, a-z, 0-9, @ # $ _
func GenerateIBMiPassword(length int) (string, error) {
	if length < 10 {
		length = 10 // IBM i minimum password length
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(ibmiPasswordChars)))

	// Ensure at least one uppercase, one lowercase, one digit, and one special char
	result[0] = 'A' // Uppercase
	result[1] = 'a' // Lowercase
	result[2] = '1' // Digit
	result[3] = '@' // Special char

	// Fill the rest with random characters
	for i := 4; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		result[i] = ibmiPasswordChars[num.Int64()]
	}

	// Shuffle the password to randomize the required characters
	for i := len(result) - 1; i > 0; i-- {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		j := num.Int64()
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}

// ExecutePythonFunction executes a Python function from vault_crud.py
func ExecutePythonFunction(pythonExecutable, pythonScriptPath, host, port, username, password, functionName string, args ...interface{}) (*PythonResult, error) {
	// Build Python command
	pythonCode := fmt.Sprintf(`
import sys
import json
import os
sys.path.insert(0, '%s')

# Set environment variables for IBM i connection
os.environ['IBMi_HOST'] = '%s'
os.environ['IBMi_PORT'] = '%s'
os.environ['IBMi_USERNAME'] = '%s'
os.environ['IBMi_PASSWORD'] = '%s'

from vault_crud import %s

# Call the function
`, getPythonScriptDir(pythonScriptPath), host, port, username, password, functionName)

	// Add function call with arguments
	argStrings := make([]string, len(args))
	for idx, arg := range args {
		switch v := arg.(type) {
		case string:
			argStrings[idx] = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
		case int:
			argStrings[idx] = fmt.Sprintf("%d", v)
		default:
			argStrings[idx] = fmt.Sprintf("'%v'", v)
		}
	}

	pythonCode += fmt.Sprintf("result = %s(%s)\n", functionName, strings.Join(argStrings, ", "))
	pythonCode += "print(json.dumps(result))"

	// Execute Python command
	cmd := exec.Command(pythonExecutable, "-c", pythonCode)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python execution failed: %w, output: %s", err, string(output))
	}

	// The Python script may output log messages to stderr (which CombinedOutput captures)
	// followed by JSON on the last line. Extract only the last line for JSON parsing.
	outputStr := strings.TrimSpace(string(output))
	lines := strings.Split(outputStr, "\n")

	// Find the last non-empty line which should be the JSON
	var jsonLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			jsonLine = line
			break
		}
	}

	if jsonLine == "" {
		return nil, fmt.Errorf("no JSON output from Python script, output: %s", outputStr)
	}

	// Parse result from the JSON line
	var result PythonResult
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Python result: %w, json line: %s, full output: %s", err, jsonLine, outputStr)
	}

	return &result, nil
}

// getPythonScriptDir returns the directory containing the Python script
func getPythonScriptDir(pythonScriptPath string) string {
	lastSlash := strings.LastIndex(pythonScriptPath, "/")
	if lastSlash == -1 {
		return "."
	}
	return pythonScriptPath[:lastSlash]
}
