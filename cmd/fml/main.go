package main

import (
	"fmt"
	"github.com/mitlibraries/fml"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Usage = "The fml command provides a set of utilities for working with MARC 21 files"

	app.Commands = []cli.Command{
		{
			Name:      "pick",
			Usage:     "Pull a single MARC record from the data by control number",
			ArgsUsage: "[controlnum] [file]",
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				file, err := os.Open(c.Args().Get(1))
				if err != nil {
					return err
				}
				m := fml.NewMarcIterator(file)
				for m.Next() {
					record, _ := m.Value()
					if record.ControlNum() == id {
						os.Stdout.Write(record.Data)
						break
					}
				}
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
