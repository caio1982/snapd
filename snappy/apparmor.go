package snappy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type apparmorJSONTemplate struct {
	Template      string   `json:"template"`
	PolicyGroups  []string `json:"policy_groups"`
	PolicyVendor  string   `json:"policy_vendor"`
	PolicyVersion float64  `json:"policy_version"`
}

func (s *SecurityDefinitions) generateApparmorJSONContent() ([]byte, error) {
	t := apparmorJSONTemplate{
		Template:     s.SecurityTemplate,
		PolicyGroups: s.SecurityCaps,
		// TODO: this won't work with Ubuntu Personal, etc
		PolicyVendor: "ubuntu-core",
		// TODO: this should perhaps be autodetected
		PolicyVersion: 15.04,
	}

	// FIXME: this is snappy specific, on other systems like the
	//        phone we may want different defaults.
	if t.Template == "" && t.PolicyGroups == nil {
		t.PolicyGroups = []string{"networking"}
	}

	// never write a null value out into the json
	if t.PolicyGroups == nil {
		t.PolicyGroups = []string{}
	}

	if t.Template == "" {
		t.Template = "default"
	}

	outStr, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return nil, err
	}

	return outStr, nil
}

func handleApparmor(buildDir string, m *packageYaml, hookName string, s *SecurityDefinitions) error {

	// ensure we have a hook
	if _, ok := m.Integration[hookName]; !ok {
		m.Integration[hookName] = clickAppHook{}
	}

	// legacy use of "Integration" - the user should
	// use the new format, nothing needs to be done
	_, hasApparmor := m.Integration[hookName]["apparmor"]
	_, hasApparmorProfile := m.Integration[hookName]["apparmor-profile"]
	if hasApparmor || hasApparmorProfile {
		return nil
	}

	// see if we have a custom security policy
	if s.SecurityPolicy != nil && s.SecurityPolicy.Apparmor != "" {
		m.Integration[hookName]["apparmor-profile"] = s.SecurityPolicy.Apparmor
		return nil
	}

	// see if we have a security override
	if s.SecurityOverride != nil && s.SecurityOverride.Apparmor != "" {
		m.Integration[hookName]["apparmor"] = s.SecurityOverride.Apparmor
		return nil
	}

	// generate apparmor template
	apparmorJSONFile := filepath.Join("meta", hookName+".apparmor")
	securityJSONContent, err := s.generateApparmorJSONContent()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(buildDir, apparmorJSONFile), securityJSONContent, 0644); err != nil {
		return err
	}

	m.Integration[hookName]["apparmor"] = apparmorJSONFile

	return nil
}

func getAaProfile(m *packageYaml, appName string) string {
	cleanedName := strings.Replace(appName, "/", "-", -1)
	return fmt.Sprintf("%s_%s_%s", m.Name, cleanedName, m.Version)
}