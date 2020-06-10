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
	"fmt"

	"encoding/xml"
	"github.com/Brian-Williams/qualsy/cmd/internal/qualys"
	"github.com/spf13/cobra"
)

var (
	name       string
	color      string
	addr       string
	idempotent bool
)

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Create a tag",
	RunE: func(cmd *cobra.Command, args []string) error {
		q := qualys.New(username, password, apiUrl)
		var tagID string
		if idempotent {
			tagID, _ = q.SearchTagExists(name)
		}
		if tagID != "" {
			fmt.Printf("Tag '%s' found skipping creation", tagID)
		} else {
			body := qualys.CreateTag{
				XMLName: xml.Name{Local: "ServiceRequest"},
				Tag: qualys.TagInfo{
					Name:  name,
					Color: color,
				},
			}
			tagID, err := q.CreateTag(body)
			if err != nil {
				return err
			}
			fmt.Printf("Successfully created tag %s with id %s\n", body.Tag.Name, tagID)
		}

		if addr != "" {
			idCrit := qualys.CriteriaServiceRequest{
				Criteria: []qualys.Criteria{
					{
						Field:    "address",
						Operator: "EQUALS",
						Criteria: addr,
					},
				},
			}
			id, err := q.IdFromCriteria(idCrit)
			if err != nil {
				return err
			}
			sr := qualys.UpdateAsset{
				Criteria: []qualys.Criteria{
					{
						Field:    "id",
						Operator: "EQUALS",
						Criteria: id,
					},
				},
				Id: tagID,
			}
			err = q.UpdateAsset(sr)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.PersistentFlags().StringVar(&name, "name", "", "name of the tag")
	tagCmd.MarkPersistentFlagRequired("name")
	tagCmd.Flags().StringVar(&color, "color", "#FFFFFF", "color of the tag")
	tagCmd.Flags().StringVar(&addr, "tag-addr", "", "addr to tag with new tag")
	tagCmd.Flags().BoolVar(&idempotent, "idempotent", false,
		"idempotent creation of tag, there is no color guarantee to this action")
}
