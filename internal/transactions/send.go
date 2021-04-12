/*
 * Flow CLI
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transactions

import (
	"fmt"

	"github.com/onflow/flow-cli/internal/command"
	"github.com/onflow/flow-cli/pkg/flowcli/services"
	"github.com/spf13/cobra"
)

type flagsSend struct {
	ArgsJSON string   `default:"" flag:"args-json" info:"arguments in JSON-Cadence format"`
	Arg      []string `default:"" flag:"arg" info:"argument in Type:Value format"`
	Signer   string   `default:"emulator-account" flag:"signer" info:"Account name from configuration used to sign the transaction"`
	Code     string   `default:"" flag:"code" info:"⚠️  Deprecated: use filename argument"`
	Results  bool     `default:"" flag:"results" info:"⚠️  Deprecated: all transactions will provide result"`
	Args     string   `default:"" flag:"args" info:"⚠️  Deprecated: use arg or args-json flag"`
	Payload  string   `flag:"payload" info:"path to the transaction payload file"`
}

var sendFlags = flagsSend{}

var SendCommand = &command.Command{
	Cmd: &cobra.Command{
		Use:     "send <filename>",
		Short:   "Send a transaction",
		Args:    cobra.MaximumNArgs(1),
		Example: `flow transactions send tx.cdc --arg String:"Hello world"`,
	},
	Flags: &sendFlags,
	Run: func(
		cmd *cobra.Command,
		args []string,
		globalFlags command.GlobalFlags,
		services *services.Services,
	) (command.Result, error) {
		if sendFlags.Results {
			fmt.Println("⚠️  DEPRECATION WARNING: all transactions will provide results")
		}

		if sendFlags.Args != "" {
			fmt.Println("⚠️  DEPRECATION WARNING: use arg flag in Type:Value format or arg-json for JSON format")

			if len(sendFlags.Arg) == 0 && sendFlags.ArgsJSON == "" {
				sendFlags.ArgsJSON = sendFlags.Args // backward compatible, args was in json format
			}
		}

		filename := ""
		if len(args) == 1 {
			filename = args[0]
		} else if sendFlags.Code != "" {
			fmt.Println("⚠️  DEPRECATION WARNING: use filename as a command argument <filename>")
			filename = sendFlags.Code
		} else {
			return nil, fmt.Errorf("provide a valide filename command argument")
		}

		tx, result, err := services.Transactions.Send(
			filename,
			sendFlags.Payload,
			sendFlags.Signer,
			sendFlags.Arg,
			sendFlags.ArgsJSON,
		)
		if err != nil {
			return nil, err
		}

		return &TransactionResult{
			result: result,
			tx:     tx,
		}, nil
	},
}
