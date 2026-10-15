# ws-graph
Generate a graph of terraform workspaces with remote-state dependencies

```
Usage: ws-graph [args] hostname orgname projectname
  -c float
    	Connectivity probability (default 0.3)
  -d string
    	Output directory (default "graph-workspaces")
  -n int
    	Number of workspaces (default 10)
  -w string
    	TFC workspace name prefix (default "gws")
```

The program generates terraform definitions for multiple (`-n`) workspaces.
The definitions are in a sub-directory (`-d`) of the current directory.

The workspaces will all be on the same TFC host, and in the same organization and project.
These are the required `hostname orgname projectname` arguments.
Note that the organization and project must be created before running `terraform apply` on the workspaces.

Workspace names are the workspace prefix (`-w`) followed by an index number, thus `gws01`, `gws02`, ....
The sub-directory (`-d`) has one directory per workspace, named with the workspace name.
Each directory has a `main.tf` that declared the workspace in the given host, organization, and project.
The `main.tf` also has a random ID resource and an output.

The workspaces have dependencies on one another, implemented by declaring `terraform_remote_state` data resources
for the state of lower-numbered workspaces.
The remote-state outputs are used in the `keepers` of the workspace random resource.
The connectivity (`-c`) is used to calculate the probability that a workspace has a dependency on a lower-numbered workspace.
The connectivity is not the probability itself, but is scaled in such a way that workspaces
generally have between 0 and 5 dependencies (though some might have more).

Since the workspaces only depend on lower-numbered workspaces,
they should be created or updated in numerical order and destroyed in reverse order.
The program writes three utility files, `init.sh`, `apply.sh`, and `destroy.sh`,
to perform the terraform init, apply, and destroy commands in the correct order.

In production (`app.terraform.io`) workspaces do not share remote-state by default.
The `init.sh` script will enable remote-state sharing on the workspaces,
if the environment variable `TOKEN` is set to a TF API token for the host.

The `workspace-destroy.sh` script force-deletes all the workspaces.
It should be used only after destroying all resources.
The script requires environment variable `TOKEN` is set.
