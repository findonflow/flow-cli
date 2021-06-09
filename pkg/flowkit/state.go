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

package flowkit

import (
	"errors"
	"fmt"
	"path"

	"github.com/onflow/cadence"

	"github.com/onflow/flow-cli/pkg/flowkit/util"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/spf13/afero"
	"github.com/thoas/go-funk"

	"github.com/onflow/flow-cli/pkg/flowkit/config"
	"github.com/onflow/flow-cli/pkg/flowkit/config/json"
)

// State contains the configuration for a Flow project.
type State struct {
	loader   *config.Loader
	conf     *config.Config
	accounts []*Account
}

// Contract is a Cadence contract definition for a project.
type Contract struct {
	Name   string
	Source string
	Target flow.Address
	Args   []cadence.Value
}

// refactor to config loader

// Load loads a project configuration and returns the resulting project.
func Load(configFilePaths []string) (*State, error) {
	loader := config.NewLoader(afero.NewOsFs())

	// here we add all available parsers (more to add yaml etc...)
	loader.AddConfigParser(json.NewParser())
	conf, err := loader.Load(configFilePaths)

	if err != nil {
		if errors.Is(err, config.ErrDoesNotExist) {
			return nil, err
		}

		return nil, err
	}

	proj, err := newProject(conf, loader)
	if err != nil {
		return nil, fmt.Errorf("invalid project configuration: %s", err)
	}

	return proj, nil
}

// refactor to config loader

// SaveDefault saves configuration to default path
func (p *State) SaveDefault() error {
	return p.Save(config.DefaultPath)
}

// refactor to config loader

// Save saves the project configuration to the given path.
func (p *State) Save(path string) error {
	p.conf.Accounts = accountsToConfig(p.accounts)
	err := p.loader.Save(p.conf, path)

	if err != nil {
		return fmt.Errorf("failed to save project configuration to: %s", path)
	}

	return nil
}

// refactor to config loader

// Exists checks if a project configuration exists.
func Exists(path string) bool {
	return config.Exists(path)
}

// refactor

// Init initializes a new Flow project.
func Init(sigAlgo crypto.SignatureAlgorithm, hashAlgo crypto.HashAlgorithm) (*State, error) {
	emulatorServiceAccount, err := generateEmulatorServiceAccount(sigAlgo, hashAlgo)
	if err != nil {
		return nil, err
	}

	composer := config.NewLoader(afero.NewOsFs())
	composer.AddConfigParser(json.NewParser())

	return &State{
		loader:   composer,
		conf:     config.DefaultConfig(),
		accounts: []*Account{emulatorServiceAccount},
	}, nil
}

// refactor to get config

// newProject creates a new project from a configuration object.
func newProject(conf *config.Config, composer *config.Loader) (*State, error) {
	accounts, err := accountsFromConfig(conf)
	if err != nil {
		return nil, err
	}

	return &State{
		loader:   composer,
		conf:     conf,
		accounts: accounts,
	}, nil
}

// refactor to contracts ?

// CheckContractConflict returns true if the same contract is configured to deploy
// to more than one account in the same network.
//
// The CLI currently does not allow the same contract to be deployed to multiple
// accounts in the same network.
func (p *State) ContractConflictExists(network string) bool {
	contracts, err := p.DeploymentContractsByNetwork(network)
	if err != nil {
		return false
	}

	uniq := funk.Uniq(
		funk.Map(contracts, func(c Contract) string {
			return c.Name
		}).([]string),
	).([]string)

	all := funk.Map(contracts, func(c Contract) string {
		return c.Name
	}).([]string)

	return len(all) != len(uniq)
}

// Networks get network configuration
func (p *State) Networks() *config.Networks {
	return &p.conf.Networks
}

// Deployments get deployments configuration
func (p *State) Deployments() *config.Deployments {
	return &p.conf.Deployments
}

// Contracts get contracts configuration
func (p *State) Contracts() *config.Contracts {
	return &p.conf.Contracts
}

// refactor to accounts ?

// EmulatorServiceAccount returns the service account for the default emulator profilee.
func (p *State) EmulatorServiceAccount() (*Account, error) {
	emulator := p.conf.Emulators.Default()
	acc := p.conf.Accounts.GetByName(emulator.ServiceAccount)
	return accountFromConfig(*acc)
}

// refactor to accounts ?

// SetEmulatorServiceKey sets the default emulator service account private key.
func (p *State) SetEmulatorServiceKey(privateKey crypto.PrivateKey) {
	acc := p.AccountByName(config.DefaultEmulatorServiceAccountName)
	acc.SetKey(
		NewHexAccountKeyFromPrivateKey(
			acc.Key().Index(),
			acc.Key().HashAlgo(),
			privateKey,
		),
	)
}

// refactor ????

// DeploymentContractsByNetwork returns all contracts for a network.
func (p *State) DeploymentContractsByNetwork(network string) ([]Contract, error) {
	contracts := make([]Contract, 0)

	// get deployments for the specified network
	for _, deploy := range p.conf.Deployments.GetByNetwork(network) {
		account := p.AccountByName(deploy.Account)
		if account == nil {
			return nil, fmt.Errorf("could not find account with name %s in the configuration", deploy.Account)
		}

		// go through each contract in this deployment
		for _, deploymentContract := range deploy.Contracts {
			c := p.conf.Contracts.GetByNameAndNetwork(deploymentContract.Name, network)
			if c == nil {
				return nil, fmt.Errorf("could not find contract with name name %s in the configuration", deploymentContract.Name)
			}

			contract := Contract{
				Name:   c.Name,
				Source: path.Clean(c.Source),
				Target: account.address,
				Args:   deploymentContract.Args,
			}

			contracts = append(contracts, contract)
		}
	}

	return contracts, nil
}

// refactor to config deployments

// AccountNamesForNetwork returns all configured account names for a network.
func (p *State) AccountNamesForNetwork(network string) []string {
	names := make([]string, 0)

	for _, account := range p.accounts {
		if len(p.conf.Deployments.GetByAccountAndNetwork(account.name, network)) > 0 {
			if !util.ContainsString(names, account.name) {
				names = append(names, account.name)
			}
		}
	}

	return names
}

// refactor to accounts

// AddOrUpdateAccount adds or updates an account.
func (p *State) AddOrUpdateAccount(account *Account) {
	for i, existingAccount := range p.accounts {
		if existingAccount.name == account.name {
			(*p).accounts[i] = account
			return
		}
	}

	p.accounts = append(p.accounts, account)
}

// refactor to accounts

// RemoveAccount removes an account from configuration
func (p *State) RemoveAccount(name string) error {
	account := p.AccountByName(name)
	if account == nil {
		return fmt.Errorf("account named %s does not exist in configuration", name)
	}

	for i, account := range p.accounts {
		if account.name == name {
			(*p).accounts = append(p.accounts[0:i], p.accounts[i+1:]...) // remove item
		}
	}

	return nil
}

// refactor to accounts

// AccountByAddress returns an account by address.
func (p *State) AccountByAddress(address string) *Account {
	for _, account := range p.accounts {
		if account.address.String() == flow.HexToAddress(address).String() {
			return account
		}
	}

	return nil
}

// refactor to accounts

// AccountByName returns an account by name.
func (p *State) AccountByName(name string) *Account {
	var account *Account

	for _, acc := range p.accounts {
		if acc.name == name {
			account = acc
		}
	}

	return account
}

// refactor to contracts

type Aliases map[string]string

// AliasesForNetwork returns all deployment aliases for a network.
func (p *State) AliasesForNetwork(network string) Aliases {
	aliases := make(Aliases)

	// get all contracts for selected network and if any has an address as target make it an alias
	for _, contract := range p.conf.Contracts.GetByNetwork(network) {
		if contract.IsAlias() {
			aliases[path.Clean(contract.Source)] = contract.Alias
		}
	}

	return aliases
}
