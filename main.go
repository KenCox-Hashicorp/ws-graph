package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
)

type (
	args struct {
		hostname     string
		organization string
		project      string
		workspace    string
		outdir       string
		connectivity float64
		count        int
	}
)

func main() {
	data := parseArgs(os.Args)

	if err := os.MkdirAll(data.outdir, 0777); err != nil {
		fmt.Printf("Error creating %s: %v\n", data.outdir, err)
	}

	for i := 1; i <= data.count; i++ {
		if err := data.generate(i); err != nil {
			fmt.Printf("Error generating graph %d: %v\n", i, err)
			return
		}
	}
}

func parseArgs(cmdline []string) *args {
	a := &args{}
	fs := flag.NewFlagSet("tfgen", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [args] hostname orgname projectname\n", cmdline[0])
		fs.PrintDefaults()
	}
	fs.StringVar(&a.workspace, "w", "gws", "TFC workspace name prefix")
	fs.StringVar(&a.outdir, "d", "graph-workspaces", "Output directory")
	fs.Float64Var(&a.connectivity, "c", 0.3, "Connectivity probability")
	fs.IntVar(&a.count, "n", 10, "Number of workspaces")

	fs.Parse(cmdline[1:])

	if fs.NArg() != 3 {
		fs.Usage()
		fmt.Fprintln(fs.Output(), "\nRequired arguments hostname orgname projectname not found")
		os.Exit(2)
	}
	a.hostname = fs.Arg(0)
	a.organization = fs.Arg(1)
	a.project = fs.Arg(2)
	if a.connectivity < 0.0 || a.connectivity > 1.0 {
		fs.Usage()
		fmt.Fprintln(fs.Output(), "\nConnectivity must be between 0 and 1")
		os.Exit(2)
	}
	return a
}

func (a *args) generate(index int) error {
	workspaceName := a.graphname(index)
	dependencies := a.dependencies(index)
	dirName := path.Join(a.outdir, workspaceName)
	if err := os.MkdirAll(dirName, 0777); err != nil {
		return fmt.Errorf("Error creating directory %s: %v\n", dirName, err)
	}
	fileName := path.Join(dirName, "main.tf")
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("Error creating file %s: %v\n", fileName, err)
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, terraformBlockFmt,
		a.hostname,
		a.organization,
		a.project,
		workspaceName); err != nil {
		return fmt.Errorf("Error writing file (terraform) %s: %v\n", fileName, err)
	}

	for _, dep := range dependencies {
		depName := a.graphname(dep)
		if _, err := fmt.Fprintf(file, remoteDependencyFmt,
			depName,
			a.hostname,
			a.organization,
			depName); err != nil {
			return fmt.Errorf("Error writing file (dependencies) %s: %v\n", fileName, err)
		}
	}

	if _, err := fmt.Fprintf(file, randomIdFmt, workspaceName); err != nil {
		return fmt.Errorf("Error writing file (randomid) %s: %v\n", fileName, err)
	}
	for ix, dep := range dependencies {
		depName := a.graphname(dep)
		if _, err := fmt.Fprintf(file, keeperFmt,
			ix,
			depName); err != nil {
			return fmt.Errorf("Error writing file (keeper) %s: %v\n", fileName, err)
		}
	}
	if _, err := fmt.Fprintf(file, outputFmt, workspaceName); err != nil {
		return fmt.Errorf("Error writing file (output) %s: %v\n", fileName, err)
	}

	fmt.Printf("Created %s: %v\n", fileName, dependencies)
	return nil
}

func (a *args) graphname(index int) string {
	if a.count < 100 {
		return fmt.Sprintf("%s%02d", a.workspace, index)
	}
	if a.count < 1000 {
		return fmt.Sprintf("%s%03d", a.workspace, index)
	}
	return fmt.Sprintf("%s%04d", a.workspace, index)
}

func (a *args) dependencies(index int) []int {
	result := []int{}
	prob := a.connectivity / ((10.0 + float64(index)) / 10.0)
	for v := 1; v < index; v++ {
		if rand.Float64() < prob {
			result = append(result, v)
		}
	}
	return result
}

const (
	terraformBlockFmt = `
terraform {
  cloud {
    hostname = "%s"
    organization = "%s"

    workspaces {
      project = "%s"
      name = "%s"
    }
  }

  required_providers {
    random = {
      source = "hashicorp/random"
      version = "3.7.2"
    }
  }
}
`

	remoteDependencyFmt = `
data "terraform_remote_state" "remote_%s" {
  backend = "remote"
  config = {
    hostname = "%s"
    organization = "%s"
    workspaces = {
      name = "%s"
    }
  }
}
`
	randomIdFmt = `
resource "random_id" "r_%s" {
  byte_length = 8
  keepers = {
`
	keeperFmt = `
	id%d = data.terraform_remote_state.remote_%s.outputs.theoutput
`

	outputFmt = `
  }
}

output "theoutput" {
  value    = random_id.r_%s.hex
}
`
)
