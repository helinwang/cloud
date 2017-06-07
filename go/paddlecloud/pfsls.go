package paddlecloud

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	pfsmod "github.com/PaddlePaddle/cloud/go/filemanager/pfsmodules"
	log "github.com/golang/glog"
	"github.com/google/subcommands"
)

// LsCommand represents ls command.
type LsCommand struct {
	cmd pfsmod.LsCmd
}

// Name returns LsCommand's name.
func (*LsCommand) Name() string { return "ls" }

// Synopsis returns Synopsis of LsCommand.
func (*LsCommand) Synopsis() string { return "List files on PaddlePaddle Cloud" }

// Usage returns usage of LsCommand.
func (*LsCommand) Usage() string {
	return `ls [-r] <pfspath>:
	List files on PaddlePaddleCloud
	Options:
`
}

// SetFlags sets LsCommand's parameters.
func (p *LsCommand) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.cmd.R, "r", false, "list files recursively")
}

// getFormatPrint gets max width of filesize and return format string to print.
func getFormatString(result []pfsmod.LsResult) string {
	max := 0
	for _, t := range result {
		str := fmt.Sprintf("%d", t.Size)

		if len(str) > max {
			max = len(str)
		}
	}

	return fmt.Sprintf("%%s %%s %%%dd %%s\n", max)
}

func formatPrint(result []pfsmod.LsResult) {
	formatStr := getFormatString(result)

	for _, t := range result {
		timeStr := time.Unix(0, t.ModTime).Format("2006-01-02 15:04:05")

		if t.IsDir {
			fmt.Printf(formatStr, timeStr, "d", t.Size, t.Path)
		} else {
			fmt.Printf(formatStr, timeStr, "f", t.Size, t.Path)
		}
	}

	fmt.Printf("\n")
}

// RemoteLs gets LsCmd result from cloud.
func RemoteLs(cmd *pfsmod.LsCmd) ([]pfsmod.LsResult, error) {
	t := fmt.Sprintf("%s/api/v1/files", config.ActiveConfig.Endpoint)
	body, err := GetCall(t, cmd.ToURLParam())
	if err != nil {
		return nil, err
	}

	type lsResponse struct {
		Err     string            `json:"err"`
		Results []pfsmod.LsResult `json:"results"`
	}

	resp := lsResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return resp.Results, err
	}

	if len(resp.Err) == 0 {
		return resp.Results, nil
	}

	return resp.Results, errors.New(resp.Err)
}

func remoteLs(cmd *pfsmod.LsCmd) error {
	for _, arg := range cmd.Args {
		subcmd := pfsmod.NewLsCmd(
			cmd.R,
			arg,
		)
		result, err := RemoteLs(subcmd)

		fmt.Printf("%s :\n", arg)
		if err != nil {
			fmt.Printf("  error:%s\n\n", err.Error())
			return err
		}

		formatPrint(result)
	}
	return nil
}

// Execute runs a LsCommand.
func (p *LsCommand) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() < 1 {
		f.Usage()
		return subcommands.ExitFailure
	}

	cmd, err := pfsmod.NewLsCmdFromFlag(f)
	if err != nil {
		return subcommands.ExitFailure
	}
	log.V(1).Infof("%#v\n", cmd)

	if err := remoteLs(cmd); err != nil {
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}