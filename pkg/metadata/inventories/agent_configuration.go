// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package inventories

import (
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/util/scrubber"
	"gopkg.in/yaml.v2"
)

// AgentConfiguration holds the configuration information for the inventory product
type AgentConfiguration struct {
	// AgentConfiguration is the entire Agent configuration scrubbed as YAML
	RuntimeConfiguration string `json:"runtime"`
	// ProvidedConfiguration are the settings set by the users scrubbed as YAML (ie: not the default)
	ProvidedConfiguration string `json:"provided"`
}

func getAgentConfiguration() *AgentConfiguration {
	if !config.Datadog.GetBool("inventories_configuration_enabled") {
		return nil
	}

	flareScrubber := scrubber.NewWithDefaults()

	conf, err := yaml.Marshal(config.Datadog.AllSettings())
	if err != nil {
		log.Errorf("could not marshal agent configuration: %s", err)
		return nil
	}

	scrubbedConf, err := flareScrubber.ScrubBytes(conf)
	if err != nil {
		log.Errorf("could not scrubb agent configuration: %s", err)
		return nil
	}

	provided, err := yaml.Marshal(config.Datadog.AllSettingsWithoutDefault())
	if err != nil {
		log.Errorf("could not marshal agent configuration: %s", err)
		return nil
	}

	scrubbedProvided, err := flareScrubber.ScrubBytes(provided)
	if err != nil {
		log.Errorf("could not scrubb agent configuration: %s", err)
		return nil
	}

	return &AgentConfiguration{
		RuntimeConfiguration:  string(scrubbedConf),
		ProvidedConfiguration: string(scrubbedProvided),
	}
}
