package main

import (
	"fmt"
	"github.com/T-TRz879/scpw"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strings"
)

const (
	cliName        = "scpw"
	cliDescription = "Simplify scp operations"
)

func main() {
	//if err := agent.Listen(agent.Options{
	//	ShutdownCleanup: true, // automatically closes on os.Interrupt
	//}); err != nil {
	//	log.Fatal(err)
	//}
	cli.VersionFlag = &cli.BoolFlag{
		Name: "version", Aliases: []string{"V"},
		Usage: "print version only",
	}
	app := &cli.App{
		Name:  cliName,
		Usage: cliDescription,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "keep-time",
				Usage: "keep file or dir atime and mtime",
				Value: true,
			},
		},
		Action:               Run,
		HideHelpCommand:      true,
		EnableBashCompletion: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func Run(ctx *cli.Context) error {
	nodes, err := scpw.LoadConfig()
	if err != nil {
		return err
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "🎈 {{ .Name | cyan }} ({{ .Host | red }} - {{ .Typ | green }})",
		Inactive: "  {{ .Name | cyan }} ({{ .Host | red }} - {{ .Typ | green }})",
		Selected: " {{ .Name | red | cyan }} ({{ .Host | red }} - {{ .Typ | green }})",
		Details: `
--------- SCPW Config ----------
{{ "Name:" | faint }}	{{ .Name }}
{{ "Address:" | faint }}	{{ .Host }}{{":"}}{{ .Port }}
{{ "User:" | faint }}	{{ .User }}
{{ "Type:" | faint }}   {{ .Typ }}
{{ range $k, $v := .LRMap }} 
{{ "Local:" | faint }} {{ $v.Local }}  {{ "Remote:" | faint }} {{ $v.Remote -}} 
{{ end }}
`,
	}

	searcher := func(input string, index int) bool {
		pepper := nodes[index]
		name := strings.Replace(strings.ToLower(pepper.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Select SCPW Config",
		Items:     nodes,
		Templates: templates,
		Size:      6,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return err
	}
	return initScpCli(ctx, nodes[i])
}

func initScpCli(ctx *cli.Context, node *scpw.Node) error {
	ssh, err := scpw.NewSSH(node)
	if err != nil {
		return err
	}
	keepTime := ctx.Bool("keep-time")
	scpwCli := scpw.NewSCP(ssh, keepTime)
	errCh := make(chan error, len(node.LRMap))
	for _, lr := range node.LRMap {
		go func(lr scpw.LRMap) {
			local, remote := lr.Local, lr.Remote
			fmt.Printf("local:[%s] remote:[%s]\n", local, remote)
			errCh <- scpwCli.SwitchScpwFunc(ctx.Context, local, remote, node.Typ)
		}(lr)
	}
	for i := 0; i < len(node.LRMap); i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}
