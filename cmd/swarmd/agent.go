package main

import (
	engineapi "github.com/docker/engine-api/client"
	"github.com/docker/swarm-v2/agent"
	"github.com/docker/swarm-v2/agent/exec/container"
	"github.com/docker/swarm-v2/ca"
	"github.com/docker/swarm-v2/identity"
	"github.com/docker/swarm-v2/log"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var (
	agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Run the swarm agent",
		Long: `Start a swarm agent with the provided path. If starting from an
empty path, the agent will allocate an identity and startup. If data is
already present, the agent will recover and startup.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			hostname, err := cmd.Flags().GetString("hostname")
			if err != nil {
				return err
			}

			id, err := cmd.Flags().GetString("id")
			if err != nil {
				return err
			}

			if id == "" {
				log.G(ctx).Debugf("agent: generated random identifier")
				id = identity.NewID()
			}

			managerAddrs, err := cmd.Flags().GetStringSlice("manager")
			if err != nil {
				return err
			}

			engineAddr, err := cmd.Flags().GetString("engine-addr")
			if err != nil {
				return err
			}

			log.G(ctx).Debugf("managers: %v", managerAddrs)
			stateDir, err := cmd.Flags().GetString("state-dir")
			if err != nil {
				return err
			}

			token, err := cmd.Flags().GetString("token")
			if err != nil {
				return err
			}

			managers := agent.NewManagers(managerAddrs...)

			managerAddr, err := managers.Select()
			if err != nil {
				return err
			}

			securityConfig, err := ca.LoadOrCreateAgentSecurityConfig(stateDir, token, managerAddr)
			if err != nil {
				return err
			}

			client, err := engineapi.NewClient(engineAddr, "", nil, nil)
			if err != nil {
				return err
			}

			executor := container.NewExecutor(client)

			ag, err := agent.New(&agent.Config{
				ID:             id,
				Hostname:       hostname,
				Managers:       managers,
				Executor:       executor,
				SecurityConfig: securityConfig,
			})
			if err != nil {
				log.G(ctx).Fatalln(err)
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			if err := ag.Start(ctx); err != nil {
				return err
			}

			// TODO(stevvooe): Register signal to gracefully shutdown agent.

			return ag.Err()
		},
	}
)

func init() {
	agentCmd.Flags().String("id", "", "Specifies the identity of the node")
	agentCmd.Flags().String("engine-addr", "unix:///var/run/docker.sock", "Address of engine instance of agent.")
	agentCmd.Flags().String("hostname", "", "Override reported agent hostname")
	agentCmd.Flags().StringSliceP("manager", "m", []string{"localhost:4242"}, "Specify one or more manager addresses")
}
