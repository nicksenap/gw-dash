package grove

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ZellijAvailable checks if we're running inside Zellij.
func ZellijAvailable() bool {
	return os.Getenv("ZELLIJ_SESSION_NAME") != ""
}

// ZellijListTabNames returns all tab names in the current Zellij session.
func ZellijListTabNames() []string {
	cmd := exec.Command("zellij", "action", "query-tab-names")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var tabs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tabs = append(tabs, line)
		}
	}
	return tabs
}

// ZellijGoToTabName switches to a Zellij tab by name.
func ZellijGoToTabName(name string) bool {
	cmd := exec.Command("zellij", "action", "go-to-tab-name", name)
	return cmd.Run() == nil
}

// ZellijWriteChars sends text to the focused pane.
func ZellijWriteChars(text string) bool {
	cmd := exec.Command("zellij", "action", "write-chars", text)
	return cmd.Run() == nil
}

// ZellijSendEnter sends Enter key to the focused pane.
func ZellijSendEnter() bool {
	cmd := exec.Command("zellij", "action", "write", "13")
	return cmd.Run() == nil
}

// ZellijApprove sends 'y' + Enter to approve a permission request.
func ZellijApprove() bool {
	return ZellijWriteChars("y") && ZellijSendEnter()
}

// ZellijDeny sends 'n' + Enter to deny a permission request.
func ZellijDeny() bool {
	return ZellijWriteChars("n") && ZellijSendEnter()
}

// ZellijNewTab opens a new Zellij tab with a name and optionally runs a command.
func ZellijNewTab(name, cwd, command string) bool {
	cmd := exec.Command("zellij", "action", "new-tab", "--name", name)
	if err := cmd.Run(); err != nil {
		return false
	}

	time.Sleep(1 * time.Second)

	ZellijWriteChars("cd " + shellQuote(cwd))
	ZellijSendEnter()
	time.Sleep(300 * time.Millisecond)

	if command != "" {
		ZellijWriteChars(command)
		ZellijSendEnter()
		time.Sleep(3 * time.Second)
		ZellijSendEnter() // Accept workspace trust prompt
	}

	return true
}

// extractWorkspaceName extracts the Grove workspace name from a CWD path.
func extractWorkspaceName(cwd string) string {
	parts := strings.Split(cwd, "/")
	for i, p := range parts {
		if p == "workspaces" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// zellijTabCWDMap parses Zellij layout to build {tab_name: cwd} mapping.
func zellijTabCWDMap() map[string]string {
	cmd := exec.Command("zellij", "action", "dump-layout")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	mapping := make(map[string]string)
	currentTab := ""
	baseCWD := ""

	tabRe := regexp.MustCompile(`tab\s+name="([^"]+)"`)
	cwdRe := regexp.MustCompile(`cwd="([^"]+)"`)
	baseCWDRe := regexp.MustCompile(`^\s+cwd\s+"([^"]+)"`)

	for _, line := range strings.Split(string(out), "\n") {
		// Top-level layout cwd
		if currentTab == "" && baseCWD == "" {
			if m := baseCWDRe.FindStringSubmatch(line); m != nil {
				baseCWD = m[1]
				continue
			}
		}
		if m := tabRe.FindStringSubmatch(line); m != nil {
			currentTab = m[1]
			continue
		}
		if currentTab != "" {
			if _, exists := mapping[currentTab]; !exists {
				if m := cwdRe.FindStringSubmatch(line); m != nil {
					cwd := m[1]
					if !strings.HasPrefix(cwd, "/") && baseCWD != "" {
						cwd = baseCWD + "/" + cwd
					}
					mapping[currentTab] = cwd
				}
			}
		}
	}

	return mapping
}

// ZellijJumpToAgent jumps to an agent's Zellij tab using multiple matching strategies.
func ZellijJumpToAgent(projectName, cwd string) bool {
	tabs := ZellijListTabNames()
	if len(tabs) == 0 {
		return false
	}

	// 1. Exact match
	for _, tab := range tabs {
		if tab == projectName {
			return ZellijGoToTabName(tab)
		}
	}

	// 2. Case-insensitive match
	lower := strings.ToLower(projectName)
	for _, tab := range tabs {
		if strings.ToLower(tab) == lower {
			return ZellijGoToTabName(tab)
		}
	}

	// 3. Match workspace name from CWD
	if cwd != "" {
		wsName := extractWorkspaceName(cwd)
		if wsName != "" {
			wsLower := strings.ToLower(wsName)
			for _, tab := range tabs {
				if strings.ToLower(tab) == wsLower {
					return ZellijGoToTabName(tab)
				}
			}
			for _, tab := range tabs {
				tabLower := strings.ToLower(tab)
				if strings.Contains(tabLower, wsLower) || strings.Contains(wsLower, tabLower) {
					return ZellijGoToTabName(tab)
				}
			}
		}
	}

	// 4. Parse layout for CWD matching
	tabCWDs := zellijTabCWDMap()
	for tabName, tabCWD := range tabCWDs {
		if cwd != "" && (strings.HasPrefix(cwd, tabCWD+"/") || cwd == tabCWD || strings.HasPrefix(tabCWD, cwd+"/")) {
			return ZellijGoToTabName(tabName)
		}
	}
	for tabName, tabCWD := range tabCWDs {
		parts := strings.Split(strings.ToLower(tabCWD), "/")
		for _, part := range parts {
			if part == lower {
				return ZellijGoToTabName(tabName)
			}
		}
	}

	// 5. Substring match
	for _, tab := range tabs {
		if strings.Contains(strings.ToLower(tab), lower) {
			return ZellijGoToTabName(tab)
		}
	}

	return false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
