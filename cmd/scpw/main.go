package main

import (
	"github.com/T-TRz879/scpw"
	"github.com/google/gops/agent"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	cliName        = "scpw"
	cliDescription = "Simplify scp operations"
	threads        = 10
)

func main() {
	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true, // automatically closes on os.Interrupt
	}); err != nil {
		log.Fatal(err)
	}
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
	p := scpw.NewProgress()
	keepTime := ctx.Bool("keep-time")
	wg := sync.WaitGroup{}
	todo := make(chan scpw.LRMap, 5)
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			ssh, err := scpw.NewSSH(node)
			if err != nil {
				panic(err)
			}
			scpwCli := scpw.NewSCP(ssh, keepTime)
			defer func() {
				ssh.Close()
				wg.Done()
			}()
			for lr := range todo {
				local, remote := lr.Local, lr.Remote
				scpwCtx := scpw.Context{Ctx: ctx.Context, Bar: p.NewInfiniteByesBar(local)}
				err = scpwCli.SwitchScpwFunc(scpwCtx, local, remote, node.Typ)
				scpwCtx.Bar.SetTotal(-1, true)
				if err != nil {
					panic(err)
				}
			}
		}()
	}
	for _, lr := range node.LRMap {
		todo <- lr
	}
	close(todo)
	wg.Wait()
	p.Wait()
	return nil
}
