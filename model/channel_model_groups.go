package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

// ModelGroups returns the configured routes, or the legacy model list shared by every group.
func (channel *Channel) ModelGroups() []dto.ChannelModelGroup {
	var settings struct {
		ModelGroups []dto.ChannelModelGroup `json:"model_groups"`
	}
	if channel.OtherSettings != "" &&
		common.UnmarshalJsonStr(channel.OtherSettings, &settings) == nil &&
		len(settings.ModelGroups) > 0 {
		return settings.ModelGroups
	}
	models := channel.GetModels()
	if len(models) == 0 {
		return nil
	}
	groups := channel.GetGroups()
	rows := make([]dto.ChannelModelGroup, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, dto.ChannelModelGroup{Group: group, Models: models})
	}
	return rows
}

// SetModelGroupSummary updates the fields used for flat channel searches and model enumeration.
func (channel *Channel) SetModelGroupSummary(rows []dto.ChannelModelGroup) {
	groups := make([]string, 0, len(rows))
	var models []string
	seen := make(map[string]bool)
	for _, row := range rows {
		groups = append(groups, row.Group)
		for _, name := range row.Models {
			if !seen[name] {
				models = append(models, name)
				seen[name] = true
			}
		}
	}
	channel.Group = strings.Join(groups, ",")
	channel.Models = strings.Join(models, ",")
}

func validateChannelModelGroups(rows []dto.ChannelModelGroup) error {
	groups := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.Group == "" || row.Group == "auto" || len(row.Group) > 64 ||
			row.Group != strings.TrimSpace(row.Group) || strings.Contains(row.Group, ",") {
			return common.NewMessage("Invalid channel model group")
		}
		if groups[row.Group] {
			return common.NewMessage("Each group can only be added once")
		}
		groups[row.Group] = true
		for _, name := range row.Models {
			if name == "" || len(name) > 255 ||
				name != strings.TrimSpace(name) || strings.Contains(name, ",") {
				return common.NewMessage("Invalid model name in channel model group")
			}
		}
	}
	return nil
}
