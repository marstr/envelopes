// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/marstr/envelopes/persist"
	"github.com/spf13/viper"

	"github.com/marstr/envelopes"

	"github.com/spf13/cobra"
)

// debitCmd represents the debit command
var debitCmd = &cobra.Command{
	Use:   "debit",
	Short: "A brief description of your command",
	Args: func(cmd *cobra.Command, args []string) {
		amountContender := args[0]
		accountContender := args[1]

		_, err := envelopes.ParseAmount(amountContender)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q not recognized as an amount.", amountContender)
			os.Exit(1)
		}

		fs := persist.FileSystem{
			Root: viper.Get("location").(string),
		}

		loader := persist.DefaultLoader{
			Fetcher: fs,
		}

		currentID, err := fs.LoadCurrent(context.Background())
		if err != nil {
			os.Exit(1)
		}

		currentTrans, currentState, currentAccounts, currentBudget := loader.LoadAll(context.Background(), currentID)
		if !currentAccounts.HasAccount(accountContender) {
			fmt.Fprintf(os.Stderr, "%q isn't an account in the current ledger", accountContender)
			os.Exit(1)
		}
	},
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("debit called")
	},
}

func init() {
	RootCmd.AddCommand(debitCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// debitCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// debitCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
