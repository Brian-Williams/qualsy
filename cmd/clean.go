/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/Brian-Williams/qualsy/cmd/internal/qualys"
	"github.com/spf13/cobra"
)

var (
	tag        string
	deactivate bool
	uninstall  bool
	deleteTag  bool
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean data with a criteria",
	RunE: func(cmd *cobra.Command, args []string) error {
		q := qualys.New(username, password, apiUrl)
		crit := qualys.CriteriaServiceRequest{
			Criteria: []qualys.Criteria{
				{
					Field:    "tagName",
					Operator: "EQUALS",
					Criteria: tag,
				},
			},
		}
		if deactivate {
			err := qualys.PostCritChecker(q, "qps/rest/2.0/deactivate/am/asset?=&module=AGENT_VM%2CAGENT_PC", crit)
			if err != nil {
				return err
			}
		}
		if uninstall {
			err := qualys.PostCritChecker(q, "qps/rest/2.0/uninstall/am/asset?=", crit)
			if err != nil {
				return err
			}
		}
		if deleteTag {
			crit.Criteria[0].Field = "name"
			err := qualys.PostCritChecker(q, "qps/rest/2.0/delete/am/tag/", crit)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.PersistentFlags().StringVar(&tag, "tag", "", "search tag for cleaning")
	cleanCmd.PersistentFlags().BoolVar(&deactivate, "deactivate", false, "deactivate matching agents")
	cleanCmd.PersistentFlags().BoolVar(&uninstall, "uninstall", false, "uninstall matching agents")
	cleanCmd.PersistentFlags().BoolVar(&deleteTag, "delete", false, "delete matching tag")
}
