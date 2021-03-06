package main

import (
	"fmt"
	"log"
	"os"

	"github.com/AlexAkulov/clickhouse-backup/config"
	"github.com/AlexAkulov/clickhouse-backup/pkg/backup"
	"github.com/AlexAkulov/clickhouse-backup/pkg/server"

	"github.com/urfave/cli"
)

const (
	defaultConfigPath = "/etc/clickhouse-backup/config.yml"
)

var (
	version   = "unknown"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func main() {
	log.SetOutput(os.Stdout)
	cliapp := cli.NewApp()
	cliapp.Name = "clickhouse-backup"
	cliapp.Usage = "Tool for easy backup of ClickHouse with cloud support"
	cliapp.UsageText = "clickhouse-backup <command> [-t, --tables=<db>.<table>] <backup_name>"
	cliapp.Description = "Run as 'root' or 'clickhouse' user"
	cliapp.Version = version

	cliapp.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  defaultConfigPath,
			Usage:  "Config `FILE` name.",
			EnvVar: "CLICKHOUSE_BACKUP_CONFIG",
		},
	}
	cliapp.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Printf("Error. Unknown command: '%s'\n\n", command)
		cli.ShowAppHelpAndExit(c, 1)
	}

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println("Version:\t", c.App.Version)
		fmt.Println("Git Commit:\t", gitCommit)
		fmt.Println("Build Date:\t", buildDate)
	}

	cliapp.Commands = []cli.Command{
		{
			Name:      "tables",
			Usage:     "Print list of tables",
			UsageText: "clickhouse-backup tables",
			Action: func(c *cli.Context) error {
				return backup.PrintTables(*getConfig(c))
			},
			Flags: cliapp.Flags,
		},
		{
			Name:        "create",
			Usage:       "Create new backup",
			UsageText:   "clickhouse-backup create [-t, --tables=<db>.<table>] <backup_name>",
			Description: "Create new backup",
			Action: func(c *cli.Context) error {
				return backup.CreateBackup(*getConfig(c), c.Args().First(), c.String("t"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
			),
		},
		{
			Name:      "upload",
			Usage:     "Upload backup to remote storage",
			UsageText: "clickhouse-backup upload [--diff-from=<backup_name>] <backup_name>",
			Action: func(c *cli.Context) error {
				return backup.Upload(*getConfig(c), c.Args().First(), c.String("diff-from"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "diff-from",
					Hidden: false,
				},
			),
		},
		{
			Name:        "create_remote",
			Usage:       "Create and upload",
			UsageText:   "clickhouse-backup create_remote [-t, --tables=<db>.<table>] [--diff-from=<backup_name>] [--delete] <backup_name>",
			Description: "Create and upload",
			Action: func(c *cli.Context) error {
				return backup.CreateToRemote(*getConfig(c), c.Args().First(), c.String("t"), c.String("diff-from"), true)
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
				cli.StringFlag{
					Name:   "diff-from",
					Hidden: false,
				},
			),
		},
		{
			Name:      "list",
			Usage:     "Print list of backups",
			UsageText: "clickhouse-backup list [all|local|remote] [latest|penult]",
			Action: func(c *cli.Context) error {
				config := getConfig(c)
				switch c.Args().Get(0) {
				case "local":
					return backup.PrintLocalBackups(*config, c.Args().Get(1))
				case "remote":
					return backup.PrintRemoteBackups(*config, c.Args().Get(1))
				case "all", "":
					fmt.Println("Local backups:")
					if err := backup.PrintLocalBackups(*config, c.Args().Get(1)); err != nil {
						return err
					}
					if config.General.RemoteStorage != "none" {
						fmt.Println("Remote backups:")
						if err := backup.PrintRemoteBackups(*config, c.Args().Get(1)); err != nil {
							return err
						}
					}
				default:
					log.Printf("Unknown command '%s'\n", c.Args().Get(0))
					cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
				}
				return nil
			},
			Flags: cliapp.Flags,
		},
		{
			Name:      "download",
			Usage:     "Download backup from remote storage",
			UsageText: "clickhouse-backup download <backup_name>",
			Action: func(c *cli.Context) error {
				return backup.Download(*getConfig(c), c.Args().First())
			},
			Flags: cliapp.Flags,
		},
		{
			Name:      "restore",
			Usage:     "Create schema and restore data from backup",
			UsageText: "clickhouse-backup restore [--schema] [--data] [-t, --tables=<db>.<table>] <backup_name>",
			Action: func(c *cli.Context) error {
				return backup.Restore(*getConfig(c), c.Args().First(), c.String("t"), c.Bool("s"), c.Bool("d"), c.Bool("rm"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
				cli.BoolFlag{
					Name:   "schema, s",
					Hidden: false,
					Usage:  "Restore schema only",
				},
				cli.BoolFlag{
					Name:   "data, d",
					Hidden: false,
					Usage:  "Restore data only",
				},
				cli.BoolFlag{
					Name:   "rm, drop",
					Hidden: false,
					Usage:  "Drop table before restore",
				},
			),
		},
		{
			Name:      "restore_remote",
			Usage:     "Download and restore",
			UsageText: "clickhouse-backup restore_remote [--schema] [--data] [-t, --tables=<db>.<table>] <backup_name>",
			Action: func(c *cli.Context) error {
				return backup.RestoreFromRemote(*getConfig(c), c.Args().First(), c.String("t"), c.Bool("s"), c.Bool("d"), c.Bool("rm"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
				cli.BoolFlag{
					Name:   "schema, s",
					Hidden: false,
					Usage:  "Restore schema only",
				},
				cli.BoolFlag{
					Name:   "data, d",
					Hidden: false,
					Usage:  "Restore data only",
				},
				cli.BoolFlag{
					Name:   "rm, drop",
					Hidden: false,
					Usage:  "Drop table before restore",
				},
			),
		},
		{
			Name:      "flashback",
			Usage:     "flashback to backup",
			UsageText: "clickhouse-backup flashback [-t, --tables=<db>.<table>] <backup_name>",
			Action: func(c *cli.Context) error {
				return backup.Flashback(*getConfig(c), c.Args().First(), c.String("t"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
			),
		},
		{
			Name:      "delete",
			Usage:     "Delete specific backup",
			UsageText: "clickhouse-backup delete <local|remote> <backup_name>",
			Action: func(c *cli.Context) error {
				cfg := getConfig(c)
				if c.Args().Get(1) == "" {
					log.Println("Backup name must be defined")
					cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
				}
				switch c.Args().Get(0) {
				case "local":
					return backup.RemoveBackupLocal(*cfg, c.Args().Get(1))
				case "remote":
					return backup.RemoveBackupRemote(*cfg, c.Args().Get(1))
				default:
					log.Printf("Unknown command '%s'\n", c.Args().Get(0))
					cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
				}
				return nil
			},
			Flags: cliapp.Flags,
		},
		{
			Name:  "default-config",
			Usage: "Print default config",
			Action: func(*cli.Context) {
				config.PrintDefaultConfig()
			},
			Flags: cliapp.Flags,
		},
		{
			Name:        "freeze",
			Usage:       "Freeze tables",
			UsageText:   "clickhouse-backup freeze [-t, --tables=<db>.<table>] <backup_name>",
			Description: "Freeze tables",
			Action: func(c *cli.Context) error {
				return backup.Freeze(*getConfig(c), c.String("t"))
			},
			Flags: append(cliapp.Flags,
				cli.StringFlag{
					Name:   "table, tables, t",
					Hidden: false,
				},
			),
		},
		{
			Name:  "clean",
			Usage: "Remove data in 'shadow' folder",
			Action: func(c *cli.Context) error {
				return backup.Clean(*getConfig(c))
			},
			Flags: cliapp.Flags,
		},
		{
			Name:  "server",
			Usage: "Run API server",
			Action: func(c *cli.Context) error {
				return server.Server(cliapp, getConfig(c), getConfigPath(c))
			},
			Flags: cliapp.Flags,
		},
	}
	if err := cliapp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func getConfig(ctx *cli.Context) *config.Config {
	configPath := getConfigPath(ctx)
	config, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func getConfigPath(ctx *cli.Context) string {
	if ctx.String("config") != defaultConfigPath {
		return ctx.String("config")
	}
	if ctx.GlobalString("config") != defaultConfigPath {
		return ctx.GlobalString("config")
	}
	if os.Getenv("CLICKHOUSE_BACKUP_CONFIG") != "" {
		return os.Getenv("CLICKHOUSE_BACKUP_CONFIG")
	}
	return defaultConfigPath
}
