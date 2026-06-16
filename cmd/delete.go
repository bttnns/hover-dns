package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <record-id>",
	Short: "Delete a DNS record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClientFromConfig()
		if err != nil {
			return err
		}
		if err := c.Delete(args[0]); err != nil {
			return err
		}
		fmt.Printf("deleted %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
