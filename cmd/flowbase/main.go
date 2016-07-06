package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"strings"
)

func main() {
	errLog := log.New(os.Stderr, "", 0)
	app := cli.NewApp()
	app.Name = "FlowBase helper tool"
	app.Usage = "A helper tool to ease working with FlowBase programs"
	app.Version = "0.0.1"
	app.Commands = []cli.Command{
		{
			Name:    "new-component",
			Aliases: []string{"nc"},
			Usage:   "Create a new component, with the (CamelCased) name taken from the first argument.\nThe component is saved in a separate file named as the component, with all the boiler plate code and an empty Run() method to fill in with your code.",
			Action: func(c *cli.Context) error {
				componentTemplate := `// Component for use with the FlowBase FBP micro-framework
// For more information about FlowBase, see: http://flowbase.org
package changethis

import "github.com/flowbase/flowbase"

type %s struct {
	In  chan string
	Out chan string
}

func New%s() *%s {
	return &%s{
		In: make(chan string, flowbase.BUFSIZE),
		Out: make(chan string, flowbase.BUFSIZE),
	}
}

func (p *%s) Run() {
	defer close(p.Out)
	for line := range p.In {
		p.Out <- line
	}
}
`
				componentName := c.Args().First()
				if componentName == "" {
					componentName = "ChangeThis"
					fmt.Printf("No component name specified, so using the default '%s' ...\n", componentName)
				}

				fileName := strings.ToLower(componentName) + ".go"
				f, err := os.Create(fileName)
				if err != nil {
					errLog.Println("Could not create file:", fileName)
					os.Exit(1)
				}
				defer f.Close()

				componentCode := fmt.Sprintf(componentTemplate, componentName, componentName, componentName, componentName, componentName)

				_, err = f.Write([]byte(componentCode))
				if err != nil {
					errLog.Println("Could not write to file:", fileName)
					os.Exit(1)
				}

				fmt.Printf("Successfully wrote new component %s to: %s\n", componentName, fileName)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
