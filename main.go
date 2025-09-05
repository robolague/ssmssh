package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/ini.v1"
)

// Loading spinner style
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)

// Lipgloss styles
var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF")).Background(lipgloss.Color("#1a1a1a")).Padding(0, 1)
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Padding(0, 1)
	borderStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#00BFFF")).Padding(1, 2)
	itemStyle     = lipgloss.NewStyle().Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1a1a1a")).Background(lipgloss.Color("#00BFFF")).Padding(0, 1)
	quitStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Italic(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)
)

type Tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type state int

const (
	stateProfile state = iota
	stateRegion
	stateInstance
	stateDone
)

type model struct {
	profiles          []string
	regions           []string
	instances         []string
	selectedProfile   string
	selectedRegion    string
	selectedInstance  string
	cursor            int
	err               error
	step              state
	filter            string
	filteredProfiles  []string
	filteredRegions   []string
	filteredInstances []string
	loading           bool
	spinnerFrame      int
	previewTags       []Tag
	previewLoading    bool
	previewInstanceId string
}

func getProfiles() ([]string, error) {
	cfg, err := ini.Load(os.ExpandEnv("$HOME/.aws/credentials"))
	if err != nil {
		return nil, err
	}
	profiles := []string{}
	for _, section := range cfg.Sections() {
		if section.Name() != "DEFAULT" {
			profiles = append(profiles, section.Name())
		}
	}
	return profiles, nil
}

func getRegions(profile string) ([]string, error) {
	cmd := exec.Command("aws", "ec2", "describe-regions", "--profile", profile, "--region", "us-west-2", "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result struct {
		Regions []struct {
			RegionName string `json:"RegionName"`
		}
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	regions := []string{}
	for _, r := range result.Regions {
		regions = append(regions, r.RegionName)
	}
	return regions, nil
}

func getInstances(profile, region string) ([]string, error) {
	cmd := exec.Command("aws", "ec2", "describe-instances", "--profile", profile, "--region", region, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result struct {
		Reservations []struct {
			Instances []struct {
				InstanceId string `json:"InstanceId"`
				Tags       []struct {
					Key   string `json:"Key"`
					Value string `json:"Value"`
				} `json:"Tags"`
			}
		}
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	instances := []string{}
	for _, res := range result.Reservations {
		for _, inst := range res.Instances {
			name := ""
			for _, tag := range inst.Tags {
				if tag.Key == "Name" {
					name = tag.Value
				}
			}
			display := inst.InstanceId
			if name != "" {
				display += " (" + name + ")"
			}
			instances = append(instances, display)
		}
	}
	return instances, nil
}

func startSession(profile, region, instanceId string) error {
	cmd := exec.Command("aws", "ssm", "start-session", "--profile", profile, "--region", region, "--target", instanceId)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Printf("Running: aws ssm start-session --profile %s --region %s --target %s\n", profile, region, instanceId)
	return cmd.Run()
}

func getInstanceTags(profile, region, instanceId string) ([]Tag, error) {
	cmd := exec.Command("aws", "ec2", "describe-instances", "--profile", profile, "--region", region, "--instance-ids", instanceId, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var result struct {
		Reservations []struct {
			Instances []struct {
				Tags []Tag `json:"Tags"`
			}
		}
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	tags := []Tag{}
	for _, res := range result.Reservations {
		for _, inst := range res.Instances {
			for _, tag := range inst.Tags {
				tags = append(tags, tag)
			}
		}
	}
	return tags, nil
}

func regionsCmd(profile string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan struct {
			regions []string
			err     error
		}, 1)
		go func() {
			regions, err := getRegions(profile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "getRegions error: %v\n", err)
			}
			ch <- struct {
				regions []string
				err     error
			}{regions, err}
		}()
		select {
		case msg := <-ch:
			return msg
		case <-time.After(15 * time.Second):
			fmt.Fprintf(os.Stderr, "timeout loading regions\n")
			return struct {
				regions []string
				err     error
			}{nil, fmt.Errorf("timeout loading regions")}
		}
	}
}

func instancesCmd(profile, region string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan struct {
			instances []string
			err       error
		}, 1)
		go func() {
			instances, err := getInstances(profile, region)
			if err != nil {
				fmt.Fprintf(os.Stderr, "getInstances error: %v\n", err)
			}
			ch <- struct {
				instances []string
				err       error
			}{instances, err}
		}()
		select {
		case msg := <-ch:
			return msg
		case <-time.After(15 * time.Second):
			fmt.Fprintf(os.Stderr, "timeout loading instances\n")
			return struct {
				instances []string
				err       error
			}{nil, fmt.Errorf("timeout loading instances")}
		}
	}
}

func previewTagsCmd(profile, region, instanceId string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan struct {
			tags       []Tag
			instanceId string
			err        error
		}, 1)
		go func() {
			tags, err := getInstanceTags(profile, region, instanceId)
			ch <- struct {
				tags       []Tag
				instanceId string
				err        error
			}{tags, instanceId, err}
		}()
		select {
		case msg := <-ch:
			return msg
		case <-time.After(5 * time.Second):
			return struct {
				tags       []Tag
				instanceId string
				err        error
			}{nil, instanceId, fmt.Errorf("timeout loading tags")}
		}
	}
}

func initialModel() model {
	profiles, err := getProfiles()
	return model{
		profiles:         profiles,
		filteredProfiles: profiles,
		cursor:           0,
		err:              err,
		step:             stateProfile,
		filter:           "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		// Only allow quit on command-q and command-c and esc
		switch s {
		case "cmd+q", "cmd+c", "esc":
			return m, tea.Quit
		}
		if m.loading {
			// Ignore other input while loading
			return m, nil
		}
		// Up/down arrows always work for navigation, k/j only if filter is empty
		switch s {
		case "up":
			switch m.step {
			case stateProfile:
				if m.cursor > 0 {
					m.cursor--
				}
			case stateRegion:
				if m.cursor > 0 {
					m.cursor--
				}
			case stateInstance:
				if m.cursor > 0 {
					m.cursor--
					m.previewLoading = true
					m.previewInstanceId = strings.Split(m.filteredInstances[m.cursor], " ")[0]
					return m, previewTagsCmd(m.selectedProfile, m.selectedRegion, m.previewInstanceId)
				}
			}
		case "down":
			switch m.step {
			case stateProfile:
				if m.cursor < len(m.filteredProfiles)-1 {
					m.cursor++
				}
			case stateRegion:
				if m.cursor < len(m.filteredRegions)-1 {
					m.cursor++
				}
			case stateInstance:
				if m.cursor < len(m.filteredInstances)-1 {
					m.cursor++
					m.previewLoading = true
					m.previewInstanceId = strings.Split(m.filteredInstances[m.cursor], " ")[0]
					return m, previewTagsCmd(m.selectedProfile, m.selectedRegion, m.previewInstanceId)
				}
			}
		}
		if m.filter == "" {
			switch s {
			case "k":
				switch m.step {
				case stateProfile:
					if m.cursor > 0 {
						m.cursor--
					}
				case stateRegion:
					if m.cursor > 0 {
						m.cursor--
					}
				case stateInstance:
					if m.cursor > 0 {
						m.cursor--
						m.previewLoading = true
						m.previewInstanceId = strings.Split(m.filteredInstances[m.cursor], " ")[0]
						return m, previewTagsCmd(m.selectedProfile, m.selectedRegion, m.previewInstanceId)
					}
				}
			case "j":
				switch m.step {
				case stateProfile:
					if m.cursor < len(m.filteredProfiles)-1 {
						m.cursor++
					}
				case stateRegion:
					if m.cursor < len(m.filteredRegions)-1 {
						m.cursor++
					}
				case stateInstance:
					if m.cursor < len(m.filteredInstances)-1 {
						m.cursor++
						m.previewLoading = true
						m.previewInstanceId = strings.Split(m.filteredInstances[m.cursor], " ")[0]
						return m, previewTagsCmd(m.selectedProfile, m.selectedRegion, m.previewInstanceId)
					}
				}
			}
		}
		switch s {
		case "enter":
			switch m.step {
			case stateProfile:
				if len(m.filteredProfiles) == 0 {
					m.err = fmt.Errorf("no AWS profiles found")
					return m, tea.Quit
				}
				m.selectedProfile = m.filteredProfiles[m.cursor]
				m.loading = true
				return m, regionsCmd(m.selectedProfile)
			case stateRegion:
				if len(m.filteredRegions) == 0 {
					m.err = fmt.Errorf("no regions found")
					return m, tea.Quit
				}
				m.selectedRegion = m.filteredRegions[m.cursor]
				m.loading = true
				return m, instancesCmd(m.selectedProfile, m.selectedRegion)
			case stateInstance:
				if len(m.filteredInstances) == 0 {
					m.err = fmt.Errorf("no instances found")
					return m, tea.Quit
				}
				m.selectedInstance = strings.Split(m.filteredInstances[m.cursor], " ")[0]
				m.step = stateDone
				return m, tea.Quit
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
		default:
			// Only filter on printable runes
			if len(s) == 1 && s[0] >= 32 && s[0] <= 126 {
				m.filter += s
			}
		}
		// Update filtered lists
		switch m.step {
		case stateProfile:
			m.filteredProfiles = filterList(m.profiles, m.filter)
			if len(m.filteredProfiles) == 0 {
				m.cursor = 0
			} else {
				if m.cursor >= len(m.filteredProfiles) {
					m.cursor = len(m.filteredProfiles) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case stateRegion:
			m.filteredRegions = filterList(m.regions, m.filter)
			if len(m.filteredRegions) == 0 {
				m.cursor = 0
			} else {
				if m.cursor >= len(m.filteredRegions) {
					m.cursor = len(m.filteredRegions) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case stateInstance:
			m.filteredInstances = filterList(m.instances, m.filter)
			if len(m.filteredInstances) == 0 {
				m.cursor = 0
			} else {
				if m.cursor >= len(m.filteredInstances) {
					m.cursor = len(m.filteredInstances) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
				// Preview loading
				m.previewLoading = true
				m.previewInstanceId = strings.Split(m.filteredInstances[m.cursor], " ")[0]
				return m, previewTagsCmd(m.selectedProfile, m.selectedRegion, m.previewInstanceId)
			}
		}
	case struct {
		regions []string
		err     error
	}:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.regions = msg.regions
		m.filteredRegions = msg.regions
		m.cursor = 0
		m.filter = ""
		m.step = stateRegion
	case struct {
		instances []string
		err       error
	}:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.instances = msg.instances
		m.filteredInstances = msg.instances
		m.cursor = 0
		m.filter = ""
		m.step = stateInstance
	case struct {
		tags       []Tag
		instanceId string
		err        error
	}:
		if msg.instanceId == m.previewInstanceId {
			m.previewTags = msg.tags
			m.previewLoading = false
		}
	case tea.Msg:
		// Spinner tick: use a custom message type
		if m.loading {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, spinnerTick()
		}
	}
	// If loading, keep ticking for spinner
	if m.loading {
		return m, spinnerTick()
	}
	return m, nil
}

func spinnerTick() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(80 * time.Millisecond)
		return struct{}{} // empty struct as tick message
	}
}

func filterList(list []string, filter string) []string {
	if filter == "" {
		return list
	}
	f := strings.ToLower(filter)
	out := []string{}
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), f) {
			out = append(out, item)
		}
	}
	return out
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: "+m.err.Error()) + "\n"
	}
	var content string
	if m.loading {
		spinner := spinnerStyle.Render(spinnerFrames[m.spinnerFrame])
		msg := infoStyle.Render("Loading...")
		return borderStyle.Render(fmt.Sprintf("%s %s", spinner, msg))
	}
	switch m.step {
	case stateProfile:
		content += headerStyle.Render("Select AWS profile") + "\n"
		content += infoStyle.Render("Search:"+m.filter) + "\n"
		windowSize := 20
		start := m.cursor - windowSize/2
		if start < 0 {
			start = 0
		}
		end := start + windowSize
		if end > len(m.filteredProfiles) {
			end = len(m.filteredProfiles)
			start = end - windowSize
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			p := m.filteredProfiles[i]
			var line string
			if m.cursor == i {
				line = selectedStyle.Render("> " + p)
			} else {
				line = itemStyle.Render("  " + p)
			}
			content += line + "\n"
		}
		content += quitStyle.Render("esc: quit")
		return borderStyle.Render(content)
	case stateRegion:
		content += headerStyle.Render("Select AWS region") + "\n"
		content += infoStyle.Render("Profile:"+m.selectedProfile) + "\n"
		content += infoStyle.Render("Search:"+m.filter) + "\n"
		windowSize := 20
		start := m.cursor - windowSize/2
		if start < 0 {
			start = 0
		}
		end := start + windowSize
		if end > len(m.filteredRegions) {
			end = len(m.filteredRegions)
			start = end - windowSize
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			r := m.filteredRegions[i]
			var line string
			if m.cursor == i {
				line = selectedStyle.Render("> " + r)
			} else {
				line = itemStyle.Render("  " + r)
			}
			content += line + "\n"
		}
		content += quitStyle.Render("esc: quit")
		return borderStyle.Render(content)
	case stateInstance:
		// Left: instance list
		left := headerStyle.Render("Select EC2 instance") + "\n"
		left += infoStyle.Render("Profile:"+m.selectedProfile+" | Region:"+m.selectedRegion) + "\n"
		left += infoStyle.Render("Search:"+m.filter) + "\n"
		windowSize := 20
		start := m.cursor - windowSize/2
		if start < 0 {
			start = 0
		}
		end := start + windowSize
		if end > len(m.filteredInstances) {
			end = len(m.filteredInstances)
			start = end - windowSize
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			inst := m.filteredInstances[i]
			var line string
			if m.cursor == i {
				line = selectedStyle.Render("> " + inst)
			} else {
				line = itemStyle.Render("  " + inst)
			}
			left += line + "\n"
		}
		left += quitStyle.Render("esc: quit")
		left = borderStyle.Render(left)
		// Right: preview window
		var right string
		if m.previewLoading {
			right = spinnerStyle.Render(spinnerFrames[m.spinnerFrame]) + " " + infoStyle.Render("Loading tags...")
		} else if len(m.previewTags) > 0 {
			right = headerStyle.Render("Instance Tags") + "\n"
			for _, tag := range m.previewTags {
				right += infoStyle.Render(fmt.Sprintf("%s: %s", tag.Key, tag.Value)) + "\n"
			}
		} else {
			right = infoStyle.Render("No tags found.")
		}
		right = borderStyle.Render(right)
		// Layout: side by side
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	case stateDone:
		content += headerStyle.Render("Session Starting") + "\n"
		content += infoStyle.Render(fmt.Sprintf("Selected: Profile=%s, Region=%s, Instance=%s", m.selectedProfile, m.selectedRegion, m.selectedInstance)) + "\n"
		content += infoStyle.Render("Starting SSM session...") + "\n"
		return borderStyle.Render(content)
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel())
	m, err := p.Run()
	if err != nil {
		fmt.Println("Error running Bubble Tea program:", err)
		os.Exit(1)
	}
	final := m.(model)
	if final.err != nil || final.step != stateDone {
		os.Exit(1)
	}
	// Start SSM session
	err = startSession(final.selectedProfile, final.selectedRegion, final.selectedInstance)
	if err != nil {
		fmt.Println("Error starting SSM session:", err)
		os.Exit(1)
	}
}
